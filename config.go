package revealjs

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/ghodss/yaml"
)

type Config struct {
	Slides          []string               `json:"slides"`
	Title           string                 `json:"title"`
	Theme           string                 `json:"theme"`
	BuildDirectory  string                 `json:"buildDir"`
	RevealJS        map[string]interface{} `json:"revealjs"`
	InternalPlugins []interface{}          `json:"plugins"`
}

type Plugin struct {
	Name string `json:"name"`
	Src  string `json:"src"`
}

func LoadConfigFile(reader io.Reader) (*Config, error) {
	c, err := doLoadConfigFile(reader)
	if err != nil {
		return nil, err
	}

	// Derive from default config
	defaultConfigFile, err := configYamlFS().Open("config.yml")
	if err != nil {
		log.Fatal(err)
	}
	defaultConfig, err := doLoadConfigFile(defaultConfigFile)
	if err != nil {
		log.Fatal(err)
	}
	if c.Title == "" {
		c.Title = defaultConfig.Title
	}
	if c.Theme == "" {
		c.Theme = defaultConfig.Theme
	}
	if c.BuildDirectory == "" {
		c.BuildDirectory = defaultConfig.BuildDirectory
	}
	if c.InternalPlugins == nil {
		c.InternalPlugins = defaultConfig.InternalPlugins
	}
	mergedRevealJS := defaultConfig.RevealJS
	for k, v := range c.RevealJS {
		mergedRevealJS[k] = v
	}
	c.RevealJS = mergedRevealJS
	return c, nil
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

func (c *Config) RevealJSConfig() map[string]string {
	m := map[string]string{}

	// config from env
	p := regexp.MustCompile(`REVEALJS_(.*?)=(.*)`)
	for _, env := range os.Environ() {
		if match := p.FindStringSubmatch(env); len(match) > 0 {
			key := match[1]
			value := match[2]
			m[key] = c.valueToString(key, value)
		}
	}
	// config from file
	for k, v := range c.RevealJS {
		m[k] = c.valueToString(k, v)
	}
	return m
}

func (c *Config) valueToString(k string, v interface{}) string {
	switch k {
	case "controlsLayout", "controlsBackArrows", "transition", "transitionSpeed", "backgroundTransition", "parallaxBackgroundImage", "parallaxBackgroundSize", "display", "showSlideNumber", "parallaxBackgroundPosition", "parallaxBackgroundRepeat", "autoAnimateMatcher", "navigationMode", "preloadIframes", "autoAnimateEasing":
		if v == nil {
			return "null"
		}
		return fmt.Sprintf(`'%v'`, v)
	case "autoPlayMedia", "autoSlideMethod", "defaultTiming", "parallaxBackgroundHorizontal", "parallaxBackgroundVertical", "keyboardCondition":
		if v == nil {
			return "null"
		}
		return fmt.Sprint(v)
	case "autoAnimateStyles":
		b, _ := json.Marshal(v)
		return string(b)
	case "plugins":
		b, _ := json.Marshal(v)
		return strings.ReplaceAll(string(b), `"`, ``)
	default:
		return fmt.Sprint(v)
	}
}
