import { createRxDatabase, addRxPlugin } from 'rxdb';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { getRxStorageDexie } from 'rxdb/plugins/storage-dexie';
import { taskSchema } from './schemas/task';
import { projectSchema } from './schemas/project';
import { areaSchema } from './schemas/area';
import type { RxDatabase, RxCollection, RxConflictHandler } from 'rxdb';
import type { Task } from '$lib/types/task';
import type { Project } from '$lib/types/project';
import type { Area } from '$lib/types/area';

if (import.meta.env.DEV) {
	addRxPlugin(RxDBDevModePlugin);
}

// RxDB's built-in default conflict handler always discards the local (fork)
// state and keeps whatever master last reported — so a local edit that races
// with a replication pull (poll or window-focus resync) can get silently
// reverted. Resolve by updated_at instead: whichever side was written more
// recently wins, only falling back to master when timestamps can't be compared.
//
// The server always overwrites updated_at with its own clock on write (see
// upsert.go), and echoes the resulting row back through this same conflict
// channel because RxDB's push protocol has no other way to report "accepted,
// but here's the corrected timestamp" (see push.go). That echo is not a real
// conflict — every other field is identical to what we just pushed — so it's
// detected separately and always accepted, rather than going through the
// timestamp race above (which clock skew between client and server could
// otherwise resolve the wrong way).
function newerWinsConflictHandler<T extends { updated_at: string }>(): RxConflictHandler<T> {
	return {
		// Must be synchronous and fast — RxDB calls this on every potential
		// conflict before ever calling resolve().
		isEqual: (a, b) => JSON.stringify(a) === JSON.stringify(b),
		resolve: async (i) => {
			const { updated_at: _localTs, ...localRest } = i.newDocumentState;
			const { updated_at: _masterTs, ...masterRest } = i.realMasterState;
			if (JSON.stringify(localRest) === JSON.stringify(masterRest)) {
				return i.realMasterState;
			}

			const localTime = Date.parse(i.newDocumentState.updated_at);
			const masterTime = Date.parse(i.realMasterState.updated_at);
			const localWins = !Number.isNaN(localTime) && !Number.isNaN(masterTime) && localTime > masterTime;
			return localWins ? i.newDocumentState : i.realMasterState;
		}
	};
}

export type ThufirCollections = {
	tasks: RxCollection<Task>;
	projects: RxCollection<Project>;
	areas: RxCollection<Area>;
};

export type ThufirDatabase = RxDatabase<ThufirCollections>;

let dbPromise: Promise<ThufirDatabase> | null = null;

export async function getDB(): Promise<ThufirDatabase> {
	if (dbPromise) return dbPromise;

	dbPromise = createRxDatabase<ThufirCollections>({
		name: 'thufirdb',
		storage: getRxStorageDexie(),
		// ignoreDuplicate is dev-mode-only as of RxDB 16 and throws in
		// production. closeDuplicates actually closes the stale instance
		// (e.g. left over from HMR re-running this module) instead of just
		// suppressing the check, and works in both dev and prod.
		closeDuplicates: true
	}).then(async (db) => {
		await db.addCollections({
			tasks: { schema: taskSchema, conflictHandler: newerWinsConflictHandler<Task>() },
			projects: { schema: projectSchema, conflictHandler: newerWinsConflictHandler<Project>() },
			areas: { schema: areaSchema, conflictHandler: newerWinsConflictHandler<Area>() }
		});
		return db;
	});

	return dbPromise;
}
