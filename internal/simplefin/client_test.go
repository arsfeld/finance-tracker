package simplefin

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func fetchFrom(t *testing.T, body string) ([]byte, *Client, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, body)
	}))
	return nil, NewClient(srv.URL), srv.Close
}

// A credit card paid off to exactly zero is the normal end state of a billing
// cycle, not a dead account. Dropping it took a whole cycle of spending with
// it, and the TD card is the only account that matters here.
func TestFetchKeepsZeroBalanceAccountsAndTheirTransactions(t *testing.T) {
	_, client, done := fetchFrom(t, `{"accounts":[{
		"id":"ACT-td","name":"TD AEROPLAN VISA INFINITE (4520)",
		"balance":"0.00","balance-date":1756000000,
		"org":{"name":"TD Canada Trust"},
		"transactions":[
			{"id":"t1","description":"METRO BLANCHARD","amount":"-67.05","posted":1755000000},
			{"id":"t2","description":"PAYMENT - THANK YOU","amount":"67.05","posted":1755500000}
		]}]}`)
	defer done()

	accounts, apiErrors, err := client.fetch(time.Now().AddDate(0, 0, -90), time.Now())
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(apiErrors) != 0 {
		t.Errorf("unexpected api errors: %v", apiErrors)
	}
	if len(accounts) != 1 {
		t.Fatalf("a zero-balance account must survive the fetch, got %d accounts", len(accounts))
	}
	if got := len(accounts[0].Transactions); got != 2 {
		t.Errorf("want both transactions on the zero-balance account, got %d", got)
	}
}

// API-level errors must reach the caller so a de-authorized connection can be
// reported rather than silently producing an empty, healthy-looking sync.
func TestFetchReturnsAPIErrors(t *testing.T) {
	_, client, done := fetchFrom(t, `{"accounts":[],"errors":[
		"Connection to TD Canada Trust may need attention. Auth required"]}`)
	defer done()

	_, apiErrors, err := client.fetch(time.Now().AddDate(0, 0, -90), time.Now())
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if len(apiErrors) != 1 {
		t.Fatalf("want the auth error surfaced, got %v", apiErrors)
	}
}

// The bridge URL carries the SimpleFin access credentials. They were printed in
// full to the debug log, where anyone reading the container logs could lift them
// and read every account.
func TestFetchKeepsCredentialsOutOfTheLog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"accounts":[]}`)
	}))
	defer srv.Close()

	var logged bytes.Buffer
	prev := log.Logger
	log.Logger = zerolog.New(&logged).Level(zerolog.DebugLevel)
	defer func() { log.Logger = prev }()

	bridge := strings.Replace(srv.URL, "http://", "http://ACCESSUSER:ACCESSSECRET@", 1)
	if _, _, err := NewClient(bridge).fetch(time.Now().AddDate(0, 0, -30), time.Now()); err != nil {
		t.Fatalf("fetch: %v", err)
	}

	if out := logged.String(); strings.Contains(out, "ACCESSSECRET") || strings.Contains(out, "ACCESSUSER") {
		t.Errorf("credentials leaked into the log:\n%s", out)
	}
}

// A failed request's error ends up in sync_log and the UI. net/http masks the
// password there but not the username, which is half of the credential.
func TestFetchKeepsCredentialsOutOfRequestErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	bridge := strings.Replace(srv.URL, "http://", "http://ACCESSUSER:ACCESSSECRET@", 1)
	srv.Close() // connection refused

	_, _, err := NewClient(bridge).fetch(time.Now().AddDate(0, 0, -30), time.Now())

	if err == nil {
		t.Fatal("expected a connection error")
	}
	if msg := err.Error(); strings.Contains(msg, "ACCESSUSER") || strings.Contains(msg, "ACCESSSECRET") {
		t.Errorf("credentials leaked into the error: %s", msg)
	}
}
