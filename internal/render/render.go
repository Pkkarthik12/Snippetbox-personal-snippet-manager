package render

import (
	"bytes"
	"html/template"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
)

type Renderer struct {
	templates *template.Template
}

func New(dir string) (*Renderer, error) {
	funcs := template.FuncMap{
		"highlight": highlight,
		"join":      strings.Join,
	}

	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".html" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("").Funcs(funcs).ParseFiles(files...)
	if err != nil {
		return nil, err
	}

	return &Renderer{templates: tpl}, nil
}

func (r *Renderer) HTML(w http.ResponseWriter, status int, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := r.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template render error", http.StatusInternalServerError)
	}
}

func highlight(language, content string) template.HTML {
	var buf bytes.Buffer
	lang := strings.TrimSpace(language)
	if lang == "" {
		lang = "plaintext"
	}
	if err := quick.Highlight(&buf, content, lang, "html", "github"); err != nil {
		template.HTMLEscape(&buf, []byte(content))
	}
	return template.HTML(buf.String())
}
