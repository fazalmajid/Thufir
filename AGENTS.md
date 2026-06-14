# Thufir — Agent Reference

Thufir is a local-first, self-hosted PWA task manager modelled after Cultured Code's Things app. It is a single-user app with offline-first sync via RxDB.

## Tech Stack

| Layer | Technology |
|-------|------------|
| Frontend | Svelte 5 (runes), SvelteKit 2 (static adapter, SSR disabled) |
| Styling | TailwindCSS 3 + typography plugin |
| Client DB | RxDB 15 with Dexie/IndexedDB storage |
| Markdown | marked.js |
| DnD | svelte-dnd-action |
| Auth (client) | @simplewebauthn/browser |
| Backend | Go 1.22+ with chi v5 router |
| Server DB | PostgreSQL (pgx pool, Goose migrations) |
| Auth (server) | go-webauthn passkeys |

## Repository Layout

```
src/
  routes/           SvelteKit pages (no SSR — all client-rendered)
  lib/
    types/          TypeScript interfaces for the three core entities
    db/             RxDB init, schemas, and replication logic
    stores/         Svelte 5 rune-based state ($state, $derived)
    services/       REST API clients (api.ts)
    components/
      layout/       Header, Sidebar, SidebarDropZone
      task/         TaskItem, TaskList, TaskQuickAdd
      ui/           DateInput (custom date picker)
server/
  cmd/server/       main.go — chi router, embedded frontend FS
  internal/
    auth/           WebAuthn handlers, session management
    config/         Env-based config (DATABASE_URL, PORT, RP_*)
    db/             Connection pool, Goose migrations + SQL files
    middleware/     RequireAuth session check
    sync/           RxDB pull/push handlers, quick-add endpoint
```

## Data Model

### Three Core Entities

All entities have `id` (UUID), `created_at`, `updated_at`, `deleted_at` (soft delete), and `sort_order` (integer, ascending = top first).

#### Task

```typescript
interface Task {
  id: string;
  title: string;
  notes?: string;            // Markdown
  status: 'inbox' | 'today' | 'upcoming' | 'anytime' | 'someday' | 'completed';
  is_completed: boolean;
  is_flagged: boolean;
  priority: 0 | 1 | 2 | 3;
  sort_order: number;
  tags: string[];
  area_id?: string;          // Direct area link (no project)
  project_id?: string;       // Project link (project carries area_id)
  parent_task_id?: string;   // Subtask parent (schema only, not used in UI)
  start_date?: string;       // ISO date string YYYY-MM-DD
  deadline?: string;         // ISO date string YYYY-MM-DD
  scheduled_date?: string;
  start_time?: string;
  reminder_time?: string;    // ISO 8601 datetime
  completed_at?: string;
  deleted_at?: string;
  created_at: string;
  updated_at: string;
}
```

Context hierarchy: Task → Project → Area. A task can belong to an area directly (area_id set, project_id null), to a project (project_id set), or to neither (pure inbox/status-based). Never set both area_id and project_id on the same task — use project_id only; the area is derived from the project.

#### Project

```typescript
interface Project {
  id: string;
  name: string;
  notes?: string;
  status: 'active' | 'completed' | 'archived';
  area_id?: string;
  tags: string[];
  deadline?: string;
  sort_order: number;
  deleted_at?: string;
  created_at: string;
  updated_at: string;
}
```

#### Area

```typescript
interface Area {
  id: string;
  name: string;
  color?: string;
  icon?: string;
  sort_order: number;
  deleted_at?: string;
  created_at: string;
  updated_at: string;
}
```

### PostgreSQL Schema (server/internal/db/migrations/)

The server mirrors these three tables plus auth tables (`name`, `credential`, `session`). All three entity tables have an `updated_at` index used by the checkpoint-based replication pull query. Migrations are embedded via Goose and run automatically at server start.

## Stores (src/lib/stores/)

All stores are Svelte 5 class instances exported as singletons. Use `$state.raw` for arrays to avoid deep reactivity overhead.

| Store | Key state | Key methods |
|-------|-----------|-------------|
| `taskStore` | `tasks[]`, derived view getters | `create()`, `update()`, `toggleComplete()`, `delete()`, `restore()`, `reorder()` |
| `projectStore` | `projects[]` | `create()` |
| `areaStore` | `areas[]` | `create()` |
| `dragStore` | DnD transient state | internal only |

