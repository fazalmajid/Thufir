"""Exercises the real WebAuthn ceremony end to end, via a Chrome DevTools
Protocol virtual authenticator — not the direct-session-seeding shortcut the
other tests use. This is the one test that proves login itself still works;
everything else in the suite deliberately bypasses it to stay fast and
focused on RxDB behaviour.

The virtual authenticator's key material only exists for the lifetime of
this test's browser context, so it enrolls its own throwaway passkey
(named 'e2e-virtual-authenticator', cleaned up by the autouse fixture)
rather than trying to reuse the account's real one.
"""

from playwright.sync_api import expect

from helpers import add_virtual_authenticator, seed_session, delete_session


def test_login_with_passkey(server, db, test_user_id, context):
    page = context.new_page()
    add_virtual_authenticator(context, page)

    # Bootstrap: log in via a seeded session just long enough to enroll a
    # passkey on our virtual authenticator through the real settings-page
    # ceremony, then drop that session so login has to work for real.
    session_id = seed_session(context, db, server, test_user_id)
    page.goto(f"{server}/settings")
    page.get_by_placeholder("Device name (optional, e.g. iPhone)").fill("e2e-virtual-authenticator")
    page.get_by_role("button", name="Enroll passkey").click()
    expect(page.get_by_text("Passkey enrolled successfully.")).to_be_visible(timeout=10_000)
    delete_session(db, session_id)

    page.goto(f"{server}/login")
    page.get_by_role("button", name="Sign in with passkey").click()
    page.wait_for_url(f"{server}/inbox", timeout=10_000)
