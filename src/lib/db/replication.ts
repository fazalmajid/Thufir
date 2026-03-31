import { replicateRxCollection } from 'rxdb/plugins/replication';
import { interval } from 'rxjs';
import type { ThufirDatabase } from './index';
import { syncError, lastSync } from '$lib/stores/sync';

const API_BASE = '';
const POLL_INTERVAL_MS = 10_000;

async function pullHandler(collection: string) {
	return async (lastCheckpoint: unknown, batchSize: number) => {
		const res = await fetch(`${API_BASE}/api/rxdb/${collection}/pull`, {
			method: 'POST',
			credentials: 'include',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ checkpoint: lastCheckpoint ?? null, limit: batchSize })
		});
		if (!res.ok) throw new Error(`Pull ${collection} failed: ${res.status}`);
		syncError.set(null);
		lastSync.set(new Date());
		return res.json();
	};
}

async function pushHandler(collection: string) {
	return async (rows: unknown[]) => {
		const res = await fetch(`${API_BASE}/api/rxdb/${collection}/push`, {
			method: 'POST',
			credentials: 'include',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(rows)
		});
		if (!res.ok) throw new Error(`Push ${collection} failed: ${res.status}`);
		return res.json();
	};
}

export async function startReplication(db: ThufirDatabase) {
	const collections = ['tasks', 'projects', 'areas'] as const;
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	const states: any[] = [];

	for (const name of collections) {
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		const col = (db as any)[name];
		if (!col) throw new Error(`RxDB collection "${name}" not found`);

		const state = replicateRxCollection({
			collection: col,
			replicationIdentifier: `thufir-${name}-v1`,
			pull: {
				handler: await pullHandler(name),
				batchSize: 100
			},
			push: {
				handler: await pushHandler(name),
				batchSize: 50
			},
			live: true,
			retryTime: 10_000
		});

		state.error$.subscribe((err: unknown) => {
			const msg = err instanceof Error ? err.message : String(err);
			syncError.set(msg);
		});

		states.push(state);
	}

	const triggerSync = () => states.forEach((s) => s.reSync());

	// Poll every POLL_INTERVAL_MS so changes from other devices are picked up
	// even when this client has no local writes to push.
	interval(POLL_INTERVAL_MS).subscribe(triggerSync);

	// Also sync immediately when the tab/window regains focus.
	if (typeof window !== 'undefined') {
		window.addEventListener('focus', triggerSync);
	}
}
