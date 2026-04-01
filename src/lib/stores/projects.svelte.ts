import { getDB } from '$lib/db/index';
import type { Project } from '$lib/types/project';

class ProjectStore {
	projects = $state.raw<Project[]>([]);
	loading = $state(false);
	error = $state<string | null>(null);

	get activeProjects() {
		return this.projects.filter((p) => p.status === 'active' && !p.deleted_at);
	}

	get completedProjects() {
		return this.projects.filter((p) => p.status === 'completed' && !p.deleted_at);
	}

	projectsByArea(areaId: string | null) {
		return this.activeProjects.filter((p) => p.area_id === areaId);
	}

	async init() {
		this.loading = true;
		try {
			const db = await getDB();
			db.projects
				.find({ selector: {}, sort: [{ sort_order: 'asc' }] })
				.$.subscribe((docs) => {
					this.projects = docs.map((d) => d.toJSON() as Project);
					this.loading = false;
				});
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to init projects';
			this.loading = false;
		}
	}

	async create(data: Partial<Project>) {
		const db = await getDB();
		const now = new Date().toISOString();
		const doc = {
			id: crypto.randomUUID(),
			name: data.name ?? 'New Project',
			status: 'active' as const,
			sort_order: 0,
			tags: [],
			created_at: now,
			updated_at: now,
			...data
		};
		try {
			await db.projects.insert(doc);
			return doc as Project;
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to create project';
			throw err;
		}
	}
}

export const projectStore = new ProjectStore();
