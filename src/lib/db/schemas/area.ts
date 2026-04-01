import type { RxJsonSchema } from 'rxdb';
import type { Area } from '$lib/types/area';

export const areaSchema: RxJsonSchema<Area> = {
	version: 0,
	primaryKey: 'id',
	type: 'object',
	properties: {
		id: { type: 'string', maxLength: 36 },
		name: { type: 'string', maxLength: 255 },
		color: { type: ['string', 'null'], maxLength: 7 },
		icon: { type: ['string', 'null'], maxLength: 50 },
		sort_order: { type: 'integer' },
		created_at: { type: 'string' },
		updated_at: { type: 'string' },
		deleted_at: { type: ['string', 'null'] }
	},
	required: ['id', 'name', 'sort_order', 'created_at', 'updated_at'],
	indexes: ['sort_order', 'updated_at']
};
