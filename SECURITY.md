# Security

Do not put taxpayer records, API keys, journals, or private server information in public issues or pull requests.

FIREtoIRIS inspects files locally. Tax data is sent only by explicit preparation or filing commands to the configured API origin. HTTPS certificate verification remains enabled. There is no telemetry or automatic IRS transmission. Input, destination and environment are bound to a private recovery journal.

Use your secret manager or process environment for API keys. Keep original tax exports and recovery journals in private folders on a local filesystem. The command is a developer preview, not a production filing authorization.

For a security issue, use GitHub's private vulnerability reporting when enabled. Otherwise contact the maintainers privately through an existing verified channel; do not post secrets or exploitable details in a public issue.
