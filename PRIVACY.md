# Privacy Policy

**Product:** TheSecondBrain  
**Developer:** A S M Sayem  
**Organisation:** ORG028658 (company-provided GitHub account, Rakuten)  
**Repository:** https://github.com/ORG028658/TheSecondBrain  
**Last updated:** 2026-04-13  

---

## Overview

TheSecondBrain is a locally-run, terminal-native knowledge vault. It processes files on your machine and sends their content to a configurable LLM API to build and query a wiki. This policy explains exactly what data moves where.

**Short version:** Your files and queries leave your machine only to reach the LLM API you configure. The developer collects nothing.

---

## 1. What This Tool Does with Your Data

### Data that stays on your machine

| Location | Contents |
|----------|----------|
| `raw/` | Your source files — never modified, never uploaded |
| `wiki/` | All generated wiki pages (markdown files) |
| `knowledge-base/embeddings/` | Vector embeddings of your wiki pages |
| `knowledge-base/metadata/` | Content hashes for change detection |
| `knowledge-base/amendments/` | Correction audit trail |
| `~/.config/secondbrain/` | Your API key and model settings |

All of the above is stored locally on your filesystem. The developer has no access to any of it.

### Data that leaves your machine

When you ingest a file or ask a question, the following is sent to the LLM API endpoint configured in `~/.config/secondbrain/config.yaml`:

- **During ingestion:** The full text content of the source file being processed
- **During queries:** Your query text and relevant wiki page chunks (retrieved via similarity search)
- **During wiki corrections:** The page content being corrected and the proposed correction

This data is sent solely for the purpose of LLM inference. No copy is retained by the developer.

---

## 2. Default API Endpoint

By default, TheSecondBrain is configured to use the **Rakuten AI Gateway**:

```
https://api.ai.public.rakuten-it.com/openai/v1
```

When using this default, your data is subject to Rakuten's own data handling and privacy policies in addition to this document. Refer to your organisation's internal documentation for the applicable Rakuten AI Gateway data retention and processing terms.

### Using a different API provider

You can point TheSecondBrain at any OpenAI-compatible API by editing `~/.config/secondbrain/config.yaml`:

```yaml
llm:
  base_url: "https://your-api-provider.com/v1"
embeddings:
  base_url: "https://your-api-provider.com/v1"
```

If you use a third-party provider (e.g. OpenAI, Azure OpenAI, Anthropic), your data is subject to that provider's privacy policy instead.

---

## 3. API Key Storage

Your API key is stored locally in:

```
~/.config/secondbrain/.env
```

This file is created with `600` permissions (readable only by you). It is never committed to version control, never transmitted to the developer, and never included in any logs or exports produced by this tool.

---

## 4. No Telemetry or Analytics

TheSecondBrain collects **no usage data, no crash reports, no analytics, and no metrics** of any kind. There are no background network calls, no update checks, and no phone-home mechanisms. The only outbound network traffic the tool generates is the LLM API calls described in Section 1.

---

## 5. Claude Code Plugin Component

TheSecondBrain includes a Claude Code plugin (`plugin.json`, `skills/`, `commands/`, `agents/`, `hooks.json`). These components run locally inside the Claude Code CLI. They do not introduce any additional network endpoints beyond what the base Claude Code application already uses. The hooks operate entirely locally, intercepting tool calls before they execute.

---

## 6. Data You Are Responsible For

Because this tool processes whatever files you place in `raw/`, you are responsible for ensuring that:

- You have the right to process and transmit any file you add to the vault
- You comply with applicable data protection regulations (e.g. GDPR, APPI) when processing files that contain personal data about others
- You follow your organisation's policies regarding sending data to external AI services

If you are processing files that contain personal or sensitive information, review your organisation's AI usage policy before using the default Rakuten AI Gateway endpoint.

---

## 7. Open Source

TheSecondBrain is open source. The full source code is available at:  
https://github.com/ORG028658/TheSecondBrain

You can audit exactly what data is sent and when by reviewing `tui/internal/embeddings/client.go` and the LLM call sites in the codebase.

---

## 8. Changes to This Policy

Updates to this policy will be reflected by a change to the **Last updated** date at the top of this document and committed to the repository. Significant changes will be noted in the commit message.

---

## 9. Contact

For privacy-related questions or concerns:

**A S M Sayem**  
sadat.sayem@rakuten.com  
https://github.com/ORG028658