**Derived task views** (`taskStore` getters):  
`inboxTasks`, `todayTasks`, `upcomingTasks`, `anytimeTasks`, `somedayTasks`, `completedTasks`, `trashedTasks` — all filter out `is_completed`/`deleted_at` as appropriate and sort by `sort_order` ascending.

**sort_order convention:** Lower value = higher in list. New tasks placed at the top of a context should use `Math.min(...contextTasks.map(t => t.sort_order)) - 1`. Default to `0` when the context is empty.

## Routes (src/routes/)

| Route | Purpose |
|-------|---------|
| `/inbox` | Status-based inbox |
| `/today` | Today focus list |
| `/upcoming` | Scheduled tasks |
| `/anytime` | Flexible tasks (paginated, 100/page) |
| `/someday` | Future ideas (paginated, 100/page) |
| `/logbook` | Completed tasks, reverse-chronological |
| `/trash` | Soft-deleted; restore or permanent delete |
| `/search?q=` | Full-text across title, notes, tags |
| `/areas/[id]` | Area tasks + project list |
| `/projects/[id]` | Project task list |
| `/settings` | Passkey/session management, bookmarklet |
| `/login` | WebAuthn setup or login |
| `/quick-add` | Bookmarklet landing page |

The root `+layout.svelte` checks auth (`GET /api/auth/me`), redirects to `/login` if unauthenticated, then initialises all stores and starts RxDB replication.

## RxDB & Sync

**Client DB** (`src/lib/db/index.ts`): database name `thufirdb`, Dexie adapter, collections `tasks`/`projects`/`areas`. Live RxDB queries (`.find().$.subscribe(...)`) drive the stores and auto-update all components.

**Replication** (`src/lib/db/replication.ts`):
- Checkpoint-based pull/push over REST
- Pull: `GET /api/rxdb/{collection}/pull?checkpoint=...&limit=...`
- Push: `POST /api/rxdb/{collection}/push` (array of changed docs)
- Live mode enabled; polls every 10 s; re-syncs on `window focus`

**Server handlers** (`server/internal/sync/`):
- Pull uses `(updated_at, id)` cursor ordering
- Push does row-level conflict resolution (last-write-wins by `updated_at`)
- Soft deletes propagate via `deleted_at` field

## Authentication

Single-user WebAuthn (passkeys). Session cookie (90 days, SameSite=Lax dev / None+Secure prod). All `/api/rxdb/*` endpoints require a valid session. The bookmarklet endpoint `POST /api/tasks/quick-add` also requires the session cookie.

## Key Patterns & Conventions

- **Svelte 5 runes only** — no Svelte 4 stores. Use `$state`, `$derived`, `$props`, `$effect`.
- **Soft deletes everywhere** — set `deleted_at` to delete; never hard-delete client-side.
- **UUIDs client-side** — `crypto.randomUUID()` at creation; server accepts them as-is.
- **Timestamps as ISO 8601 strings** — `created_at`, `updated_at`, dates, reminder_time.
- **Optimistic UI** — write to RxDB first; replication syncs in the background.
- **No comments on obvious code** — add a comment only for a non-obvious invariant or workaround.
- **Task context rule** — when creating a task inside an area or project, always pass `area_id` or `project_id` to `taskStore.create()`. Never rely on the `status` field alone to associate a task with a context.
- **TaskQuickAdd props** — accepts `status`, `area_id`, `project_id`. Pass the appropriate ID from the enclosing route page.

## Build & Run

```bash
make dev        # Build frontend, run Go dev server (port 3001)
npm run dev     # Vite dev server for hot module reload (separate process)
make            # Production build — embeds frontend into Go binary
./thufir        # Run production binary
make clean      # Remove build artefacts
```

**Required env vars:**
- `DATABASE_URL` — PostgreSQL DSN
- `RP_ID` — WebAuthn relying party domain (default: `thufir.majid.org`)
- `RP_ORIGIN` — WebAuthn origin URL
- `GO_ENV=production` — enables secure cookies

## What Is Not Yet Implemented

- Subtasks (schema has `parent_task_id`; no UI)
- Recurring tasks
- Tag browsing/faceting in UI (tags exist on tasks but no tag list route)
- Auto-purge of trash
- Multi-user support
