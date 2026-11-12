# Contributing

Contributions are welcome under Apache 2.0. Preserve LICENSE, NOTICE and third-party notices when distributing modified copies.

Use Go 1.26 or newer. Run `go test -race ./...` and `go vet ./...`. Windows-specific checks run on native Windows CI; local cross-compilation is not proof of runtime behavior.

Keep changes focused. Include tests for parsing, amount precision, privacy, consent and recovery changes. Use only fictional data. Do not add production credentials, a private TaxComposer module dependency, IRS schema packages or real return examples.

You may modify the converter, interface or backend connection, use it commercially, or publish a fork under the license's terms. A fork must not misrepresent its operator or imply that TaxComposer endorses it. Connecting to another backend does not grant access to TaxComposer accounts or IRS credentials.
