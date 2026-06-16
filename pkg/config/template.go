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
	"github.com/pastdev/configloader/pkg/xdg"
	"gopkg.in/yaml.v3"
)

func DefaultFuncMap() map[string]any {
	funcs := map[string]any{}
	bitwarden.New().AddFuncs(funcs)
	lastpass.New().AddFuncs(funcs)
	xdg.AddFuncs(funcs)
	return funcs
}

type Template struct {
	funcMap map[string]any
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

// YamlValueTemplateUnmarshal returns an Unmarshal function that unmarshals from
// yaml, then processes each _value_ individually through the go template engine.
// The unmarshal function expects to be given a reference to one of:
//
//	*any
//	*map[any]any
//	*map[string]any
//	*[]any
func YamlValueTemplateUnmarshal(executor Executor) func(b []byte, cfg any) error {
	return func(b []byte, doc any) error {
		err := yaml.Unmarshal(b, doc)
		if err != nil {
			return fmt.Errorf("yamlunmarshal: %w", err)
		}

		exec := executor
		if exec == nil {
			exec = NewTemplate(DefaultFuncMap())
		}

		switch typed := doc.(type) {
		case *any:
			// this case uses the internal helper method because it needs to
			// replace the root of the value in case it was a scalar. all other
			// cases just replace within the map/slice so no need to replace
			// root
			value, err := walk(exec, *typed, []string{})
			if err != nil {
				return fmt.Errorf("yamlunmarshal walk valueMap: %w", err)
			}
			*typed = value
			return nil

		case *map[any]any:
			err := Walk(exec, *typed)
			if err != nil {
				return fmt.Errorf("yamlunmarshal walk valueMap: %w", err)
			}
			return nil

		case *map[string]any:
			err := Walk(exec, *typed)
			if err != nil {
				return fmt.Errorf("yamlunmarshal walk valueMap: %w", err)
			}
			return nil

		case *[]any:
			err := Walk(exec, *typed)
			if err != nil {
				return fmt.Errorf("yamlunmarshal walk valueMap: %w", err)
			}
			return nil

		default:
			return fmt.Errorf("yamlunmarshal unsupported destination type %T", doc)
		}
	}
}
