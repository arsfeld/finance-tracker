package llm

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"finance_tracker/internal/billing"
	"finance_tracker/internal/models"
)

// SystemPrompt is the default LLM system prompt for spending analysis.
const SystemPrompt = `You are an expert financial analyst specializing in personal finance and spending pattern analysis for families.
You are analyzing spending for a Canadian family of 4: 2 adults, 2 daughters (born 2021 and 2024, currently ~4 and ~1 years old).
Your role is to provide clear, actionable insights from transaction data. Focus on identifying trends,
highlighting what's improving and what needs attention, and providing family-relevant suggestions.
Be concise, specific, and use the pre-calculated data provided — do not recalculate totals or percentages.`

// GeneratePrompt builds an analysis prompt from database transactions.
func GeneratePrompt(
	transactions []models.DBTransaction,
	accounts []models.DBAccount,
	stale []models.StaleConnection,
	drifted []models.UnreconciledAccount,
	startDate, endDate time.Time,
	now time.Time,
	billingDay int,
	isMultiPeriod bool,
	budgets ...models.Budget,
) string {
	dataLagWarning := buildDataLagWarning(stale, drifted, now)
	// Filter to expenses only.
	var expenses []models.DBTransaction
	for _, t := range transactions {
		if t.Amount < 0 {
			expenses = append(expenses, t)
		}
	}

	txnTable := formatDBTransactions(expenses)
	acctTable := formatDBAccounts(accounts)
	totalExpenses := calcTotalExpenses(expenses)

	var prompt string
	if isMultiPeriod {
		prompt = generateMultiPeriodPrompt(expenses, accounts, startDate, endDate, now, billingDay, acctTable, txnTable, totalExpenses, dataLagWarning)
	} else {
		prompt = generateSinglePeriodPrompt(expenses, startDate, endDate, now, acctTable, txnTable, totalExpenses, dataLagWarning)
	}

	// Append budget status if budgets are configured.
	if len(budgets) > 0 {
		prompt += "\n\n" + buildBudgetSection(budgets, expenses, billingDay)
	}

	return prompt
}

func generateSinglePeriodPrompt(
	expenses []models.DBTransaction,
	startDate, endDate time.Time,
	now time.Time,
	acctTable, txnTable string,
	totalExpenses float64,
	dataLagWarning string,
) string {
	calDays := int(endDate.Sub(startDate).Hours()/24) + 1
	txnDays := countTxnDays(expenses, startDate, endDate)

	burnRate := 0.0
	if txnDays > 0 {
		burnRate = totalExpenses / float64(txnDays)
	}
	monthlyProjection := burnRate * 30

	topExpenses := formatTopExpenses(expenses, 10)

	catBreakdown := buildCategoryBreakdown(expenses)

	return fmt.Sprintf(`## Financial Transaction Analysis — Family of 4

Household: 2 adults, 2 children (born 2021 and 2024, currently ~4 and ~1 years old).

Billing Period: %s to %s (%d calendar days, %d transaction days)
Total Expenses: $%.2f
Daily Burn Rate: $%.2f/day (based on transaction days)
Monthly Projection: $%.2f (at current rate)

%s
%s

Please create a concise report (~250 words) with:

### Summary
Provide a human-friendly overview of spending patterns during this period.

### Analysis Breakdown
1. **Total Expenses**: $%.2f
2. **Major Categories**: Use the pre-calculated category totals above
3. **Top 10 Largest Expenses**:
%s4. **Key Insights**: 1-2 actionable insights referencing the burn rate and projection above.

Notes:
- Consider only outgoing expenses (ignore income/credits/refunds)
- Format all monetary values consistently (e.g., $1,234.56)
- Use CAD ($) for all amounts

Accounts Information:
%s

All Transactions:
%s`,
		startDate.Format("2006-01-02"), endDate.Format("2006-01-02"),
		calDays, txnDays, totalExpenses, burnRate, monthlyProjection,
		catBreakdown, dataLagWarning,
		totalExpenses, topExpenses, acctTable, txnTable)
}

