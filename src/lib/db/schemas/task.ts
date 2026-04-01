import type { RxJsonSchema } from 'rxdb';
import type { Task } from '$lib/types/task';

export const taskSchema: RxJsonSchema<Task> = {
	version: 0,
	primaryKey: 'id',
	type: 'object',
	properties: {
		id: { type: 'string', maxLength: 36 },
		title: { type: 'string', maxLength: 500 },
		notes: { type: ['string', 'null'] },
		project_id: { type: ['string', 'null'], maxLength: 36 },
		area_id: { type: ['string', 'null'], maxLength: 36 },
		parent_task_id: { type: ['string', 'null'], maxLength: 36 },
		status: {
			type: 'string',
			enum: ['inbox', 'today', 'upcoming', 'anytime', 'someday', 'completed']
		},
		is_completed: { type: 'boolean' },
		completed_at: { type: ['string', 'null'] },
		start_date: { type: ['string', 'null'] },
		deadline: { type: ['string', 'null'] },
		scheduled_date: { type: ['string', 'null'] },
		start_time: { type: ['string', 'null'] },
		reminder_time: { type: ['string', 'null'] },
		is_flagged: { type: 'boolean' },
		priority: { type: 'integer', minimum: 0, maximum: 3 },
		tags: { type: 'array', items: { type: 'string' } },
		sort_order: { type: 'integer' },
		created_at: { type: 'string' },
		updated_at: { type: 'string' },
		deleted_at: { type: ['string', 'null'] }
	},
	required: ['id', 'title', 'status', 'is_completed', 'is_flagged', 'priority', 'sort_order', 'created_at', 'updated_at'],
	indexes: ['status', 'sort_order', 'updated_at']
};
