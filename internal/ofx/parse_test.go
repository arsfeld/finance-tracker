package ofx

import (
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

func parseFixture(t *testing.T, name string) *Statement {
	t.Helper()

	f, err := os.Open("testdata/" + name)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	stmt, err := Parse(f)
	if err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	return stmt
}

func TestParseQFX(t *testing.T) {
	stmt := parseFixture(t, "accountactivity.qfx")

	if stmt.AccountID != "4520883071874533" {
		t.Errorf("account id = %q, want 4520883071874533", stmt.AccountID)
	}
	if stmt.Currency != "CAD" {
		t.Errorf("currency = %q, want CAD", stmt.Currency)
	}
	if len(stmt.Transactions) != 50 {
		t.Fatalf("transactions = %d, want 50", len(stmt.Transactions))
	}
	if stmt.LedgerBalance == nil || *stmt.LedgerBalance != -3187.08 {
		t.Errorf("ledger balance = %v, want -3187.08", stmt.LedgerBalance)
	}
	wantDate := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	if !stmt.BalanceDate.Equal(wantDate) {
		t.Errorf("balance date = %v, want %v", stmt.BalanceDate, wantDate)
	}

	first := stmt.Transactions[0]
	if first.FITID != "2026-10-15~2026-10-27~1817474" {
		t.Errorf("first fitid = %q", first.FITID)
	}
	if first.Name != "AMZN Mktp CA*7Q32G2B93" {
		t.Errorf("first name = %q", first.Name)
	}
	if first.Amount != -98.71 {
		t.Errorf("first amount = %v", first.Amount)
	}
	if !first.Posted.Equal(wantDate) {
		t.Errorf("first posted = %v, want %v", first.Posted, wantDate)
	}

	last := stmt.Transactions[len(stmt.Transactions)-1]
	if last.FITID != "2026-10-15~2026-10-15~1819510" {
		t.Errorf("last fitid = %q", last.FITID)
	}
}

func TestParseMaskedOFX(t *testing.T) {
	stmt := parseFixture(t, "accountactivity.ofx")

	if stmt.AccountID != "452088******4533" {
		t.Errorf("account id = %q", stmt.AccountID)
	}
	if len(stmt.Transactions) != 50 {
		t.Fatalf("transactions = %d, want 50", len(stmt.Transactions))
	}

	qfx := parseFixture(t, "accountactivity.qfx")
	qfxIDs := make(map[string]bool, len(qfx.Transactions))
	for _, tx := range qfx.Transactions {
		qfxIDs[tx.FITID] = true
	}
	for _, tx := range stmt.Transactions {
		if !qfxIDs[tx.FITID] {
			t.Errorf("masked OFX has FITID %q not present in QFX", tx.FITID)
		}
	}
}

func TestParseQBOAndASO(t *testing.T) {
	for _, name := range []string{"accountactivity.qbo", "accountactivity.aso"} {
		stmt := parseFixture(t, name)
		if len(stmt.Transactions) != 50 {
			t.Errorf("%s transactions = %d, want 50", name, len(stmt.Transactions))
		}
	}
}

func TestParseRejectsNonOFX(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "not-ofx")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("date,description,amount\n2026-10-01,Coffee,-4.50\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if _, err := Parse(f); err == nil {
		t.Fatal("expected an error for a non-OFX file")
	} else if !strings.Contains(err.Error(), "not an OFX") {
		t.Errorf("error = %q, want it to mention not an OFX", err)
	}
}

func TestParseOFXDate(t *testing.T) {
	for _, raw := range []string{"20261027020000[-5:EST]", "20261027020000", "20261027"} {
		got, err := parseOFXDate(raw)
		if err != nil {
			t.Fatalf("parse %q: %v", raw, err)
		}
		want := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
		if !got.Equal(want) {
			t.Errorf("parse %q = %v, want %v", raw, got, want)
		}
	}
}

// OFX 1.x commonly omits closing tags on aggregate elements. The parser must
// rely on the next aggregate, not on closing tags that may not exist.
func TestParseWithoutAggregateClosingTags(t *testing.T) {
	data, err := os.ReadFile("testdata/accountactivity.qfx")
	if err != nil {
		t.Fatal(err)
	}
	stripped := regexp.MustCompile(`</[A-Za-z0-9.]+>`).ReplaceAllString(string(data), "")
	stmt, err := Parse(strings.NewReader(stripped))
	if err != nil {
		t.Fatalf("parse stripped OFX: %v", err)
	}
	if len(stmt.Transactions) != 50 {
		t.Errorf("transactions = %d, want 50", len(stmt.Transactions))
	}
	if stmt.LedgerBalance == nil || *stmt.LedgerBalance != -3187.08 {
		t.Errorf("ledger balance = %v", stmt.LedgerBalance)
	}
}

func TestParseRejectsMultipleStatements(t *testing.T) {
	body := `OFXHEADER:100
DATA:OFXSGML
<OFX>
<STMTRS><CURDEF>CAD</CURDEF></STMTRS>
<STMTRS><CURDEF>CAD</CURDEF></STMTRS>
</OFX>`
	if _, err := Parse(strings.NewReader(body)); err == nil {
		t.Fatal("expected multiple statements to be rejected")
	}
}

func TestParseBankStatement(t *testing.T) {
	body := `OFXHEADER:100
DATA:OFXSGML
<OFX>
<BANKMSGSRSV1>
<STMTTRNRS>
<STMTRS>
<CURDEF>CAD
<BANKACCTFROM>
<ACCTID>123456789
<ACCTTYPE>CHECKING
</BANKACCTFROM>
<BANKTRANLIST>
<DTSTART>20261001
<DTEND>20261031
<STMTTRN>
<TRNTYPE>DEBIT
<DTPOSTED>20261002
<TRNAMT>-10.00
<FITID>abc
<NAME>COFFEE
</STMTTRN>
</BANKTRANLIST>
<LEDGERBAL>
<BALAMT>100.00
<DTASOF>20261031
</LEDGERBAL>
</STMTRS>
</STMTTRNRS>
</BANKMSGSRSV1>
</OFX>`
	stmt, err := Parse(strings.NewReader(body))
	if err != nil {
		t.Fatalf("parse bank statement: %v", err)
	}
	if stmt.AccountID != "123456789" || stmt.AccountType != "CHECKING" {
		t.Errorf("account = %q/%q", stmt.AccountID, stmt.AccountType)
	}
	if len(stmt.Transactions) != 1 || stmt.Transactions[0].FITID != "abc" {
		t.Errorf("transactions = %+v", stmt.Transactions)
	}
	if stmt.LedgerBalance == nil || *stmt.LedgerBalance != 100 {
		t.Errorf("ledger balance = %v", stmt.LedgerBalance)
	}
}