func generateMultiPeriodPrompt(
	expenses []models.DBTransaction,
	accounts []models.DBAccount,
	startDate, endDate time.Time,
	now time.Time,
	billingDay int,
	acctTable, txnTable string,
	totalExpenses float64,
	dataLagWarning string,
) string {
	periods := calcBillingPeriods(startDate, endDate, billingDay)
	periodTotals := calcPeriodTotals(expenses, periods)

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Multi-Cycle Analysis (%d Billing Periods):\n", len(periods)))

	var completedBurnRates []float64
	for i, p := range periods {
		calDays := int(p.End.Sub(p.Start).Hours()/24) + 1
		txnDays := countTxnDays(expenses, p.Start, p.End)
		burnRate := 0.0
		if txnDays > 0 {
			burnRate = periodTotals[i] / float64(txnDays)
		}
		status := "completed"
		if !p.IsComplete {
			status = "in progress"
		}
		focus := ""
		if p.IsFocus {
			focus = " [FOCUS]"
		}
		change := ""
		if i > 0 && periodTotals[i-1] > 0 {
			pct := ((periodTotals[i] - periodTotals[i-1]) / periodTotals[i-1]) * 100
			dir := "no change"
			if pct > 0 {
				dir = "increase"
			} else if pct < 0 {
				dir = "decrease"
			}
			change = fmt.Sprintf(" - Change: %+.1f%% (%s)", pct, dir)
		}
		summary.WriteString(fmt.Sprintf("- Period %d: %s (%d cal/%d txn days) - $%.2f [%s] - Burn: $%.2f/day%s%s\n",
			i+1, p.Label, calDays, txnDays, periodTotals[i], status, burnRate, change, focus))
		if p.IsComplete {
			completedBurnRates = append(completedBurnRates, burnRate)
		}
	}

	avg := 0.0
	if len(periodTotals) > 0 {
		s := 0.0
		for _, t := range periodTotals {
			s += t
		}
		avg = s / float64(len(periodTotals))
	}
	avgBurn := 0.0
	if len(completedBurnRates) > 0 {
		s := 0.0
		for _, r := range completedBurnRates {
			s += r
		}
		avgBurn = s / float64(len(completedBurnRates))
	}

	summary.WriteString(fmt.Sprintf("- Grand Total: $%.2f\n", totalExpenses))
	summary.WriteString(fmt.Sprintf("- %d-Period Average: $%.2f\n", len(periods), avg))
	summary.WriteString(fmt.Sprintf("- Average Burn Rate (completed): $%.2f/day\n", avgBurn))
	summary.WriteString(fmt.Sprintf("- Monthly Projection: $%.2f\n", avgBurn*30))

	catBreakdown := buildCategoryBreakdown(expenses)
	topExpenses := formatTopExpenses(expenses, 10)

	return fmt.Sprintf(`## Financial Transaction Analysis — Family of 4

Household: 2 adults, 2 children (born 2021 and 2024, currently ~4 and ~1 years old).

%s
%s
%s

### Instructions

Analyze this family's spending. Write a concise report (~250 words) with:

1. **Summary**: Overview of the last 3 billing cycles. What's the overall trajectory?
2. **What's Improving**: Categories or habits trending positively.
3. **What Needs Attention**: Categories growing faster than average, unusual spikes.
4. **Top Expenses**: The 10 largest individual charges:
%s5. **Family-Specific Insights**: Note anything relevant to a family with young children.
6. **Actionable Suggestions**: 2-3 concrete things to try next billing cycle.

Notes:
- All calculations are pre-computed. Use them directly.
- Focus narrative on FOCUS periods.
- Use CAD ($) for all amounts.
- Consider only outgoing expenses.

Accounts Information:
%s

All Transactions:
%s`, summary.String(), catBreakdown, dataLagWarning, topExpenses, acctTable, txnTable)
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

func formatDBAccounts(accounts []models.DBAccount) string {
	var b strings.Builder
	b.WriteString("| Account | Balance | Last Synced |\n")
	b.WriteString("|---------|---------|-------------|\n")
	for _, a := range accounts {
		b.WriteString(fmt.Sprintf("| %s | %.2f | %s |\n", a.Name, a.Balance, time.Unix(a.BalanceDate, 0).Format("2006-01-02")))
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

func buildBudgetSection(budgets []models.Budget, expenses []models.DBTransaction, billingDay int) string {
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
	b.WriteString("\nComment on budget adherence where budgets are set. Note any categories that are significantly over budget.")
	return b.String()
}

func countTxnDays(txns []models.DBTransaction, start, end time.Time) int {
	days := make(map[string]bool)
	for _, t := range txns {
		ts := t.Posted
		if t.TransactedAt != nil {
			ts = *t.TransactedAt
		}
		d := time.Unix(ts, 0)
		if !d.Before(start) && !d.After(end) {
			days[d.Format("2006-01-02")] = true
		}
	}
	return len(days)
}

func calcBillingPeriods(start, end time.Time, billingDay int) []models.BillingPeriod {
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

func calcPeriodTotals(txns []models.DBTransaction, periods []models.BillingPeriod) []float64 {
	totals := make([]float64, len(periods))
	for _, t := range txns {
		ts := t.Posted
		if t.TransactedAt != nil {
			ts = *t.TransactedAt
		}
		d := time.Unix(ts, 0)
		for i, p := range periods {
			if !d.Before(p.Start) && !d.After(p.End) {
				totals[i] += math.Abs(t.Amount)
				break
			}
		}
	}
	return totals
}

// FilterExcludedCategories removes transactions belonging to categories the user
// has excluded from analysis. Exclusion is independent of the sign of the amount:
// both legs of an excluded transfer must disappear, otherwise the negative leg
// would be counted as spending while the positive leg is silently dropped.
func FilterExcludedCategories(txns []models.DBTransaction, excluded []string) []models.DBTransaction {
	if len(excluded) == 0 {
		return txns
	}

	set := make(map[string]bool, len(excluded))
	for _, c := range excluded {
		if c != "" {
			set[c] = true
		}
	}
	if len(set) == 0 {
		return txns
	}

	filtered := make([]models.DBTransaction, 0, len(txns))
	for _, t := range txns {
		if !set[t.Category] {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// buildDataLagWarning describes the accounts whose bank connection has stopped
// refreshing, so the model can tell missing data from a genuinely quiet month.
//
// The trigger is the connection, not the transaction feed. A gap since the last
// charge means nothing on its own — a card can go a fortnight without a
// purchase — and a global "newest transaction" check stays silent as long as
// any one account is still live, which is exactly how five dead accounts went
// unnoticed for months.
func buildDataLagWarning(stale []models.StaleConnection, drifted []models.UnreconciledAccount, now time.Time) string {
	if len(stale) == 0 && len(drifted) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n> [!WARNING]\n> **DATA LAG DETECTED**: ")
	b.WriteString(fmt.Sprintf("%d account(s) are not reporting in full, so this analysis is based on INCOMPLETE data. ", len(stale)+len(drifted)))
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

	for _, u := range drifted {
		fmt.Fprintf(&b, "> - **%s** (%s): still syncing, but its balance has moved $%.2f more than its transactions account for, so that spending is missing from this report\n",
			u.Name, u.OrgName, math.Abs(u.Unexplained))
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
