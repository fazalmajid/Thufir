"""Round-trips through the RxDB <-> Postgres replication protocol
(replicateRxCollection + the Go /api/rxdb/*/push and /pull handlers), in
both directions."""

from playwright.sync_api import expect

from helpers import poll_db_row


def test_push_local_write_reaches_postgres(authed_page, db, test_user_id, marker):
    page = authed_page
    title = f"{marker} push replication"

    page.get_by_placeholder("Add a new task...").fill(title)
    page.get_by_placeholder("Add a new task...").press("Enter")
    expect(page.get_by_text(title, exact=True)).to_be_visible()

    poll_db_row(page, db, "SELECT id FROM task WHERE user_id = %s::uuid AND title = %s", (test_user_id, title))


def test_pull_remote_insert_reaches_ui(authed_page, db, test_user_id, marker):
    page = authed_page
    title = f"{marker} pull replication"

    with db.cursor() as cur:
        cur.execute(
            "INSERT INTO task (user_id, title, status) VALUES (%s::uuid, %s, 'inbox') RETURNING id",
            (test_user_id, title),
        )
        assert cur.fetchone() is not None

    # replication.ts triggers an immediate resync on window focus rather than
    # waiting for the 10s poll interval.
    page.evaluate("window.dispatchEvent(new Event('focus'))")

    expect(page.get_by_text(title, exact=True)).to_be_visible(timeout=10_000)
