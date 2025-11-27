export type ProjectStatus = 'active' | 'completed' | 'archived';

export interface Project {
	id: string;
	name: string;
	notes?: string;
	area_id?: string | null;
	status: ProjectStatus;
	deadline?: string | null;
	tags?: string[];
	sort_order: number;
	created_at: string;
	updated_at: string;
	completed_at?: string | null;
	deleted_at?: string | null;
}
