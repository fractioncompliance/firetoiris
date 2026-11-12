package firetoiris

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const DefaultAPIURL = "https://taxcomposer.com"
const DefaultProvider = "taxcomposer"

type Client struct {
	base string
	key  string
	http *http.Client
}

type APIError struct {
	Status     int
	Code       string
	Retryable  bool
	RetryAfter string
}

func (e *APIError) Error() string {
	out := fmt.Sprintf("TaxComposer HTTP %d (%s); retryable=%t", e.Status, e.Code, e.Retryable)
	if e.RetryAfter != "" {
		out += "; retry_after_seconds=" + e.RetryAfter
	}
	return out
}

func NewClient(base, key string, allowLoopbackHTTP bool) (*Client, error) {
	if base == "" {
		base = DefaultAPIURL
	}
	u, err := url.Parse(base)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, fmt.Errorf("API_requires_an_origin_without_credentials_path_query_or_fragment")
	}
	if u.Scheme != "https" {
		ip, e := netip.ParseAddr(u.Hostname())
		if u.Scheme != "http" || !allowLoopbackHTTP || e != nil || !ip.IsLoopback() {
			return nil, fmt.Errorf("HTTPS_required_except_explicit_literal_loopback_simulation")
		}
	}
	if strings.TrimSpace(key) == "" || strings.ContainsAny(key, "\r\n") {
		return nil, fmt.Errorf("API_key_required_in_environment")
	}
	baseTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		baseTransport = &http.Transport{}
	}
	transport := baseTransport.Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	transport.Proxy = nil
	return &Client{base: strings.TrimSuffix(u.String(), "/"), key: key, http: &http.Client{Transport: transport, Timeout: 45 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (c *Client) call(ctx context.Context, method, path, key string, input, output any) error {
	var raw []byte
	var err error
	if input != nil {
		raw, err = json.Marshal(input)
		if err != nil {
			return fmt.Errorf("request_encoding_failed")
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("request_invalid")
	}
	if method != "GET" {
		req.GetBody = nil
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "FIREtoIRIS/"+Version)
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	response, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("network_outcome_unknown: resume_with_the_same_journal")
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 8<<20+1))
	if err != nil || len(data) > 8<<20 {
		return fmt.Errorf("response_unreadable_or_oversized: resume_with_the_same_journal")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var body struct {
			Error struct {
				Code      string `json:"code"`
				Retryable bool   `json:"retryable"`
			} `json:"error"`
		}
		_ = json.Unmarshal(data, &body)
		code := body.Error.Code
		if !regexp.MustCompile(`^[a-z_]{1,80}$`).MatchString(code) {
			code = "request_rejected"
		}
		after := response.Header.Get("Retry-After")
		if !regexp.MustCompile(`^[0-9]{1,6}$`).MatchString(after) {
			after = ""
		}
		return &APIError{response.StatusCode, code, body.Error.Retryable, after}
	}
	if err = json.Unmarshal(data, output); err != nil {
		return fmt.Errorf("response_invalid_JSON: resume_with_the_same_journal")
	}
	return nil
}

type Resource struct {
	ID             string `json:"id"`
	Tenant         string `json:"tenant"`
	FilingID       string `json:"filing_id"`
	SHA256         string `json:"sha256"`
	Mode           string `json:"mode"`
	Status         string `json:"status"`
	TransmissionID string `json:"transmission_id"`
	Validation     *struct {
		Valid           bool `json:"valid"`
		SchemaValidated bool `json:"schema_validated"`
	} `json:"validation,omitempty"`
	Approval *struct {
		SHA256 string `json:"sha256"`
	} `json:"approval,omitempty"`
	SentToIRS             bool   `json:"sent_to_irs"`
	IRSAcceptanceVerified bool   `json:"irs_acceptance_verified"`
	NextAction            string `json:"next_action,omitempty"`
}

func (c *Client) checkMode(ctx context.Context, expected string) error {
	if expected != "simulation" && expected != "preparation" && expected != "ats" {
		return fmt.Errorf("mode_must_be_simulation_preparation_or_ats")
	}
	var caps struct {
		Mode string `json:"mode"`
	}
	if err := c.call(ctx, "GET", "/v1/capabilities", "", nil, &caps); err != nil {
		return err
	}
	if caps.Mode != expected {
		return fmt.Errorf("server_mode_mismatch")
	}
	if strings.HasPrefix(c.base, "http:") && expected != "simulation" {
		return fmt.Errorf("HTTP_is_only_for_loopback_simulation")
	}
	return nil
}

var safeID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,100}$`)
var safeHash = regexp.MustCompile(`^[a-f0-9]{64}$`)

func validResource(r Resource, mode string) bool {
	return safeID.MatchString(r.ID) && safeHash.MatchString(r.SHA256) && r.Mode == mode && r.Tenant != ""
}

// InputCheck contains only non-sensitive validation flags. Detailed findings
// remain in TaxComposer; error text from the service is never echoed here.
type InputCheck struct {
	InputSHA256                 string `json:"input_sha256"`
	Valid                       bool   `json:"valid"`
	SchemaValidated             bool   `json:"schema_validated"`
	CompleteIRSBusinessRules    bool   `json:"complete_irs_business_rules"`
	ProductionSubmissionAllowed bool   `json:"production_submission_allowed"`
}

func (c *Client) CheckInput(ctx context.Context, input Input) (InputCheck, error) {
	if err := CheckEnvelope(input); err != nil {
		return InputCheck{}, err
	}
	var response struct {
		InputSHA256                 string `json:"input_sha256"`
		Valid                       *bool  `json:"valid"`
		SchemaValidated             *bool  `json:"schema_validated"`
		CompleteIRSBusinessRules    *bool  `json:"complete_irs_business_rules"`
		ProductionSubmissionAllowed *bool  `json:"production_submission_allowed"`
	}
	if err := c.call(ctx, "POST", "/v1/filings/form-input/check", "", input, &response); err != nil {
		return InputCheck{}, err
	}
	if !safeHash.MatchString(response.InputSHA256) || response.Valid == nil || response.SchemaValidated == nil || response.CompleteIRSBusinessRules == nil || response.ProductionSubmissionAllowed == nil {
		return InputCheck{}, fmt.Errorf("invalid_service_validation_response")
	}
	return InputCheck{response.InputSHA256, *response.Valid, *response.SchemaValidated, *response.CompleteIRSBusinessRules, *response.ProductionSubmissionAllowed}, nil
}
