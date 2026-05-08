CREATE TABLE IF NOT EXISTS snippets (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    language TEXT NOT NULL DEFAULT 'plaintext',
    content TEXT NOT NULL,
    search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('simple', coalesce(content, '')), 'B') ||
        setweight(to_tsvector('simple', coalesce(language, '')), 'C')
    ) STORED,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tags (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE CHECK (name = lower(name) AND name <> '')
);

CREATE TABLE IF NOT EXISTS snippet_tags (
    snippet_id BIGINT NOT NULL REFERENCES snippets(id) ON DELETE CASCADE,
    tag_id BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (snippet_id, tag_id)
);

CREATE INDEX IF NOT EXISTS snippets_search_idx ON snippets USING GIN (search_vector);
CREATE INDEX IF NOT EXISTS snippets_updated_at_idx ON snippets (updated_at DESC);
CREATE INDEX IF NOT EXISTS tags_name_idx ON tags (name);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS snippets_set_updated_at ON snippets;
CREATE TRIGGER snippets_set_updated_at
BEFORE UPDATE ON snippets
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();
