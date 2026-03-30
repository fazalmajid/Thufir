import { Hono } from 'hono';
import { cors } from 'hono/cors';
import { getCookie, setCookie, deleteCookie } from 'hono/cookie';
import { zValidator } from '@hono/zod-validator';
import { z } from 'zod';
import postgres from 'postgres';
import {
	generateRegistrationOptions,
	verifyRegistrationResponse,
	generateAuthenticationOptions,
	verifyAuthenticationResponse,
} from '@simplewebauthn/server';
import type { AuthenticatorTransportFuture } from '@simplewebauthn/server';
import crypto from 'node:crypto';

type Variables = {
	userId: string;
	displayName: string;
};

const app = new Hono<{ Variables: Variables }>();

// Config
const RP_NAME = 'Thufir';
const RP_ID = process.env.RP_ID || 'localhost';
const RP_ORIGIN = process.env.RP_ORIGIN || 'http://localhost:5173';
const IS_PROD = process.env.NODE_ENV === 'production';

const ALLOWED_ORIGINS = Array.from(new Set([
	'http://localhost:5173',
	'http://localhost:4173',
	RP_ORIGIN,
]));

// Database
const sql = postgres(process.env.DATABASE_URL!, {
	max: 10,
	idle_timeout: 20,
	connect_timeout: 10,
});

// In-memory challenge store — entries expire after 5 minutes
const challenges = new Map<string, { challenge: string; userId?: string; expires: number }>();
setInterval(() => {
	const now = Date.now();
	for (const [key, val] of challenges.entries()) {
		if (val.expires < now) challenges.delete(key);
	}
}, 60_000);

// CORS
app.use('/*', cors({
	origin: ALLOWED_ORIGINS,
	credentials: true,
}));

// Session middleware — protects all /api/* except /api/auth/*
app.use('/api/*', async (c, next) => {
	if (c.req.path.startsWith('/api/auth/')) return next();

	const sessionId = getCookie(c, 'session');
	if (!sessionId) return c.json({ error: 'Unauthorized' }, 401);

	const rows = await sql`
		SELECT s.user_id, u.display_name
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.id = ${sessionId} AND s.expires_at > NOW()
	`;

	if (!rows.length) return c.json({ error: 'Unauthorized' }, 401);

	c.set('userId', rows[0].user_id);
	c.set('displayName', rows[0].display_name);
	return next();
});

// ── Auth helpers ──────────────────────────────────────────────────────────────

function setSessionCookie(c: any, sessionId: string) {
	setCookie(c, 'session', sessionId, {
		httpOnly: true,
		secure: IS_PROD,
		sameSite: 'Lax',
		maxAge: 90 * 24 * 60 * 60, // 90 days
		path: '/',
	});
}

async function createSession(userId: string): Promise<string> {
	const [session] = await sql`
		INSERT INTO sessions (user_id, expires_at)
		VALUES (${userId}, NOW() + INTERVAL '90 days')
		RETURNING id
	`;
	return session.id;
}

function storeChallengeToken(
	c: any,
	token: string,
	data: { challenge: string; userId?: string },
) {
	challenges.set(token, { ...data, expires: Date.now() + 5 * 60 * 1000 });
	setCookie(c, 'challenge', token, {
		httpOnly: true,
		secure: IS_PROD,
		sameSite: 'Lax',
		maxAge: 300,
		path: '/',
	});
}

// ── Auth routes ───────────────────────────────────────────────────────────────

app.get('/health', (c) => c.json({ status: 'ok' }));

// Whether any users exist — used by frontend to detect first-run setup
app.get('/api/auth/status', async (c) => {
	const [{ count }] = await sql`SELECT COUNT(*)::int AS count FROM users`;
	return c.json({ hasUsers: count > 0 });
});

// Current session user
app.get('/api/auth/me', async (c) => {
	const sessionId = getCookie(c, 'session');
	if (!sessionId) return c.json({ user: null }, 401);

	const rows = await sql`
		SELECT s.user_id AS id, u.display_name
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.id = ${sessionId} AND s.expires_at > NOW()
	`;

	if (!rows.length) return c.json({ user: null }, 401);
	return c.json({ user: { id: rows[0].id, displayName: rows[0].display_name } });
});

