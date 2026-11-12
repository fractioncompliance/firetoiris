package firetoiris

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestClientProtectsCredentialsAndSanitizesErrors(t *testing.T) {
	for _, u := range []string{"http://example.com", "https://user:pass@example.com", "https://example.com/path", "https://example.com?key=secret", "http://localhost:8099"} {
		if _, err := NewClient(u, "secret", false); err == nil {
			t.Fatalf("accepted %s", u)
		}
	}
	leaked := false
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked = true }))
	defer destination.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, destination.URL, 307)
			return
		}
		w.WriteHeader(422)
		fmt.Fprint(w, `{"code":"bad_input","message":"000654321 secret"}`)
	}))
	defer server.Close()
	c, err := NewClient(server.URL, "secret", true)
	if err != nil {
		t.Fatal(err)
	}
	err = c.call(context.Background(), "POST", "/redirect", "key", map[string]string{}, &map[string]any{})
	if err == nil || leaked {
		t.Fatal("followed redirect")
	}
	err = c.call(context.Background(), "POST", "/bad", "key", nil, &map[string]any{})
	if err == nil || strings.Contains(err.Error(), "000654321") || strings.Contains(err.Error(), "secret") {
		t.Fatalf("unsafe error %v", err)
	}
}

func TestJournalPrepareResumesAfterLostResponse(t *testing.T) {
	rows := fixture("NE")
	b, _ := Parse(strings.NewReader(strings.Join(rows, "\n")), 2026)
	inputs, _ := Convert(b, Options{})
	keys := []string{}
	attempt := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/capabilities" {
			fmt.Fprint(w, `{"mode":"simulation","production_enabled":false}`)
			return
		}
		if r.URL.Path == "/v1/filings/form-input" {
			keys = append(keys, r.Header.Get("Idempotency-Key"))
			attempt++
			if attempt == 1 {
				h := w.(http.Hijacker)
				conn, _, _ := h.Hijack()
				conn.Close()
				return
			}
			json.NewEncoder(w).Encode(Resource{ID: "fil_test", SHA256: strings.Repeat("a", 64), Mode: "simulation", Status: "validated", Tenant: "test", Validation: &struct {
				Valid           bool `json:"valid"`
				SchemaValidated bool `json:"schema_validated"`
			}{true, true}})
			return
		}
		t.Errorf("unexpected route %s", r.URL.Path)
	}))
	defer server.Close()
	c, _ := NewClient(server.URL, "secret", true)
	path := filepath.Join(t.TempDir(), "state.json")
	j, err := OpenJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = j.Prepare(context.Background(), c, inputs, "simulation"); err == nil {
		t.Fatal("lost response reported success")
	}
	j.Close()
	j, err = OpenJournal(path)
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	if err = j.Prepare(context.Background(), c, inputs, "simulation"); err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 || keys[0] != keys[1] {
		t.Fatal("retry changed idempotency key")
	}
	if len(j.State.Items) != 1 || j.State.Items[0].FilingID != "fil_test" {
		t.Fatal("missing recovery state")
	}
	if err = j.Prepare(context.Background(), c, inputs, "simulation"); err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 {
		t.Fatal("recreated completed draft")
	}
	inputs[0].Records[0]["NonemployeeCompensationAmt"] = "9999.00"
	if err = j.Prepare(context.Background(), c, inputs, "simulation"); err == nil {
		t.Fatal("journal accepted changed payload")
	}
	if other, err := OpenJournal(path); err == nil {
		other.Close()
		t.Fatal("concurrent journal writer accepted")
	}
}

func TestPreparationValidatesWholeBatchBeforeNetwork(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
	defer server.Close()
	c, _ := NewClient(server.URL, "secret", true)
	j, err := OpenJournal(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	b, _ := Parse(strings.NewReader(strings.Join(fixture("NE"), "\n")), 2026)
	inputs, _ := Convert(b, Options{})
	inputs = append(inputs, Input{TaxYear: 2026, Form: "UNKNOWN"})
	if err = j.Prepare(context.Background(), c, inputs, "simulation"); err == nil || calls != 0 {
		t.Fatal("invalid later batch sent earlier batch")
	}
}

func TestReviewedInputRejectsDuplicateKeysAndUnknownFields(t *testing.T) {
	for _, raw := range []string{`{"tax_year":2026,"tax_year":2025}`, `{"inputs":[],"version":"0.1.0","secret":"private"}`, `{"tax_year":2026} {"tax_year":2025}`} {
		if _, _, err := ReadInputs(strings.NewReader(raw)); err == nil {
			t.Fatal("accepted malformed JSON")
		}
	}
}
