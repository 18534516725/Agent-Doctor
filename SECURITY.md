# Security policy

## Reporting a vulnerability

Please report suspected vulnerabilities privately through GitHub Security
Advisories after the public repository is available. Until then, contact the
maintainer through a verified NexoToken support channel.

Do not include complete prompts, source code, API keys, bearer tokens, cookies,
passwords, private repository URLs, raw request headers, or full diagnostic logs
in a public issue. Create the smallest synthetic reproduction possible.

Agent Doctor is designed to fail open when an adapter cannot reach its local
service: a diagnostic failure must not block the user's AI coding task. Reports
about a client-blocking hook, credential persistence, non-loopback listener,
unauthorized data export, or destructive uninstall are treated as high priority.

## Supported versions

Security fixes are provided for the latest stable release. Before the first
stable release, only source builds from this repository are in scope.
