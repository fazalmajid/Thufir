export interface ChecklistItem {
	id: string;
	task_id: string;
	title: string;
	is_completed: boolean;
	sort_order: number;
	created_at: string;
	updated_at: string;
	completed_at?: string | null;
	deleted_at?: string | null;
}
