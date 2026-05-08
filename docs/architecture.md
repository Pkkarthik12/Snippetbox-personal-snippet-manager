# Architecture

Snippetbox is intentionally small:

- `cmd/server`: process entrypoint, config loading, graceful shutdown.
- `internal/handlers`: HTML routes, REST API routes, auth, request limits.
- `internal/models`: pgx-backed persistence and search queries.
- `internal/render`: template parsing and Chroma syntax highlighting.
- `schema`: SQL migrations.

PostgreSQL owns search through a generated `tsvector` column. Tags are normalized into a join table so exact tag filters remain fast and simple.
