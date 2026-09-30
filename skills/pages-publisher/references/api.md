# Pages HTTP API

## Authentication and conventions

Set these locally rather than placing secrets in commands or tracked files:

```bash
export PAGES_BASE_URL="http://localhost:8080"
export PAGES_TOKEN="<workspace-token>"
```

Send `Authorization: Bearer $PAGES_TOKEN`. A workspace token is restricted to its workspace and scopes. The platform `PAGE_TOKEN` can administer all workspaces and should be reserved for bootstrap or emergency administration.

Scopes:

- `read`: list the assigned workspace, projects, pages, and versions.
- `write`: includes read and can upload, publish, and delete pages.
- `admin`: includes read/write and can manage projects and workspace tokens.

Only the platform token can create or delete workspaces.

## Common endpoints

| Operation | Method and path | Minimum scope |
|---|---|---|
| Health | `GET /healthz` | none |
| List workspaces | `GET /api/workspaces` | read |
| Create workspace | `POST /api/workspaces` | platform token |
| List projects | `GET /api/workspaces/{workspace}/projects` | read |
| Create project | `POST /api/workspaces/{workspace}/projects` | admin |
| List pages | `GET /api/workspaces/{workspace}/projects/{project}/pages` | read |
| Upload preview | `POST /api/workspaces/{workspace}/projects/{project}/pages` | write |
| List versions | `GET /api/workspaces/{workspace}/projects/{project}/pages/{slug}/versions` | read |
| Publish version | `POST /api/workspaces/{workspace}/projects/{project}/pages/{slug}/publish` | write |
| Delete page | `DELETE /api/workspaces/{workspace}/projects/{project}/pages/{slug}` | write |

The helper covers the non-destructive common operations:

```bash
bash skills/pages-publisher/scripts/pages-client.sh health
bash skills/pages-publisher/scripts/pages-client.sh workspaces
bash skills/pages-publisher/scripts/pages-client.sh projects acme
bash skills/pages-publisher/scripts/pages-client.sh pages acme website
bash skills/pages-publisher/scripts/pages-client.sh upload acme website home "Home" ./index.html
bash skills/pages-publisher/scripts/pages-client.sh versions acme website home
bash skills/pages-publisher/scripts/pages-client.sh publish acme website home ver_xxx
```

## Create destinations

Create a workspace with the platform token:

```http
POST /api/workspaces
Content-Type: application/json

{"slug":"acme","name":"Acme Team"}
```

Create a project with an admin-capable token:

```http
POST /api/workspaces/acme/projects
Content-Type: application/json

{"slug":"website","name":"Brand Website","description":"Public launch pages"}
```

Create a workspace token:

```http
POST /api/workspaces/acme/tokens
Content-Type: application/json

{"name":"AI Publisher","scopes":["read","write"]}
```

The `secret` in the response is shown once. Store it in a secret manager or local environment variable.

## Upload and publish

For a file upload, use multipart fields `slug`, `title`, and `file`. JSON uploads accept `slug`, `title`, and `html`. Raw HTML is also accepted with `slug` and `title` query parameters.

The HTML limit is 5 MB. A slug may contain lowercase ASCII letters, digits, and hyphens and may be no longer than 63 characters. Titles may be no longer than 120 characters.

The upload response is the page record and includes the new `latestVersion` and `latestPreviewUrl`. Publish that exact version:

```http
POST /api/workspaces/acme/projects/website/pages/home/publish
Content-Type: application/json

{"version":"ver_xxx"}
```

URL forms:

- Preview: `/preview/{workspace}/{project}/{slug}/{version}`
- Stable published page: `/p/{workspace}/{project}/{slug}`

Preview and public page HTML run under the service's restrictive CSP sandbox. External assets must use HTTPS and remain subject to that policy.

## Error handling

The API returns a JSON error body with an error code and message. Preserve that body when reporting failures. On a network timeout during upload or publish, list versions before retrying to avoid creating an unintended extra version or hiding a successful publish.
