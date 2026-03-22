#!/usr/bin/env python3
"""
Import Things 3 SQLite database (things.db) into Thufir's PostgreSQL schema.

Things 3 date encoding:
  Integer dates (startDate, deadline) use a compact format:
    year  = (v >> 16) & 0xFFFF
    month = (v >> 12) & 0xF
    day   = (v & 0xFFF) // 128

  Timestamp fields (creationDate, stopDate, etc.) are Unix timestamps (floats).

Things 3 type/status/start mappings:
  type:   0=task, 1=project, 2=heading (in-project section header)
  status: 0=open, 2=cancelled, 3=completed
  start:  0=inbox, 1=someday, 2=anytime/scheduled

Usage:
  pip install psycopg2-binary
  python import_things.py [things.db] [postgres_dsn]

Defaults:
  things.db path: ./things.db
  postgres_dsn:   postgresql://thufir:changeme@localhost:5432/thufir
"""

import sys
import sqlite3
import uuid
import datetime
import os
from typing import Optional

try:
    import psycopg2
except ImportError:
    print("ERROR: psycopg2 not found. Install with: pip install psycopg2-binary")
    sys.exit(1)


THINGS_DB = sys.argv[1] if len(sys.argv) > 1 else "things.db"
POSTGRES_DSN = sys.argv[2] if len(sys.argv) > 2 else os.environ.get(
    "DATABASE_URL", "postgresql://thufir:changeme@localhost:5432/thufir"
)


def decode_things_date(v: Optional[int]) -> Optional[datetime.date]:
    """Decode Things 3 compact integer date format to a Python date."""
    if v is None or v == 0:
        return None
    year = (v >> 16) & 0xFFFF
    month = (v >> 12) & 0xF
    day = (v & 0xFFF) // 128
    if year == 0 or month == 0 or month > 12 or day == 0 or day > 31:
        return None
    try:
        return datetime.date(year, month, day)
    except ValueError:
        return None


def from_unix(ts: Optional[float]) -> Optional[datetime.datetime]:
    """Convert Unix timestamp to UTC datetime."""
    if ts is None or ts == 0:
        return None
    return datetime.datetime.fromtimestamp(ts, tz=datetime.timezone.utc)


def map_task_status(
    start: int,
    status: int,
    start_date: Optional[datetime.date],
    today_index: Optional[int],
) -> str:
    """Map Things 3 start/status/todayIndex to Thufir task status."""
    if status == 3:
        return "completed"
    # Things marks tasks as Today by setting todayIndex > 0
    if today_index and today_index > 0:
        return "today"
    if start == 0:
        return "inbox"
    if start == 1:
        return "someday"
    # start == 2: anytime or upcoming (has a future start date)
    if start_date and start_date > datetime.date.today():
        return "upcoming"
    return "anytime"


