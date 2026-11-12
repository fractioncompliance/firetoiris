package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/fractioncompliance/firetoiris/pkg/firetoiris"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, out, errOut io.Writer) int {
	if len(args) == 0 {
		return runBridge(ctx, nil, out, errOut)
	}
	if args[0] == "bridge" {
		return runBridge(ctx, args[1:], out, errOut)
	}
	if args[0] == "help" || args[0] == "--help" {
		fmt.Fprintln(out, "FIREtoIRIS — keep your FIRE export, prepare through TaxComposer.\nCommands: bridge, catalog, schema, inspect, convert, prepare, approve, challenge, submit, status\nUse COMMAND --help for options. Conversion and preparation never submit a return.\nKeys: FIRETOIRIS_API_KEY or TAXCOMPOSER_API_KEY; submission tokens: FIRETOIRIS_CHALLENGE and FIRETOIRIS_CAPTCHA_TOKEN.")
		return 0
	}
	command := args[0]
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	fs.SetOutput(errOut)
	input := fs.String("input", "", "Existing 750-byte FIRE export")
	formInput := fs.String("form-input", "", "Reviewed TaxComposer form-input JSON or converted bundle")
	output := fs.String("output", "", "New private converted JSON file")
	year := fs.Int("tax-year", 2026, "Tax year, distinct from the calendar year of submission")
	form := fs.String("form", "", "TaxComposer form code, for example 1099NEC")
	order := fs.String("name-order", "", "Explicit personal name order: last-first or first-last; two words only")
	api := fs.String("api", firetoiris.DefaultAPIURL, "TaxComposer API origin; override for your own compatible service")
	mode := fs.String("mode", "simulation", "Expected server mode: simulation, preparation or ats")
	state := fs.String("state", "", "Private recovery journal path; reuse for retries")
	loopback := fs.Bool("allow-loopback-http", false, "Allow literal loopback HTTP for simulation only")
	reference := fs.String("authorization-reference", "", "Reference to actual external filer authorization")
	confirm := fs.Bool("confirm-reviewed", false, "Confirm review before owner approval or submission")
	batch := fs.Int("batch", 1, "One-based batch number for challenge or submit")
	if err := fs.Parse(args[1:]); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(errOut, "Unexpected positional arguments.")
		return 2
	}
	emit := func(value any) error { e := json.NewEncoder(out); e.SetIndent("", "  "); return e.Encode(value) }
	var err error
	switch command {
	case "catalog":
		var rows []firetoiris.Capability
		rows, err = firetoiris.Catalog()
		if err == nil {
			err = emit(rows)
		}
	case "schema":
		var schema any
		schema, err = firetoiris.Schema(*year, *form)
		if err == nil {
			err = emit(schema)
		}
	case "inspect", "convert", "prepare":
		var bundle firetoiris.Bundle
		var source string
		bundle, source, err = load(*input, *formInput, *year, *order)
		if err != nil {
			break
		}
		if command == "inspect" {
			err = emit(summary(bundle, source))
			break
		}
		if command == "convert" {
			if *output == "" {
				err = fmt.Errorf("--output is required")
				break
			}
			if err = firetoiris.WriteBundle(*output, bundle); err == nil {
				err = emit(map[string]any{"written": true, "source_sha256": source, "batches": len(bundle.Inputs), "notice": "Contains private tax data. No network request was made."})
			}
			break
		}
		var c *firetoiris.Client
		var j *firetoiris.Journal
		c, j, err = connect(*api, *state, *loopback)
		if err != nil {
			break
		}
		defer j.Close()
		if err = j.Prepare(ctx, c, bundle.Inputs, *mode, source); err == nil {
			err = emit(map[string]any{"prepared_batches": len(j.State.Items), "source_sha256": source, "mode": j.State.Mode, "notice": "Saved drafts only. No return was submitted."})
		}
	case "approve", "challenge", "submit", "status":
		if (command == "approve" || command == "submit") && !*confirm {
			err = fmt.Errorf("--confirm-reviewed is required")
			break
		}
		var c *firetoiris.Client
		var j *firetoiris.Journal
		c, j, err = connect(*api, *state, *loopback)
		if err != nil {
			break
		}
		defer j.Close()
		switch command {
		case "approve":
			err = j.Approve(ctx, c, *reference)
			if err == nil {
				err = emit(map[string]any{"approved": true, "mode": j.State.Mode, "notice": "Approved current hashes. No return was submitted."})
			}
		case "challenge":
			var result firetoiris.Challenge
			result, err = j.Challenge(ctx, c, *batch-1)
			if err == nil {
				err = emit(result)
			}
		case "submit":
			err = j.Submit(ctx, c, *batch-1, os.Getenv("FIRETOIRIS_CHALLENGE"), os.Getenv("FIRETOIRIS_CAPTCHA_TOKEN"))
			if err == nil {
				err = emit(map[string]any{"batch": *batch, "transmission_id": j.State.Items[*batch-1].TransmissionID, "mode": j.State.Mode, "notice": "Read status for the result. A queue entry is not IRS acceptance."})
			}
		case "status":
			var resources []firetoiris.Resource
			resources, err = j.Status(ctx, c)
			if err == nil {
				rows := []map[string]any{}
				for _, r := range resources {
					rows = append(rows, map[string]any{"id": r.ID, "status": r.Status, "mode": r.Mode, "sent_to_irs": r.SentToIRS, "irs_acceptance_verified": r.IRSAcceptanceVerified, "next_action": r.NextAction})
				}
				err = emit(rows)
			}
		}
	default:
		err = fmt.Errorf("unknown command; run FIREtoIRIS help")
	}
	if err != nil {
		fmt.Fprintln(errOut, "FIREtoIRIS:", err)
		return 1
	}
	return 0
}

