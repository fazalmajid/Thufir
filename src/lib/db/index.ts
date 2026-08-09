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
function newerWinsConflictHandler<T extends { updated_at: string }>(): RxConflictHandler<T> {
	return async (i) => {
		if (JSON.stringify(i.newDocumentState) === JSON.stringify(i.realMasterState)) {
			return { isEqual: true };
		}
		const localTime = Date.parse(i.newDocumentState.updated_at);
		const masterTime = Date.parse(i.realMasterState.updated_at);
		const localWins = !Number.isNaN(localTime) && !Number.isNaN(masterTime) && localTime > masterTime;
		return {
			isEqual: false,
			documentData: localWins ? i.newDocumentState : i.realMasterState
		};
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
		ignoreDuplicate: true
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
