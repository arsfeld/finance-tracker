package llm

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"finance_tracker/internal/billing"
	"finance_tracker/internal/ledger"
	"finance_tracker/internal/models"
)

// SystemPrompt is the default LLM system prompt for spending analysis.
const SystemPrompt = `You are an expert financial analyst specializing in personal finance and spending pattern analysis for families.
You are analyzing credit card spending for a Canadian family of 4: 2 adults, 2 daughters (born 2021 and 2024, currently ~4 and ~1 years old).
The family pays the card in full every month, so card spending is what the household spends day to day.
Spending totals and daily burn rates come from card balances and are authoritative, even when individual transactions are missing.
Categories and individual charges describe only the itemized part of that spending.
Be concise, specific, and use the pre-calculated data provided — do not recalculate totals or percentages.`

// PromptInput is everything the analysis prompt is built from.
type PromptInput struct {
	Periods    []ledger.PeriodSpend   // one per billing cycle, oldest first
	Charges    []models.DBTransaction // itemized card charges, excluded categories removed
	Stale      []models.StaleConnection
	Budgets    []models.Budget
	BillingDay int
	Now        time.Time
}

const readingGuide = `How to read these numbers:
- "Total" and "Daily burn" come from the card balance: debt going up is spending, debt going down is a payment. They are authoritative even when transactions are missing.
- "Itemized" is the part of that spending with individual transactions. Categories and top expenses describe only this part.
- "Not itemized" is spending the balance shows but the transaction feed did not deliver. It is real spending with an unknown category — never treat it as savings.
- Card payments and transfers are not expenses and are not shown.
- Periods marked "itemized only" have no balance history; their totals are a lower bound. Do not compare them with balance periods or read a trend across that boundary.
- A balance period marked "balance (X of Y days)" is covered for only part of its days, so its Total is a lower bound. When coverage differs, compare cycles by Daily burn, not Total.
`

const reportInstructions = `### Instructions

Analyze this family's credit card spending. Write a concise report (~250 words) with:

1. **Summary**: The last 3 billing cycles by daily burn (and total where fully covered). What's the trajectory?
2. **Burn Rate**: Is the current cycle's daily burn higher or lower than the last completed cycle? Say so even when the reason can't be itemized.
3. **What's Improving / What Needs Attention**: From the itemized categories, framed as a share of what is itemized.
4. **Top Expenses**: The 10 largest itemized charges:
%s5. **Family-Specific Insights**: Anything relevant to a family with young children.
6. **Actionable Suggestions**: 2-3 concrete things to try next billing cycle.

Notes:
- All calculations are pre-computed. Use them directly.
- Focus the narrative on FOCUS periods.
- Use CAD ($) for all amounts.
- When a period has "Not itemized" spending, say how much could not be broken down instead of guessing what it was.
`

// GeneratePrompt builds the analysis prompt from balance-derived card spending.
func GeneratePrompt(in PromptInput) string {
	// Only charges are itemized spending; a credit reaching this point would be
	// a payment or refund and has no place in the prompt.
	var charges []models.DBTransaction
	for _, t := range in.Charges {
		if t.Amount < 0 {
			charges = append(charges, t)
		}
	}

	var b strings.Builder
	b.WriteString("## Credit Card Spending Analysis — Family of 4\n\n")
	b.WriteString("Household: 2 adults, 2 children (born 2021 and 2024, currently ~4 and ~1 years old).\n\n")
	b.WriteString(readingGuide)
	b.WriteString("\n")
	b.WriteString(buildPeriodTable(in.Periods))
	b.WriteString(buildBurnTrend(in.Periods))
	b.WriteString(buildDataLagWarning(in.Stale, in.Now))
	b.WriteString("\n")
	b.WriteString(buildCoverageLine(in.Periods))
	b.WriteString(buildChargeSpan(charges, currentPeriod(in.Periods)))
	b.WriteString(buildCategoryBreakdown(charges))
	b.WriteString("\n")
	fmt.Fprintf(&b, reportInstructions, formatTopExpenses(charges, 10))
	if len(in.Budgets) > 0 {
		b.WriteString("\n" + buildBudgetSection(in.Budgets, charges, in.BillingDay, currentPeriod(in.Periods)) + "\n")
	}
	b.WriteString("\nCard Charges:\n")
	b.WriteString(formatDBTransactions(charges))
	return b.String()
}

func money(v float64) string { return fmt.Sprintf("$%.2f", v) }