// Logout
app.post('/api/auth/logout', async (c) => {
	const sessionId = getCookie(c, 'session');
	if (sessionId) {
		await sql`DELETE FROM sessions WHERE id = ${sessionId}`;
		deleteCookie(c, 'session', { path: '/' });
	}
	return c.json({ success: true });
});

// ── First-time setup ──────────────────────────────────────────────────────────

app.post(
	'/api/auth/setup/options',
	zValidator('json', z.object({ displayName: z.string().min(1).max(100) })),
	async (c) => {
		const { displayName } = c.req.valid('json');

		const [{ count }] = await sql`SELECT COUNT(*)::int AS count FROM users`;
		if (count > 0) return c.json({ error: 'Setup already complete' }, 403);

		const userId = crypto.randomUUID();
		const options = await generateRegistrationOptions({
			rpName: RP_NAME,
			rpID: RP_ID,
			userID: new TextEncoder().encode(userId),
			userName: displayName,
			userDisplayName: displayName,
			attestationType: 'none',
			authenticatorSelection: {
				residentKey: 'required',
				userVerification: 'required',
			},
		});

		const token = crypto.randomUUID();
		storeChallengeToken(c, token, { challenge: options.challenge, userId });

		return c.json({ options, userId });
	},
);

app.post(
	'/api/auth/setup/verify',
	zValidator('json', z.object({
		userId: z.string().uuid(),
		displayName: z.string().min(1).max(100),
		deviceName: z.string().max(100).optional(),
		response: z.any(),
	})),
	async (c) => {
		const [{ count }] = await sql`SELECT COUNT(*)::int AS count FROM users`;
		if (count > 0) return c.json({ error: 'Setup already complete' }, 403);

		const { userId, displayName, deviceName, response } = c.req.valid('json');

		const token = getCookie(c, 'challenge');
		if (!token) return c.json({ error: 'No challenge found' }, 400);

		const stored = challenges.get(token);
		if (!stored || stored.userId !== userId || stored.expires < Date.now()) {
			return c.json({ error: 'Challenge expired or invalid' }, 400);
		}

		let verification;
		try {
			verification = await verifyRegistrationResponse({
				response,
				expectedChallenge: stored.challenge,
				expectedOrigin: RP_ORIGIN,
				expectedRPID: RP_ID,
				requireUserVerification: true,
			});
		} catch (err) {
			console.error('Registration verification error:', err);
			return c.json({ error: 'Verification failed' }, 400);
		}

		challenges.delete(token);
		deleteCookie(c, 'challenge', { path: '/' });

		const { verified, registrationInfo } = verification;
		if (!verified || !registrationInfo) return c.json({ error: 'Verification failed' }, 400);

		const { credential } = registrationInfo;

		await sql.begin(async (sql) => {
			await sql`INSERT INTO users (id, display_name) VALUES (${userId}, ${displayName})`;
			await sql`
				INSERT INTO credentials (user_id, credential_id, public_key, sign_count, transports, device_name)
				VALUES (
					${userId},
					${credential.id},
					${Buffer.from(credential.publicKey)},
					${credential.counter},
					${sql.array(credential.transports ?? [])},
					${deviceName ?? null}
				)
			`;
		});

		const sessionId = await createSession(userId);
		setSessionCookie(c, sessionId);

		return c.json({ success: true });
	},
);

// ── Login ─────────────────────────────────────────────────────────────────────

app.post('/api/auth/login/options', async (c) => {
	const options = await generateAuthenticationOptions({
		rpID: RP_ID,
		userVerification: 'required',
	});

	const token = crypto.randomUUID();
	storeChallengeToken(c, token, { challenge: options.challenge });

	return c.json(options);
});

