export type TaskStatus = 'inbox' | 'today' | 'upcoming' | 'anytime' | 'someday' | 'completed';

export interface Task {
	id: string;
	title: string;
	notes?: string;

	// Hierarchy
	project_id?: string | null;
	area_id?: string | null;
	parent_task_id?: string | null;

	// Status & timing
	status: TaskStatus;
	is_completed: boolean;
	completed_at?: string | null;

	// Dates
	start_date?: string | null;
	deadline?: string | null;
	scheduled_date?: string | null;

	// Time
	start_time?: string | null;
	reminder_time?: string | null;

	// Flags & priority
	is_flagged: boolean;
	priority: number;

	// Tags
	tags?: string[];

	// Ordering
	sort_order: number;

	// Checklist summary
	checklist_total: number;
	checklist_completed: number;

	// Metadata
	created_at: string;
	updated_at: string;
	deleted_at?: string | null;
}

export interface CreateTaskInput {
	id: string;
	title: string;
	notes?: string;
	project_id?: string | null;
	area_id?: string | null;
	status?: TaskStatus;
	scheduled_date?: string | null;
	deadline?: string | null;
	tags?: string[];
	is_flagged?: boolean;
	priority?: number;
}

export interface UpdateTaskInput {
	title?: string;
	notes?: string;
	project_id?: string | null;
	area_id?: string | null;
	status?: TaskStatus;
	scheduled_date?: string | null;
	deadline?: string | null;
	tags?: string[];
	is_flagged?: boolean;
	priority?: number;
	is_completed?: boolean;
	sort_order?: number;
}
