package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fractioncompliance/firetoiris/pkg/firetoiris"
)

//go:embed web/*
var bridgeAssets embed.FS

type bridgeConfig struct {
	APIOrigin, Mode, Workspace string
	AllowLoopbackHTTP          bool
}
type bridge struct {
	config        bridgeConfig
	origin, token string
	mu            sync.Mutex
	bundle        *firetoiris.Bundle
	source        string
}

func newBridge(config bridgeConfig, origin, token string) *bridge {
	return &bridge{config: config, origin: origin, token: token}
}
func (b *bridge) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'self'; style-src 'self'; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
	if r.Host != strings.TrimPrefix(b.origin, "http://") || (r.Header.Get("Origin") != "" && r.Header.Get("Origin") != b.origin) {
		http.Error(w, "Local bridge access only", 403)
		return
	}
	if r.Method == "GET" {
		names := map[string]string{"/": "index.html", "/app.js": "app.js", "/app.css": "app.css"}
		name, ok := names[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		raw, err := bridgeAssets.ReadFile("web/" + name)
		if err != nil {
			http.Error(w, "Page unavailable", 500)
			return
		}
		if name == "index.html" {
			raw = bytes.ReplaceAll(raw, []byte("BRIDGE_TOKEN"), []byte(b.token))
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		}
		if name == "app.js" {
			w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		}
		if name == "app.css" {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		}
		w.Write(raw)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "Method unavailable", 405)
		return
	}
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-Bridge-Token")), []byte(b.token)) != 1 {
		http.Error(w, "Refresh this local bridge window", 403)
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/import":
		b.importFiles(w, r)
	case "/prepare":
		b.prepare(w, r)
	case "/clear":
		b.bundle = nil
		b.source = ""
		json.NewEncoder(w).Encode(map[string]bool{"cleared": true})
	default:
		http.NotFound(w, r)
	}
}
func bridgeError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
func (b *bridge) importFiles(w http.ResponseWriter, r *http.Request) {
	b.bundle = nil
	b.source = ""
	r.Body = http.MaxBytesReader(w, r.Body, 128<<20)
	reader, err := r.MultipartReader()
	if err != nil {
		bridgeError(w, 400, "Choose FIRE exports to import.")
		return
	}
	var files [][]byte
	year := 2026
	order := ""
	for {
		part, e := reader.NextPart()
		if e == io.EOF {
			break
		}
		if e != nil {
			bridgeError(w, 400, "Import could not be read. Maximum combined size is 128 MiB.")
			return
		}
		raw, e := io.ReadAll(io.LimitReader(part, 128<<20+1))
		part.Close()
		if e != nil || len(raw) > 128<<20 {
			bridgeError(w, 400, "Import exceeds the 128 MiB limit.")
			return
		}
		switch part.FormName() {
		case "tax_year":
			year, e = strconv.Atoi(string(raw))
			if e != nil {
				bridgeError(w, 400, "Choose a supported tax year.")
				return
			}
		case "name_order":
			order = string(raw)
		case "files":
			files = append(files, raw)
			if len(files) > 50 {
				bridgeError(w, 400, "Import up to 50 exports at a time.")
				return
			}
		default:
			bridgeError(w, 400, "Unexpected import field.")
			return
		}
	}
	if len(files) == 0 {
		bridgeError(w, 400, "Choose at least one FIRE export.")
		return
	}
	inputs := []firetoiris.Input{}
	hashes := []string{}
	seen := map[string]bool{}
	for i, raw := range files {
		parsed, e := firetoiris.Parse(bytes.NewReader(raw), year)
		if e != nil {
			bridgeError(w, 422, fmt.Sprintf("Export %d: %s", i+1, e))
			return
		}
		if seen[parsed.SHA256] {
			bridgeError(w, 422, fmt.Sprintf("Export %d duplicates another selected export. Remove the duplicate.", i+1))
			return
		}
		seen[parsed.SHA256] = true
		converted, e := firetoiris.Convert(parsed, firetoiris.Options{NameOrder: order})
		if e != nil {
			bridgeError(w, 422, fmt.Sprintf("Export %d: %s", i+1, e))
			return
		}
		for _, input := range converted {
			if e = firetoiris.CheckEnvelope(input); e != nil {
				bridgeError(w, 422, fmt.Sprintf("Export %d: %s", i+1, e))
				return
			}
		}
		inputs = append(inputs, converted...)
		hashes = append(hashes, parsed.SHA256)
		if len(inputs) > 100 {
			bridgeError(w, 422, "Import up to 100 payer batches at a time.")
			return
		}
	}
	b.source = bridgeDigest(hashes)
	b.bundle = &firetoiris.Bundle{Version: firetoiris.Version, SourceSHA256: b.source, Inputs: inputs}
	report := summary(*b.bundle, b.source)
	report["export_count"] = len(files)
	report["filing_service"] = b.config.APIOrigin
	report["mode"] = b.config.Mode
	json.NewEncoder(w).Encode(report)
}
func bridgeDigest(value any) string {
	raw, _ := json.Marshal(value)
	return fmt.Sprintf("%x", sha256.Sum256(raw))
}
func (b *bridge) prepare(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Key       string `json:"key"`
		Confirmed bool   `json:"confirmed"`
		Source    string `json:"source_sha256"`
	}
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	d.DisallowUnknownFields()
	if d.Decode(&request) != nil || d.Decode(&struct{}{}) != io.EOF || !request.Confirmed || b.bundle == nil || request.Source != b.source {
		bridgeError(w, 400, "Import and review the selected exports before preparing.")
		return
	}
	c, err := firetoiris.NewClient(b.config.APIOrigin, request.Key, b.config.AllowLoopbackHTTP)
	request.Key = ""
	if err != nil {
		bridgeError(w, 400, "Enter a valid TaxComposer API key.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()
	// All form validation stays in the private service. Check every batch before
	// saving any draft; retrying preparation reuses durable idempotency keys.
	for i, input := range b.bundle.Inputs {
		check, e := c.CheckInput(ctx, input)
		if e != nil {
			bridgeError(w, 422, fmt.Sprintf("Batch %d: %s", i+1, e))
			return
		}
		if !check.Valid || !check.SchemaValidated {
			bridgeError(w, 422, fmt.Sprintf("Batch %d needs attention in TaxComposer before it can be prepared.", i+1))
			return
		}
	}
	if err = os.MkdirAll(b.config.Workspace, 0700); err != nil {
		bridgeError(w, 500, "The private recovery folder could not be created.")
		return
	}
	journalPath := filepath.Join(b.config.Workspace, "bridge-"+b.source+"-state.json")
	journal, err := firetoiris.OpenJournal(journalPath)
	if err != nil {
		bridgeError(w, 409, "Recovery state is unavailable or already in use. Preserve the recovery folder and try again.")
		return
	}
	defer journal.Close()
	if err = journal.Prepare(ctx, c, b.bundle.Inputs, b.config.Mode, b.source); err != nil {
		bridgeError(w, 422, err.Error())
		return
	}
	ids := []string{}
	for _, item := range journal.State.Items {
		ids = append(ids, item.FilingID)
	}
	json.NewEncoder(w).Encode(map[string]any{"prepared_batches": len(ids), "filing_ids": ids, "status": "prepared_for_owner_review", "filing_service": b.config.APIOrigin, "mode": b.config.Mode, "sent_to_irs": false, "irs_acceptance_verified": false, "notice": "Prepared in TaxComposer. Continue there to review, authorize and send when filing is available."})
}

func runBridge(ctx context.Context, args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("bridge", flag.ContinueOnError)
	fs.SetOutput(errOut)
	api := fs.String("api", firetoiris.DefaultAPIURL, "Filing service origin")
	mode := fs.String("mode", "preparation", "Expected service mode: preparation, simulation or ats")
	workspace := fs.String("workspace", "", "Private recovery folder")
	noOpen := fs.Bool("no-open", false, "Print the local address without opening a browser")
	loopback := fs.Bool("allow-loopback-http", false, "Allow an explicitly selected local simulation service")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 || (*mode != "preparation" && *mode != "simulation" && *mode != "ats") {
		fmt.Fprintln(errOut, "Choose preparation, simulation or ats.")
		return 2
	}
	if _, err := firetoiris.NewClient(*api, "origin-check", *loopback); err != nil {
		fmt.Fprintln(errOut, "Invalid filing service origin.")
		return 2
	}
	if *workspace == "" {
		folder, err := os.UserConfigDir()
		if err != nil {
			fmt.Fprintln(errOut, "Choose a private folder with --workspace.")
			return 1
		}
		*workspace = filepath.Join(folder, "FIREtoIRIS")
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintln(errOut, "Local bridge could not start.")
		return 1
	}
	defer listener.Close()
	secret := make([]byte, 32)
	if _, err = rand.Read(secret); err != nil {
		fmt.Fprintln(errOut, "Local bridge could not start securely.")
		return 1
	}
	origin := "http://" + listener.Addr().String()
	server := &http.Server{Handler: newBridge(bridgeConfig{*api, *mode, *workspace, *loopback}, origin, hex.EncodeToString(secret)), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 90 * time.Second, WriteTimeout: 6 * time.Minute, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			server.Close()
		case <-done:
		}
	}()
	fmt.Fprintf(out, "FIREtoIRIS bridge: %s\nKeep this window open. Press Ctrl+C to close.\n", origin)
	if !*noOpen {
		var command *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			command = exec.Command("open", origin)
		case "windows":
			command = exec.Command("rundll32", "url.dll,FileProtocolHandler", origin)
		default:
			command = exec.Command("xdg-open", origin)
		}
		if err = command.Start(); err == nil {
			go command.Wait()
		}
	}
	err = server.Serve(listener)
	if err != nil && err != http.ErrServerClosed && ctx.Err() == nil {
		fmt.Fprintln(errOut, "Local bridge stopped.")
		return 1
	}
	return 0
}
