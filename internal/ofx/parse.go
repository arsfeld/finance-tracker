// Package ofx parses the OFX family of bank statement exports: OFX, QFX,
// QBO and ASO. They share the same SGML/XML payload, and only the wrapper
// differs.
package ofx

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"strconv"
	"strings"
	"time"
)

// Statement is the single account statement contained in an export.
type Statement struct {
	AccountID     string
	AccountType   string
	Currency      string
	LedgerBalance *float64
	BalanceDate   time.Time
	Start         time.Time
	End           time.Time
	Transactions  []Transaction
}

// Transaction is one posted row from an OFX statement.
type Transaction struct {
	FITID  string
	Type   string
	Name   string
	Memo   string
	Amount float64
	Posted time.Time
}

type token struct {
	name    string
	value   string
	closing bool
}

// Parse reads one OFX statement. It accepts both OFX 1.x SGML, where most
// aggregate tags have no closing tag, and OFX 2.x XML.
func Parse(r io.Reader) (*Statement, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read OFX: %w", err)
	}
	if !bytes.Contains(bytes.ToUpper(data), []byte("<OFX")) {
		return nil, fmt.Errorf("not an OFX file: missing <OFX> header")
	}

	stmt := &Statement{}
	statements := 0
	var cur *Transaction
	var inLedger bool

	flush := func() {
		if cur == nil {
			return
		}
		if cur.FITID != "" || cur.Name != "" || cur.Amount != 0 {
			stmt.Transactions = append(stmt.Transactions, *cur)
		}
		cur = nil
	}

	for _, tok := range tokenize(data) {
		up := strings.ToUpper(tok.name)
		if tok.closing {
			switch up {
			case "STMTTRN":
				flush()
			case "LEDGERBAL":
				inLedger = false
			case "CREDITCARDMSGSRSV1", "BANKMSGSRSV1", "CCSTMTRS", "STMTRS", "BANKTRANLIST":
				flush()
			}
			continue
		}

		switch up {
		case "CCSTMTRS", "STMTRS":
			statements++
		case "ACCTID":
			if stmt.AccountID == "" {
				stmt.AccountID = strings.TrimSpace(tok.value)
			}
		case "ACCTTYPE":
			if stmt.AccountType == "" {
				stmt.AccountType = strings.TrimSpace(tok.value)
			}
		case "CURDEF":
			if stmt.Currency == "" {
				stmt.Currency = strings.TrimSpace(tok.value)
			}
		case "LEDGERBAL":
			inLedger = true
		case "AVAILBAL":
			inLedger = false
		case "BALAMT":
			if inLedger && stmt.LedgerBalance == nil {
				amount, err := parseAmount(tok.value)
				if err != nil {
					return nil, fmt.Errorf("parse ledger balance: %w", err)
				}
				stmt.LedgerBalance = &amount
			}
		case "DTASOF":
			if inLedger && stmt.BalanceDate.IsZero() {
				date, err := parseOFXDate(tok.value)
				if err != nil {
					return nil, fmt.Errorf("parse balance date: %w", err)
				}
				stmt.BalanceDate = date
			}
		case "DTSTART":
			if stmt.Start.IsZero() {
				if date, err := parseOFXDate(tok.value); err == nil {
					stmt.Start = date
				}
			}
		case "DTEND":
			if stmt.End.IsZero() {
				if date, err := parseOFXDate(tok.value); err == nil {
					stmt.End = date
				}
			}
		case "STMTTRN":
			flush()
			cur = &Transaction{}
		case "TRNTYPE":
			if cur != nil {
				cur.Type = strings.TrimSpace(tok.value)
			}
		case "DTPOSTED":
			if cur != nil {
				date, err := parseOFXDate(tok.value)
				if err != nil {
					return nil, fmt.Errorf("parse transaction date %q: %w", tok.value, err)
				}
				cur.Posted = date
			}
		case "TRNAMT":
			if cur != nil {
				amount, err := parseAmount(tok.value)
				if err != nil {
					return nil, fmt.Errorf("parse transaction amount %q: %w", tok.value, err)
				}
				cur.Amount = amount
			}
		case "FITID":
			if cur != nil {
				cur.FITID = strings.TrimSpace(tok.value)
			}
		case "NAME":
			if cur != nil {
				cur.Name = strings.TrimSpace(tok.value)
			}
		case "MEMO":
			if cur != nil {
				cur.Memo = strings.TrimSpace(tok.value)
			}
		}
	}
	flush()

	if statements > 1 {
		return nil, fmt.Errorf("statement contains %d accounts; upload one account at a time", statements)
	}
	if statements == 0 {
		return nil, fmt.Errorf("not an OFX statement: no account statement found")
	}
	if stmt.Start.After(stmt.End) && !stmt.End.IsZero() {
		stmt.Start, stmt.End = stmt.End, stmt.Start
	}
	return stmt, nil
}

// parseOFXDate reads the date portion of an OFX timestamp. Times are dropped
// deliberately: the app stores transaction dates at noon UTC so a bank's
// timezone cannot move a purchase to another day.
func parseOFXDate(raw string) (time.Time, error) {
	digits := make([]byte, 0, 8)
	for _, r := range raw {
		if r < '0' || r > '9' {
			if len(digits) == 8 {
				break
			}
			continue
		}
		digits = append(digits, byte(r))
		if len(digits) == 8 {
			break
		}
	}
	if len(digits) != 8 {
		return time.Time{}, fmt.Errorf("invalid OFX date %q", raw)
	}
	date, err := time.Parse("20060102", string(digits))
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid OFX date %q: %w", raw, err)
	}
	return time.Date(date.Year(), date.Month(), date.Day(), 12, 0, 0, 0, time.UTC), nil
}

func parseAmount(raw string) (float64, error) {
	clean := strings.ReplaceAll(strings.TrimSpace(raw), ",", "")
	return strconv.ParseFloat(clean, 64)
}

func tokenize(data []byte) []token {
	var tokens []token
	for i := 0; i < len(data); {
		if data[i] != '<' {
			i++
			continue
		}
		i++
		if i >= len(data) {
			break
		}
		if data[i] == '?' || data[i] == '!' {
			for i < len(data) && data[i] != '>' {
				i++
			}
			i++
			continue
		}

		closing := false
		if data[i] == '/' {
			closing = true
			i++
		}

		start := i
		for i < len(data) && data[i] != '>' && data[i] != ' ' && data[i] != '\n' && data[i] != '\r' && data[i] != '\t' {
			i++
		}
		name := strings.TrimSpace(string(data[start:i]))
		for i < len(data) && data[i] != '>' {
			i++
		}
		i++ // consume '>'

		if closing || name == "" {
			if name != "" {
				tokens = append(tokens, token{name: name, closing: closing})
			}
			continue
		}

		start = i
		for i < len(data) && data[i] != '<' {
			i++
		}
		value := html.UnescapeString(strings.TrimSpace(string(data[start:i])))
		tokens = append(tokens, token{name: name, value: value})
	}
	return tokens
}
