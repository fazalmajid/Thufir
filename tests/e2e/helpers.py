"""Shared helpers for the Playwright suite: WebAuthn virtual authenticator
setup and direct-DB session seeding (used to bypass the passkey ceremony for
tests that aren't specifically about login)."""

from __future__ import annotations

import time
from urllib.parse import urlparse

VIRTUAL_AUTHENTICATOR_OPTIONS = {
    "protocol": "ctap2",
    "transport": "internal",
    "hasResidentKey": True,
    "hasUserVerification": True,
    "isUserVerified": True,
    "automaticPresenceSimulation": True,
}


def add_virtual_authenticator(context, page):
    """Registers a CDP virtual WebAuthn authenticator on the given context.

    Scoped to the browser context that `page` belongs to — other pages opened
    in the same context share it, pages in other contexts do not.
    """
    cdp = context.new_cdp_session(page)
    cdp.send("WebAuthn.enable")
    result = cdp.send("WebAuthn.addVirtualAuthenticator", {"options": VIRTUAL_AUTHENTICATOR_OPTIONS})
    return cdp, result["authenticatorId"]


def seed_session(context, db, base_url: str, user_id: str) -> str:
    """Inserts a session row directly in Postgres and injects it as a cookie,
    bypassing the WebAuthn login ceremony. Returns the session id so the
    caller can delete it during teardown."""
    with db.cursor() as cur:
        cur.execute(
            """
            INSERT INTO session (user_id, expires_at, user_agent, ua_display, ip_address)
            VALUES (%s::uuid, NOW() + INTERVAL '1 hour', 'pytest-e2e', 'pytest-e2e', '127.0.0.1')
            RETURNING id::text
            """,
            (user_id,),
        )
        session_id = cur.fetchone()[0]

    host = urlparse(base_url).hostname
    context.add_cookies([
        {"name": "session", "value": session_id, "domain": host, "path": "/", "httpOnly": True, "secure": False}
    ])
    return session_id


def delete_session(db, session_id: str) -> None:
    with db.cursor() as cur:
        cur.execute("DELETE FROM session WHERE id = %s::uuid", (session_id,))


def poll_db_row(page, db, query: str, params: tuple, predicate=lambda row: row is not None,
                 timeout_ms: int = 8000, interval_ms: int = 200):
    """Retries a query until `predicate(row)` is true or the timeout expires.

    Replication push is asynchronous (RxDB batches/debounces it internally),
    so a write landing in the UI does not mean it has reached Postgres yet —
    every assertion against the server's copy of the data needs to tolerate
    that lag instead of checking once immediately.
    """
    deadline = time.monotonic() + timeout_ms / 1000
    last = None
    while time.monotonic() < deadline:
        with db.cursor() as cur:
            cur.execute(query, params)
            last = cur.fetchone()
        if predicate(last):
            return last
        page.wait_for_timeout(interval_ms)
    raise AssertionError(f"condition on {query!r} {params!r} never became true; last row = {last!r}")
