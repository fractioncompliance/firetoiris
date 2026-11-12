# FIREtoIRIS

An Apache-2.0 migration bridge for software that already exports IRS FIRE-format information returns. Keep your export routine, check and convert its output, and connect to a filing service. TaxComposer is the default service; you can modify the program or replace the connection.

**Developer preview.** The downloadable app opens a local browser window with an import, review and prepare flow. A command-line interface and Go library are also included. It does not include an IRS transmitter, TaxComposer's backend, IRS XSD packages or filing credentials. Local checks cover the source export and request envelope. TaxComposer’s private filing module performs form validation through the API; it is not reimplemented in this repository. Production filing is not enabled by this preview.

## What it does

- Reads legacy Publication 1220 files with ordering, ASCII/length, tax-year, amount-code, exact-cents, count and control-total checks.
- Converts original domestic records using profiles for all 32 Publication 1220 form codes. A representative export for each form is tested in TY2025 and TY2026.
- Accepts reviewed JSON inputs for 83 snapshotted form/year API contracts, including 44 TY2026 forms.
- Creates drafts through the default TaxComposer connector; explicit authorization, approval, submission and status are separate commands.
- Preserves idempotency keys and recovery state so interrupted requests can be reconciled without blindly duplicating filings.
- Builds for Windows, macOS and Linux on x64 and ARM64.

