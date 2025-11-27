import { Hono } from 'hono';
import { cors } from 'hono/cors';
import { zValidator } from '@hono/zod-validator';
import { z } from 'zod';
import postgres from 'postgres';

const app = new Hono();

// Database connection
const sql = postgres(process.env.DATABASE_URL!, {
	max: 10,
	idle_timeout: 20,
	connect_timeout: 10
});

// Middleware
app.use('/*', cors({
	origin: ['http://localhost:5173', 'http://localhost:4173'],
	credentials: true
}));

// Validation schemas
const createTaskSchema = z.object({
	id: z.string().uuid(),
	title: z.string().min(1).max(500),
	notes: z.string().optional(),
	project_id: z.string().uuid().nullable().optional(),
	area_id: z.string().uuid().nullable().optional(),
	status: z.enum(['inbox', 'today', 'upcoming', 'anytime', 'someday', 'completed']).default('inbox'),
	scheduled_date: z.string().nullable().optional(),
	deadline: z.string().nullable().optional(),
	tags: z.array(z.string()).optional(),
	is_flagged: z.boolean().optional(),
	priority: z.number().min(0).max(3).optional()
});

const updateTaskSchema = z.object({
	title: z.string().min(1).max(500).optional(),
	notes: z.string().optional(),
	project_id: z.string().uuid().nullable().optional(),
	area_id: z.string().uuid().nullable().optional(),
	status: z.enum(['inbox', 'today', 'upcoming', 'anytime', 'someday', 'completed']).optional(),
	scheduled_date: z.string().nullable().optional(),
	deadline: z.string().nullable().optional(),
	tags: z.array(z.string()).optional(),
	is_flagged: z.boolean().optional(),
	priority: z.number().min(0).max(3).optional(),
	is_completed: z.boolean().optional(),
	sort_order: z.number().optional()
});

// Health check
app.get('/health', (c) => {
	return c.json({ status: 'ok' });
});

// Task routes
app.get('/api/tasks', async (c) => {
	try {
		const tasks = await sql`
			SELECT * FROM tasks
			WHERE deleted_at IS NULL
			ORDER BY sort_order ASC, created_at DESC
		`;
		return c.json(tasks);
	} catch (error) {
		console.error('Error fetching tasks:', error);
		return c.json({ error: 'Failed to fetch tasks' }, 500);
	}
});

app.get('/api/tasks/:id', async (c) => {
	const id = c.req.param('id');

	try {
		const [task] = await sql`
			SELECT * FROM tasks
			WHERE id = ${id} AND deleted_at IS NULL
		`;

		if (!task) {
			return c.json({ error: 'Task not found' }, 404);
		}

		return c.json(task);
	} catch (error) {
		console.error('Error fetching task:', error);
		return c.json({ error: 'Failed to fetch task' }, 500);
	}
});

app.post('/api/tasks', zValidator('json', createTaskSchema), async (c) => {
	const task = c.req.valid('json');

	try {
		const [created] = await sql`
			INSERT INTO tasks ${sql(task)}
			RETURNING *
		`;

		return c.json(created, 201);
	} catch (error) {
		console.error('Error creating task:', error);
		return c.json({ error: 'Failed to create task' }, 500);
	}
});

app.patch('/api/tasks/:id', zValidator('json', updateTaskSchema), async (c) => {
	const id = c.req.param('id');
	const updates = c.req.valid('json');

	try {
		// Handle completion
		if (updates.is_completed !== undefined) {
			if (updates.is_completed) {
				updates.completed_at = new Date().toISOString();
				updates.status = 'completed';
			} else {
				updates.completed_at = null;
			}
		}

		const [updated] = await sql`
			UPDATE tasks
			SET ${sql(updates)}, updated_at = NOW()
			WHERE id = ${id} AND deleted_at IS NULL
			RETURNING *
		`;

		if (!updated) {
			return c.json({ error: 'Task not found' }, 404);
		}

		return c.json(updated);
	} catch (error) {
		console.error('Error updating task:', error);
		return c.json({ error: 'Failed to update task' }, 500);
	}
});