def main():
    print(f"Opening Things DB: {THINGS_DB}")
    sq = sqlite3.connect(THINGS_DB)
    sq.row_factory = sqlite3.Row

    print(f"Connecting to PostgreSQL: {POSTGRES_DSN}")
    pg = psycopg2.connect(POSTGRES_DSN)
    pg.autocommit = False

    # Maps from Things UUID (text) -> Thufir UUID
    area_map: dict[str, str] = {}
    project_map: dict[str, str] = {}
    task_map: dict[str, str] = {}
    tag_map: dict[str, str] = {}

    try:
        with pg.cursor() as cur:
            # ------------------------------------------------------------------
            # 1. Areas
            # ------------------------------------------------------------------
            print("\n--- Importing areas ---")
            areas = sq.execute(
                "SELECT uuid, title, visible, \"index\" FROM TMArea ORDER BY \"index\""
            ).fetchall()

            for i, row in enumerate(areas):
                new_id = str(uuid.uuid4())
                area_map[row["uuid"]] = new_id
                cur.execute(
                    """
                    INSERT INTO areas (id, name, sort_order)
                    VALUES (%s, %s, %s)
                    """,
                    (new_id, row["title"] or "Unnamed Area", row["index"] or i),
                )
            print(f"  Inserted {len(areas)} areas")

            # ------------------------------------------------------------------
            # 2. Tags
            # ------------------------------------------------------------------
            print("\n--- Importing tags ---")
            tags = sq.execute("SELECT uuid, title FROM TMTag WHERE title IS NOT NULL").fetchall()

            tag_count = len(tags)
            for row in tags:
                new_id = str(uuid.uuid4())
                tag_map[row["uuid"]] = row["title"]  # map uuid -> name (tags are stored by name)
                # INSERT OR IGNORE equivalent: use ON CONFLICT
                cur.execute(
                    """
                    INSERT INTO tags (name)
                    VALUES (%s)
                    ON CONFLICT (name) DO NOTHING
                    """,
                    (row["title"],),
                )
            print(f"  Processed {tag_count} tags")

            # ------------------------------------------------------------------
            # 3. Projects (Things type=1 tasks)
            # ------------------------------------------------------------------
            print("\n--- Importing projects ---")
            projects = sq.execute(
                """
                SELECT uuid, title, notes, area, status, stopDate, creationDate,
                       "index", trashed
                FROM TMTask
                WHERE type = 1
                ORDER BY "index"
                """
            ).fetchall()

            for i, row in enumerate(projects):
                new_id = str(uuid.uuid4())
                project_map[row["uuid"]] = new_id

                area_id = area_map.get(row["area"]) if row["area"] else None
                completed_at = from_unix(row["stopDate"]) if row["status"] == 3 else None
                deleted_at = from_unix(row["creationDate"]) if row["trashed"] else None

                if row["status"] == 3:
                    proj_status = "completed"
                elif row["trashed"]:
                    proj_status = "archived"
                else:
                    proj_status = "active"

                cur.execute(
                    """
                    INSERT INTO projects (id, name, notes, area_id, status, sort_order,
                                         completed_at, deleted_at)
                    VALUES (%s, %s, %s, %s, %s, %s, %s, %s)
                    """,
                    (
                        new_id,
                        row["title"] or "Unnamed Project",
                        row["notes"],
                        area_id,
                        proj_status,
                        row["index"] or i,
                        completed_at,
                        deleted_at,
                    ),
                )
            print(f"  Inserted {len(projects)} projects")

            # ------------------------------------------------------------------
            # 4. Tasks (Things type=0 and type=2 headings)
            #    Pass 1: insert all tasks (parent_task_id handled in pass 2)
            # ------------------------------------------------------------------
            print("\n--- Importing tasks (pass 1: insert) ---")

            # Fetch tag associations for tasks
            task_tags_raw = sq.execute(
                "SELECT tasks, tags FROM TMTaskTag"
            ).fetchall()
            task_to_tags: dict[str, list[str]] = {}
            for r in task_tags_raw:
                name = tag_map.get(r["tags"])
                if name:
                    task_to_tags.setdefault(r["tasks"], []).append(name)

            # Fetch checklist items and build Markdown per task
            checklist_rows = sq.execute(
                """
                SELECT task, title, status, "index"
                FROM TMChecklistItem
                WHERE task IS NOT NULL
                ORDER BY task, "index"
                """
            ).fetchall()
            task_to_checklist: dict[str, str] = {}
            for r in checklist_rows:
                mark = "[x]" if r["status"] == 3 else "[ ]"
                line = f"- {mark} {r['title']}"
                task_to_checklist[r["task"]] = task_to_checklist.get(r["task"], "") + line + "\n"

            tasks = sq.execute(
                """
                SELECT uuid, title, notes, area, project, heading,
                       status, start, startDate, deadline, stopDate,
                       reminderTime, creationDate, userModificationDate,
                       "index", todayIndex, trashed, type
                FROM TMTask
                WHERE type IN (0, 2)
                ORDER BY "index"
                """
            ).fetchall()

            for i, row in enumerate(tasks):
                new_id = str(uuid.uuid4())
                task_map[row["uuid"]] = new_id

                area_id = area_map.get(row["area"]) if row["area"] else None
                project_id = project_map.get(row["project"]) if row["project"] else None

                start_date = decode_things_date(row["startDate"])
                deadline = decode_things_date(row["deadline"])
                is_completed = row["status"] == 3
                completed_at = from_unix(row["stopDate"]) if is_completed else None
                reminder = from_unix(row["reminderTime"]) if row["reminderTime"] else None
                created_at = from_unix(row["creationDate"]) or datetime.datetime.now(datetime.timezone.utc)
                updated_at = from_unix(row["userModificationDate"]) or created_at
                deleted_at = updated_at if row["trashed"] else None

                status = map_task_status(
                    row["start"] or 0,
                    row["status"] or 0,
                    start_date,
                    row["todayIndex"],
                )

                task_tag_names = task_to_tags.get(row["uuid"], [])
                sort_order = row["todayIndex"] if (status == "today" and row["todayIndex"] and row["todayIndex"] > 0) else row["index"] or i

                # Merge checklist items as Markdown into notes
                checklist_md = task_to_checklist.get(row["uuid"], "")
                if checklist_md:
                    notes = ((row["notes"] or "").rstrip() + "\n\n" + checklist_md).lstrip() if row["notes"] else checklist_md
                else:
                    notes = row["notes"]

                cur.execute(
                    """
                    INSERT INTO tasks (
                        id, title, notes,
                        project_id, area_id,
                        status, is_completed, completed_at,
                        start_date, deadline,
                        reminder_time,
                        tags,
                        sort_order,
                        created_at, updated_at, deleted_at
                    ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                    """,
                    (
                        new_id,
                        row["title"] or "(no title)",
                        notes,
                        project_id,
                        area_id,
                        status,
                        is_completed,
                        completed_at,
                        start_date,
                        deadline,
                        reminder,
                        task_tag_names if task_tag_names else None,
                        sort_order,
                        created_at,
                        updated_at,
                        deleted_at,
                    ),
                )

            print(f"  Inserted {len(tasks)} tasks")

            checklist_count = len(task_to_checklist)
            print(f"\n--- Converted {checklist_count} tasks' checklists to Markdown ---")

            # ------------------------------------------------------------------
            # 5. Update tag usage counts
            # ------------------------------------------------------------------
            print("\n--- Updating tag usage counts ---")
            cur.execute(
                """
                UPDATE tags t
                SET usage_count = (
                    SELECT COUNT(*)
                    FROM tasks
                    WHERE t.name = ANY(tasks.tags)
                      AND tasks.deleted_at IS NULL
                )
                """
            )
            print(f"  Updated usage counts for {tag_count} tags")

        pg.commit()
        print("\n✓ Import complete!")
        print(f"  Areas:    {len(areas)}")
        print(f"  Projects: {len(projects)}")
        print(f"  Tasks:    {len(tasks)}")
        print(f"  Checklists: {checklist_count}")
        print(f"  Tags:     {tag_count}")

    except Exception as e:
        pg.rollback()
        print(f"\nERROR: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)
    finally:
        sq.close()
        pg.close()


if __name__ == "__main__":
    main()
