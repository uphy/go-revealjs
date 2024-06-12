package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/uphy/go-revealjs"
)

type (
	BuildResultAsserter struct {
		Dir string
	}
	IndexHTMLAsserter struct {
		HTML string
	}
)

func Run(t *testing.T, check func(asserter *BuildResultAsserter)) {
	wd, err := os.Getwd()
	if err != nil {
		t.Errorf("failed to get working directory: %s", err)
	}
	t.Run(wd, func(t *testing.T) {
		if err := build(wd, check); err != nil {
			t.Log(err)
			t.Fail()
		}
	})
}

func build(wd string, check func(result *BuildResultAsserter)) error {
	testName := filepath.Base(wd)
	dataDir := filepath.Join(wd, "testdata")
	r, err := revealjs.NewRevealJS(dataDir)
	if err != nil {
		return err
	}
	r.EmbedHTML = true
	r.EmbedMarkdown = true

	dir, err := os.MkdirTemp("", fmt.Sprintf("revealjs-test-%s-*", testName))
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	if err := r.Build(dir); err != nil {
		return err
	}
	check(&BuildResultAsserter{Dir: dir})
	return nil
}

func (r *BuildResultAsserter) HasRevealJSFiles(t *testing.T) {
	r.HasDirectory(t, "dist")
	r.HasFile(t, "dist/reveal.css")
	r.HasFile(t, "dist/reveal.js")
	r.HasFile(t, "dist/theme/black.css")

	r.HasDirectory(t, "plugin")
	r.HasFile(t, "plugin/markdown/markdown.js")

	r.HasFile(t, "index.html")
}

func (r *BuildResultAsserter) HasDirectory(t *testing.T, name string) {
	if s, err := os.Stat(filepath.Join(r.Dir, name)); err != nil || !s.IsDir() {
		t.Errorf("directory %s not found", name)
	}
}

func (r *BuildResultAsserter) HasFile(t *testing.T, name string) {
	if _, err := os.Stat(filepath.Join(r.Dir, name)); err != nil {
		t.Errorf("file %s not found", name)
	}
}

func (r *BuildResultAsserter) NotHasFile(t *testing.T, pattern string) {
	matches, err := filepath.Glob(filepath.Join(r.Dir, pattern))
	if err != nil {
		t.Errorf("failed to glob: %s", err)
	}
	if len(matches) > 0 {
		t.Errorf("file %s found: %v", pattern, matches)
	}
}

func (r *BuildResultAsserter) IndexHTML(t *testing.T) *IndexHTMLAsserter {
	indexHTML := filepath.Join(r.Dir, "index.html")
	b, err := os.ReadFile(indexHTML)
	if err != nil {
		t.Errorf("failed to read index.html: %s", err)
	}
	html := normalizeText(string(b))
	return &IndexHTMLAsserter{HTML: html}
}

func normalizeText(s string) string {
	// remove all whitespace of each line start
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	return strings.Join(lines, "\n")
}

func (a *IndexHTMLAsserter) HasString(t *testing.T, s string) {
	if !strings.Contains(a.HTML, normalizeText(s)) {
		t.Errorf("string %s not found", s)
		t.Errorf("index.html: %s", a.HTML)
	}
}

func (a *IndexHTMLAsserter) HasTitle(t *testing.T, title string) {
	a.HasString(t, fmt.Sprintf("<title>%s</title>", title))
}

func (a *IndexHTMLAsserter) HasTheme(t *testing.T, theme string) {
	a.HasString(t, fmt.Sprintf(`<link rel="stylesheet" href="dist/theme/%s.css">`, theme))
}

func (a *IndexHTMLAsserter) HasScriptTag(t *testing.T, src string) {
	a.HasString(t, fmt.Sprintf(`<script src="%s"></script>`, src))
}

func (a *IndexHTMLAsserter) HasStandardScriptTags(t *testing.T) {
	a.HasScriptTag(t, "dist/reveal.js")
	a.HasScriptTag(t, "plugin/markdown/markdown.js")
	a.HasScriptTag(t, "plugin/highlight/highlight.js")
	a.HasScriptTag(t, "plugin/search/search.js")
	a.HasScriptTag(t, "plugin/notes/notes.js")
	a.HasScriptTag(t, "plugin/math/math.js")
	a.HasScriptTag(t, "plugin/zoom/zoom.js")
}

func (a *IndexHTMLAsserter) HasConfigProperty(t *testing.T, key, value string) {
	a.HasString(t, fmt.Sprintf(`%s: %s`, key, value))
}
