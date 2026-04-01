import { createRxDatabase, addRxPlugin } from 'rxdb';
import { RxDBDevModePlugin } from 'rxdb/plugins/dev-mode';
import { getRxStorageDexie } from 'rxdb/plugins/storage-dexie';
import { taskSchema } from './schemas/task';
import { projectSchema } from './schemas/project';
import { areaSchema } from './schemas/area';
import type { RxDatabase, RxCollection } from 'rxdb';
import type { Task } from '$lib/types/task';
import type { Project } from '$lib/types/project';
import type { Area } from '$lib/types/area';

if (import.meta.env.DEV) {
	addRxPlugin(RxDBDevModePlugin);
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
			tasks: { schema: taskSchema },
			projects: { schema: projectSchema },
			areas: { schema: areaSchema }
		});
		return db;
	});

	return dbPromise;
}