app.delete('/api/tasks/:id', async (c) => {
	const id = c.req.param('id');

	try {
		await sql`
			UPDATE tasks
			SET deleted_at = NOW()
			WHERE id = ${id}
		`;

		return c.json({ success: true });
	} catch (error) {
		console.error('Error deleting task:', error);
		return c.json({ error: 'Failed to delete task' }, 500);
	}
});

app.post('/api/tasks/:id/restore', async (c) => {
	const id = c.req.param('id');

	try {
		const [restored] = await sql`
			UPDATE tasks
			SET deleted_at = NULL
			WHERE id = ${id}
			RETURNING *
		`;

		if (!restored) {
			return c.json({ error: 'Task not found' }, 404);
		}

		return c.json(restored);
	} catch (error) {
		console.error('Error restoring task:', error);
		return c.json({ error: 'Failed to restore task' }, 500);
	}
});

// Project routes
app.get('/api/projects', async (c) => {
	try {
		const projects = await sql`
			SELECT * FROM projects
			WHERE deleted_at IS NULL
			ORDER BY sort_order ASC, created_at DESC
		`;
		return c.json(projects);
	} catch (error) {
		console.error('Error fetching projects:', error);
		return c.json({ error: 'Failed to fetch projects' }, 500);
	}
});

app.post('/api/projects', async (c) => {
	const data = await c.req.json();

	try {
		const [project] = await sql`
			INSERT INTO projects ${sql(data)}
			RETURNING *
		`;

		return c.json(project, 201);
	} catch (error) {
		console.error('Error creating project:', error);
		return c.json({ error: 'Failed to create project' }, 500);
	}
});

// Area routes
app.get('/api/areas', async (c) => {
	try {
		const areas = await sql`
			SELECT * FROM areas
			WHERE deleted_at IS NULL
			ORDER BY sort_order ASC
		`;
		return c.json(areas);
	} catch (error) {
		console.error('Error fetching areas:', error);
		return c.json({ error: 'Failed to fetch areas' }, 500);
	}
});

app.post('/api/areas', async (c) => {
	const data = await c.req.json();

	try {
		const [area] = await sql`
			INSERT INTO areas ${sql(data)}
			RETURNING *
		`;

		return c.json(area, 201);
	} catch (error) {
		console.error('Error creating area:', error);
		return c.json({ error: 'Failed to create area' }, 500);
	}
});

// Checklist item routes
app.post('/api/checklist-items', async (c) => {
	const data = await c.req.json();

	try {
		const [item] = await sql`
			INSERT INTO checklist_items ${sql(data)}
			RETURNING *
		`;

		return c.json(item, 201);
	} catch (error) {
		console.error('Error creating checklist item:', error);
		return c.json({ error: 'Failed to create checklist item' }, 500);
	}
});

app.patch('/api/checklist-items/:id', async (c) => {
	const id = c.req.param('id');
	const updates = await c.req.json();

	try {
		const [updated] = await sql`
			UPDATE checklist_items
			SET ${sql(updates)}, updated_at = NOW()
			WHERE id = ${id} AND deleted_at IS NULL
			RETURNING *
		`;

		if (!updated) {
			return c.json({ error: 'Checklist item not found' }, 404);
		}

		return c.json(updated);
	} catch (error) {
		console.error('Error updating checklist item:', error);
		return c.json({ error: 'Failed to update checklist item' }, 500);
	}
});

// Start server
import { serve } from '@hono/node-server';

const port = Number(process.env.PORT) || 3001;

console.log(`🚀 Thufir API server starting on port ${port}...`);

serve({
	fetch: app.fetch,
	port
}, (info) => {
	console.log(`✓ Server listening on http://localhost:${info.port}`);
});
