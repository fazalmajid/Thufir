import { areaAPI } from '$lib/services/api';
import type { Area } from '$lib/types/area';

class AreaStore {
	areas = $state<Area[]>([]);
	loading = $state(false);
	error = $state<string | null>(null);

	get activeAreas() {
		return this.areas.filter((a) => !a.deleted_at).sort((a, b) => a.sort_order - b.sort_order);
	}

	async load() {
		this.loading = true;
		this.error = null;

		try {
			this.areas = await areaAPI.list();
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to load areas';
			console.error('Failed to load areas:', err);
		} finally {
			this.loading = false;
		}
	}

	async create(data: Partial<Area>) {
		try {
			const newArea = await areaAPI.create(data);
			this.areas = [...this.areas, newArea];
			return newArea;
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to create area';
			console.error('Failed to create area:', err);
			throw err;
		}
	}
}

export const areaStore = new AreaStore();
