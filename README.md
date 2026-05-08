# Snippetbox

Self-hosted, lightning-fast personal code snippet manager. Save, tag, search, and reuse your code without giving it to anyone else.

Snippetbox is a lightweight alternative to GitHub Gist. It stores snippets in PostgreSQL, renders syntax highlighting with Chroma, and serves a minimal HTML interface plus a small REST API.

## Features

- Syntax highlighting for 50+ languages powered by Chroma.
- Tag-based organization with multiple tags per snippet.
- Full-text search by title, code content, language, or tag.
- REST API for editor, terminal, and script integrations.
- GitHub Gist import command.
- Minimal server-rendered UI with progressive HTMX-style search hooks.
- Single compiled Go binary for deployment.
- Optional single-user HTTP Basic Auth.

## Tech Stack

| Layer | Technology |
| --- | --- |
| Backend | Go 1.22+ |
| Database | PostgreSQL via pgx |
| Frontend | Server-rendered HTML, HTMX hooks, vanilla CSS |
| Highlighting | Chroma |
| Auth | Optional HTTP Basic Auth |
| Deployment | Single binary or Docker |

## Getting Started

### Prerequisites

- Go 1.22+
- PostgreSQL 14+
- `make` optional

### 1. Clone the repo

```sh
git clone https://github.com/Pkkarthik12/Snippetbox-personal-snippet-manager.git
cd Snippetbox-personal-snippet-manager
```

### 2. Set up the database

```sh
createdb snippetbox
psql snippetbox < schema/001_init.sql
```

### 3. Configure environment

```sh
cp .env.example .env
```

Edit `.env`:

```env
DATABASE_URL=postgres://localhost/snippetbox?sslmode=disable
PORT=8080
AUTH_ENABLED=false
AUTH_USERNAME=admin
AUTH_PASSWORD=changeme
MAX_BODY_SIZE=1MB
```

### 4. Run

```sh
go run ./cmd/server
```

Visit `http://localhost:8080`.

## Build

```sh
make build
./bin/snippetbox
```

## Docker

```sh
docker compose up -d
```

The compose file runs PostgreSQL, applies the SQL schema, and starts Snippetbox. Data is stored in a named Docker volume.

## REST API

List snippets:

```sh
curl http://localhost:8080/api/snippets
```

Create a snippet:

```sh
curl -X POST http://localhost:8080/api/snippets \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Debounce in JavaScript",
    "language": "javascript",
    "content": "function debounce(fn, delay) { ... }",
    "tags": ["js", "utils"]
  }'
```

Search snippets:

```sh
curl "http://localhost:8080/api/snippets?q=debounce&tag=js"
```

Read the full reference in [docs/api.md](docs/api.md).

## Import GitHub Gists

```sh
export GITHUB_TOKEN=ghp_yourtoken
go run ./cmd/import-gists --username yourname
```

`GITHUB_TOKEN` is optional for public gists, but recommended to avoid strict API limits and to access secret gists visible to your account.

## Project Structure

```text
snippetbox/
├── cmd/
│   ├── server/
│   └── import-gists/
├── internal/
│   ├── config/
│   ├── handlers/
│   ├── models/
│   └── render/
├── schema/
├── templates/
├── static/
├── docs/
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── README.md
```

## Makefile Commands

```sh
make run
make build
make test
make lint
make db-migrate
make db-reset
```

## Roadmap

- VS Code extension.
- Neovim Lua plugin.
- Multi-user snippet visibility.
- Webhook on snippet create/update.
- Encrypted snippets at rest.
- Markdown preview mode.

## License

MIT. See [LICENSE](LICENSE).
