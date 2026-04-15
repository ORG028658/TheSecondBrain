# Security Policy

## Supported Scope

TheSecondBrain is an open-source beta tool that stores user data locally and sends selected content to a configured OpenAI-compatible API endpoint for inference.

Security-sensitive areas include:

- API key storage in `~/.config/secondbrain/.env`
- path traversal and vault-boundary checks
- shell passthrough behavior
- accidental writes outside the vault
- unintended transmission of local content to external providers

## Reporting a Vulnerability

Please do **not** open a public issue for suspected security problems.

Report privately to:

- Email: `sadat.sayem@rakuten.com`
- GitHub: `https://github.com/ORG028658`

Include:

- affected version or commit
- reproduction steps
- impact
- any suggested mitigation

I will acknowledge the report, investigate it, and coordinate disclosure once a fix or mitigation is available.

## Security Expectations

- Secrets should never be committed to the repository.
- Vault file operations must stay inside the project root.
- Behavior documented in `PRIVACY.md` should remain accurate.
- New external integrations must document what data leaves the machine and when.
