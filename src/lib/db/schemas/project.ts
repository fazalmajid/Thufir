import type { RxJsonSchema } from 'rxdb';
import type { Project } from '$lib/types/project';

export const projectSchema: RxJsonSchema<Project> = {
	version: 0,
	primaryKey: 'id',
	type: 'object',
	properties: {
		id: { type: 'string', maxLength: 36 },
		name: { type: 'string', maxLength: 255 },
		notes: { type: ['string', 'null'] },
		area_id: { type: ['string', 'null'], maxLength: 36 },
		status: { type: 'string', enum: ['active', 'completed', 'archived'] },
		deadline: { type: ['string', 'null'] },
		tags: { type: 'array', items: { type: 'string' } },
		sort_order: { type: 'integer' },
		created_at: { type: 'string' },
		updated_at: { type: 'string' },
		completed_at: { type: ['string', 'null'] },
		deleted_at: { type: ['string', 'null'] }
	},
	required: ['id', 'name', 'status', 'sort_order', 'created_at', 'updated_at'],
	indexes: ['status', 'sort_order', 'updated_at']
};
