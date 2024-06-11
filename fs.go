package revealjs

import (
	"io/fs"
	"path/filepath"
	"strings"
)

const (
	FileNameConfig        = "config.yml"
	FileNameIndexHTMLTmpl = "index.html.tmpl"
	FileNameIndexHTML     = "index.html"
	DirNameSlides         = "slides"
	DirNameAssets         = "assets"
)

// SlideResourceFS is a file system that provides slides and assets.
// - *.md
// - *.html
// - slides/*.md
// - slides/*.html
// - assets/**/*
// - config.yml
// - index.html.tmpl
type SlideResourceFS struct {
	fs fs.FS
}

func NewSlideResourceFS(fs fs.FS) *SlideResourceFS {
	return &SlideResourceFS{fs}
}

func (s *SlideResourceFS) Open(name string) (fs.File, error) {
	if name == "." {
		return s.fs.Open(name)
	}

	dir, file := filepath.Split(name)
	if dir == "" || dir == DirNameSlides {
		if IsMarkdown(file) || IsHTML(file) {
			return s.fs.Open(name)
		}
	}
	if s.hasPrefix(name, DirNameAssets) {
		return s.fs.Open(name)
	}
	if dir == "" && (file == FileNameConfig || file == FileNameIndexHTMLTmpl) {
		return s.fs.Open(name)
	}
	return nil, fs.ErrNotExist
}

func (f *SlideResourceFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == "." || name == DirNameSlides {
		entries, err := fs.ReadDir(f.fs, name)
		if err != nil {
			return nil, err
		}
		var filteredEntries []fs.DirEntry
		for _, entry := range entries {
			if name == "." && entry.IsDir() && (entry.Name() == DirNameSlides || entry.Name() == DirNameAssets) {
				filteredEntries = append(filteredEntries, entry)
			} else if !entry.IsDir() && (IsMarkdown(entry.Name()) || IsHTML(entry.Name())) {
				filteredEntries = append(filteredEntries, entry)
			} else if name == "." && (entry.Name() == FileNameConfig || entry.Name() == FileNameIndexHTMLTmpl) {
				filteredEntries = append(filteredEntries, entry)
			}
		}

		return filteredEntries, nil
	}
	if f.hasPrefix(name, DirNameAssets) {
		return fs.ReadDir(f.fs, name)
	}

	return nil, nil
}

func (f *SlideResourceFS) hasPrefix(path string, prefix string) bool {
	if path == prefix {
		return true
	}
	return strings.HasPrefix(path, prefix+filepath.FromSlash("/"))
}

func IsMarkdown(path string) bool {
	return filepath.Ext(path) == ".md"
}

func IsHTML(path string) bool {
	return filepath.Ext(path) == ".html"
}
