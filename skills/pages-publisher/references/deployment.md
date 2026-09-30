# Deployment and operations

## Runtime configuration

| Variable | Default | Purpose |
|---|---|---|
| `PAGE_TOKEN` | `dev-token` | Platform automation and emergency administration token |
| `PAGE_ADDR` | `:8080` | HTTP listen address |
| `PAGE_DATA` | `./data` | Persistent metadata and page storage |

Always replace `dev-token` in production. Keep secrets in an untracked `.env`, environment manager, or secret store.

## Docker Compose

From the repository root:

```bash
docker compose up -d --build
curl --fail http://localhost:8080/healthz
docker compose ps
```

The supplied Compose file maps host port 8080 and mounts `./data` at `/app/data`. To expose a different host port, change only the host side, for example `8082:8080`; the container continues listening on 8080.

Before an upgrade:

1. Confirm the absolute deployment directory and active Compose project.
2. Back up `.env` and `data` without printing secret values.
3. Pull or copy the intended revision.
4. Run `docker compose up -d --build`.
5. Verify `/healthz`, container status, and one known published page.

Never delete or replace `data` during a routine deployment. Do not run broad cleanup commands that could remove unrelated images, volumes, or services.

## Native Go process

For local development:

```bash
PAGE_TOKEN='<strong-secret>' PAGE_ADDR=':8080' PAGE_DATA='./data' go run .
```

For production, use a service manager, a dedicated unprivileged account, a persistent absolute `PAGE_DATA` path, and a TLS-terminating reverse proxy.

## Verification and diagnosis

Use this order:

1. `GET /healthz` confirms the HTTP process responds.
2. Inspect process or Compose status and recent application logs.
3. Confirm the requested host port maps to the container's 8080 port.
4. Confirm the data mount points to the expected persistent directory.
5. Test an authenticated read using a least-privilege token.
6. Test an existing published URL.

Do not include `.env` contents, bearer tokens, password hashes, or session cookies in diagnostic output.
