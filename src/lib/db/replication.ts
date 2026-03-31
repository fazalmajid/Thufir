import { replicateRxCollection } from 'rxdb/plugins/replication';
import type { ThufirDatabase } from './index';

// When deployed, API and frontend share the same origin so API_BASE is empty.
// During `npm run dev`, Vite proxies /api/* so this also works.
const API_BASE = '';

async function pullHandler(collection: string) {
	return async (lastCheckpoint: unknown, batchSize: number) => {
		const res = await fetch(`${API_BASE}/api/rxdb/${collection}/pull`, {
			method: 'POST',
			credentials: 'include',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ checkpoint: lastCheckpoint ?? null, limit: batchSize })
		});
		if (!res.ok) throw new Error(`Pull ${collection} failed: ${res.status}`);
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

	for (const name of collections) {
		replicateRxCollection({
			collection: db[name] as any, // eslint-disable-line @typescript-eslint/no-explicit-any
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
			retryTime: 10_000 // retry/re-poll every 10 seconds
		});
	}
}
