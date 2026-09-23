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
	SourceItemizedOnly Source = "itemized_only"
)

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
// spending. So a negative interval is first netted against the positive
// intervals within PaymentMatchWindow of it, nearest first, before anything is
// clamped.
//
// Whatever a drop cannot net against is itself a payment the feeds missed, and
// is dropped: spending never goes negative. The card is paid in full monthly,
// and the feeds do miss payments.
func Intervals(snaps []models.BalanceSnapshot, payments []Payment) []Interval {
	var out []Interval
	for i := 1; i < len(snaps); i++ {
		s0, s1 := snaps[i-1], snaps[i]
		spend := -s1.Balance - -s0.Balance
		for _, p := range payments {
			if p.At > s0.BalanceDate && p.At <= s1.BalanceDate {
				spend += p.Amount
			}
		}
		out = append(out, Interval{From: s0.BalanceDate, To: s1.BalanceDate, Spend: spend})
	}

	window := int64(PaymentMatchWindow / time.Second)
	// absorb moves as much of the drop at i as neighbour j can take. Only
	// positive neighbours give anything, so a drop never feeds another drop.
	absorb := func(i, j int) {
		if take := min(-out[i].Spend, out[j].Spend); take > 0 {
			out[j].Spend -= take
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
	Total        float64 // headline: balance spending plus itemized charges outside the covered window
	BalanceSpend float64
	CoveredDays  float64
	DailyBurn    float64
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

// CardPeriods measures one card's spending in each billing period. Where the
// balance history covers the period, the balance decides the total, and an
// interval crossing a period boundary is split in proportion to time. Where it
// doesn't, the period falls back to its itemized charges and is labeled as
// such. coverageEnd is the card's latest balance_date: an unchanged balance is
// not stored as a snapshot but still counts as coverage.
func CardPeriods(periods []models.BillingPeriod, snaps []models.BalanceSnapshot, payments []Payment,
	coverageEnd int64, charges []models.DBTransaction) []PeriodSpend {
	intervals := Intervals(snaps, payments)
	var covFrom, covTo int64
	if len(snaps) >= 2 {
		covFrom = snaps[0].BalanceDate
		covTo = max(coverageEnd, snaps[len(snaps)-1].BalanceDate)
	}

	out := make([]PeriodSpend, len(periods))
	for i, p := range periods {
		from, to := PeriodBounds(p)
		ps := PeriodSpend{Period: p}

		for _, iv := range intervals {
			if ov := overlap(iv.From, iv.To, from, to); ov > 0 {
				ps.BalanceSpend += iv.Spend * float64(ov) / float64(iv.To-iv.From)
			}
		}

		coveredFrom, coveredTo := max(from, covFrom), min(to, covTo)
		covered := len(snaps) >= 2 && coveredTo > coveredFrom
		if covered {
			ps.CoveredDays = float64(coveredTo-coveredFrom) / 86400
		}

		var inside float64
		for _, t := range charges {
			d := EffectiveDate(t)
			if t.Amount >= 0 || d < from || d >= to {
				continue
			}
			ps.Itemized += -t.Amount
			if covered && d >= coveredFrom && d < coveredTo {
				inside += -t.Amount
			}
		}

		if covered {
			ps.Source = SourceBalance
			ps.Total = ps.BalanceSpend + (ps.Itemized - inside)
			ps.DailyBurn = ps.BalanceSpend / ps.CoveredDays
			if gap := ps.BalanceSpend - inside; gap > NotItemizedMin {
				ps.NotItemized = gap
			}
		} else {
			ps.Source = SourceItemizedOnly
			ps.Total = ps.Itemized
			if days := float64(to-from) / 86400; days > 0 {
				ps.DailyBurn = ps.Itemized / days
			}
		}
		out[i] = ps
	}
	return out
}

// Combine adds up several cards' periods. Totals and burn rates add; coverage
// is the widest card's; a period has balance data if any card does.
func Combine(perCard [][]PeriodSpend) []PeriodSpend {
	if len(perCard) == 0 {
		return nil
	}
	out := make([]PeriodSpend, len(perCard[0]))
	for i := range out {
		out[i] = PeriodSpend{Period: perCard[0][i].Period, Source: SourceItemizedOnly}
		for _, card := range perCard {
			c := card[i]
			out[i].Total += c.Total
			out[i].BalanceSpend += c.BalanceSpend
			out[i].Itemized += c.Itemized
			out[i].NotItemized += c.NotItemized
			out[i].DailyBurn += c.DailyBurn
			out[i].CoveredDays = max(out[i].CoveredDays, c.CoveredDays)
			if c.Source == SourceBalance {
				out[i].Source = SourceBalance
			}
		}
	}
	return out
}