The source layout is [Publication 1220, Rev. May 2026](https://www.irs.gov/pub/irs-pdf/p1220.pdf), which describes TY2025 FIRE layouts. TY2026 values can use that legacy export layout and the TY2026 target API contract. This is not a claim that IRS published a TY2026 FIRE layout. New-year fields absent from an old layout need reviewed input.

## Download or build

Choose the ZIP for your OS and processor in [Releases](https://github.com/fractioncompliance/firetoiris/releases). Extract it and launch FIREtoIRIS. It opens the guided bridge in your browser. If your OS requires a terminal, run `./FIREtoIRIS bridge` (Windows: `.\FIREtoIRIS.exe bridge`). Keep its window open while using the bridge.

| Target | Download suffix |
|---|---|
| Windows Intel/AMD x64 | windows-amd64 |
| Windows ARM64 | windows-arm64 |
| Mac Apple silicon | macos-arm64 |
| Mac Intel | macos-amd64 |
| Linux Intel/AMD x64 | linux-amd64 |
| Linux ARM64 | linux-arm64 |

No Go installation is needed for a download. Preview binaries are unsigned. Use the normal operating-system checks; do not disable security software. A developer can build with Go 1.26.11 or a newer patched release:

```sh
go build -trimpath -o FIREtoIRIS ./cmd/firetoiris
go test -race ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
```

## The guided bridge

1. Select your tax year and import all the FIRE exports for the batch (up to 50 files / 128 MiB).
2. Review return counts, form types and exact amount totals. Duplicate exports and unsupported data block preparation.
3. Paste a TaxComposer preparation API key, confirm the review and select **Prepare in TaxComposer**.
4. The private API checks all batches before creating drafts, then saves schema-validated drafts with durable recovery keys. Continue in TaxComposer for owner review, authorization and sending when available.

The bridge binds only to a random port on `127.0.0.1`. Imports are held in memory until replaced or the bridge closes. Keys are not stored. Recovery journals contain IDs and digests in the operating system’s user configuration folder under `FIREtoIRIS`; use `bridge --workspace PATH` to choose another private folder. Source data is sent only when you select Prepare. No return is sent to IRS by the guided bridge.

An unsupported record stops the import with its file/record position and a reason. The tool does not silently drop data or label an unresolved return ready. Forms using a different legacy specification still need a corresponding converter; the reviewed API input path is available for all listed installed forms.

## Try a fictional export offline

```sh
./FIREtoIRIS inspect --input nec.fire.txt --tax-year 2026
./FIREtoIRIS convert --input nec.fire.txt --tax-year 2026 --output reviewed.json
./FIREtoIRIS catalog
./FIREtoIRIS schema --tax-year 2026 --form 1099NEC
```

From a source checkout, the sample path is `examples/nec.fire.txt`. Inspection prints counts, totals, assumptions and a source digest without names or TINs. Conversion writes a new private JSON file and refuses to overwrite an existing file. These commands make no network request.

Personal names require an explicit `--name-order last-first` or `--name-order first-last`; only unambiguous two-word names are automatically split. Business issuer TINs are assumed by the legacy profile. Sole proprietors using an SSN, joint names, suffixes and other ambiguous identities require reviewed input. Blank legacy indicators become false, so newly introduced tax-year fields require review.

## Prepare through TaxComposer

The default API origin is `https://taxcomposer.com`. The connector uses its public REST contract, including `POST /v1/filings/form-input`. You need a TaxComposer account/key with the appropriate access. Put the key in `FIRETOIRIS_API_KEY` or `TAXCOMPOSER_API_KEY` using your secret manager or process environment, not command arguments or source files.

```sh
./FIREtoIRIS prepare --input existing-fire.txt --tax-year 2026 \
  --mode preparation --state batch-state.json
```

This sends converted data to create drafts. It does not transmit a return to IRS. A preparation key cannot approve or submit. Source and envelope checks happen locally. TaxComposer’s private filing module builds and validates the XML and applies its authorization gates. A converted file may still be rejected by the service.

For forms or fields needing review, author a JSON input with `tax_year`, `form`, `header` and `records` following the `schema` command. Scalars are lexical strings, groups are objects, and repeating fields are arrays. Do not supply automatic fields. Each form/year has a fictional example under `pkg/firetoiris/testdata/inputs/`.

```sh
./FIREtoIRIS prepare --form-input reviewed-form.json \
  --mode preparation --state batch-state.json
```

## Approval, test submission and results

Only an authorized owner credential can approve the exact saved filing. `--authorization-reference` records an actual external review/authorization; the string does not itself establish consent.

```sh
./FIREtoIRIS approve --state batch-state.json \
  --authorization-reference YOUR-REVIEW-RECORD --confirm-reviewed
./FIREtoIRIS challenge --state batch-state.json --batch 1
./FIREtoIRIS submit --state batch-state.json --batch 1 --confirm-reviewed
./FIREtoIRIS status --state batch-state.json
```

The configured service may require a fresh browser CAPTCHA. Supply its challenge and token using `FIRETOIRIS_CHALLENGE` and `FIRETOIRIS_CAPTCHA_TOKEN`; the command does not solve or bypass CAPTCHA. The implemented environments are preparation, simulation and IRS test (ATS), subject to the service's own gates. There is no production-mode switch in this preview.

Queued, received and accepted are distinct states. A simulated acceptance is identified as simulation and must not be represented as IRS acceptance. This repository makes no claim that any user's filing has been authorized or accepted by IRS.

## Use your own service or fork

TaxComposer is a default, not a license requirement. Point `--api https://YOUR-SERVICE` at a compatible REST implementation. For a different protocol, replace or adapt `pkg/firetoiris/client.go`; the parser and converter are separate. The API contract is described by the client methods and embedded public-facing field snapshots. Forks can change the interface, mappings and connection, and can be used commercially under Apache 2.0.

```sh
./FIREtoIRIS prepare --input existing-fire.txt --tax-year 2026 \
  --api https://YOUR-SERVICE --mode preparation --state own-service-state.json
```

Changing a client does not grant access to TaxComposer accounts, infrastructure, trademarks, IRS enrollment or credentials. A direct IRS backend is a separate implementation and requires the applicable [IRS enrollment and testing](https://www.irs.gov/e-file-providers/iris-assurance-testing-system-ats). The hosted TaxComposer backend is not part of this open-source project.

## Coverage and limits

Foreign/free-form addresses, state elections and K totals, corrections/replacements, combined name/phone fields, special-data fields and unmapped material data are blocked rather than silently discarded. FIRE formats for 1042-S, 8027 and 8955-SSA are separate specifications; use reviewed API inputs for those forms. See [coverage](COVERAGE.md).

Input is limited to 128 MiB; converted groups split at 5,000 recipients. Supported records are continuous 750-byte records, 750-byte records with LF/CRLF separators, or 748-byte data plus embedded CRLF. Unsupported encodings and overpunch amounts are rejected.

## Recovery and privacy

Keep original exports and the journal. Reuse the same source, API origin, environment, preparation key and journal after a timeout. Completed drafts are retained, uncertain requests reuse stable idempotency keys, and submit checks for an existing transmission before another attempt. The journal stores IDs and digests, not API keys or raw tax records.

Unix uses private file modes and flock; Windows uses protected current-user/SYSTEM access lists and LockFileEx. Use a local filesystem with these semantics. HTTPS verification stays enabled; redirects, HTTP proxies and automatic write retries are disabled. Literal loopback HTTP is allowed only by explicit simulation selection. The tool has no telemetry or automatic cloud sync.

Do not put real tax data, credentials or journals in public issues. See [security](SECURITY.md).

## Contract snapshots and dependencies

`pkg/firetoiris/contracts/` contains only client-facing form-input metadata: fields, groups, choices and lexical constraints. It does not contain IRS XSD files, signing material, server XML construction, government connectivity or proprietary business-rule execution. These snapshots guide field mapping only. There is no independent form validator or XML builder. The private TaxComposer module is a required service dependency for validation and preparation. A manifest pins each snapshot.

The only external module is `golang.org/x/sys`, used for Windows system calls under its BSD license. Go and dependency notices are included under `third_party/` and in every binary ZIP.

## License

[Apache License 2.0](LICENSE), with [NOTICE](NOTICE) and third-party notices. You may modify, redistribute and use the project commercially under those terms. Trademark rights and hosted TaxComposer services are not granted by the code license. See [contributing](CONTRIBUTING.md).
