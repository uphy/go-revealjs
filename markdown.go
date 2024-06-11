package revealjs

import (
	"regexp"

	"github.com/ghodss/yaml"
)

var yamlHeaderRegexp = regexp.MustCompile(`(?s)^---\n(.*?)\n---\n+`)

type Markdown struct {
	content string
}

func NewMarkdown(content string) *Markdown {
	return &Markdown{content}
}

func (m *Markdown) WithoutYAMLHeader() string {
	return yamlHeaderRegexp.ReplaceAllString(m.content, "")
}

func (m *Markdown) YAMLHeader() (map[string]interface{}, error) {
	header := make(map[string]interface{})
	matches := yamlHeaderRegexp.FindStringSubmatch(m.content)
	if len(matches) == 2 {
		if err := yaml.Unmarshal([]byte(matches[1]), &header); err != nil {
			return nil, err
		}
	}
	return header, nil
}
