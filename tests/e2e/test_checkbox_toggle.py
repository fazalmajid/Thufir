"""Regression coverage for the markdown-checklist checkbox bug fixed on main:
clicking a checkbox in a task's notes would sometimes visually revert and
require a second click. Two independent causes were fixed:

  1. tasks.svelte.ts kept replacing every task's object reference on any
     RxDB emission, forcing needless re-renders (the "wiggle").
  2. RxDB's default conflict handler let a stale replication pull clobber a
     fresh local write; db/index.ts now has a conflictHandler that keeps
     whichever side has the newer updated_at.

This is the test that must keep passing across the RxDB 15->17 migration —
conflictHandler is exactly the API that migration has to rewrite.
"""

import re

from playwright.sync_api import expect

from helpers import poll_db_row


def _create_task_with_checklist(page, db, test_user_id, title: str, notes: str):
    page.get_by_placeholder("Add a new task...").fill(title)
    page.get_by_placeholder("Add a new task...").press("Enter")
    row = page.get_by_text(title, exact=True)
    expect(row).to_be_visible()

    row.dblclick()
    page.get_by_placeholder("Notes (Markdown supported)").fill(notes)
    page.get_by_role("button", name="Save").click()

    # Same push-handler gap noted in test_task_crud.py: the create's push
    # response never echoes the server-assigned updated_at, so editing again
    # (adding notes) right after creating can be flagged as a false
    # conflict. Wait for the edit to land, then force a resync so the
    # checkbox click below starts from a known-fresh local state.
    poll_db_row(page, db, "SELECT notes FROM task WHERE user_id = %s::uuid AND title = %s", (test_user_id, title),
                predicate=lambda row: row is not None and row[0] == notes)
    page.evaluate("window.dispatchEvent(new Event('focus'))")
    page.wait_for_timeout(500)


def _assert_stays_checked(page, checkbox, settle_checks=15, interval_ms=200):
    """Polls across the whole window the old bug reverted in, rather than
    asserting once right after the click."""
    expect(checkbox).to_be_checked(timeout=3000)
    for _ in range(settle_checks):
        page.wait_for_timeout(interval_ms)
        if not checkbox.is_checked():
            raise AssertionError("checkbox reverted to unchecked after initially becoming checked")
    expect(checkbox).to_be_checked()


def test_checking_a_checklist_item_persists_and_does_not_revert(authed_page, db, test_user_id, marker):
    page = authed_page
    title = f"{marker} groceries checklist"
    _create_task_with_checklist(page, db, test_user_id, title, "- [ ] milk\n- [ ] eggs")

    checkbox = page.locator("input.task-checkbox").first
    expect(checkbox).to_be_visible()
    expect(checkbox).not_to_be_checked()

    checkbox.click()
    _assert_stays_checked(page, checkbox)

    with db.cursor() as cur:
        cur.execute("SELECT notes FROM task WHERE user_id = %s::uuid AND title = %s", (test_user_id, title))
        (notes,) = cur.fetchone()
    assert re.search(r"-\s*\[x\]\s*milk", notes, re.IGNORECASE), f"checkbox state never reached Postgres: {notes!r}"


def test_checkbox_survives_a_replication_resync_race(authed_page, db, test_user_id, marker):
    """Specifically targets the conflictHandler fix: fire the same
    window-focus resync that replication.ts listens for right as the click
    lands, reproducing the race the old default conflict handler lost."""
    page = authed_page
    title = f"{marker} race condition checklist"
    _create_task_with_checklist(page, db, test_user_id, title, "- [ ] first item")

    checkbox = page.locator("input.task-checkbox").first
    expect(checkbox).to_be_visible()

    checkbox.click()
    page.evaluate("window.dispatchEvent(new Event('focus'))")
    _assert_stays_checked(page, checkbox)

    with db.cursor() as cur:
        cur.execute("SELECT notes FROM task WHERE user_id = %s::uuid AND title = %s", (test_user_id, title))
        (notes,) = cur.fetchone()
    assert re.search(r"-\s*\[x\]\s*first item", notes, re.IGNORECASE), (
        f"replication resync reverted the checkbox: {notes!r}"
    )