app.post(
	'/api/auth/login/verify',
	zValidator('json', z.object({ response: z.any() })),
	async (c) => {
		const { response } = c.req.valid('json');

		const token = getCookie(c, 'challenge');
		if (!token) return c.json({ error: 'No challenge found' }, 400);

		const stored = challenges.get(token);
		if (!stored || stored.expires < Date.now()) {
			return c.json({ error: 'Challenge expired' }, 400);
		}

		const rows = await sql`SELECT * FROM credentials WHERE credential_id = ${response.id}`;
		if (!rows.length) return c.json({ error: 'Credential not found' }, 400);

		const cred = rows[0];

		let verification;
		try {
			verification = await verifyAuthenticationResponse({
				response,
				expectedChallenge: stored.challenge,
				expectedOrigin: RP_ORIGIN,
				expectedRPID: RP_ID,
				credential: {
					id: cred.credential_id,
					publicKey: new Uint8Array(cred.public_key),
					counter: Number(cred.sign_count),
					transports: cred.transports as AuthenticatorTransportFuture[],
				},
				requireUserVerification: true,
			});
		} catch (err) {
			console.error('Authentication verification error:', err);
			return c.json({ error: 'Authentication failed' }, 401);
		}

		challenges.delete(token);
		deleteCookie(c, 'challenge', { path: '/' });

		const { verified, authenticationInfo } = verification;
		if (!verified || !authenticationInfo) return c.json({ error: 'Authentication failed' }, 401);

		await sql`
			UPDATE credentials SET sign_count = ${authenticationInfo.newCounter}
			WHERE credential_id = ${response.id}
		`;

		const sessionId = await createSession(cred.user_id);
		setSessionCookie(c, sessionId);

		const [user] = await sql`SELECT id, display_name FROM users WHERE id = ${cred.user_id}`;
		return c.json({ success: true, user: { id: user.id, displayName: user.display_name } });
	},
);

// ── Add device (requires active session) ─────────────────────────────────────

app.post('/api/auth/device/options', async (c) => {
	const sessionId = getCookie(c, 'session');
	if (!sessionId) return c.json({ error: 'Unauthorized' }, 401);

	const rows = await sql`
		SELECT s.user_id, u.display_name
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.id = ${sessionId} AND s.expires_at > NOW()
	`;
	if (!rows.length) return c.json({ error: 'Unauthorized' }, 401);

	const { user_id, display_name } = rows[0];

	const existingCreds = await sql`
		SELECT credential_id, transports FROM credentials WHERE user_id = ${user_id}
	`;

	const options = await generateRegistrationOptions({
		rpName: RP_NAME,
		rpID: RP_ID,
		userID: new TextEncoder().encode(user_id),
		userName: display_name,
		userDisplayName: display_name,
		attestationType: 'none',
		excludeCredentials: existingCreds.map((cr: any) => ({
			id: cr.credential_id,
			transports: cr.transports,
		})),
		authenticatorSelection: {
			residentKey: 'required',
			userVerification: 'required',
		},
	});

	const token = crypto.randomUUID();
	storeChallengeToken(c, token, { challenge: options.challenge, userId: user_id });

	return c.json(options);
});

