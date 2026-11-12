package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func bridgeUpload(t *testing.T, copies int) *bytes.Buffer {
	t.Helper()
	raw, err := os.ReadFile("../../examples/nec.fire.txt")
	if err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	w.WriteField("tax_year", "2026")
	for i := 0; i < copies; i++ {
		f, _ := w.CreateFormFile("files", fmt.Sprintf("export-%d.txt", i))
		f.Write(raw)
	}
	w.Close()
	// Store the boundary in a test-only prefix for the request helper.
	return bytes.NewBuffer(append([]byte(w.FormDataContentType()+"\n"), body.Bytes()...))
}

func TestBridgeLocalReviewAndDuplicateProtection(t *testing.T) {
	b := newBridge(bridgeConfig{APIOrigin: "https://taxcomposer.com", Mode: "preparation", Workspace: t.TempDir()}, "http://127.0.0.1:12345", "test-token")
	for _, copies := range []int{1, 2} {
		upload := bridgeUpload(t, copies)
		content, _ := upload.ReadString('\n')
		req := httptest.NewRequest("POST", b.origin+"/import", upload)
		req.Header.Set("Content-Type", strings.TrimSpace(content))
		req.Header.Set("X-Bridge-Token", b.token)
		response := httptest.NewRecorder()
		b.ServeHTTP(response, req)
		if copies == 1 && response.Code != 200 {
			t.Fatal(response.Code, response.Body.String())
		}
		if copies == 2 && response.Code == 200 {
			t.Fatal("duplicate export accepted")
		}
		if strings.Contains(response.Body.String(), "000654321") {
			t.Fatal("identity in review response")
		}
	}
}

func TestBridgeRejectsCrossSiteAndUnconfirmedPreparation(t *testing.T) {
	b := newBridge(bridgeConfig{APIOrigin: "https://taxcomposer.com", Mode: "preparation", Workspace: t.TempDir()}, "http://127.0.0.1:12345", "test-token")
	for _, tc := range []struct{ host, origin, token string }{
		{"evil.example", "", "test-token"}, {"127.0.0.1:12345", "https://evil.example", "test-token"}, {"127.0.0.1:12345", "", ""},
	} {
		req := httptest.NewRequest("POST", b.origin+"/prepare", strings.NewReader(`{}`))
		req.Host = tc.host
		req.Header.Set("Origin", tc.origin)
		req.Header.Set("X-Bridge-Token", tc.token)
		response := httptest.NewRecorder()
		b.ServeHTTP(response, req)
		if response.Code != 403 {
			t.Fatal("cross-site request accepted")
		}
	}
	req := httptest.NewRequest("POST", b.origin+"/prepare", strings.NewReader(`{"confirmed":false}`))
	req.Header.Set("X-Bridge-Token", b.token)
	response := httptest.NewRecorder()
	b.ServeHTTP(response, req)
	var result map[string]any
	json.Unmarshal(response.Body.Bytes(), &result)
	if response.Code == 200 {
		t.Fatal("unreviewed upload prepared")
	}
}

func TestBridgePreparesThroughPrivateAPIAndResumes(t *testing.T) {
	checks, drafts := 0, 0
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/capabilities":
			fmt.Fprint(w, `{"mode":"simulation"}`)
		case "/v1/filings/form-input/check":
			checks++
			fmt.Fprintf(w, `{"input_sha256":%q,"valid":true,"schema_validated":true,"complete_irs_business_rules":false,"production_submission_allowed":false}`, strings.Repeat("a", 64))
		case "/v1/filings/form-input":
			drafts++
			if r.Header.Get("Idempotency-Key") == "" {
				t.Error("missing recovery key")
			}
			fmt.Fprintf(w, `{"id":"fil_test","tenant":"test","sha256":%q,"mode":"simulation","validation":{"valid":true,"schema_validated":true}}`, strings.Repeat("b", 64))
		default:
			t.Errorf("unexpected service request %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer service.Close()
	b := newBridge(bridgeConfig{APIOrigin: service.URL, Mode: "simulation", Workspace: t.TempDir(), AllowLoopbackHTTP: true}, "http://127.0.0.1:12345", "test-token")
	upload := bridgeUpload(t, 1)
	content, _ := upload.ReadString('\n')
	req := httptest.NewRequest("POST", b.origin+"/import", upload)
	req.Header.Set("Content-Type", strings.TrimSpace(content))
	req.Header.Set("X-Bridge-Token", b.token)
	response := httptest.NewRecorder()
	b.ServeHTTP(response, req)
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	if checks != 0 || drafts != 0 {
		t.Fatal("import sent tax data")
	}
	for i := 0; i < 2; i++ {
		body, _ := json.Marshal(map[string]any{"key": "test-key", "confirmed": true, "source_sha256": b.source})
		req = httptest.NewRequest("POST", b.origin+"/prepare", bytes.NewReader(body))
		req.Header.Set("X-Bridge-Token", b.token)
		response = httptest.NewRecorder()
		b.ServeHTTP(response, req)
		if response.Code != 200 || !strings.Contains(response.Body.String(), `"sent_to_irs":false`) {
			t.Fatal(response.Code, response.Body.String())
		}
	}
	if checks != 2 || drafts != 1 {
		t.Fatalf("unsafe recovery: checks=%d drafts=%d", checks, drafts)
	}
}

func TestBridgeImportsMultipleDistinctExports(t *testing.T) {
	raw, _ := os.ReadFile("../../examples/nec.fire.txt")
	second := bytes.ReplaceAll(raw, []byte("000654321"), []byte("000654322"))
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	writer.WriteField("tax_year", "2026")
	for i, data := range [][]byte{raw, second} {
		file, _ := writer.CreateFormFile("files", fmt.Sprintf("file-%d.txt", i))
		file.Write(data)
	}
	writer.Close()
	b := newBridge(bridgeConfig{}, "http://127.0.0.1:12345", "test-token")
	req := httptest.NewRequest("POST", b.origin+"/import", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-Bridge-Token", b.token)
	response := httptest.NewRecorder()
	b.ServeHTTP(response, req)
	var report struct {
		Records int `json:"records"`
		Exports int `json:"export_count"`
	}
	json.Unmarshal(response.Body.Bytes(), &report)
	if response.Code != 200 || report.Records != 2 || report.Exports != 2 {
		t.Fatal(response.Code, response.Body.String())
	}
}
