# Thufir E2E suite

Playwright + pytest tests driving the real app (built binary + real
Postgres) through a browser. Built to validate behaviour across the pending
RxDB 15 -> 17 migration: local persistence (Dexie storage), the replication
push/pull round trip, and the checkbox-revert regression fixed on `main`.

## Setup

```
cd tests/e2e
uv sync
uv run playwright install chromium
cp .env.example .env   # fill in TEST_DATABASE_URL
```

`TEST_DATABASE_URL` must point at a disposable clone of the production
database — the suite refuses to start unless the connection string contains
`thufirdev`. It creates and deletes real rows (tagged with a per-test
`E2E-<random>` prefix) and never truncates or touches anything else.

## Running

```
cd tests/e2e
uv run pytest -v
```

This runs `make build` in the repo root once per session, starts the
resulting binary against `TEST_DATABASE_URL` on a scratch port, and tears it
down afterward. No dev server needs to be running separately.

Most tests authenticate by seeding a session row directly in Postgres and
injecting it as a cookie — fast, and avoids WebAuthn ceremony flakiness.
`test_auth.py` is the exception: it drives the real passkey ceremony through
a Chrome DevTools Protocol virtual authenticator, enrolling its own
throwaway credential (`e2e-virtual-authenticator`, cleaned up automatically)
rather than touching the account's real passkey.

## Known gotchas (found while building this)

- **thufirdev is a ~6k-task real dataset.** Every fresh browser context does
  a full initial RxDB sync (paginated ~100 rows/request), which visibly
  churns the DOM for a couple of seconds after page load — `authed_page` in
  `conftest.py` waits this out. If tests start flaking on actionability
  timeouts, that settle window is the first thing to look at.
- **The push handler's success response never echoes back the
  server-assigned `updated_at`** (`server/internal/sync/push.go`). RxDB's
  local "assumedMasterState" for a just-created or just-edited document is
  therefore stale until the next pull corrects it, and a second edit inside
  that window gets flagged as a false conflict. Tests that mutate the same
  task twice in quick succession (create-then-edit, delete-then-restore)
  work around it by forcing a resync (`window.dispatchEvent(new
  Event('focus'))`) and waiting for it to land in between. This is a real,
  pre-existing bug independent of the RxDB version — worth fixing
  separately, not patched here.
- Playwright pinned to `1.47.0`: newer Playwright/Chrome-for-Testing
  combinations were seen returning incorrect `is_enabled`/`is_editable`
  results for elements that were genuinely interactable (confirmed via
  direct DOM inspection and `force=True`). Revisit the pin if it's ever
  bumped.
