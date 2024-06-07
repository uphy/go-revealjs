package revealjs

import (
	"errors"
	"fmt"
	"html"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/uphy/go-revealjs/vfs"
)

type RevealJS struct {
	config        *Config
	dataDirectory string
	EmbedHTML     bool
	EmbedMarkdown bool
	fs            fs.FS
}

func NewRevealJS(dataDirectory string) (*RevealJS, error) {
	absDataDir, err := filepath.Abs(dataDirectory)
	if err != nil {
		return nil, err
	}
	if !exist(absDataDir) {
		return nil, errors.New("`dir` not exist")
	}
	mfs := vfs.NewMergeFS(os.DirFS(absDataDir), indexHTMLTmplFS(), configYamlFS(), revealjsFS())
	revealJS := &RevealJS{nil, absDataDir, true, false, mfs}
	if err := revealJS.ReloadConfig(); err != nil {
		return nil, err
	}
	return revealJS, nil
}

func (r *RevealJS) ReloadConfig() error {
	configFile, err := r.fs.Open("config.yml")
	if err != nil {
		return err
	}
	c, err := LoadConfigFile(configFile)
	defer configFile.Close()
	if err != nil {
		return err
	}
	r.config = c
	return nil
}

type HTMLGeneratorParams struct {
	HotReload bool
	Revision  *string
}

func (r *RevealJS) GenerateIndexHTML(w io.Writer, params *HTMLGeneratorParams) error {
	b, err := fs.ReadFile(r.fs, "index.html.tmpl")
	if err != nil {
		return err
	}
	tmpl, err := template.New("index.html.tmpl").Parse(string(b))
	if err != nil {
		return err
	}
	var hotReloadScript string
	if params.HotReload {
		hotReloadScript = `<script>
const revision = "__REVISION__";
async function reloadCheck() {
	const baseUrl = window.location.href.split("/").slice(0, 3).join("/");
	const response = await fetch(baseUrl + "/revision");
	const newRevision = await response.text();
	if (revision !== newRevision) {
		window.location.reload();
	}
}
setInterval(function() {
		reloadCheck().catch(console.error);
}, 1000);
</script>`
		if params.Revision != nil {
			hotReloadScript = strings.ReplaceAll(hotReloadScript, "__REVISION__", *params.Revision)
		}
	} else {
		hotReloadScript = ""
	}
	if err := tmpl.Execute(w, map[string]interface{}{
		"config":          r.config,
		"sections":        r.generateSections(),
		"hotReloadScript": hotReloadScript,
	}); err != nil {
		return err
	}
	return nil
}

func (r *RevealJS) generateSections() []string {
	sections := make([]string, 0)
	if r.config.Slides == nil || len(r.config.Slides) == 0 {
		buildDir := r.BuildDirectory()
		if err := filepath.Walk(r.dataDirectory, func(path string, info os.FileInfo, err error) error {
			if strings.HasPrefix(path, buildDir) {
				return nil
			}

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

func (r *RevealJS) DataDirectory() string {
	return r.dataDirectory
}

func (r *RevealJS) BuildDirectory() string {
	return filepath.Join(r.dataDirectory, r.config.BuildDirectory)
}

func (r *RevealJS) FileSystem() fs.FS {
	return r.fs
}

func (r *RevealJS) Build() error {
	dst := r.BuildDirectory()

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
	if err := r.GenerateIndexHTML(f, &HTMLGeneratorParams{
		HotReload: false,
		Revision:  nil,
	}); err != nil {
		return err
	}
	defer f.Close()

	// copy fs files
	return extractFile(r.fs, ".", dst, func(path string) bool {
		// Skip paths under dst directory
		absSrc := filepath.Join(r.dataDirectory, path)
		if strings.HasPrefix(absSrc, dst) {
			return true
		}

		// Skip index.html.tmpl and config.yml
		filename := filepath.Base(path)
		if filename == "index.html.tmpl" || filename == "config.yml" {
			return true
		}

		return false
	})
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
