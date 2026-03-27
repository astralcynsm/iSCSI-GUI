# iSCSIGUI

Mode B scaffold for a long-term ZimaOS iSCSI GUI project.

## Layout

- `agent/`: host-level Go service (systemd), owns iSCSI operations.
- `web/gateway/`: web gateway that serves UI and proxies `/api` to agent via Unix socket.
- `web/frontend/`: placeholder for Vue frontend.
- `packaging/raw/`: raw package root for ZimaOS module distribution.
- `deploy/`: install/uninstall scripts for host agent.
- `info/`: product and implementation docs.

## Quick start

1. Build binaries:

```bash
make build-agent
make build-gateway
```

2. Run agent locally (TCP mode):

```bash
make run-agent
curl -s http://127.0.0.1:18080/health
```

3. Build raw package:

```bash
make build-raw
```

This generates `./iscsi_gui.raw`.

## Install on ZimaOS

```bash
# copy file to ZimaOS host first
zpkg install /var/lib/extensions/iscsi_gui.raw
```

Installed services:

- `iscsi-agent` (Unix socket API backend)
- `iscsi-web-gateway` (HTTP `:18081` proxy to agent Unix socket)

## Implemented API (agent)

- `GET /health`
- `GET /api/v1/system/health`
- `GET/POST/DELETE /api/v1/targets`
- `GET/POST/DELETE /api/v1/backstores`
- `GET/POST/DELETE /api/v1/mappings`
- `GET/POST/DELETE /api/v1/acls`
- `GET /api/v1/sessions`

## Notes

- `sessions` endpoint reads from `targetcli sessions` output and supports `target_iqn` filter.
- Frontend module is still pending; current focus is backend API closure and packaging/release workflow.

## GitHub Release Automation

Workflow file:

- `.github/workflows/release-raw.yml`

What it does:

1. Builds `iscsi_gui.raw` on GitHub Actions.
2. Publishes release artifact to GitHub Release.
3. Supports:
   - tag push trigger (`v*` / `V*`)
   - manual trigger (`workflow_dispatch`)
4. Manual trigger can also publish/update the `latest` release tag.

Recommended release flow (`v1.0.0Beta`):

```bash
git tag v1.0.0Beta
git push origin v1.0.0Beta
```

Or use Actions UI:

1. Open `release-raw` workflow
2. Run workflow
3. Set `release_tag=v1.0.0Beta`
4. Keep `publish_latest=true` if you want Mod-Store-compatible latest tag release

After release:

```bash
zpkg update
zpkg list-remote
zpkg install iscsi_gui
```
