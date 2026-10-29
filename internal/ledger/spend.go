package ledger

import (
	"time"

	"finance_tracker/internal/models"
)

// NotItemizedMin is the smallest gap between balance spending and itemized
// charges worth reporting. Below it the gap is the normal lag between a charge
// hitting the balance and posting.
const NotItemizedMin = 50.0

// Source says where a period's headline figure came from.
type Source string

const (
	SourceBalance      Source = "balance"
	SourceTransactions Source = "transactions"
	SourceMixed        Source = "mixed"
)

// MaxTrustedGap is the longest stretch between two SimpleFin readings whose
// balance change is trusted as spending. Sync runs every two days, so a missed
// run or two still fits. A longer gap means the connection was failing, and
// the reading at its end may be a frozen balance: TD's Sep 23 reading predated
// $8,450 of payments, and booking them as spending zeroed the next cycle.
const MaxTrustedGap = 7 * 24 * time.Hour

// ImportSnapshotSource marks a balance read from an imported statement. It is
// kept as a record but never measured: one reading per import spans weeks,
// and the statement's own transactions are complete.
const ImportSnapshotSource = "import"

// Interval is the spending between two consecutive balance snapshots.
type Interval struct {
	From, To int64
	Spend    float64
}

// Intervals turns a card's snapshots into spending. Between two readings,
// spending is the growth in debt plus whatever was paid in between.
//
// A payment seen from the paying side is dated by the bank debit, but the card
// balance drops one to three days later. When a snapshot falls between the
// two, one interval gets the payment added without the drop and a neighbour
// gets the drop without the payment. Clamping each interval on its own would
// keep the inflated one and zero the other, counting the whole payment as
// spending. So a negative interval is first netted against the intervals
// within PaymentMatchWindow of it, nearest first, before anything is clamped.
//
// A neighbour gives up at most the payments that were added to it, never its
// debt growth. Netting exists only to move a payment back to the interval
// whose drop it explains; the growth is real charges. A drop with no detected
// payment nearby is a payment the feeds missed, and payments never change
// spending, so the rest of the drop is discarded: spending never goes
// negative. The card is paid in full monthly, and the feeds do miss payments.
func Intervals(snaps []models.BalanceSnapshot, payments []Payment) []Interval {
	var out []Interval
	var paid []float64 // payments added to each interval that netting may still undo
	for i := 1; i < len(snaps); i++ {
		s0, s1 := snaps[i-1], snaps[i]
		var inflation float64
		for _, p := range payments {
			if p.At > s0.BalanceDate && p.At <= s1.BalanceDate {
				inflation += p.Amount
			}
		}
		out = append(out, Interval{From: s0.BalanceDate, To: s1.BalanceDate, Spend: -s1.Balance - -s0.Balance + inflation})
		paid = append(paid, inflation)
	}

	window := int64(PaymentMatchWindow / time.Second)
	// absorb moves as much of the drop at i as neighbour j can give: no more
	// than the payments still attributed to it, and never below zero, so a drop
	// neither eats real charges nor feeds another drop.
	absorb := func(i, j int) {
		if take := min(-out[i].Spend, out[j].Spend, paid[j]); take > 0 {
			out[j].Spend -= take
			paid[j] -= take
			out[i].Spend += take
		}
	}
	// Intervals are contiguous and in order, so each scan can stop at the first
	// neighbour outside the window. Drops are handled oldest first; a neighbour
	// shared by two drops gives the second only what the first left.
	for i := range out {
		if out[i].Spend >= 0 {
			continue
		}
		for j := i - 1; j >= 0 && out[i].Spend < 0 && out[j].To >= out[i].From-window; j-- {
			absorb(i, j)
		}
		for j := i + 1; j < len(out) && out[i].Spend < 0 && out[j].From <= out[i].To+window; j++ {
			absorb(i, j)
		}
		out[i].Spend = 0
	}
	return out
}

// PeriodSpend is one billing period's card spending.
type PeriodSpend struct {
	Period       models.BillingPeriod
	Source       Source
	Total        float64 // headline: balance spending plus itemized charges outside the balance-measured days
	BalanceSpend float64
	CoveredDays  float64 // days of the period up to the card's data-through date
	BalanceDays  float64 // of those, days measured by the balance
	DailyBurn    float64 // Total over CoveredDays
	Itemized     float64
	NotItemized  float64 // balance spending the transactions don't account for; 0 when under NotItemizedMin
}

// PeriodBounds returns a period as a half-open [from, to) range. A completed
// period's End is midnight of its last day, so the day itself is added back.
func PeriodBounds(p models.BillingPeriod) (from, to int64) {
	end := p.End
	if p.IsComplete {
		end = end.Add(24 * time.Hour)
	}
	return p.Start.Unix(), end.Unix()
}

