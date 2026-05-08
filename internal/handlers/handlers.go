package handlers

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Pkkarthik12/Snippetbox-personal-snippet-manager/internal/config"
	"github.com/Pkkarthik12/Snippetbox-personal-snippet-manager/internal/models"
	"github.com/Pkkarthik12/Snippetbox-personal-snippet-manager/internal/render"
)

type Dependencies struct {
	Config   config.Config
	Store    *models.Store
	Renderer *render.Renderer
}

type App struct {
	cfg      config.Config
	store    *models.Store
	renderer *render.Renderer
}

type pageData struct {
	Snippets []models.Snippet
	Snippet  models.Snippet
	Tags     []string
	Query    string
	Tag      string
	Error    string
}

func New(deps Dependencies) *App {
	return &App{cfg: deps.Config, store: deps.Store, renderer: deps.Renderer}
}

func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))

	mux.HandleFunc("GET /", a.home)
	mux.HandleFunc("GET /snippets/new", a.newSnippet)
	mux.HandleFunc("POST /snippets", a.createSnippet)
	mux.HandleFunc("GET /snippets/{id}", a.showSnippet)
	mux.HandleFunc("DELETE /snippets/{id}", a.deleteSnippet)
	mux.HandleFunc("POST /snippets/{id}/delete", a.deleteSnippet)

	mux.HandleFunc("GET /api/snippets", a.apiListSnippets)
	mux.HandleFunc("POST /api/snippets", a.apiCreateSnippet)
	mux.HandleFunc("GET /api/snippets/{id}", a.apiGetSnippet)
	mux.HandleFunc("DELETE /api/snippets/{id}", a.apiDeleteSnippet)

	var handler http.Handler = maxBody(a.cfg.MaxBodySize, mux)
	if a.cfg.AuthEnabled {
		handler = a.basicAuth(handler)
	}
	return handler
}

func (a *App) home(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	tag := r.URL.Query().Get("tag")
	snippets, err := a.store.ListSnippets(r.Context(), models.ListFilters{Query: query, Tag: tag})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tags, err := a.store.Tags(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	templateName := "index.html"
	if r.Header.Get("HX-Request") == "true" {
		templateName = "partials/list.html"
	}
	a.renderer.HTML(w, http.StatusOK, templateName, pageData{Snippets: snippets, Tags: tags, Query: query, Tag: tag})
}

func (a *App) newSnippet(w http.ResponseWriter, r *http.Request) {
	a.renderer.HTML(w, http.StatusOK, "new.html", nil)
}

func (a *App) createSnippet(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.renderer.HTML(w, http.StatusBadRequest, "new.html", pageData{Error: "Could not read form"})
		return
	}

	snippet, err := a.store.CreateSnippet(r.Context(), models.CreateSnippetParams{
		Title:    r.FormValue("title"),
		Language: r.FormValue("language"),
		Content:  r.FormValue("content"),
		Tags:     splitTags(r.FormValue("tags")),
	})
	if err != nil {
		a.renderer.HTML(w, http.StatusBadRequest, "new.html", pageData{Error: err.Error()})
		return
	}

	http.Redirect(w, r, "/snippets/"+strconv.FormatInt(snippet.ID, 10), http.StatusSeeOther)
}

func (a *App) showSnippet(w http.ResponseWriter, r *http.Request) {
	snippet, err := a.getSnippetFromPath(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	a.renderer.HTML(w, http.StatusOK, "show.html", pageData{Snippet: snippet})
}

func (a *App) deleteSnippet(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := a.store.DeleteSnippet(r.Context(), id); err != nil {
		if errors.Is(err, models.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *App) apiListSnippets(w http.ResponseWriter, r *http.Request) {
	snippets, err := a.store.ListSnippets(r.Context(), models.ListFilters{
		Query: r.URL.Query().Get("q"),
		Tag:   r.URL.Query().Get("tag"),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, snippets)
}

func (a *App) apiCreateSnippet(w http.ResponseWriter, r *http.Request) {
	var params models.CreateSnippetParams
	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	snippet, err := a.store.CreateSnippet(r.Context(), params)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, snippet)
}

func (a *App) apiGetSnippet(w http.ResponseWriter, r *http.Request) {
	snippet, err := a.getSnippetFromPath(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, snippet)
}

func (a *App) apiDeleteSnippet(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err := a.store.DeleteSnippet(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) getSnippetFromPath(r *http.Request) (models.Snippet, error) {
	id, err := parseID(r)
	if err != nil {
		return models.Snippet{}, err
	}
	return a.store.GetSnippet(r.Context(), id)
}

func (a *App) basicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		validUser := subtle.ConstantTimeCompare([]byte(username), []byte(a.cfg.AuthUsername)) == 1
		validPass := subtle.ConstantTimeCompare([]byte(password), []byte(a.cfg.AuthPassword)) == 1
		if !ok || !validUser || !validPass {
			w.Header().Set("WWW-Authenticate", `Basic realm="Snippetbox"`)
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func maxBody(size int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if size > 0 {
			r.Body = http.MaxBytesReader(w, r.Body, size)
		}
		next.ServeHTTP(w, r)
	})
}

func parseID(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func splitTags(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == '\t'
	})
	return fields
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
