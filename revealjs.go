package revealjs

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

type RevealJS struct {
	config        *Config
	dataDirectory string
	indexTemplate string
	EmbedHTML     bool
	EmbedMarkdown bool
}

const (
	dataDirectoryName = "data"
	markdownSection   = `<section data-markdown="%s" data-separator="^\r?\n---\r?\n$" data-separator-vertical="^\r?\n~~~\r?\n$"></section>`
)

func NewRevealJS(dataDirectory string) (*RevealJS, error) {
	if !exist(dataDirectory) {
		return nil, errors.New("`dir` not exist")
	}
	indexTemplate := filepath.Join(dataDirectory, "index.html.tmpl")
	return &RevealJS{nil, dataDirectory, indexTemplate, true, false}, nil
}

func (r *RevealJS) reloadConfig() error {
	configFile := filepath.Join(r.dataDirectory, "config.yml")
	if !exist(configFile) {
		fs := NewDefaultPreset()
		if err := fs.Generate(r.dataDirectory, false); err != nil {
			return err
		}
	}
	c, err := LoadConfigFile(configFile)
	if err != nil {
		return err
	}
	r.config = c
	return nil
}

func (r *RevealJS) Start() error {
	r.Reconfigure()
	// TODO 終了処理いらないっけ？
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
			// Is index.html
			if req.URL.Path == "/" {
				// Generate index.html
				buf := &bytes.Buffer{}
				if err := r.generateIndexHTML(buf); err != nil {
					http.Error(w, "failed to generate index.html", http.StatusInternalServerError)
					return
				}
				http.ServeContent(w, req, "index.html", time.Now(), bytes.NewReader(buf.Bytes()))
				return
			}

			// Host user data files
			// TODO user dataとreveal.jsのファイルが重複する場合の処理 (/data以下でホストするとか。外部mdを読み込むケースのパス指定方法変更も必要)
			dataFile := filepath.Join(r.dataDirectory, req.URL.Path)
			if exist(dataFile) {
				http.ServeFile(w, req, dataFile)
				return
			}

			// Fallback to reveal.js files
			http.ServeFileFS(w, req, revealjsFS(), req.URL.Path)
		})
		log.Println("Start server on http://localhost:8080")
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			log.Fatal("Failed to start server: ", err)
		}
	}()
	watcher, err := NewWatcher(r)
	if err != nil {
		return err
	}
	go watcher.Start()
	return nil
}

func (r *RevealJS) Reconfigure() {
	if err := r.reloadConfig(); err != nil {
		log.Println("failed to reload config.yml: ", err)
	}
}

func (r *RevealJS) generateIndexHTML(w io.Writer) error {
	b, err := os.ReadFile(r.indexTemplate)
	if err != nil {
		return err
	}
	tmpl, err := template.New("index.html.tmpl").Parse(string(b))
	if err != nil {
		return err
	}
	if err := tmpl.Execute(w, map[string]interface{}{
		"config":   r.config,
		"sections": r.generateSections(),
	}); err != nil {
		return err
	}
	return nil
}

func (r *RevealJS) generateSections() []string {
	sections := make([]string, 0)
	if r.config.Slides == nil || len(r.config.Slides) == 0 {
		if err := filepath.Walk(r.dataDirectory, func(path string, info os.FileInfo, err error) error {
			p, _ := filepath.Rel(r.dataDirectory, path)
			section := r.sectionFor(p)
			if len(section) > 0 {
				sections = append(sections, section)
			}
			return nil
		}); err != nil {
			log.Println("failed to walk data directory: ", err)
		}
	} else {
		for _, s := range r.config.Slides {
			section := r.sectionFor(s)
			if len(section) > 0 {
				sections = append(sections, section)
			} else {
				log.Println("unsupported slide file: ", s)
			}
		}
	}
	return sections
}

func (r *RevealJS) sectionFor(relPathFromDataDirectory string) string {
	path := filepath.Join(r.dataDirectory, relPathFromDataDirectory)

	switch filepath.Ext(path) {
	case ".html":
		if r.EmbedHTML {
			content, err := os.ReadFile(path)
			if err != nil {
				log.Printf("failed to load file %s: %s", path, err)
				return ""
			}
			return string(content)
		}
		return fmt.Sprintf(`<section data-external="%s"></section>`, relPathFromDataDirectory)
	case ".md":
		if r.EmbedMarkdown {
			b, err := os.ReadFile(path)
			if err != nil {
				log.Printf("failed to read markdown file: %s", path)
			}
			return fmt.Sprintf(`<section data-markdown data-separator="^\r?\n---\r?\n$" data-separator-vertical="^\r?\n~~~\r?\n$">%s</section>`, html.EscapeString(string(b)))
		}
		return fmt.Sprintf(`<section data-markdown="%s" data-separator="^\r?\n---\r?\n$" data-separator-vertical="^\r?\n~~~\r?\n$"></section>`, relPathFromDataDirectory)
	default:
		return ""
	}
}

func (r *RevealJS) UpdateSlideFile(file string) {
	r.Reconfigure()
}

func (r *RevealJS) DataDirectory() string {
	return r.dataDirectory
}

func (r *RevealJS) Build(dst string) error {
	r.Reconfigure()

	// Make 'build' directory if not exist
	if err := os.MkdirAll(dst, 0700); err != nil {
		return err
	}
	// clean
	files, err := os.ReadDir(dst)
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := os.RemoveAll(filepath.Join(dst, f.Name())); err != nil {
			return err
		}
	}

	// generate index.html
	f, err := os.Create(filepath.Join(dst, "index.html"))
	if err != nil {
		return err
	}
	if err := r.generateIndexHTML(f); err != nil {
		return err
	}
	defer f.Close()

	// copy embed files
	for _, src := range []string{"dist", "css", "plugin"} {
		if err := extractFile(revealjsFS(), src, filepath.Join(dst, src), func(path string) bool {
			return false
		}); err != nil {
			return err
		}
	}

	// copy user files
	absDst, err := filepath.Abs(dst)
	if err != nil {
		return err
	}
	absDataDir, err := filepath.Abs(r.dataDirectory)
	if err != nil {
		return err
	}
	if err := extractFile(os.DirFS(absDataDir), ".", dst, func(path string) bool {
		// Skip paths under dst directory
		absSrc := filepath.Join(absDataDir, path)
		if strings.HasPrefix(absSrc, absDst) {
			return true
		}

		// Skip index.html.tmpl and config.yml
		filename := filepath.Base(path)
		if filename == "index.html.tmpl" || filename == "config.yml" {
			return true
		}

		return false
	}); err != nil {
		return err
	}

	return nil
}

// extractFile copies files from src to dst.
// src is a path from the root of the file system
// dst is a path of the local file system
func extractFile(fileSystem fs.FS, src, dst string, skip func(path string) bool) error {
	return fs.WalkDir(fileSystem, src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if skip(path) {
			return nil
		}

		relPath, _ := filepath.Rel(src, path)
		copyDst := filepath.Join(dst, relPath)
		if d.IsDir() {
			return os.MkdirAll(copyDst, 0700)
		}
		reader, err := fileSystem.Open(path)
		if err != nil {
			return err
		}
		defer reader.Close()

		writer, err := os.Create(copyDst)
		if err != nil {
			return err
		}
		defer writer.Close()

		_, err = io.Copy(writer, reader)
		return err
	})
}

func exist(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}