func buildPeriodTable(periods []ledger.PeriodSpend) string {
	var b strings.Builder
	b.WriteString("Billing Periods:\n\n")
	b.WriteString("| Period | Status | Source | Covered days | Total | Daily burn | Change | Itemized | Not itemized |\n")
	b.WriteString("|--------|--------|--------|--------------|-------|------------|--------|----------|--------------|\n")
	for i, p := range periods {
		status := "completed"
		if !p.Period.IsComplete {
			status = "in progress"
		}
		if p.Period.IsFocus {
			status += " [FOCUS]"
		}
		source, covered := "balance", fmt.Sprintf("%.1f", p.CoveredDays)
		if p.Source == ledger.SourceItemizedOnly {
			source, covered = "itemized only", "—"
		} else if days := periodDays(p.Period); p.CoveredDays < days-0.5 {
			// A cycle the balance history only partly covers has a partial
			// Total that otherwise reads as the whole cycle.
			source = fmt.Sprintf("balance (%.1f of %g days)", p.CoveredDays, math.Round(days*10)/10)
		}
		// A cycle in progress has a partial total, so only completed balance
		// cycles are compared; the burn trend covers the current one.
		change := "—"
		if i > 0 {
			prev := periods[i-1]
			if p.Period.IsComplete && p.Source == ledger.SourceBalance && prev.Source == ledger.SourceBalance && prev.Total > 0 {
				change = fmt.Sprintf("%+.1f%%", (p.Total-prev.Total)/prev.Total*100)
			}
		}
		notItemized := "—"
		if p.NotItemized > 0 {
			notItemized = money(p.NotItemized)
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s/day | %s | %s | %s |\n",
			p.Period.Label, status, source, covered, money(p.Total), money(p.DailyBurn), change, money(p.Itemized), notItemized)
	}
	return b.String()
}

// buildBurnTrend compares the current cycle's daily burn with the last
// completed cycle that has balance data. It answers "is spending going up?"
// even when nothing can be itemized.
func buildBurnTrend(periods []ledger.PeriodSpend) string {
	if len(periods) == 0 {
		return ""
	}
	cur := periods[len(periods)-1]
	if cur.Period.IsComplete || cur.Source != ledger.SourceBalance {
		return ""
	}
	for i := len(periods) - 2; i >= 0; i-- {
		prev := periods[i]
		if prev.Period.IsComplete && prev.Source == ledger.SourceBalance && prev.DailyBurn > 0 {
			return fmt.Sprintf("\nCurrent cycle burn: %s/day over %.1f days vs %s/day in %s (%+.1f%%).\n",
				money(cur.DailyBurn), cur.CoveredDays, money(prev.DailyBurn), prev.Period.Label,
				(cur.DailyBurn-prev.DailyBurn)/prev.DailyBurn*100)
		}
	}
	return fmt.Sprintf("\nCurrent cycle burn: %s/day over %.1f days. No completed cycle has balance data yet, so there is no balance-based comparison.\n",
		money(cur.DailyBurn), cur.CoveredDays)
}

func periodDays(p models.BillingPeriod) float64 {
	from, to := ledger.PeriodBounds(p)
	return float64(to-from) / 86400
}

// buildCoverageLine says how much of the balance-measured spending the
// categories explain. Itemized-only periods are left out: there the itemized
// charges are the whole total by construction, which would inflate the share.
func buildCoverageLine(periods []ledger.PeriodSpend) string {
	var total, itemized float64
	for _, p := range periods {
		if p.Source == ledger.SourceBalance {
			total += p.Total
			itemized += p.Itemized
		}
	}
	if total <= 0 {
		return ""
	}
	return fmt.Sprintf("Categories cover %s of %s (%.0f%%) of balance-tracked card spending.\n",
		money(itemized), money(total), math.Min(100, itemized/total*100))
}

// buildChargeSpan dates the itemized charges. A feed can stop while the
// balance keeps refreshing, and then the categories describe the weeks before
// it stopped, not how the family spends now.
func buildChargeSpan(charges []models.DBTransaction, current *ledger.PeriodSpend) string {
	if len(charges) == 0 {
		return ""
	}
	earliest, latest := ledger.EffectiveDate(charges[0]), ledger.EffectiveDate(charges[0])
	for _, t := range charges[1:] {
		d := ledger.EffectiveDate(t)
		earliest, latest = min(earliest, d), max(latest, d)
	}
	const layout = "Jan 2, 2006"
	line := fmt.Sprintf("Itemized charges span %s – %s (by transaction date)",
		time.Unix(earliest, 0).UTC().Format(layout), time.Unix(latest, 0).UTC().Format(layout))
	if current != nil && latest < current.Period.Start.Unix() {
		line += "; none have arrived since, so categories describe that window, not current habits"
	}
	return line + ".\n"
}

