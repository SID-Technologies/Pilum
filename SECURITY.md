# Security Policy

## Reporting a Vulnerability

Please report security vulnerabilities privately through GitHub, not as a public issue:

**[Report a vulnerability](https://github.com/SID-Technologies/Pilum/security/advisories/new)** (Security tab → Report a vulnerability)

This opens a private advisory visible only to maintainers until a fix is ready. If you'd rather not use GitHub, email **dan.flanagan93@gmail.com** with "Pilum security" in the subject.

Include what you can:
- The recipe/ingredient or code path involved
- Steps to reproduce, or a proof of concept
- What you'd expect to happen instead

You should get a response within a few days. Pilum is maintained by one person, so turnaround isn't instant, but every report gets read and taken seriously.

## Supported Versions

Pilum is pre-1.0. Only the latest tagged release is supported — fixes land as new patch releases rather than backports to older tags.

| Version | Supported |
|---------|-----------|
| Latest (`v0.7.x`) | ✅ |
| Older tags | ❌ |

## How Pilum Handles Credentials

Pilum does not store, proxy, or manage credentials of its own. Recipes generate the same provider CLI commands you'd run by hand (`gcloud run deploy`, `sam deploy`, `az containerapp up`, ...), executed with whatever credentials are already active in your environment — `gcloud auth`, AWS profiles, environment variables, or your CI's secret store. IAM/RBAC policies apply exactly as if you'd typed the command yourself; Pilum cannot do anything your existing credentials can't already do.

`pilum deploy --dry-run` prints every command before anything executes, so you can verify exactly what will run against your infrastructure.

## Automated Checks

This repo runs [`govulncheck`](https://github.com/golang/vuln) in CI on every PR, has Dependabot version updates and security updates enabled, and uses GitHub secret scanning with push protection.
