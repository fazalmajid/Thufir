import { getDB } from '$lib/db/index';
import type { Area } from '$lib/types/area';

class AreaStore {
	areas = $state.raw<Area[]>([]);
	loading = $state(false);
	error = $state<string | null>(null);

	get activeAreas() {
		return this.areas.filter((a) => !a.deleted_at).sort((a, b) => a.sort_order - b.sort_order);
	}

	async init() {
		this.loading = true;
		try {
			const db = await getDB();
			db.areas
				.find({ selector: {}, sort: [{ sort_order: 'asc' }] })
				.$.subscribe((docs) => {
					this.areas = docs.map((d) => d.toJSON() as Area);
					this.loading = false;
				});
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to init areas';
			this.loading = false;
		}
	}

	async create(data: Partial<Area>) {
		const db = await getDB();
		const now = new Date().toISOString();
		const doc = {
			id: crypto.randomUUID(),
			name: data.name ?? 'New Area',
			sort_order: 0,
			created_at: now,
			updated_at: now,
			...data
		};
		try {
			await db.areas.insert(doc);
			return doc as Area;
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to create area';
			throw err;
		}
	}
}

export const areaStore = new AreaStore();
