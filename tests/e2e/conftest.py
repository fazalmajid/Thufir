"""Playwright + pytest harness for Thufir.

Every test gets a *fresh* browser context (fresh IndexedDB, so RxDB starts
empty each time) authenticated via a directly-seeded session row — no
WebAuthn ceremony needed except in test_auth.py, which exercises the real
passkey flow through a CDP virtual authenticator.

Safety: this suite refuses to run against anything but a database whose
connection string contains "thufirdev". It never truncates or bulk-deletes
existing data — it only ever touches rows it created itself, tagged with a
per-test "E2E-<random>" marker, cleaned up after every test.
"""

from __future__ import annotations

import os
import socket
import subprocess
import tempfile
import time
import urllib.request
import uuid
from pathlib import Path

import psycopg
import pytest
from playwright.sync_api import sync_playwright

from helpers import seed_session, delete_session

REPO_ROOT = Path(__file__).resolve().parents[2]
ENV_FILE = Path(__file__).resolve().parent / ".env"


def _load_dotenv(path: Path) -> dict:
    env = {}
    if not path.exists():
        return env
    for line in path.read_text().splitlines():
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        k, v = line.split("=", 1)
        env[k.strip()] = v.strip()
    return env


@pytest.fixture(scope="session")
def db_url() -> str:
    env = _load_dotenv(ENV_FILE)
    url = os.environ.get("TEST_DATABASE_URL") or env.get("TEST_DATABASE_URL")
    if not url:
        pytest.fail(
            "TEST_DATABASE_URL not set. Copy tests/e2e/.env.example to "
            "tests/e2e/.env and fill in a thufirdev connection string."
        )
    if "thufirdev" not in url:
        pytest.fail(
            f"Refusing to run: TEST_DATABASE_URL does not mention 'thufirdev' ({url!r}). "
            "This suite creates and deletes rows and must never point at production."
        )
    return url


@pytest.fixture(scope="session")
def db(db_url):
    conn = psycopg.connect(db_url, autocommit=True)
    yield conn
    conn.close()


@pytest.fixture(scope="session")
def test_user_id(db) -> str:
    with db.cursor() as cur:
        cur.execute("SELECT id::text FROM name ORDER BY created_at LIMIT 1")
        row = cur.fetchone()
    if not row:
        pytest.fail("thufirdev has no rows in `name` — expected it to be cloned from production with an account.")
    return row[0]


def _free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


@pytest.fixture(scope="session")
def server(db_url):
    """Builds the app once and runs it against thufirdev on a scratch port."""
    subprocess.run(["make", "build"], cwd=REPO_ROOT, check=True)

    port = _free_port()
    base_url = f"http://localhost:{port}"
    env = {
        **os.environ,
        "DATABASE_URL": db_url,
        "PORT": str(port),
        "RP_ID": "localhost",
        "RP_ORIGIN": base_url,
    }
    # GO_ENV is deliberately left unset so cookies aren't marked Secure
    # (we're on plain http://localhost).
    env.pop("GO_ENV", None)

    log_file = tempfile.NamedTemporaryFile(prefix="thufir-e2e-server-", suffix=".log", delete=False, mode="w")
    proc = subprocess.Popen(
        [str(REPO_ROOT / "thufir")],
        cwd=REPO_ROOT,
        env=env,
        stdout=log_file,
        stderr=subprocess.STDOUT,
    )

    try:
        deadline = time.time() + 20
        healthy = False
        while time.time() < deadline:
            if proc.poll() is not None:
                break
            try:
                with urllib.request.urlopen(f"{base_url}/health", timeout=1) as r:
                    if r.status == 200:
                        healthy = True
                        break
            except Exception:
                time.sleep(0.2)
        if not healthy:
            log_file.flush()
            log = Path(log_file.name).read_text()
            proc.terminate()
            raise RuntimeError(f"server on {base_url} never became healthy. Log:\n{log}")
        yield base_url
    finally:
        proc.terminate()
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()
        log_file.close()


@pytest.fixture(scope="session")
def playwright_instance():
    with sync_playwright() as p:
        yield p


@pytest.fixture(scope="session")
def browser(playwright_instance):
    b = playwright_instance.chromium.launch()
    yield b
    b.close()


@pytest.fixture
def context(browser):
    ctx = browser.new_context()
    yield ctx
    ctx.close()


@pytest.fixture
def page(context):
    p = context.new_page()
    yield p
    p.close()


@pytest.fixture
def authed_page(server, db, test_user_id, context):
    """A fresh, logged-in page — auth via a directly-seeded session row, not
    the WebAuthn ceremony. Use this for anything that isn't testing login
    itself."""
    session_id = seed_session(context, db, server, test_user_id)
    p = context.new_page()
    p.goto(f"{server}/inbox")
    p.wait_for_load_state("networkidle")
    # thufirdev's cloned dataset is ~6k tasks; RxDB's initial sync pages it in
    # ~100-row batches (60+ round trips), and each batch re-triggers the task
    # list's flip animations client-side even after the network itself goes
    # idle. Give that render churn time to finish before a test starts
    # interacting, or actionability waits can spuriously time out.
    p.wait_for_timeout(2000)
    yield p
    delete_session(db, session_id)


@pytest.fixture
def marker() -> str:
    """A random per-test title/name prefix so cleanup can find exactly what
    this test created, and nothing else."""
    return f"E2E-{uuid.uuid4().hex[:8]}"


@pytest.fixture(autouse=True)
def cleanup_marked_data(db, test_user_id):
    """Deletes only rows tagged with our 'E2E-' marker prefix. Never touches
    pre-existing data."""
    yield
    with db.cursor() as cur:
        cur.execute("DELETE FROM task WHERE user_id = %s::uuid AND title LIKE 'E2E-%%'", (test_user_id,))
        cur.execute("DELETE FROM project WHERE user_id = %s::uuid AND name LIKE 'E2E-%%'", (test_user_id,))
        cur.execute("DELETE FROM area WHERE user_id = %s::uuid AND name LIKE 'E2E-%%'", (test_user_id,))
        cur.execute(
            "DELETE FROM credential WHERE user_id = %s::uuid AND device_name = 'e2e-virtual-authenticator'",
            (test_user_id,),
        )
        cur.execute("DELETE FROM session WHERE user_agent = 'pytest-e2e'")
