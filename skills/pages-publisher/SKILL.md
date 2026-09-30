---
name: pages-publisher
description: "Operate this repository's self-hosted Pages service: create workspaces and projects, upload HTML previews, inspect versions, publish exact versions, and deploy or troubleshoot the Go/Docker service. Use for this Pages product, its /api/workspaces/... or /p/... URLs, and Pages workspace tokens; do not use for GitHub Pages, ChatGPT Pages, or pages.houtai.io."
---

# Pages Publisher

Use the Pages API and deployment workflow without exposing credentials or bypassing preview review.

## Choose the workflow

- For listing workspaces, projects, pages, or versions, or for uploading and publishing HTML, read [references/api.md](references/api.md).
- For installing, upgrading, configuring, backing up, or diagnosing the service, read [references/deployment.md](references/deployment.md).
- Prefer [scripts/pages-client.sh](scripts/pages-client.sh) for routine API calls so authentication and URL construction remain consistent.

## Publishing workflow

1. Resolve the service URL, workspace slug, project slug, page slug, and title. Ask only when a missing value materially changes the destination.
2. Obtain a least-privilege workspace token through `PAGES_TOKEN`; prefer `read,write` over the platform token.
3. Validate that the HTML is self-contained enough for its target and is no larger than 5 MB. Slugs use lowercase letters, digits, and hyphens and are at most 63 characters.
4. Upload the HTML. Treat the returned preview URL and version as the authoritative result.
5. Review or report the preview. Uploading never changes the stable published URL.
6. Publish the exact returned version only when the user asked to publish. Report both the version and stable `/p/{workspace}/{project}/{slug}` URL.

Uploading the same slug creates a new immutable version. Do not infer that the latest version should replace the live page.

## Safety rules

- Never print, commit, paste into HTML, or store API tokens in tracked files. Redact tokens in logs and examples.
- Do not create a platform-wide token when a workspace token is sufficient.
- Before retrying a write after a timeout, list the page or its versions to detect whether the first request succeeded.
- Treat deletion of a page, project, workspace, token, or member as destructive. Resolve the exact target and require an explicit user request.
- Preserve the `data` directory during deploys and upgrades; it contains metadata, previews, published content, and version history.
- Keep uploaded HTML untrusted. Do not weaken the service's CSP sandbox unless the user explicitly requests and understands the security impact.

## Response expectations

Lead with the result. For publish operations, include the destination, created version, preview URL, and published URL when available. For deployment work, include the health-check result, listen address, and persistence location without revealing secrets.
