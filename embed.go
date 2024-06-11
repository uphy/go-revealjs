package revealjs

import (
	"embed"
	"errors"
	"io/fs"
	"path/filepath"
)

//go:embed assets/presets
var _initFS embed.FS

//go:embed assets/index.html.tmpl assets/config.yml
var _defaultFS embed.FS

//go:embed assets/reveal.js/css assets/reveal.js/dist assets/reveal.js/plugin
var _revealjsFS embed.FS

var PresetNames = []string{"default", "demo"}

func presetFS(name string) (fs.FS, error) {
	if !isSupportedPresetName(name) {
		return nil, errors.New("unsupported preset name: " + name)
	}
	return fs.Sub(_initFS, filepath.Join("assets", "presets", name))
}

// defaultFS returns the default files
// - index.html.tmpl
// - config.yml
func defaultFS() fs.FS {
	f, _ := fs.Sub(_defaultFS, "assets")
	return f
}

func defaultConfigYAML() (fs.File, error) {
	return defaultFS().Open(FileNameConfig)
}

func revealjsFS() fs.FS {
	f, _ := fs.Sub(_revealjsFS, "assets/reveal.js")
	return f
}

func isSupportedPresetName(name string) bool {
	for _, n := range PresetNames {
		if name == n {
			return true
		}
	}
	return false
}
