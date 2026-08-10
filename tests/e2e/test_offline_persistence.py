"""Local-first behaviour: RxDB's Dexie-backed IndexedDB storage must survive
a reload with no network, and a write made while offline must sync once
connectivity returns. This is the most direct exercise of the RxStorage
adapter (getRxStorageDexie), which is one of the two axes of the pending
RxDB 15->17 migration."""

from playwright.sync_api import expect

from helpers import poll_db_row


def test_data_survives_reload_with_backend_unreachable(authed_page, db, test_user_id, marker):
    """Simulates "no connectivity to the server" rather than a literal
    browser-level offline state: a fresh Playwright context has never had a
    chance to install/activate the app's service worker, so a real
    `context.set_offline(True)` blocks the initial document fetch itself and
    can't reload at all — which isn't the thing this test cares about. What
    matters for the RxDB migration is that the app's *data* layer (Dexie/
    IndexedDB) serves the last-synced state without the API being reachable,
    so only /api/ is cut here."""
    page = authed_page
    title = f"{marker} offline reload"

    page.get_by_placeholder("Add a new task...").fill(title)
    page.get_by_placeholder("Add a new task...").press("Enter")
    expect(page.get_by_text(title, exact=True)).to_be_visible()

    # Let the push replication land before cutting the API, otherwise this
    # would just be testing "did the write happen at all".
    poll_db_row(page, db, "SELECT 1 FROM task WHERE user_id = %s::uuid AND title = %s", (test_user_id, title))

    # Block only the replication endpoints, not /api/auth/me — the root
    # layout's auth guard (+layout.svelte) redirects to /login if that call
    # fails, which is a real but separate behaviour from what this test
    # cares about (does RxDB's local storage serve data without a fresh
    # pull).
    page.route("**/api/rxdb/**", lambda route: route.abort())
    try:
        page.reload()
        page.wait_for_load_state("domcontentloaded")
        expect(page.get_by_text(title, exact=True)).to_be_visible(timeout=10_000)
    finally:
        page.unroute("**/api/rxdb/**")


def test_write_made_while_offline_syncs_after_reconnect(authed_page, db, test_user_id, marker):
    page = authed_page
    title = f"{marker} created offline"

    page.context.set_offline(True)
    try:
        page.get_by_placeholder("Add a new task...").fill(title)
        page.get_by_placeholder("Add a new task...").press("Enter")
        expect(page.get_by_text(title, exact=True)).to_be_visible()

        with db.cursor() as cur:
            cur.execute("SELECT 1 FROM task WHERE user_id = %s::uuid AND title = %s", (test_user_id, title))
            assert cur.fetchone() is None, "write reached Postgres while the browser context was offline"
    finally:
        page.context.set_offline(False)

    page.evaluate("window.dispatchEvent(new Event('focus'))")

    with db.cursor() as cur:
        for _ in range(30):
            cur.execute("SELECT 1 FROM task WHERE user_id = %s::uuid AND title = %s", (test_user_id, title))
            if cur.fetchone():
                return
            page.wait_for_timeout(300)
    raise AssertionError("offline-created task never synced to Postgres after reconnecting")
