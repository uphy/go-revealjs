package revealjs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Slides          []string               `yaml:"slides"`
	Title           string                 `yaml:"title"`
	Theme           string                 `yaml:"theme"`
	RevealJS        map[string]interface{} `yaml:"revealjs"`
	InternalPlugins []interface{}          `yaml:"plugins"`
}

type Plugin struct {
	Name string `yaml:"name"`
	Src  string `yaml:"src"`
}

func LoadConfigFile(reader io.Reader) (*Config, error) {
	loadedConfig, err := doLoadConfigFile(reader)
	if err != nil {
		return nil, err
	}

	// Derive from default config
	defaultConfigFile, err := defaultConfigYAML()
	if err != nil {
		log.Fatal(err)
	}
	cfg, err := doLoadConfigFile(defaultConfigFile)
	if err != nil {
		log.Fatal(err)
	}
	cfg.OverrideWith(loadedConfig)
	return cfg, nil
}

func LoadConfigFromMarkdown(content string) (*Config, error) {
	md := NewMarkdown(content)
	if header, err := md.YAMLHeader(); err != nil {
		return nil, err
	} else {
		b, err := yaml.Marshal(header)
		if err != nil {
			return nil, err
		}
		return doLoadConfigFile(bytes.NewReader(b))
	}
}

func (c *Config) OverrideWith(other *Config) {
	if other.Title != "" {
		c.Title = other.Title
	}
	if other.Theme != "" {
		c.Theme = other.Theme
	}
	if other.InternalPlugins != nil {
		c.InternalPlugins = other.InternalPlugins
	}
	if c.RevealJS == nil {
		c.RevealJS = map[string]interface{}{}
	}
	for k, v := range other.RevealJS {
		c.RevealJS[k] = v
	}
}

func doLoadConfigFile(reader io.Reader) (*Config, error) {
	var c Config
	b, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) Plugins() []Plugin {
	plugins := []Plugin{}
	for _, v := range c.InternalPlugins {
		var plugin Plugin
		if src, ok := v.(string); ok {
			plugin = Plugin{
				Name: src,
				Src:  "",
			}
		} else {
			plugin = Plugin{}
			b, _ := json.Marshal(v)
			json.Unmarshal(b, &plugin)
		}
		if plugin.Src == "" {
			switch plugin.Name {
			case "RevealHighlight":
				plugin.Src = "plugin/highlight/highlight.js"
			case "RevealMarkdown":
				plugin.Src = "plugin/markdown/markdown.js"
			case "RevealSearch":
				plugin.Src = "plugin/search/search.js"
			case "RevealNotes":
				plugin.Src = "plugin/notes/notes.js"
			case "RevealMath":
				plugin.Src = "plugin/math/math.js"
			case "RevealZoom":
				plugin.Src = "plugin/zoom/zoom.js"
			default:
				log.Fatalf("plugin %s is not supported", plugin.Name)
			}
		}
		plugins = append(plugins, plugin)
	}
	return plugins
}

func (c *Config) RevealJSConfig() (map[string]string, error) {
	m := map[string]string{}

	// config from file
	for k, v := range c.RevealJS {
		s, err := c.valueToString(k, v)
		if err != nil {
			return nil, fmt.Errorf("error in config '%s': %v", k, err)
		}
		m[k] = s
	}
	return m, nil
}

func (c *Config) valueToString(k string, v interface{}) (string, error) {
	if p := configProperty(k); p != nil {
		s, err := p.ToString(v)
		if err != nil {
			return "", err
		}
		return s, nil
	}
	if k == "plugins" {
		return "", errors.New("'revealjs.plugins' is not supported, use 'plugins' instead")
	}
	return fmt.Sprint(v), nil
}
