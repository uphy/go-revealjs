package revealjs

import (
	"encoding/json"
	"fmt"
)

var properties = map[string]Property{
	"autoAnimateMatcher":   stringProperty(),
	"autoAnimateEasing":    stringProperty(),
	"autoAnimateStyles":    jsonProperty(),
	"autoPlayMedia":        boolProperty(),
	"autoSlideMethod":      stringProperty(),
	"backgroundTransition": choiceProperty([]string{"none", "fade", "slide", "convex", "concave", "zoom"}),
	"controlsLayout":       choiceProperty([]string{"bottom-right", "edges"}),
	"controlsBackArrows":   choiceProperty([]string{"faded", "hidden", "visible"}),
	"defaultTiming":        numberProperty(),
	"display":              stringProperty(),
	"keyboardCondition":    stringProperty(),
	"navigationMode":       choiceProperty([]string{"default", "linear", "grid"}),
	"preloadIframes":       boolProperty(),
	"showSlideNumber":      choiceProperty([]string{"all", "print", "speaker"}),
	"transition":           choiceProperty([]string{"none", "fade", "slide", "convex", "concave", "zoom"}),
	"transitionSpeed":      choiceProperty([]string{"default", "fast", "slow"}),
}

type (
	Property interface {
		ToString(v interface{}) (string, error)
	}
	StringProperty struct {
		validValues []string
	}
	BoolProperty struct {
	}
	NumberProperty struct {
	}
	JSONProperty struct {
	}
)

func configProperty(k string) Property {
	if p, ok := properties[k]; ok {
		return p
	}
	return nil
}

func stringProperty() *StringProperty {
	return &StringProperty{}
}

func choiceProperty(validValues []string) *StringProperty {
	return &StringProperty{validValues: validValues}
}

func boolProperty() *BoolProperty {
	return &BoolProperty{}
}

func numberProperty() *NumberProperty {
	return &NumberProperty{}
}

func jsonProperty() *JSONProperty {
	return &JSONProperty{}
}

func (p *StringProperty) ToString(v interface{}) (string, error) {
	if v == nil {
		return "null", nil
	}
	if len(p.validValues) == 0 {
		return fmt.Sprintf(`'%v'`, v), nil
	}

	// choice
	for _, vv := range p.validValues {
		if v == vv {
			return fmt.Sprintf(`'%v'`, v), nil
		}
	}
	return "", fmt.Errorf("invalid value %v (valid values: %v)", v, p.validValues)
}

func (p *BoolProperty) ToString(v interface{}) (string, error) {
	if v == nil {
		return "null", nil
	}
	if b, ok := v.(bool); ok {
		return fmt.Sprint(b), nil
	}
	return "", fmt.Errorf("invalid value %v, expected boolean value", v)
}

func (p *NumberProperty) ToString(v interface{}) (string, error) {
	if v == nil {
		return "null", nil
	}
	if n, ok := v.(int); ok {
		return fmt.Sprint(n), nil
	}
	if f, ok := v.(float64); ok {
		return fmt.Sprint(f), nil
	}
	if f, ok := v.(float32); ok {
		return fmt.Sprint(f), nil
	}
	return "", fmt.Errorf("invalid value %v, expected number value", v)
}

func (p *JSONProperty) ToString(v interface{}) (string, error) {
	if v == nil {
		return "null", nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
