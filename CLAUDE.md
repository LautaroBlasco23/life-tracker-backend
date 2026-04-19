# life-tracker-backend

## Stack

- **Go** with Gin HTTP framework
- **PostgreSQL** via GORM (ORM layer only — no AutoMigrate)
- **MongoDB** for activity records, transactions, and payments
- **golang-migrate** for versioned SQL migrations

## Architecture

Domain-based structure under `internal/domain/{domain}/`:

```
internal/
├── domain/{domain}/
│   ├── controller/   ← HTTP handlers
│   ├── service/      ← business logic
│   ├── repository/   ← data access
│   ├── model/        ← DB structs (GORM/BSON)
│   ├── dto/          ← request/response types
│   └── routes/       ← route registration
├── database/
│   ├── migrations/   ← versioned SQL files (*.up.sql / *.down.sql)
│   ├── migrator.go   ← runs migrations on startup via embed.FS
│   ├── postgresql.go
│   └── mongo.go
├── config/
├── middleware/
└── infrastructure/
```

## Database Migrations

Migrations live in `internal/database/migrations/` and run automatically on startup.

**To add a new migration**, create two files:

```
000003_your_change.up.sql
000003_your_change.down.sql
```

Naming convention: `{6-digit-number}_{description}.{up|down}.sql`

The migration state is tracked in the `schema_migrations` table in PostgreSQL.

**To rollback manually** (dev only):

```bash
migrate -path ./internal/database/migrations -database "postgres://..." down 1
```

MongoDB does not use file-based migrations. Index creation is managed in `internal/database/mongo.go`.

## Key Commands

```bash
make start        # setup .env + start app with hot reload
make dev          # start app (requires .env)
make test         # run tests with isolated test DB
make code-check   # format + lint
make db-up        # start dev databases only
make db-remove    # wipe dev databases
```

## Preflight

**Ecosystem**: Go 1.26.2
**Config**: go.mod, Makefile
**Status**: ready

| Category | Status | Command |
|----------|--------|---------|
| Build    | ready  | `go build ./...` |
| Check    | ready  | `make code-check` |
| Test     | ready  | `make test` |

**Blockers**: none
**Warnings**: none
