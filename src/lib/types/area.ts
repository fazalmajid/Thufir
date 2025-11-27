export interface Area {
	id: string;
	name: string;
	color?: string | null;
	icon?: string | null;
	sort_order: number;
	created_at: string;
	updated_at: string;
	deleted_at?: string | null;
}
