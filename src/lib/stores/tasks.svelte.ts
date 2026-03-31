import { getDB } from '$lib/db/index';
import type { Task, CreateTaskInput, UpdateTaskInput } from '$lib/types/task';

class TaskStore {
	tasks = $state.raw<Task[]>([]);
	loading = $state(false);
	error = $state<string | null>(null);

	// ── Derived views ────────────────────────────────────────────────────────

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
			.sort((a, b) => new Date(b.completed_at!).getTime() - new Date(a.completed_at!).getTime());
	}

	get trashedTasks() {
		return this.tasks.filter((t) => !!t.deleted_at);
	}

	// ── Initialisation (call once after auth) ─────────────────────────────────

	async init() {
		this.loading = true;
		try {
			const db = await getDB();
			// Live query — re-runs whenever any task changes locally or syncs.
			db.tasks
				.find({ selector: {}, sort: [{ sort_order: 'asc' }] })
				.$.subscribe((docs) => {
					this.tasks = docs.map((d) => d.toJSON() as Task);
					this.loading = false;
				});
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to init tasks';
			this.loading = false;
		}
	}

	// ── Mutations ─────────────────────────────────────────────────────────────

	async create(input: Omit<CreateTaskInput, 'id'>) {
		const db = await getDB();
		const now = new Date().toISOString();
		const doc = {
			id: crypto.randomUUID(),
			status: 'inbox' as const,
			is_completed: false,
			is_flagged: false,
			priority: 0,
			sort_order: 0,
			tags: [],
			created_at: now,
			updated_at: now,
			...input
		};
		try {
			await db.tasks.insert(doc);
			return doc;
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to create task';
			throw err;
		}
	}

	async update(id: string, updates: UpdateTaskInput) {
		const db = await getDB();
		const doc = await db.tasks.findOne(id).exec();
		if (!doc) throw new Error(`Task ${id} not found`);
		try {
			await doc.patch(updates);
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to update task';
			throw err;
		}
	}

	async toggleComplete(id: string) {
		const task = this.tasks.find((t) => t.id === id);
		if (!task) return;
		const now = new Date().toISOString();
		return this.update(id, {
			is_completed: !task.is_completed,
			completed_at: !task.is_completed ? now : null,
			status: !task.is_completed ? 'completed' : task.status
		});
	}

	async delete(id: string) {
		const db = await getDB();
		const doc = await db.tasks.findOne(id).exec();
		if (!doc) return;
		try {
			// Soft delete: set deleted_at; RxDB replication pushes the change.
			await doc.patch({ deleted_at: new Date().toISOString() });
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to delete task';
			throw err;
		}
	}

	async restore(id: string) {
		const db = await getDB();
		const doc = await db.tasks.findOne(id).exec();
		if (!doc) return;
		try {
			await doc.patch({ deleted_at: null });
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to restore task';
			throw err;
		}
	}

	async reorder(reorderedTasks: Task[]) {
		const db = await getDB();
		await Promise.all(
			reorderedTasks.map((task, index) =>
				db.tasks.findOne(task.id).exec().then((doc) => doc?.patch({ sort_order: index }))
			)
		);
	}
}

export const taskStore = new TaskStore();