app.post(
	'/api/auth/device/verify',
	zValidator('json', z.object({
		deviceName: z.string().max(100).optional(),
		response: z.any(),
	})),
	async (c) => {
		const sessionId = getCookie(c, 'session');
		if (!sessionId) return c.json({ error: 'Unauthorized' }, 401);

		const rows = await sql`
			SELECT user_id FROM sessions WHERE id = ${sessionId} AND expires_at > NOW()
		`;
		if (!rows.length) return c.json({ error: 'Unauthorized' }, 401);

		const userId = rows[0].user_id;
		const { deviceName, response } = c.req.valid('json');

		const token = getCookie(c, 'challenge');
		if (!token) return c.json({ error: 'No challenge found' }, 400);

		const stored = challenges.get(token);
		if (!stored || stored.userId !== userId || stored.expires < Date.now()) {
			return c.json({ error: 'Challenge expired or invalid' }, 400);
		}

		let verification;
		try {
			verification = await verifyRegistrationResponse({
				response,
				expectedChallenge: stored.challenge,
				expectedOrigin: RP_ORIGIN,
				expectedRPID: RP_ID,
				requireUserVerification: true,
			});
		} catch (err) {
			console.error('Device registration error:', err);
			return c.json({ error: 'Verification failed' }, 400);
		}

		challenges.delete(token);
		deleteCookie(c, 'challenge', { path: '/' });

		const { verified, registrationInfo } = verification;
		if (!verified || !registrationInfo) return c.json({ error: 'Verification failed' }, 400);

		const { credential } = registrationInfo;

		await sql`
			INSERT INTO credentials (user_id, credential_id, public_key, sign_count, transports, device_name)
			VALUES (
				${userId},
				${credential.id},
				${Buffer.from(credential.publicKey)},
				${credential.counter},
				${sql.array(credential.transports ?? [])},
				${deviceName ?? null}
			)
		`;

		return c.json({ success: true });
	},
);

// List credentials for the current user
app.get('/api/auth/devices', async (c) => {
	const sessionId = getCookie(c, 'session');
	if (!sessionId) return c.json({ error: 'Unauthorized' }, 401);

	const rows = await sql`
		SELECT user_id FROM sessions WHERE id = ${sessionId} AND expires_at > NOW()
	`;
	if (!rows.length) return c.json({ error: 'Unauthorized' }, 401);

	const devices = await sql`
		SELECT id, device_name, transports, created_at
		FROM credentials
		WHERE user_id = ${rows[0].user_id}
		ORDER BY created_at ASC
	`;
	return c.json(devices);
});

// Delete a credential (cannot delete the last one)
app.delete('/api/auth/devices/:id', async (c) => {
	const sessionId = getCookie(c, 'session');
	if (!sessionId) return c.json({ error: 'Unauthorized' }, 401);

	const rows = await sql`
		SELECT user_id FROM sessions WHERE id = ${sessionId} AND expires_at > NOW()
	`;
	if (!rows.length) return c.json({ error: 'Unauthorized' }, 401);

	const userId = rows[0].user_id;
	const id = c.req.param('id');

	const [{ count }] = await sql`
		SELECT COUNT(*)::int AS count FROM credentials WHERE user_id = ${userId}
	`;
	if (count <= 1) return c.json({ error: 'Cannot remove the last passkey' }, 400);

	await sql`DELETE FROM credentials WHERE id = ${id} AND user_id = ${userId}`;
	return c.json({ success: true });
});

// ── Validation schemas ────────────────────────────────────────────────────────

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
	notes: z.string().nullable().optional(),
	project_id: z.string().uuid().nullable().optional(),
	area_id: z.string().uuid().nullable().optional(),
	status: z.enum(['inbox', 'today', 'upcoming', 'anytime', 'someday', 'completed']).optional(),
	scheduled_date: z.string().nullable().optional(),
	deadline: z.string().nullable().optional(),
	reminder_time: z.string().nullable().optional(),
	tags: z.array(z.string()).optional(),
	is_flagged: z.boolean().optional(),
	priority: z.number().min(0).max(3).optional(),
	is_completed: z.boolean().optional(),
	sort_order: z.number().optional()
});

// ── Task routes ───────────────────────────────────────────────────────────────

app.get('/api/tasks', async (c) => {
	const status = c.req.query('status');
	const projectId = c.req.query('project_id');
	const areaId = c.req.query('area_id');

	try {
		const tasks = await sql`
			SELECT * FROM tasks
			WHERE deleted_at IS NULL
			${status ? sql`AND status = ${status}` : sql``}
			${projectId ? sql`AND project_id = ${projectId}` : sql``}
			${areaId ? sql`AND area_id = ${areaId} AND project_id IS NULL` : sql``}
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
			SELECT * FROM tasks WHERE id = ${id} AND deleted_at IS NULL
		`;
		if (!task) return c.json({ error: 'Task not found' }, 404);
		return c.json(task);
	} catch (error) {
		console.error('Error fetching task:', error);
		return c.json({ error: 'Failed to fetch task' }, 500);
	}
});

