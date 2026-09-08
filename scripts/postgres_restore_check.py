#!/usr/bin/env python3
"""Isolated real PostgreSQL backup/restore acceptance. Never drops the supplied database.
Requires TEST_POSTGRES_DSN in keyword form, PG_BIN and shared Go cache environment.
"""
import json
import os
from pathlib import Path
import subprocess
import tempfile
import uuid
base = os.environ["TEST_POSTGRES_DSN"]
if "://" in base:
    raise SystemExit("Use libpq keyword DSN for this script")
pg = Path(os.environ.get("PG_BIN", "/opt/homebrew/opt/postgresql@16/bin"))
suffix = uuid.uuid4().hex[:12]
source, restored = "restore_src_" + suffix, "restore_dst_" + suffix
created = []
def run(args, env=None):
    result = subprocess.run(["rtk", "proxy", *map(str, args)], text=True, capture_output=True, env=env)
    if result.returncode:
        raise RuntimeError(result.stderr)
    return result.stdout.strip()
def sql(dsn, statement):
    return run([pg / "psql", dsn, "-XAt", "-v", "ON_ERROR_STOP=1", "-c", statement])
def snapshot(dsn):
    tables = sql(dsn, "SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename").splitlines()
    contents = {}
    for table in tables:
        quoted = chr(34) + table.replace(chr(34), chr(34)*2) + chr(34)
        contents[table] = sql(dsn, "SELECT count(*),md5(coalesce(string_agg(row_to_json(t)::text,'' ORDER BY row_to_json(t)::text),'')) FROM " + quoted + " t")
    contents["sequences"] = sql(dsn, "SELECT sequencename,last_value FROM pg_sequences WHERE schemaname='public' ORDER BY sequencename")
    return contents
try:
    print(sql(base, "SELECT version()"), flush=True)
    for name in [source, restored]:
        sql(base, "CREATE DATABASE " + name)
        created.append(name)
    src_dsn, dst_dsn = base + " dbname=" + source, base + " dbname=" + restored
    env = dict(os.environ, RUOYI_DATABASE_DSN=src_dsn)
    run(["go", "run", "./cmd/migrate"], env)
    run(["go", "run", "./cmd/bootstrap"], dict(env, RUOYI_BOOTSTRAP_PLATFORM_ADMIN="false", RUOYI_BOOTSTRAP_USERNAME="restore-fixture", RUOYI_BOOTSTRAP_PASSWORD="Isolated-Restore-Fixture-Only-392!", RUOYI_BOOTSTRAP_WEBSITE="restore.test"))
    sql(src_dsn, "INSERT INTO product_sku(spu_id,price,stock,tenant_id,properties) VALUES(1,3000000001,7,1,'[]')")
    before = snapshot(src_dsn)
    with tempfile.TemporaryDirectory(prefix="ruoyi-restore-") as temp:
        archive = Path(temp) / "baseline.dump"
        run([pg / "pg_dump", src_dsn, "--format=custom", "--no-owner", "--no-acl", "--file", archive])
        run([pg / "pg_restore", "--dbname", dst_dsn, "--no-owner", "--no-acl", "--exit-on-error", archive])
        print("Archive bytes:", archive.stat().st_size, flush=True)
    after = snapshot(dst_dsn)
    if before != after:
        raise AssertionError("restore mismatch: " + repr([k for k in before if before[k] != after.get(k)]))
    run(["go", "run", "./cmd/migrate"], dict(os.environ, RUOYI_DATABASE_DSN=dst_dsn))
    assert after == snapshot(dst_dsn), "repeat migration changed restored data"
    print(json.dumps({"result": "PASS", "tables_compared": len(before)-1, "comparison": "row count + ordered full-row MD5 for every table, and all sequence last values", "restored_migration": sql(dst_dsn,"SELECT version,dirty FROM schema_migrations"), "bootstrap_user_count": sql(dst_dsn,"SELECT count(*) FROM system_users"), "sku_fen": sql(dst_dsn,"SELECT price FROM product_sku"), "repeat_migration": "unchanged"}, indent=2), flush=True)
finally:
    for name in reversed(created):
        sql(base, "DROP DATABASE " + name + " WITH (FORCE)")
    print("Cleaned isolated rehearsal databases; supplied database untouched.", flush=True)
