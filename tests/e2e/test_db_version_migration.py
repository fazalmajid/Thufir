"""Regression test for RxDB error DM5: a local database created by an older
RxDB major version can't be opened by newer code without a storage
migration. Hit in production after the 15->17 RxDB migration — any browser
that had used the app before that deploy showed a permanent "DB init
failed: ... DM5" error on next load, because RxDB stamps the database with
the RxDB version that created it and refuses to open it otherwise.

db/index.ts now catches DM5 specifically and wipes+recreates the local
database rather than attempting RxDB's official (and much heavier) storage
migration path — safe because Postgres, not the local RxDB copy, is the
source of truth; the local database is a replicated cache.

There's no way to trigger a real DM5 by installing an old RxDB version
alongside the current one just for a test, so this reproduces the exact
condition RxDB itself checks: it directly overwrites the local database's
internal version-stamp document with an old version string, which is
exactly what a browser that last opened the app on RxDB 15.x would have
stored.
"""

from playwright.sync_api import expect


def test_recovers_from_incompatible_local_db_version(authed_page, marker):
    page = authed_page
    title = f"{marker} version stamp recovery"

    page.get_by_placeholder("Add a new task...").fill(title)
    page.get_by_placeholder("Add a new task...").press("Enter")
    expect(page.get_by_text(title, exact=True)).to_be_visible()

    downgraded = page.evaluate("""async () => {
        const req = indexedDB.open('rxdb-dexie-thufirdb--0--_rxdb_internal');
        return await new Promise((resolve, reject) => {
            req.onsuccess = () => {
                const idb = req.result;
                const tx = idb.transaction('docs', 'readwrite');
                const store = tx.objectStore('docs');
                const getReq = store.get('storage-token|storageToken');
                getReq.onsuccess = () => {
                    const doc = getReq.result;
                    if (!doc) { resolve(false); return; }
                    doc.data.rxdbVersion = '15.39.0';
                    const putReq = store.put(doc);
                    putReq.onsuccess = () => resolve(true);
                    putReq.onerror = () => reject(putReq.error);
                };
                getReq.onerror = () => reject(getReq.error);
            };
            req.onerror = () => reject(req.error);
        });
    }""")
    assert downgraded, "couldn't find the storage-token doc to downgrade — RxDB's internal store layout may have changed"

    page.reload()
    page.wait_for_load_state("networkidle")

    # Must recover on its own, not get stuck on the "DB init failed" banner
    # that shipped to production before this fix. Generous timeout: recovery
    # means a full local wipe + fresh paginated resync of thufirdev's ~6k
    # tasks from scratch (see README's note on initial-sync churn), not just
    # a quick retry.
    expect(page.get_by_text("DB init failed", exact=False)).not_to_be_visible(timeout=30_000)
    expect(page.get_by_text(title, exact=True)).to_be_visible(timeout=30_000)