func currentPeriod(periods []ledger.PeriodSpend) *ledger.PeriodSpend {
	if len(periods) == 0 {
		return nil
	}
	return &periods[len(periods)-1]
}

// Helper functions operating on DB types.

func formatDBTransactions(txns []models.DBTransaction) string {
	var b strings.Builder
	b.WriteString("| Description | Amount | Date | Category |\n")
	b.WriteString("|------------|---------|------|----------|\n")
	for _, t := range txns {
		ts := t.Posted
		if t.TransactedAt != nil {
			ts = *t.TransactedAt
		}
		cat := t.Category
		if cat == "" {
			cat = "Uncategorized"
		}
		b.WriteString(fmt.Sprintf("| %s | %.2f | %s | %s |\n", t.Description, t.Amount, time.Unix(ts, 0).Format("2006-01-02"), cat))
	}
	return b.String()
}

func calcTotalExpenses(txns []models.DBTransaction) float64 {
	total := 0.0
	for _, t := range txns {
		total += math.Abs(t.Amount)
	}
	return total
}

func formatTopExpenses(txns []models.DBTransaction, n int) string {
	sorted := make([]models.DBTransaction, len(txns))
	copy(sorted, txns)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Amount < sorted[j].Amount })
	if n > len(sorted) {
		n = len(sorted)
	}
	var b strings.Builder
	for _, t := range sorted[:n] {
		ts := t.Posted
		if t.TransactedAt != nil {
			ts = *t.TransactedAt
		}
		b.WriteString(fmt.Sprintf("   - $%.2f at %s on %s\n", math.Abs(t.Amount), t.Description, time.Unix(ts, 0).Format("Jan 2")))
	}
	return b.String()
}

func buildCategoryBreakdown(txns []models.DBTransaction) string {
	totals := make(map[string]float64)
	for _, t := range txns {
		cat := t.Category
		if cat == "" {
			cat = "Uncategorized"
		}
		totals[cat] += math.Abs(t.Amount)
	}

	type catTotal struct {
		name  string
		total float64
	}
	var sorted []catTotal
	for k, v := range totals {
		sorted = append(sorted, catTotal{k, v})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].total > sorted[j].total })

	var b strings.Builder
	b.WriteString("Category Breakdown:\n")
	grandTotal := calcTotalExpenses(txns)
	for _, c := range sorted {
		pct := 0.0
		if grandTotal > 0 {
			pct = (c.total / grandTotal) * 100
		}
		b.WriteString(fmt.Sprintf("- %s: $%.2f (%.1f%%)\n", c.name, c.total, pct))
	}
	return b.String()
}

func buildBudgetSection(budgets []models.Budget, expenses []models.DBTransaction, billingDay int, current *ledger.PeriodSpend) string {
	// Budgets are monthly, so compare against spending in the current billing period only.
	// The analysis range often spans several billing periods; including all of it would make
	// every category look massively over budget.
	periodStart, periodEnd := billing.CurrentBillingPeriod(billingDay)
	startUnix, endUnix := periodStart.Unix(), periodEnd.Unix()

	spending := make(map[string]float64)
	for _, t := range expenses {
		if t.Posted < startUnix || t.Posted > endUnix {
			continue
		}
		cat := t.Category
		if cat == "" {
			cat = "Uncategorized"
		}
		spending[cat] += math.Abs(t.Amount)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Budget Status (current billing period: %s to %s):\n",
		periodStart.Format("2006-01-02"), periodEnd.Format("2006-01-02"))

	// Budgets can only be checked against itemized charges. When most of the
	// cycle is unitemized, "under budget" means nothing, and the model has to
	// know. Below a fifth, every category is trivially under budget, so the
	// figures are withheld rather than hedged.
	itemizedShare := 1.0
	if current != nil && current.Source == ledger.SourceBalance && current.Total > 0 {
		itemizedShare = current.Itemized / current.Total
	}
	if itemizedShare < 0.2 {
		fmt.Fprintf(&b, "Budgets cannot be assessed this cycle: only %.0f%% of card spending is itemized.", itemizedShare*100)
		return b.String()
	}

	for _, budget := range budgets {
		spent := spending[budget.Category]
		// Case-insensitive fallback.
		if spent == 0 {
			for cat, val := range spending {
				if strings.EqualFold(cat, budget.Category) {
					spent = val
					break
				}
			}
		}
		pct := 0.0
		if budget.Amount > 0 {
			pct = (spent / budget.Amount) * 100
		}
		status := ""
		if pct >= 100 {
			status = fmt.Sprintf(" — OVER by $%.2f", spent-budget.Amount)
		}
		b.WriteString(fmt.Sprintf("- %s: $%.2f spent / $%.2f budget (%.0f%%)%s\n",
			budget.Category, spent, budget.Amount, pct, status))
	}
	if itemizedShare < 0.8 {
		fmt.Fprintf(&b, "\nOnly %.0f%% of this cycle's card spending is itemized, so these budget figures are lower bounds.",
			itemizedShare*100)
	}
	b.WriteString("\nComment on budget adherence where budgets are set. Note any categories that are significantly over budget.")
	return b.String()
}

