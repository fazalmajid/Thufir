import { taskAPI } from '$lib/services/api';
import type { Task, CreateTaskInput, UpdateTaskInput } from '$lib/types/task';

// Svelte 5 runes-based store
class TaskStore {
	tasks = $state.raw<Task[]>([]);
	loading = $state(false);
	error = $state<string | null>(null);

	// Derived states for different views
	get inboxTasks() {
		return this.tasks
			.filter((t) => t.status === 'inbox' && !t.is_completed && !t.deleted_at)
			.sort((a, b) => a.sort_order - b.sort_order);
	}

	get todayTasks() {
		return this.tasks
			.filter((t) => t.status === 'today' && !t.is_completed && !t.deleted_at)
			.sort((a, b) => a.sort_order - b.sort_order);
	}

	get upcomingTasks() {
		return this.tasks
			.filter((t) => t.status === 'upcoming' && !t.is_completed && !t.deleted_at)
			.sort((a, b) => a.sort_order - b.sort_order);
	}

	get anytimeTasks() {
		return this.tasks
			.filter((t) => t.status === 'anytime' && !t.is_completed && !t.deleted_at)
			.sort((a, b) => a.sort_order - b.sort_order);
	}

	get somedayTasks() {
		return this.tasks
			.filter((t) => t.status === 'someday' && !t.is_completed && !t.deleted_at)
			.sort((a, b) => a.sort_order - b.sort_order);
	}

	get completedTasks() {
		return this.tasks
			.filter((t) => t.is_completed && !t.deleted_at)
			.sort((a, b) => a.sort_order - b.sort_order);
	}

	get trashedTasks() {
		return this.tasks.filter((t) => t.deleted_at !== null);
	}

	async load(params?: { status?: string; project_id?: string; area_id?: string }) {
		this.loading = true;
		this.error = null;

		try {
			const fetched = await taskAPI.list(params);
			if (params) {
				// Merge fetched tasks into the store, replacing any with matching status/scope
				const fetchedIds = new Set(fetched.map((t) => t.id));
				this.tasks = [
					...this.tasks.filter((t) => !fetchedIds.has(t.id)),
					...fetched
				];
			} else {
				this.tasks = fetched;
			}
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to load tasks';
			console.error('Failed to load tasks:', err);
		} finally {
			this.loading = false;
		}
	}

	async create(input: Omit<CreateTaskInput, 'id'>) {
		const data: CreateTaskInput = {
			id: crypto.randomUUID(),
			...input
		};

		try {
			const newTask = await taskAPI.create(data);
			this.tasks = [...this.tasks, newTask];
			return newTask;
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to create task';
			console.error('Failed to create task:', err);
			throw err;
		}
	}

	async update(id: string, updates: UpdateTaskInput) {
		try {
			const updated = await taskAPI.update(id, updates);
			this.tasks = this.tasks.map((t) => (t.id === id ? updated : t));
			return updated;
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to update task';
			console.error('Failed to update task:', err);
			throw err;
		}
	}

	async toggleComplete(id: string) {
		const task = this.tasks.find((t) => t.id === id);
		if (!task) return;

		return this.update(id, { is_completed: !task.is_completed });
	}

	async delete(id: string) {
		try {
			await taskAPI.delete(id);
			// Update local state to mark as deleted
			this.tasks = this.tasks.map((t) =>
				t.id === id ? { ...t, deleted_at: new Date().toISOString() } : t
			);
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to delete task';
			console.error('Failed to delete task:', err);
			throw err;
		}
	}

	async restore(id: string) {
		try {
			const restored = await taskAPI.restore(id);
			this.tasks = this.tasks.map((t) => (t.id === id ? restored : t));
			return restored;
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to restore task';
			console.error('Failed to restore task:', err);
			throw err;
		}
	}

	async reorder(reorderedTasks: Task[]) {
		// Update local state immediately for responsiveness
		const updates = reorderedTasks.map((task, index) => ({
			id: task.id,
			sort_order: index
		}));

		// Optimistically update local state
		const taskMap = new Map(reorderedTasks.map((task, index) => [task.id, { ...task, sort_order: index }]));
		this.tasks = this.tasks.map((t) => taskMap.get(t.id) || t);

		try {
			await taskAPI.reorder(updates);
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to reorder tasks';
			console.error('Failed to reorder tasks:', err);
			// Reload tasks to restore correct order on error
			await this.load();
			throw err;
		}
	}
}

export const taskStore = new TaskStore();
