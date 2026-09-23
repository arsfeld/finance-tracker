package ledger

import (
	"sort"
	"time"

	"finance_tracker/internal/models"
)

// Report is the card spending picture the analysis prompt is built from.
type Report struct {
	Periods []PeriodSpend
	Charges []models.DBTransaction // itemized card charges in range, excluded categories removed, newest first
}

// Build groups accounts into cards and measures each card's spending.
//
// Only included credit card accounts are analyzed. Every non-card account,
// included or not, is scanned for payments toward the cards, because that is
// where a payment shows up when the card's own feed drops it. Charges in
// excluded categories stay out of the itemized figures, but the balance still
// counts them. Card fees and interest are real costs, and the balance is the
// authority.
func Build(periods []models.BillingPeriod, accounts []models.DBAccount, txns []models.DBTransaction,
	snapshots map[string][]models.BalanceSnapshot, patterns map[string][]string, excluded []string) Report {
	isExcluded := make(map[string]bool, len(excluded))
	for _, c := range excluded {
		if c != "" {
			isExcluded[c] = true
		}
	}

	cardOf := make(map[string]string)     // included card account ID -> card key
	isCard := make(map[string]bool)       // any card account, included or not
	coverageEnd := make(map[string]int64) // card key -> latest balance_date
	for _, a := range accounts {
		if !a.IsCreditCard || a.CardKey == "" {
			continue
		}
		isCard[a.ID] = true
		coverageEnd[a.CardKey] = max(coverageEnd[a.CardKey], a.BalanceDate)
		if a.IsIncluded {
			cardOf[a.ID] = a.CardKey
		}
	}

	byCard := make(map[string][]models.DBTransaction)
	var paying []models.DBTransaction
	for _, t := range txns {
		if key, ok := cardOf[t.AccountID]; ok {
			byCard[key] = append(byCard[key], t)
		} else if !isCard[t.AccountID] {
			paying = append(paying, t)
		}
	}

	keys := make([]string, 0, len(coverageEnd))
	for _, key := range cardOf {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	keys = dedupSorted(keys)

	var report Report
	if len(periods) == 0 {
		return report
	}
	rangeFrom, _ := PeriodBounds(periods[0])
	_, rangeTo := PeriodBounds(periods[len(periods)-1])

	var perCard [][]PeriodSpend
	for _, key := range keys {
		cardTxns := byCard[key]
		var charges []models.DBTransaction
		for _, t := range cardTxns {
			if t.Amount < 0 && !isExcluded[t.Category] {
				charges = append(charges, t)
				if d := EffectiveDate(t); d >= rangeFrom && d < rangeTo {
					report.Charges = append(report.Charges, t)
				}
			}
		}
		payments := DetectPayments(cardTxns, paying, patterns[key])
		perCard = append(perCard, CardPeriods(periods, snapshots[key], payments, coverageEnd[key], charges))
	}

	report.Periods = Combine(perCard)
	if report.Periods == nil {
		for _, p := range periods {
			report.Periods = append(report.Periods, PeriodSpend{Period: p, Source: SourceItemizedOnly})
		}
	}
	sort.SliceStable(report.Charges, func(i, j int) bool {
		return EffectiveDate(report.Charges[i]) > EffectiveDate(report.Charges[j])
	})
	return report
}

func dedupSorted(s []string) []string {
	out := s[:0]
	for i, v := range s {
		if i == 0 || v != s[i-1] {
			out = append(out, v)
		}
	}
	return out
}

// FetchFrom returns how far back transactions must be loaded to measure the
// range starting at start. The interval straddling start begins at the last
// snapshot before it, and payments settling it may post up to
// PaymentMatchWindow earlier.
func FetchFrom(snapshots map[string][]models.BalanceSnapshot, start time.Time) time.Time {
	from := start.Unix()
	for _, snaps := range snapshots {
		for i := len(snaps) - 1; i >= 0; i-- {
			if snaps[i].BalanceDate < start.Unix() {
				from = min(from, snaps[i].BalanceDate)
				break
			}
		}
	}
	return time.Unix(from, 0).UTC().Add(-PaymentMatchWindow)
}