// CyclePeriods splits [start, end] into billing cycles, oldest first; the last three are marked as focus.
func CyclePeriods(start, end time.Time, billingDay int) []models.BillingPeriod {
	if billingDay < 1 {
		billingDay = 1
	} else if billingDay > 28 {
		billingDay = 28
	}

	currentYear, currentMonth, _ := end.Date()
	var currentCycleStart time.Time
	if end.Day() >= billingDay {
		currentCycleStart = time.Date(currentYear, currentMonth, billingDay, 0, 0, 0, 0, time.UTC)
	} else {
		currentCycleStart = time.Date(currentYear, currentMonth, billingDay, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	}

	var periods []models.BillingPeriod
	cycleStart := currentCycleStart

	for cycleStart.After(start) || cycleStart.Equal(start) {
		var periodEnd time.Time
		var isComplete bool
		if cycleStart.Equal(currentCycleStart) {
			periodEnd = end
			isComplete = false
		} else {
			periodEnd = cycleStart.AddDate(0, 1, 0).Add(-24 * time.Hour)
			isComplete = true
		}
		periodStart := cycleStart
		if periodStart.Before(start) {
			periodStart = start
		}
		label := fmt.Sprintf("%s %d - %s %d", periodStart.Format("Jan"), periodStart.Day(), periodEnd.Format("Jan"), periodEnd.Day())
		periods = append([]models.BillingPeriod{{
			Label:      label,
			Start:      periodStart,
			End:        periodEnd,
			IsComplete: isComplete,
		}}, periods...)
		cycleStart = cycleStart.AddDate(0, -1, 0)
	}

	for i := len(periods) - 1; i >= 0 && i >= len(periods)-3; i-- {
		periods[i].IsFocus = true
	}
	return periods
}

// buildDataLagWarning tells the model which cards stopped refreshing. Their
// balances, and so their share of the totals, are out of date. A card whose
// balance is current but whose transactions fall short needs no warning: the
// table's Not itemized column already carries it, and a second figure from
// the drift detector, which cannot see paying-side payments, contradicted it.
func buildDataLagWarning(stale []models.StaleConnection, now time.Time) string {
	var b strings.Builder
	if len(stale) > 0 {
		b.WriteString("\n> [!WARNING]\n> **DATA LAG DETECTED**: ")
		fmt.Fprintf(&b, "%d card(s) stopped refreshing, so their figures are out of date. ", len(stale))
		b.WriteString("Do not congratulate the user on low spending, and do not read a downward trend into it.\n")
		for _, c := range stale {
			lastSync := time.Unix(c.BalanceDate, 0).UTC()
			line := fmt.Sprintf("> - **%s** (%s): last refreshed %s (%d days ago)",
				c.Name, c.OrgName, lastSync.Format("Jan 2, 2006"), daysBetween(lastSync, now))
			if c.LastTransaction > 0 {
				lastTxn := time.Unix(c.LastTransaction, 0).UTC()
				line += fmt.Sprintf("; newest transaction %s (%d days ago)",
					lastTxn.Format("Jan 2, 2006"), daysBetween(lastTxn, now))
			} else {
				line += "; no transactions on record"
			}
			b.WriteString(line + "\n")
		}
	}
	return b.String()
}

func daysBetween(from, to time.Time) int {
	d := int(to.Sub(from).Hours() / 24)
	if d < 0 {
		return 0
	}
	return d
}
