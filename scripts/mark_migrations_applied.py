#!/usr/bin/env python3
"""Mark all init-db/*.sql migrations as applied in schema_migrations.

Use when the database was restored from a dump and schema_migrations is empty,
causing the auto-migration system to re-run all migrations on startup.
Re-running 001_schema.sql after later migrations extended views causes
"cannot drop columns from view" errors.

Run with uv (no dependencies required):

    cd pos-backend
    uv run scripts/mark_migrations_applied.py
    pm2 restart pos-backend        # or: go run ./cmd/api.go

Overridable via env: POS_PG_USER (default postgres), POS_PG_DB (default pos_db).
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
# container name and works from any host that runs docker compose up.
PSQL = ["docker", "compose", "exec", "-T", "postgres", "psql", "-U", PG_USER, "-d", PG_DB]


def main() -> int:
    os.chdir(BACKEND_DIR)  # docker compose needs the compose file's directory
    files = sorted(f for f in os.listdir(INIT_DB) if f.endswith(".sql"))
    if not files:
        print(f"no .sql files found in {INIT_DB}")
        return 1

    count = 0
    for f in files:
        result = subprocess.run(
            PSQL + ["-c", f"INSERT INTO schema_migrations (filename) VALUES ('{f}') ON CONFLICT DO NOTHING;"],
            capture_output=True, text=True,
        )
        if result.returncode == 0:
            count += 1
        else:
            print(f"failed: {f}: {(result.stderr or result.stdout).strip()}")

    print(f"Marked {count}/{len(files)} migrations as applied")
    return 0 if count == len(files) else 1


if __name__ == "__main__":
    sys.exit(main())
