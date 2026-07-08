# Kandev Instance Debugging

Use this for live-instance state/logs or when a UI/browser repro needs an isolated app.

## Identify Instances

```bash
scripts/pcraft-instances
```

Columns: `PID BACKEND_PORT WEB_PORT AGENTCTL_PORT HOME_DIR REPO_PATH`.

The user's live instance usually has `HOME_DIR=/home/<user>` or backend port `38429`. Never mutate it. Read-only logs/export are allowed.

## Launch Isolated Instance

Use an isolated instance for browser/UI repros and API probing that mutates data:

```bash
# Backend only:
scripts/dev-isolated

# Backend + web frontend:
scripts/dev-isolated --web
```

On a clean checkout, pass `--install` or run `make install` once so frontend dependencies exist.

`dev-isolated` prints a `READY` block with ports, log paths, pidfile, and teardown command. Save the backend/web port and pidfile.

## Logs And Export

Use `pcraft-logs` against the relevant port:

```bash
# Full structured export:
scripts/pcraft-logs <port> --export

# Error-level only:
scripts/pcraft-logs <port> --export --level error

# Tail isolated backend stderr:
scripts/pcraft-logs <port> --tail --lines 120
```

Filter aggressively. Summarize metadata, error count, recent stack traces, and warning patterns that correlate with the report.

Prefer `scripts/pcraft-logs` over generic MCP fetches; it uses plain `curl` and avoids proxy/schema conversion issues.

## Teardown

Tear down only instances you launched:

```bash
scripts/pcraft-kill --pidfile /tmp/pcraft-dev-isolated-<port>.pid --yes
# or:
scripts/pcraft-kill <your_port> --yes
```

`pcraft-kill` refuses protected ports without `--force`. Never use `pkill kandev`.