app.post('/api/tasks', zValidator('json', createTaskSchema), async (c) => {
	const task = c.req.valid('json');

	try {
		const [created] = await sql`INSERT INTO tasks ${sql(task)} RETURNING *`;
		return c.json(created, 201);
	} catch (error) {
		console.error('Error creating task:', error);
		return c.json({ error: 'Failed to create task' }, 500);
	}
});

app.patch('/api/tasks/:id', zValidator('json', updateTaskSchema), async (c) => {
	const id = c.req.param('id');
	const updates = c.req.valid('json') as Record<string, unknown>;

	try {
		if (updates.is_completed !== undefined) {
			if (updates.is_completed) {
				updates.completed_at = new Date().toISOString();
				updates.status = 'completed';
			} else {
				updates.completed_at = null;
			}
		}

		const [updated] = await sql`
			UPDATE tasks SET ${sql(updates)}, updated_at = NOW()
			WHERE id = ${id} AND deleted_at IS NULL
			RETURNING *
		`;
		if (!updated) return c.json({ error: 'Task not found' }, 404);
		return c.json(updated);
	} catch (error) {
		console.error('Error updating task:', error);
		return c.json({ error: 'Failed to update task' }, 500);
	}
});

app.delete('/api/tasks/:id', async (c) => {
	const id = c.req.param('id');

	try {
		await sql`UPDATE tasks SET deleted_at = NOW() WHERE id = ${id}`;
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
			UPDATE tasks SET deleted_at = NULL WHERE id = ${id} RETURNING *
		`;
		if (!restored) return c.json({ error: 'Task not found' }, 404);
		return c.json(restored);
	} catch (error) {
		console.error('Error restoring task:', error);
		return c.json({ error: 'Failed to restore task' }, 500);
	}
});

const reorderTasksSchema = z.object({
	tasks: z.array(z.object({
		id: z.string().uuid(),
		sort_order: z.number()
	}))
});

app.post('/api/tasks/reorder', zValidator('json', reorderTasksSchema), async (c) => {
	const { tasks } = c.req.valid('json');

	try {
		await sql.begin(async (sql) => {
			for (const task of tasks) {
				await sql`
					UPDATE tasks SET sort_order = ${task.sort_order}, updated_at = NOW()
					WHERE id = ${task.id}
				`;
			}
		});
		return c.json({ success: true });
	} catch (error) {
		console.error('Error reordering tasks:', error);
		return c.json({ error: 'Failed to reorder tasks' }, 500);
	}
});

// ── Project routes ────────────────────────────────────────────────────────────

app.get('/api/projects', async (c) => {
	try {
		const projects = await sql`
			SELECT * FROM projects WHERE deleted_at IS NULL ORDER BY sort_order ASC, created_at DESC
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
		const [project] = await sql`INSERT INTO projects ${sql(data)} RETURNING *`;
		return c.json(project, 201);
	} catch (error) {
		console.error('Error creating project:', error);
		return c.json({ error: 'Failed to create project' }, 500);
	}
});

// ── Area routes ───────────────────────────────────────────────────────────────

app.get('/api/areas', async (c) => {
	try {
		const areas = await sql`
			SELECT * FROM areas WHERE deleted_at IS NULL ORDER BY sort_order ASC
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
		const [area] = await sql`INSERT INTO areas ${sql(data)} RETURNING *`;
		return c.json(area, 201);
	} catch (error) {
		console.error('Error creating area:', error);
		return c.json({ error: 'Failed to create area' }, 500);
	}
});

// ── Start server ──────────────────────────────────────────────────────────────

import { serve } from '@hono/node-server';

const port = Number(process.env.PORT) || 3001;

console.log(`🚀 Thufir API server starting on port ${port}...`);

serve({ fetch: app.fetch, port, hostname: '0.0.0.0' }, (info) => {
	console.log(`✓ Server listening on http://0.0.0.0:${info.port}`);
});