func load(input, reviewed string, year int, order string) (firetoiris.Bundle, string, error) {
	if (input == "") == (reviewed == "") {
		return firetoiris.Bundle{}, "", fmt.Errorf("supply exactly one of --input and --form-input")
	}
	path := input
	if reviewed != "" {
		path = reviewed
	}
	f, err := os.Open(path)
	if err != nil {
		return firetoiris.Bundle{}, "", fmt.Errorf("input file cannot be opened")
	}
	defer f.Close()
	if reviewed != "" {
		return firetoiris.ReadInputs(f)
	}
	b, err := firetoiris.Parse(f, year)
	if err != nil {
		return firetoiris.Bundle{}, "", err
	}
	inputs, err := firetoiris.Convert(b, firetoiris.Options{NameOrder: order})
	if err != nil {
		return firetoiris.Bundle{}, "", err
	}
	for i, item := range inputs {
		if err = firetoiris.CheckEnvelope(item); err != nil {
			return firetoiris.Bundle{}, "", fmt.Errorf("batch %d (%s): %w", i+1, item.Form, err)
		}
	}
	return firetoiris.Bundle{Version: firetoiris.Version, SourceSHA256: b.SHA256, Inputs: inputs}, b.SHA256, nil
}
func connect(origin, state string, loopback bool) (*firetoiris.Client, *firetoiris.Journal, error) {
	if origin == "" || state == "" {
		return nil, nil, fmt.Errorf("--api and --state are required")
	}
	key := os.Getenv("FIRETOIRIS_API_KEY")
	if key == "" {
		key = os.Getenv("TAXCOMPOSER_API_KEY")
	}
	c, err := firetoiris.NewClient(origin, key, loopback)
	if err != nil {
		return nil, nil, err
	}
	j, err := firetoiris.OpenJournal(state)
	return c, j, err
}
func summary(bundle firetoiris.Bundle, source string) map[string]any {
	forms := map[string]int{}
	years := map[string]bool{}
	count := 0
	sums := map[string]*big.Rat{}
	for _, input := range bundle.Inputs {
		for _, record := range input.Records {
			for key, raw := range record {
				if strings.HasSuffix(key, "Amt") {
					if text, ok := raw.(string); ok {
						if value, valid := new(big.Rat).SetString(text); valid {
							key = input.Form + "." + key
							if sums[key] == nil {
								sums[key] = new(big.Rat)
							}
							sums[key].Add(sums[key], value)
						}
					}
				}
			}
		}
		forms[input.Form] += len(input.Records)
		years[strconv.Itoa(input.TaxYear)] = true
		count += len(input.Records)
	}
	totals := map[string]string{}
	for key, value := range sums {
		totals[key] = value.FloatString(2)
	}
	return map[string]any{"amount_field_sums": totals, "source_layout": "Publication 1220 TY2025 (Rev. May 2026) or reviewed form-input", "source_sha256": source, "form_counts": forms, "tax_years": years, "payer_batches": len(bundle.Inputs), "records": count, "offline_source_checks_passed": true, "filing_validation_pending": true, "irs_acceptance_verified": false, "assumptions": []string{"FIRE issuer TINs are treated as business TINs; review sole proprietors separately", "Blank legacy indicators map to false; supply new tax-year fields in reviewed form-input"}, "notice": "Offline conversion only. Review converted data and server validation before authorizing a filing."}
}
