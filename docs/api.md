# Snippetbox API

All endpoints return JSON. When HTTP Basic Auth is enabled, pass credentials with each request.

## List snippets

```sh
curl http://localhost:8080/api/snippets
```

Optional query parameters:

- `q`: full-text search over title, language, content, and matching tags.
- `tag`: exact tag filter.

## Create snippet

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

## Get snippet

```sh
curl http://localhost:8080/api/snippets/42
```

## Delete snippet

```sh
curl -X DELETE http://localhost:8080/api/snippets/42
```
