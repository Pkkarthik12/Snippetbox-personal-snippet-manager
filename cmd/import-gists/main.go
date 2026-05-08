package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/Pkkarthik12/Snippetbox-personal-snippet-manager/internal/config"
	"github.com/Pkkarthik12/Snippetbox-personal-snippet-manager/internal/models"
)

type gist struct {
	ID          string              `json:"id"`
	Description string              `json:"description"`
	Files       map[string]gistFile `json:"files"`
}

type gistFile struct {
	Filename string `json:"filename"`
	Language string `json:"language"`
	Content  string `json:"content"`
}

func main() {
	_ = godotenv.Load()

	username := flag.String("username", "", "GitHub username to import gists from")
	flag.Parse()
	if *username == "" {
		slog.Error("--username is required")
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()
	store, err := models.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	token := os.Getenv("GITHUB_TOKEN")
	gists, err := fetchGists(ctx, *username, token)
	if err != nil {
		slog.Error("fetch gists", "error", err)
		os.Exit(1)
	}

	count := 0
	for _, g := range gists {
		fullGist, err := fetchGist(ctx, g.ID, token)
		if err != nil {
			slog.Warn("skip gist", "id", g.ID, "error", err)
			continue
		}
		g = fullGist
		for _, file := range g.Files {
			if strings.TrimSpace(file.Content) == "" {
				continue
			}
			title := file.Filename
			if strings.TrimSpace(g.Description) != "" {
				title = g.Description + " - " + file.Filename
			}
			_, err := store.CreateSnippet(ctx, models.CreateSnippetParams{
				Title:    title,
				Language: normalizeLanguage(file.Language),
				Content:  file.Content,
				Tags:     gistTags(g.Description, file.Language),
			})
			if err != nil {
				slog.Warn("skip gist file", "file", file.Filename, "error", err)
				continue
			}
			count++
		}
	}

	fmt.Printf("Imported %d gist files\n", count)
}

func fetchGists(ctx context.Context, username, token string) ([]gist, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	var all []gist

	for page := 1; ; page++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://api.github.com/users/%s/gists?per_page=100&page=%d", username, page), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("github returned %s", resp.Status)
		}

		var pageGists []gist
		if err := json.NewDecoder(resp.Body).Decode(&pageGists); err != nil {
			return nil, err
		}
		if len(pageGists) == 0 {
			break
		}
		all = append(all, pageGists...)
		if len(pageGists) < 100 {
			break
		}
	}

	return all, nil
}

func fetchGist(ctx context.Context, id, token string) (gist, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/gists/"+id, nil)
	if err != nil {
		return gist{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return gist{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return gist{}, fmt.Errorf("github returned %s", resp.Status)
	}

	var g gist
	if err := json.NewDecoder(resp.Body).Decode(&g); err != nil {
		return gist{}, err
	}
	return g, nil
}

func gistTags(description, language string) []string {
	tags := []string{"gist"}
	if language != "" {
		tags = append(tags, strings.ToLower(language))
	}
	for _, word := range strings.Fields(description) {
		word = strings.Trim(strings.ToLower(word), "#,.;:()[]{}")
		if strings.HasPrefix(word, "#") {
			tags = append(tags, strings.TrimPrefix(word, "#"))
		}
	}
	return tags
}

func normalizeLanguage(language string) string {
	if strings.TrimSpace(language) == "" {
		return "plaintext"
	}
	return strings.ToLower(strings.ReplaceAll(language, " ", ""))
}
