# Migration Guide

## Overview

Migrations live in `internal/db/script/migration/`. Each migration has two paired files:

- `NNNN_name.up.sql` — the forward migration (create table, add column, etc.)
- `NNNN_name.down.sql` — the rollback (undo the up migration)

The `NNNN` prefix is a zero-padded number (e.g. `001`, `002`) that controls execution order.

```
internal/db/script/migration/
  001-user_table.up.sql
  001-user_table.down.sql
  002-add_user_fields.up.sql
  002-add_user_fields.down.sql
```

## Commands

### Run pending migrations

```bash
go run ./cmd -migrate
```

Applies all pending `.up.sql` files and exits. Does **not** start the HTTP server.

### Roll back migrations

```bash
go run ./cmd -rollback=1    # roll back last 1
go run ./cmd -rollback=3    # roll back last 3
go run ./cmd -rollback=0    # roll back ALL
```

Rolls back N migrations and exits. Does **not** start the HTTP server.

### Start the server (no migration)

```bash
go run ./cmd
```

Opens the database and starts the HTTP server. **No migrations are applied.**

### Start the server with auto-migration

```bash
AUTO_MIGRATE=true go run ./cmd
```

Same as above, but also runs pending migrations before starting the server.

## Workflow examples

### First time setup

```bash
# 1. Run all migrations to create tables (from project root)
go run ./cmd -migrate

# 2. Start the server (safe — no auto-migration)
go run ./cmd
```

### Adding a new feature that needs a DB change

```bash
# 1. Create 003-some_feature.up.sql and 003-some_feature.down.sql

# 2. Apply the new migration
go run ./cmd -migrate

# 3. Verify the server still starts
go run ./cmd
```

### Rolling back a bad deployment

```bash
# 1. Roll back the last migration
go run ./cmd -rollback=1

# 2. Fix the migration files

# 3. Re-apply
go run ./cmd -migrate
```

### Development with auto-migrate

```bash
# Set the env var once per terminal session
$env:AUTO_MIGRATE = "true"
go run ./cmd   # auto-migrates every time you restart
```

## Adding a new column

### Step 1: Create a new migration pair

Example — add a `salary` column to the `user` table:

**`003-add_salary.up.sql`**

```sql
ALTER TABLE user ADD COLUMN salary INTEGER NOT NULL DEFAULT 0;
```

**`003-add_salary.down.sql`**

```sql
ALTER TABLE user DROP COLUMN salary;
```

> SQLite supports `DROP COLUMN` from version 3.35.0+ (the project uses `modernc.org/sqlite` which tracks the latest SQLite).

### Step 2: Apply it

```bash
go run ./cmd -migrate
```

## Database path

Controlled by the `DATABASE_PATH` environment variable:

```bash
# Default: internal/db/helpdesk.db (relative to project root)
# Run all commands from the project root (where go.mod lives)
go run ./cmd

# Custom path
$env:DATABASE_PATH = "custom/path/mydb.db"
go run ./cmd
```

## schema_migrations table

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY
);
```

- `version` stores the migration name without suffix (e.g. `001-user_table`)
- A row exists only after a successful up-migration
- Down-migrations delete their row

## Go API

```go
// Open database connection
config.InitDB()

// Apply all pending up-migrations
config.RunMigrations()

// Or call directly:
script.MigrateDB(db)            // up
script.RollbackDB(db, steps)    // down (0 = all)
```

## Best practices

1. **Never edit an existing migration** after it's been applied. Create a new migration file instead.
2. **Always write a down file** for every up file.
3. **Keep migrations small** — one logical change per migration.
4. **Test rollback locally** before deploying: `-migrate` → `-rollback=1` → `-migrate`.
5. **Use explicit commands in production** — never rely on `AUTO_MIGRATE` for production deployments.
