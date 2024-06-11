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
	userFS        fs.FS
}

func NewRevealJS(dataDirectory string) (*RevealJS, error) {
	absDataDir, err := filepath.Abs(dataDirectory)
	if err != nil {
		return nil, err
	}
	if !exist(absDataDir) {
		return nil, errors.New("`dir` not exist")
	}
	userFS := NewSlideResourceFS(os.DirFS(absDataDir))
	systemFS := vfs.NewMergeFS(defaultFS(), revealjsFS())
	mfs := vfs.NewMergeFS(userFS, systemFS)
	revealJS := &RevealJS{nil, absDataDir, true, false, mfs, userFS}
	if err := revealJS.ReloadConfig(); err != nil {
		return nil, err
	}
	return revealJS, nil
}

func (r *RevealJS) ReloadConfig() error {
	configFile, err := r.fs.Open(FileNameConfig)
	if err != nil {
		return err
	}
	c, err := LoadConfigFile(configFile)
	defer configFile.Close()
	if err != nil {
		return err
	}
	r.config = c
	if files, collectErr := r.collectSlideSourceFiles(); collectErr != nil {
		return collectErr
	} else {
		for _, file := range files {
			if IsMarkdown(file) {
				b, err := fs.ReadFile(r.fs, file)
				if err != nil {
					return err
				}
				configInMd, err := LoadConfigFromMarkdown(string(b))
				if err != nil {
					return err
				}
				c.OverrideWith(configInMd)
			}
		}
	}
	return nil
}

type HTMLGeneratorParams struct {
	HotReload bool
	Revision  *string
}

func (r *RevealJS) GenerateIndexHTML(w io.Writer, params *HTMLGeneratorParams) error {
	b, err := fs.ReadFile(r.fs, FileNameIndexHTMLTmpl)
	if err != nil {
		return err
	}
	tmpl, err := template.New(FileNameIndexHTMLTmpl).Parse(string(b))
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
	sections, err := r.generateSections()
	if err != nil {
		return err
	}
	if err := tmpl.Execute(w, map[string]interface{}{
		"config":          r.config,
		"sections":        sections,
		"hotReloadScript": hotReloadScript,
	}); err != nil {
		return err
	}
	return nil
}

func (r *RevealJS) generateSections() ([]string, error) {
	files, err := r.collectSlideSourceFiles()
	if err != nil {
		return nil, err
	}
	return r.doGenerateSections(files), nil
}

func (r *RevealJS) collectSlideSourceFiles() ([]string, error) {
	if r.config.Slides != nil && len(r.config.Slides) > 0 {
		return r.config.Slides, nil
	}

	files := make([]string, 0)
	fs.WalkDir(r.userFS, ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		if IsHTML(path) || IsMarkdown(path) {
			files = append(files, path)
		}
		return nil
	})
	return files, nil
}

func (r *RevealJS) doGenerateSections(files []string) []string {
	sections := make([]string, 0)
	for _, file := range files {
		section, err := r.sectionFor(file)
		if err != nil {
			log.Printf("failed to generate <section> tag for %s: %s", file, err)
		} else {
			sections = append(sections, section)
		}
	}
	return sections
}

func (r *RevealJS) sectionFor(relPathFromDataDirectory string) (string, error) {
	path := filepath.Join(r.dataDirectory, relPathFromDataDirectory)

	if IsHTML(path) {
		if r.EmbedHTML {
			content, err := os.ReadFile(path)
			if err != nil {
				return "", fmt.Errorf("failed to load file %s: %w", path, err)
			}
			return string(content), nil
		}
		return fmt.Sprintf(`<section data-external="%s"></section>`, relPathFromDataDirectory), nil
	} else if IsMarkdown(path) {
		if r.EmbedMarkdown {
			b, err := os.ReadFile(path)
			if err != nil {
				log.Printf("failed to read markdown file: %s", path)
			}
			md := NewMarkdown(string(b)).WithoutYAMLHeader()
			return fmt.Sprintf(`<section data-markdown data-separator="^\r?\n---\r?\n$" data-separator-vertical="^\r?\n~~~\r?\n$">%s</section>`, html.EscapeString(md)), nil
		}
		return fmt.Sprintf(`<section data-markdown="%s" data-separator="^\r?\n---\r?\n$" data-separator-vertical="^\r?\n~~~\r?\n$"></section>`, relPathFromDataDirectory), nil
	} else {
		return "", fmt.Errorf("unsupported slide file: %s", path)
	}
}

func (r *RevealJS) DataDirectory() string {
	return r.dataDirectory
}

func (r *RevealJS) FileSystem() fs.FS {
	return r.fs
}

func (r *RevealJS) Build(dst string) error {
	// Make 'build' directory if not exist
	if err := os.MkdirAll(dst, 0700); err != nil {
		return err
	}

	// generate index.html
	f, err := os.Create(filepath.Join(dst, FileNameIndexHTML))
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
		if filename == FileNameIndexHTMLTmpl || filename == FileNameConfig {
			return true
		}

		// Skip embedded html/md files
		if r.EmbedHTML && IsHTML(filename) {
			return true
		}
		if r.EmbedMarkdown && IsMarkdown(filename) {
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

		var reader io.ReadCloser
		reader, err = fileSystem.Open(path)
		if err != nil {
			return err
		}
		defer reader.Close()

		writer, err := os.Create(copyDst)
		if err != nil {
			return err
		}
		defer writer.Close()

		if IsMarkdown(path) {
			b, err := io.ReadAll(reader)
			if err != nil {
				return err
			}
			content := NewMarkdown(string(b)).WithoutYAMLHeader()
			_, err = writer.WriteString(content)
			if err != nil {
				return err
			}
		} else {
			_, err = io.Copy(writer, reader)
		}
		return err
	})
}

func exist(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true
}
