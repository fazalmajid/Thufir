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
- **Fixed:** the push handler's success response used to never echo back the
  server-assigned `updated_at` (`server/internal/sync/push.go`), so RxDB's
  local "assumedMasterState" for a just-created or just-edited document
  stayed stale until the next pull corrected it, and a second edit inside
  that window got flagged as a false conflict. `upsertTask`/`upsertProject`/
  `upsertArea` now `RETURNING` the written row and the push handler echoes
  it back through the conflicts channel — RxDB's push protocol has no other
  way to say "accepted, but here's the corrected state". The client's
  `conflictHandler` (`src/lib/db/index.ts`) recognizes an otherwise-identical
  document as a timestamp correction rather than a real conflict. Verified
  directly: create-then-immediately-delete with no settle wait now succeeds
  (it didn't before). The tests no longer need the resync-and-wait
  workaround this bug used to require.
- Playwright pinned to `1.47.0`: newer Playwright/Chrome-for-Testing
  combinations were seen returning incorrect `is_enabled`/`is_editable`
  results for elements that were genuinely interactable (confirmed via
  direct DOM inspection and `force=True`). Revisit the pin if it's ever
  bumped.
