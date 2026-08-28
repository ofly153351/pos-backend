#!/usr/bin/env python3
"""Prepare the schema_migrations ledger for a database restored from a dump.

When a DB is restored from a dump, the schema_migrations ledger is empty, so the
backend's auto-migrate re-runs every init-db/*.sql in order and dies on
001_schema.sql ("cannot drop columns from view" — later migrations extended the
views). This script fills the ledger for exactly the migrations the restored DB
already reflects, leaving genuinely NEW migration files unrecorded so the backend
applies them on boot.

Per-file detection (safe — every probe runs inside a rolled-back transaction):
  * probe succeeds → migration is NOT yet applied → leave unrecorded (backend
    will apply it for real)
  * probe fails with an "already applied" signature (…already exists / cannot
    drop columns / duplicate key) → restored DB already has it → record it
  * probe fails with anything else → real problem → warn, do not record

Run with uv (stdlib only, no dependencies), from the repo root:

    uv run scripts/mark_migrations_applied.py

Overridable via env: POS_PG_USER (default postgres), POS_PG_DB (default pos_db).
Requires Docker up (talks to the postgres service via `docker compose exec`).
"""
# /// script
# requires-python = ">=3.10"
# ///

import os
import subprocess
import sys

BACKEND_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
INIT_DB = os.path.join(BACKEND_DIR, "init-db")
PG_USER = os.environ.get("POS_PG_USER", "postgres")
PG_DB = os.environ.get("POS_PG_DB", "pos_db")
# `docker compose exec` resolves the postgres service regardless of the actual
# container name and works from any host that runs `docker compose up`.
PSQL = ["docker", "compose", "exec", "-T", "postgres", "psql", "-U", PG_USER, "-d", PG_DB]

# Lower-case substrings that identify a probe failure caused by the migration
# already being reflected in the database (vs. a genuine SQL problem).
ALREADY_APPLIED_MARKERS = ("already exists", "cannot drop columns", "duplicate key")


def psql(args, input_text=None):
    return subprocess.run(PSQL + args, capture_output=True, text=True, input=input_text)


def main() -> int:
    os.chdir(BACKEND_DIR)  # docker compose needs the compose file's directory
    files = sorted(f for f in os.listdir(INIT_DB) if f.endswith(".sql"))
    if not files:
        print(f"no .sql files found in {INIT_DB}")
        return 1

    ledger = psql(["-tAc", "SELECT filename FROM schema_migrations"])
    if ledger.returncode != 0:
        print(f"cannot read schema_migrations: {ledger.stderr.strip()}")
        return 1
    recorded = set(ledger.stdout.split())

    marked = fresh = warned = 0
    for f in files:
        if f in recorded:
            continue

        # Probe the migration inside a transaction that is ALWAYS rolled back:
        # explicit ROLLBACK on success; ON_ERROR_STOP aborts before it on failure
        # and the closed connection implicitly rolls back. Nothing persists.
        with open(os.path.join(INIT_DB, f), encoding="utf-8") as fh:
            probe = psql(
                ["-v", "ON_ERROR_STOP=1", "-c", "BEGIN;", "-c", "ROLLBACK;"],
                input_text=fh.read(),
            )
        if probe.returncode == 0:
            fresh += 1
            print(f"fresh (backend will apply): {f}")
            continue

        err = (probe.stderr or probe.stdout).lower()
        if not any(m in err for m in ALREADY_APPLIED_MARKERS):
            warned += 1
            print(f"WARN probe failed unexpectedly for {f} — NOT recorded:")
            print(f"      {(probe.stderr or probe.stdout).strip()[:300]}")
            continue

        r = psql(["-c", f"INSERT INTO schema_migrations (filename) VALUES ('{f}') ON CONFLICT DO NOTHING;"])
        if r.returncode == 0:
            marked += 1
            print(f"marked already-applied: {f}")
        else:
            warned += 1
            print(f"WARN failed to record {f}: {r.stderr.strip()}")

    print(f"ledger ready: marked={marked} fresh-for-runner={fresh} warned={warned} total={len(files)}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
