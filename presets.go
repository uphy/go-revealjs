package revealjs

import (
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/uphy/go-revealjs/vfs"
)

// Preset is a collection of files that are used to generate an initial Reveal.js presentation.
type Preset struct {
	fs fs.FS
}

type GenerateOptions struct {
	Force                bool
	GenerateConfig       bool
	GenerateHTMLTemplate bool
}

func NewDefaultPreset() *Preset {
	preset, _ := NewPreset(PresetNames[0])
	return preset
}

func NewPreset(name string) (*Preset, error) {
	preset, err := presetFS(name)
	if err != nil {
		return nil, err
	}
	mfs := vfs.NewMergeFS(preset, defaultFS())
	return &Preset{mfs}, nil
}

func (f *Preset) Generate(dest string, options *GenerateOptions) error {
	// Create 'dest' directory if not exist
	if err := os.MkdirAll(dest, 0700); err != nil {
		return err
	}
	if err := f.extractAll(dest, options); err != nil {
		return err
	}
	return nil
}

func (f *Preset) extractAll(destDir string, options *GenerateOptions) error {
	// Clean destDir if force==true
	if options.Force {
		if exist(destDir) {
			files, err := os.ReadDir(destDir)
			if err != nil {
				return err
			}
			for _, f := range files {
				if err := os.RemoveAll(filepath.Join(destDir, f.Name())); err != nil {
					return err
				}
			}
		}
	}
	return fs.WalkDir(f.fs, ".", func(path string, d fs.DirEntry, err error) error {
		if path == "." {
			return nil
		}
		if path == "index.html.tmpl" && !options.GenerateHTMLTemplate {
			return nil
		}
		if path == "config.yml" && !options.GenerateConfig {
			return nil
		}
		dest := filepath.Join(destDir, path)
		return extract(f.fs, path, dest)
	})
}

func extract(fs fs.FS, src string, dest string) error {
	// Get file info of src file
	file, err := fs.Open(src)
	if err != nil {
		return err
	}
	info, _ := file.Stat()
	if err := file.Close(); err != nil {
		return err
	}

	// If dest file already exist, skip
	exist := exist(dest)
	if exist {
		log.Println("Skipped.  File already exist:", dest)
		return nil
	}

	if info.IsDir() {
		// Create dir
		if err := os.MkdirAll(dest, 0700); err != nil {
			return err
		}
	} else {
		// Copy file
		in, err := fs.Open(src)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(dest)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	}
	return nil
}