// EffectiveDate is when a charge happened, falling back to when it posted.
func EffectiveDate(t models.DBTransaction) int64 {
	if t.TransactedAt != nil {
		return *t.TransactedAt
	}
	return t.Posted
}

func overlap(a0, a1, b0, b1 int64) int64 {
	lo, hi := max(a0, b0), min(a1, b1)
	if hi < lo {
		return 0
	}
	return hi - lo
}

type span struct{ from, to int64 }

func inSpans(spans []span, at int64) bool {
	for _, s := range spans {
		if at >= s.from && at < s.to {
			return true
		}
	}
	return false
}

// CardPeriods measures one card's spending in each billing period. The balance
// decides the total only between SimpleFin readings at most MaxTrustedGap
// apart; there an interval crossing a period boundary is split in proportion
// to time. Every other day, outages and imported stretches included, is
// measured by its itemized charges. dataThrough is the last moment the card
// has data for, from either source. Balance coverage ends at the last reading:
// anything after it may be an outage filled by imports.
func CardPeriods(periods []models.BillingPeriod, snaps []models.BalanceSnapshot, payments []Payment,
	dataThrough int64, charges []models.DBTransaction) []PeriodSpend {
	var readings []models.BalanceSnapshot
	for _, s := range snaps {
		if s.Source != ImportSnapshotSource {
			readings = append(readings, s)
		}
	}

	// Netting runs over the whole series so a payment can still move back to
	// the interval whose drop it explains; trust is decided afterwards.
	gap := int64(MaxTrustedGap / time.Second)
	intervals := Intervals(readings, payments)
	var trusted []Interval
	var spans []span
	for _, iv := range intervals {
		if iv.To-iv.From > gap {
			continue
		}
		trusted = append(trusted, iv)
		if n := len(spans); n > 0 && spans[n-1].to == iv.From {
			spans[n-1].to = iv.To
		} else {
			spans = append(spans, span{iv.From, iv.To})
		}
	}
	if n := len(spans); n > 0 {
		dataThrough = max(dataThrough, spans[n-1].to)
	}

	out := make([]PeriodSpend, len(periods))
	for i, p := range periods {
		from, to := PeriodBounds(p)
		ps := PeriodSpend{Period: p}

		for _, iv := range trusted {
			if ov := overlap(iv.From, iv.To, from, to); ov > 0 {
				ps.BalanceSpend += iv.Spend * float64(ov) / float64(iv.To-iv.From)
			}
		}

		dataEnd := to
		if dataThrough > 0 {
			dataEnd = min(to, dataThrough)
		}
		if dataEnd > from {
			ps.CoveredDays = float64(dataEnd-from) / 86400
		}
		var balanceSecs int64
		for _, s := range spans {
			balanceSecs += overlap(s.from, s.to, from, to)
		}
		ps.BalanceDays = float64(balanceSecs) / 86400

		var inside float64
		for _, t := range charges {
			d := EffectiveDate(t)
			if t.Amount >= 0 || d < from || d >= to {
				continue
			}
			ps.Itemized += -t.Amount
			if inSpans(spans, d) {
				inside += -t.Amount
			}
		}

		ps.Total = ps.BalanceSpend + (ps.Itemized - inside)
		if ps.CoveredDays > 0 {
			ps.DailyBurn = ps.Total / ps.CoveredDays
		}
		switch {
		case balanceSecs == 0:
			ps.Source = SourceTransactions
		case ps.BalanceDays >= ps.CoveredDays-0.5:
			ps.Source = SourceBalance
		default:
			ps.Source = SourceMixed
		}
		if balanceSecs > 0 {
			if gap := ps.BalanceSpend - inside; gap > NotItemizedMin {
				ps.NotItemized = gap
			}
		}
		out[i] = ps
	}
	return out
}

// Combine adds up several cards' periods. Totals and burn rates add; coverage
// is the widest card's. A period is balance or transactions when every card
// agrees, and mixed otherwise.
func Combine(perCard [][]PeriodSpend) []PeriodSpend {
	if len(perCard) == 0 {
		return nil
	}
	out := make([]PeriodSpend, len(perCard[0]))
	for i := range out {
		out[i] = PeriodSpend{Period: perCard[0][i].Period, Source: perCard[0][i].Source}
		for _, card := range perCard {
			c := card[i]
			out[i].Total += c.Total
			out[i].BalanceSpend += c.BalanceSpend
			out[i].Itemized += c.Itemized
			out[i].NotItemized += c.NotItemized
			out[i].DailyBurn += c.DailyBurn
			out[i].CoveredDays = max(out[i].CoveredDays, c.CoveredDays)
			out[i].BalanceDays = max(out[i].BalanceDays, c.BalanceDays)
			if c.Source != out[i].Source {
				out[i].Source = SourceMixed
			}
		}
	}
	return out
}
