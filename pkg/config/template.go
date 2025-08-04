package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"github.com/pastdev/configloader/pkg/bitwarden"
	"github.com/pastdev/configloader/pkg/lastpass"
	"gopkg.in/yaml.v3"
)

var DefaultFuncMap = map[string]any{
	"bitwardenFormat": bitwarden.GetFormat,
	"bitwardenJSON":   bitwarden.GetJSON,
	"lastpassFormat":  lastpass.GetFormat,
	"lastpassJSON":    lastpass.GetJSON,
}

type Template struct {
	funcMap template.FuncMap
}

type Executor interface {
	// Executes the template stored in value and returns the result. The name
	// value is intended for error reporting only to provide context as to
	// which template failed.
	Execute(name string, value any) (any, error)
}

func (t *Template) Execute(name string, value any) (any, error) {
	str, ok := value.(string)
	if !ok || !strings.Contains(str, "{{") {
		return value, nil
	}

	tmpl, err := template.New(name).Funcs(t.funcMap).Parse(str)
	if err != nil {
		return nil, fmt.Errorf("new template: %w", err)
	}

	var newValue bytes.Buffer
	err = tmpl.Execute(&newValue, nil)
	if err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	raw := newValue.Bytes()
	var parsed any
	err = json.Unmarshal(raw, &parsed)
	if err != nil {
		//nolint: nilerr // json parse is best effort
		return string(raw), nil
	}
	return parsed, nil
}

func NewTemplate(funcMap template.FuncMap) *Template {
	return &Template{
		funcMap: funcMap,
	}
}

// Walk will recursively iterate over all the nodes of data calling callback
// for each node.
func Walk(callback Executor, data any) error {
	_, err := walk(callback, data, []string{})
	if err != nil {
		return err
	}
	return nil
}

func walk(callback Executor, node any, keyStack []string) (any, error) {
	switch typed := node.(type) {
	case map[string]any:
		// json deserialized
		for k, v := range typed {
			keyStack := append(keyStack, fmt.Sprintf("%v", k))
			newValue, err := walk(callback, v, keyStack)
			if err != nil {
				return nil, fmt.Errorf("walk array: %w", err)
			}
			typed[k] = newValue
		}
		return typed, nil
	case map[any]any:
		// yaml deserialized
		for k, v := range typed {
			keyStack := append(keyStack, fmt.Sprintf("%v", k))
			newValue, err := walk(callback, v, keyStack)
			if err != nil {
				return nil, fmt.Errorf("walk array: %w", err)
			}
			typed[k] = newValue
		}
		return typed, nil
	case []any:
		for i, j := range typed {
			keyStack := append(keyStack, strconv.Itoa(i))
			newValue, err := walk(callback, j, keyStack)
			if err != nil {
				return nil, fmt.Errorf("walk array: %w", err)
			}
			typed[i] = newValue
		}
		return typed, nil
	default:
		v, err := callback.Execute(fmt.Sprintf("/%s", strings.Join(keyStack, "/")), node)
		if err != nil {
			return nil, fmt.Errorf("execute template: %w", err)
		}
		return v, nil
	}
}

// YamlValueTemplateUnmarshal is an Unmarshal function that unmarshals from
// yaml, then processes each _value_ individually through the go template engine
// then reserializes the result to yaml before unmarshaling into T.
func YamlValueTemplateUnmarshal[T any](executor Executor) func(b []byte, cfg *T) error {
	return func(b []byte, cfg *T) error {
		var valueMap map[any]any
		err := yaml.Unmarshal(b, &valueMap)
		if err != nil {
			return fmt.Errorf("yamlunmarshal to valueMap: %w", err)
		}

		if executor == nil {
			executor = NewTemplate(DefaultFuncMap)
		}

		// walk the map and template each value
		err = Walk(executor, valueMap)
		if err != nil {
			return fmt.Errorf("yamlunmarshal walk valueMap: %w", err)
		}

		data, err := yaml.Marshal(valueMap)
		if err != nil {
			return fmt.Errorf("yamlunmarshal from valueMap: %w", err)
		}

		err = yaml.Unmarshal(data, cfg)
		if err != nil {
			return fmt.Errorf("yamlunmarshal to type: %w", err)
		}
		return nil
	}
}
