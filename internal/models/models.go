package models

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

type Snippet struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Language  string    `json:"language"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateSnippetParams struct {
	Title    string   `json:"title"`
	Language string   `json:"language"`
	Content  string   `json:"content"`
	Tags     []string `json:"tags"`
}

type ListFilters struct {
	Query string
	Tag   string
	Limit int
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() {
	s.db.Close()
}

func (s *Store) Ping(ctx context.Context) error {
	return s.db.Ping(ctx)
}

func (s *Store) CreateSnippet(ctx context.Context, params CreateSnippetParams) (Snippet, error) {
	params.Tags = normalizeTags(params.Tags)
	if strings.TrimSpace(params.Title) == "" {
		return Snippet{}, errors.New("title is required")
	}
	if strings.TrimSpace(params.Content) == "" {
		return Snippet{}, errors.New("content is required")
	}
	if strings.TrimSpace(params.Language) == "" {
		params.Language = "plaintext"
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Snippet{}, err
	}
	defer tx.Rollback(ctx)

	var snippet Snippet
	err = tx.QueryRow(ctx, `
		INSERT INTO snippets (title, language, content)
		VALUES ($1, $2, $3)
		RETURNING id, title, language, content, created_at, updated_at
	`, strings.TrimSpace(params.Title), strings.TrimSpace(params.Language), params.Content).
		Scan(&snippet.ID, &snippet.Title, &snippet.Language, &snippet.Content, &snippet.CreatedAt, &snippet.UpdatedAt)
	if err != nil {
		return Snippet{}, err
	}

	if err := replaceTags(ctx, tx, snippet.ID, params.Tags); err != nil {
		return Snippet{}, err
	}
	snippet.Tags = params.Tags

	if err := tx.Commit(ctx); err != nil {
		return Snippet{}, err
	}
	return snippet, nil
}

func (s *Store) ListSnippets(ctx context.Context, filters ListFilters) ([]Snippet, error) {
	if filters.Limit <= 0 || filters.Limit > 200 {
		filters.Limit = 50
	}

	query := strings.TrimSpace(filters.Query)
	tag := strings.TrimSpace(filters.Tag)

	rows, err := s.db.Query(ctx, `
		SELECT s.id, s.title, s.language, s.content, s.created_at, s.updated_at,
		       COALESCE(array_agg(t.name ORDER BY t.name) FILTER (WHERE t.name IS NOT NULL), '{}') AS tags
		FROM snippets s
		LEFT JOIN snippet_tags st ON st.snippet_id = s.id
		LEFT JOIN tags t ON t.id = st.tag_id
		WHERE
			($1 = '' OR s.search_vector @@ websearch_to_tsquery('simple', $1)
			 OR EXISTS (
				SELECT 1 FROM snippet_tags st2
				JOIN tags t2 ON t2.id = st2.tag_id
				WHERE st2.snippet_id = s.id AND t2.name ILIKE '%' || $1 || '%'
			 ))
			AND ($2 = '' OR EXISTS (
				SELECT 1 FROM snippet_tags st3
				JOIN tags t3 ON t3.id = st3.tag_id
				WHERE st3.snippet_id = s.id AND t3.name = lower($2)
			))
		GROUP BY s.id
		ORDER BY s.updated_at DESC
		LIMIT $3
	`, query, tag, filters.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanSnippets(rows)
}

func (s *Store) GetSnippet(ctx context.Context, id int64) (Snippet, error) {
	row := s.db.QueryRow(ctx, `
		SELECT s.id, s.title, s.language, s.content, s.created_at, s.updated_at,
		       COALESCE(array_agg(t.name ORDER BY t.name) FILTER (WHERE t.name IS NOT NULL), '{}') AS tags
		FROM snippets s
		LEFT JOIN snippet_tags st ON st.snippet_id = s.id
		LEFT JOIN tags t ON t.id = st.tag_id
		WHERE s.id = $1
		GROUP BY s.id
	`, id)

	var snippet Snippet
	err := row.Scan(&snippet.ID, &snippet.Title, &snippet.Language, &snippet.Content, &snippet.CreatedAt, &snippet.UpdatedAt, &snippet.Tags)
	if errors.Is(err, pgx.ErrNoRows) {
		return Snippet{}, ErrNotFound
	}
	return snippet, err
}

func (s *Store) DeleteSnippet(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM snippets WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) Tags(ctx context.Context) ([]string, error) {
	rows, err := s.db.Query(ctx, `SELECT name FROM tags ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

func replaceTags(ctx context.Context, tx pgx.Tx, snippetID int64, tags []string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM snippet_tags WHERE snippet_id = $1`, snippetID); err != nil {
		return err
	}
	for _, tag := range tags {
		var tagID int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO tags (name) VALUES ($1)
			ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
			RETURNING id
		`, tag).Scan(&tagID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO snippet_tags (snippet_id, tag_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, snippetID, tagID); err != nil {
			return err
		}
	}
	return nil
}

func scanSnippets(rows pgx.Rows) ([]Snippet, error) {
	var snippets []Snippet
	for rows.Next() {
		var snippet Snippet
		if err := rows.Scan(&snippet.ID, &snippet.Title, &snippet.Language, &snippet.Content, &snippet.CreatedAt, &snippet.UpdatedAt, &snippet.Tags); err != nil {
			return nil, err
		}
		snippets = append(snippets, snippet)
	}
	return snippets, rows.Err()
}

func normalizeTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		tag = strings.Trim(tag, "#")
		if tag == "" {
			continue
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		normalized = append(normalized, tag)
	}
	return normalized
}
