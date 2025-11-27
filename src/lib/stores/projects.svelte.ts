import { projectAPI } from '$lib/services/api';
import type { Project } from '$lib/types/project';

class ProjectStore {
	projects = $state<Project[]>([]);
	loading = $state(false);
	error = $state<string | null>(null);

	get activeProjects() {
		return this.projects.filter(
			(p) => p.status === 'active' && !p.deleted_at
		);
	}

	get completedProjects() {
		return this.projects.filter(
			(p) => p.status === 'completed' && !p.deleted_at
		);
	}

	// Get projects grouped by area
	projectsByArea(areaId: string | null) {
		return this.activeProjects.filter((p) => p.area_id === areaId);
	}

	async load() {
		this.loading = true;
		this.error = null;

		try {
			this.projects = await projectAPI.list();
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to load projects';
			console.error('Failed to load projects:', err);
		} finally {
			this.loading = false;
		}
	}

	async create(data: Partial<Project>) {
		try {
			const newProject = await projectAPI.create(data);
			this.projects = [...this.projects, newProject];
			return newProject;
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to create project';
			console.error('Failed to create project:', err);
			throw err;
		}
	}
}

export const projectStore = new ProjectStore();
