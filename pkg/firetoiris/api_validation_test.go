package firetoiris

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidationDependsOnPrivateService(t *testing.T) {
	b, _ := Parse(strings.NewReader(strings.Join(fixture("NE"), "\n")), 2026)
	inputs, _ := Convert(b, Options{})
	// A filing rule is deliberately left for the private module, not replicated locally.
	inputs[0].Records[0]["NonemployeeCompensationAmt"] = "1.001"
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "POST" || r.URL.Path != "/v1/filings/form-input/check" || r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected check request")
		}
		var input Input
		if json.NewDecoder(r.Body).Decode(&input) != nil || input.Records[0]["NonemployeeCompensationAmt"] != "1.001" {
			t.Error("input changed")
		}
		fmt.Fprintf(w, `{"input_sha256":%q,"valid":false,"schema_validated":false,"complete_irs_business_rules":false,"production_submission_allowed":false,"findings":[{"message":"000654321 private"}]}`, strings.Repeat("a", 64))
	}))
	defer server.Close()
	c, _ := NewClient(server.URL, "test-key", true)
	result, err := c.CheckInput(context.Background(), inputs[0])
	if err != nil || result.Valid || result.SchemaValidated || calls != 1 {
		t.Fatalf("private check not authoritative: %+v %v", result, err)
	}
	raw, _ := json.Marshal(result)
	if strings.Contains(string(raw), "000654321") || strings.Contains(string(raw), "private") {
		t.Fatal("check disclosed server findings")
	}
}

func TestCheckRejectsMalformedServiceResponse(t *testing.T) {
	b, _ := Parse(strings.NewReader(strings.Join(fixture("NE"), "\n")), 2026)
	inputs, _ := Convert(b, Options{})
	for _, body := range []string{`{}`, `{"valid":true,"schema_validated":true}`, `{"input_sha256":"bad","valid":true,"schema_validated":true,"complete_irs_business_rules":false,"production_submission_allowed":false}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
		c, _ := NewClient(server.URL, "test-key", true)
		if _, err := c.CheckInput(context.Background(), inputs[0]); err == nil {
			t.Error("accepted incomplete response")
		}
		server.Close()
	}
}

func TestDefaultTaxComposerOriginAndOverride(t *testing.T) {
	c, err := NewClient("", "test-key", false)
	if err != nil || c.base != "https://taxcomposer.com" {
		t.Fatalf("default: %v %v", c, err)
	}
	c, err = NewClient("https://example.invalid", "test-key", false)
	if err != nil || c.base != "https://example.invalid" {
		t.Fatalf("override: %v %v", c, err)
	}
}
