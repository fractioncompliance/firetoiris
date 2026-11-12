# Changelog

## 0.1.0 — developer preview

- Standalone Apache-2.0 FIRE migration command and Go package, using TaxComposer’s private filing module through its API.
- Legacy conversion profiles for 32 Publication 1220 forms and client-contract inputs for 83 form/year pairs.
- Guided bulk-import browser interface: import exports, review totals and prepare drafts.
- TaxComposer as the default API connector, with an origin override and forkable connection code.
- Private recovery journals, stable request keys and separate approval/submission controls.
- Windows, macOS and Linux downloads on x64 and ARM64.
- Go 1.26.11 minimum runtime, recorded in package manifests, with reachable-vulnerability checks in CI.

The local browser interface prepares drafts. There is no direct IRS transmitter or enabled production filing mode in this preview.
