import type { Task, CreateTaskInput, UpdateTaskInput } from '$lib/types/task';
import type { Project } from '$lib/types/project';
import type { Area } from '$lib/types/area';
import type { ChecklistItem } from '$lib/types/checklist';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:3001';

async function fetchAPI(endpoint: string, options?: RequestInit) {
	const response = await fetch(`${API_URL}${endpoint}`, {
		...options,
		headers: {
			'Content-Type': 'application/json',
			...options?.headers
		}
	});

	if (!response.ok) {
		const error = await response.json().catch(() => ({ error: 'Unknown error' }));
		throw new Error(error.error || `HTTP ${response.status}`);
	}

	return response.json();
}

// Task API
export const taskAPI = {
	async list(): Promise<Task[]> {
		return fetchAPI('/api/tasks');
	},

	async get(id: string): Promise<Task> {
		return fetchAPI(`/api/tasks/${id}`);
	},

	async create(data: CreateTaskInput): Promise<Task> {
		return fetchAPI('/api/tasks', {
			method: 'POST',
			body: JSON.stringify(data)
		});
	},

	async update(id: string, data: UpdateTaskInput): Promise<Task> {
		return fetchAPI(`/api/tasks/${id}`, {
			method: 'PATCH',
			body: JSON.stringify(data)
		});
	},

	async delete(id: string): Promise<{ success: boolean }> {
		return fetchAPI(`/api/tasks/${id}`, {
			method: 'DELETE'
		});
	},

	async toggleComplete(id: string, isCompleted: boolean): Promise<Task> {
		return this.update(id, { is_completed: isCompleted });
	},

	async restore(id: string): Promise<Task> {
		return fetchAPI(`/api/tasks/${id}/restore`, {
			method: 'POST'
		});
	}
};

// Project API
export const projectAPI = {
	async list(): Promise<Project[]> {
		return fetchAPI('/api/projects');
	},

	async create(data: Partial<Project>): Promise<Project> {
		return fetchAPI('/api/projects', {
			method: 'POST',
			body: JSON.stringify({
				id: crypto.randomUUID(),
				...data
			})
		});
	}
};

// Area API
export const areaAPI = {
	async list(): Promise<Area[]> {
		return fetchAPI('/api/areas');
	},

	async create(data: Partial<Area>): Promise<Area> {
		return fetchAPI('/api/areas', {
			method: 'POST',
			body: JSON.stringify({
				id: crypto.randomUUID(),
				...data
			})
		});
	}
};

// Checklist API
export const checklistAPI = {
	async create(data: Partial<ChecklistItem>): Promise<ChecklistItem> {
		return fetchAPI('/api/checklist-items', {
			method: 'POST',
			body: JSON.stringify({
				id: crypto.randomUUID(),
				...data
			})
		});
	},

	async update(id: string, data: Partial<ChecklistItem>): Promise<ChecklistItem> {
		return fetchAPI(`/api/checklist-items/${id}`, {
			method: 'PATCH',
			body: JSON.stringify(data)
		});
	}
};
