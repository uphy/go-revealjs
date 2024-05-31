package revealjs

import (
	"embed"
	"errors"
	"io/fs"
	"path/filepath"
)

//go:embed assets/presets
var _initFS embed.FS

//go:embed assets/index.html.tmpl
var _indexHTMLTmpl embed.FS

//go:embed assets/config.yml
var _configYamlFS embed.FS

//go:embed assets/reveal.js/css assets/reveal.js/dist assets/reveal.js/plugin
var _revealjsFS embed.FS

var PresetNames = []string{"default", "demo"}

func presetFS(name string) (fs.FS, error) {
	if !isSupportedPresetName(name) {
		return nil, errors.New("unsupported preset name: " + name)
	}
	return fs.Sub(_initFS, filepath.Join("assets", "presets", name))
}

func indexHTMLTmplFS() fs.FS {
	f, _ := fs.Sub(_indexHTMLTmpl, "assets")
	return f
}

func configYamlFS() fs.FS {
	f, _ := fs.Sub(_configYamlFS, "assets")
	return f
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
