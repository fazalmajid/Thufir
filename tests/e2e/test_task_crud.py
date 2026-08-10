"""Basic task lifecycle through the UI: create, edit, complete, delete,
restore. These exercise RxDB's local write path (insert/patch) and the
push-replication round trip to Postgres on every step."""

from playwright.sync_api import expect

from helpers import poll_db_row


def test_create_task_persists_to_server(authed_page, db, test_user_id, marker):
    page = authed_page
    title = f"{marker} buy groceries"

    page.get_by_placeholder("Add a new task...").fill(title)
    page.get_by_placeholder("Add a new task...").press("Enter")

    expect(page.get_by_text(title, exact=True)).to_be_visible()

    row = poll_db_row(
        page, db,
        "SELECT status, is_completed FROM task WHERE user_id = %s::uuid AND title = %s",
        (test_user_id, title),
    )
    assert row == ("inbox", False)


def test_edit_title_and_notes_persist(authed_page, db, test_user_id, marker):
    page = authed_page
    title = f"{marker} original title"
    new_title = f"{marker} edited title"

    page.get_by_placeholder("Add a new task...").fill(title)
    page.get_by_placeholder("Add a new task...").press("Enter")
    row = page.get_by_text(title, exact=True)
    expect(row).to_be_visible()

    row.dblclick()
    title_input = page.get_by_placeholder("Task title")
    title_input.fill(new_title)
    page.get_by_placeholder("Notes (Markdown supported)").fill("some notes here")
    page.get_by_role("button", name="Save").click()

    expect(page.get_by_text(new_title, exact=True)).to_be_visible()

    result = poll_db_row(
        page, db,
        "SELECT title, notes FROM task WHERE user_id = %s::uuid AND title = %s",
        (test_user_id, new_title),
    )
    assert result == (new_title, "some notes here")


def test_complete_task(authed_page, db, test_user_id, marker):
    page = authed_page
    title = f"{marker} finish the report"

    page.get_by_placeholder("Add a new task...").fill(title)
    page.get_by_placeholder("Add a new task...").press("Enter")
    row = page.get_by_text(title, exact=True)
    expect(row).to_be_visible()

    # The completion checkbox is the plain one at the start of the row,
    # distinct from any markdown-checklist checkboxes in expanded notes.
    task_row = row.locator("xpath=ancestor::div[contains(@class,'group')][1]")
    task_row.locator('input[type="checkbox"]').first.click()

    result = poll_db_row(
        page, db,
        "SELECT is_completed, status FROM task WHERE user_id = %s::uuid AND title = %s",
        (test_user_id, title),
        predicate=lambda row: row is not None and row[0] is True,
    )
    assert result == (True, "completed")


def test_delete_and_restore_task(authed_page, db, test_user_id, marker):
    page = authed_page
    title = f"{marker} temporary task"

    page.get_by_placeholder("Add a new task...").fill(title)
    page.get_by_placeholder("Add a new task...").press("Enter")
    row = page.get_by_text(title, exact=True)
    expect(row).to_be_visible()
    task_row = row.locator("xpath=ancestor::div[contains(@class,'group')][1]")

    page.once("dialog", lambda d: d.accept())
    task_row.get_by_role("button", name="Delete task").click()
    expect(page.get_by_text(title, exact=True)).not_to_be_visible()

    poll_db_row(
        page, db,
        "SELECT deleted_at IS NOT NULL FROM task WHERE user_id = %s::uuid AND title = %s",
        (test_user_id, title),
        predicate=lambda row: row is not None and row[0] is True,
    )

    # Click the sidebar link rather than page.goto(): a hard navigation
    # re-bootstraps RxDB from scratch, which against thufirdev's ~6k-task
    # dataset can take well over the default actionability timeout to
    # settle (see tests/e2e/README.md) — and isn't what a real user does
    # when moving between views anyway. This is a normal in-SPA transition.
    page.get_by_role("link", name="Trash").click()
    trashed_row = page.get_by_text(title, exact=True)
    expect(trashed_row).to_be_visible(timeout=10_000)
    # There's no sort guarantee on the trash view and thufirdev has plenty
    # of other deleted tasks, so `.first` would grab whichever trashed item
    # happens to render first — not necessarily this one.
    trashed_row.locator("xpath=ancestor::div[contains(@class,'items-center')][1]").get_by_role("button", name="Restore").click()
    expect(trashed_row).not_to_be_visible()

    poll_db_row(
        page, db,
        "SELECT deleted_at IS NULL FROM task WHERE user_id = %s::uuid AND title = %s",
        (test_user_id, title),
        predicate=lambda row: row is not None and row[0] is True,
    )
