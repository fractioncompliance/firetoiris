# FIREtoIRIS

This is a standalone Apache-2.0 client project. Keep TaxComposer's backend private.

- Never add private backend imports, IRS XSD packages, signing credentials, production configuration or real taxpayer data.
- Tests use fictional records and local HTTP servers; do not call IRS or submit returns in tests.
- The default connector is TaxComposer. Preserve API-origin overrides and the right to modify or replace the connector.
- Validation and filing depend on the private TaxComposer module through its API. Do not replicate its form validator, XML builder or filing rules. Contract snapshots guide field mapping only; keep their manifest and fictional fixtures in sync.
- Run `go test -race ./...`, `go vet ./...` and the public-source audit before publishing. Cross-compile all six supported targets.
- Use Apache 2.0 for project code; preserve notices for dependencies. Never grant rights to credentials or trademarks.
- Preserve recovery journals and idempotency keys across retries. Reject changed input or destination before sending tax data.
