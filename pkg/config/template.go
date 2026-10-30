package config

import (
	"bytes"
	"context"
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
	funcs["home"] = home
	funcs["user"] = currentUsername
	funcs["userRaw"] = currentUsernameRaw
	funcs["relativeFile"] = func(ctx context.Context) any {
		return func(path string, mode ...string) (string, error) {
			return relativeFile(ctx, path, mode...)
		}
	}
	funcs["relativeDir"] = func(ctx context.Context) any {
		return func(path string, mode ...string) (string, error) {
			return relativeDir(ctx, path, mode...)
		}
	}
	return funcs
}

// Template executes values as go templates using its func map. A func map
// entry of type func(context.Context) any is treated as a factory for a
// context-bound func: on each Execute it is called with the supplied ctx and
// its result is used as the template func of that name.
type Template struct {
	funcMap    map[string]any
	contextual map[string]func(context.Context) any
}

type Executor interface {
	// Executes the template stored in value and returns the result. The name
	// value is intended for error reporting only to provide context as to
	// which template failed. The ctx is the one supplied to the
	// UnmarshalFunc.
	Execute(ctx context.Context, name string, value any) (any, error)
}

func (t *Template) Execute(ctx context.Context, name string, value any) (any, error) {
	str, ok := value.(string)
	if !ok || !strings.Contains(str, "{{") {
		return value, nil
	}

	tmpl := template.New(name).Funcs(t.funcMap)
	if len(t.contextual) > 0 {
		bound := make(template.FuncMap, len(t.contextual))
		for k, factory := range t.contextual {
			bound[k] = factory(ctx)
		}
		tmpl = tmpl.Funcs(bound)
	}

	tmpl, err := tmpl.Parse(str)
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
	contextual := map[string]func(context.Context) any{}
	for k, v := range funcMap {
		if factory, ok := v.(func(context.Context) any); ok {
			contextual[k] = factory
		}
	}
	return &Template{
		funcMap:    funcMap,
		contextual: contextual,
	}
}

// Walk will recursively iterate over all the nodes of data calling callback
// for each node.
func Walk(ctx context.Context, callback Executor, data any) error {
	_, err := walk(ctx, callback, data, []string{})
	if err != nil {
		return err
	}
	return nil
}

func walk(ctx context.Context, callback Executor, node any, keyStack []string) (any, error) {
	switch typed := node.(type) {
	case map[string]any:
		// json deserialized
		for k, v := range typed {
			keyStack := append(keyStack, fmt.Sprintf("%v", k))
			newValue, err := walk(ctx, callback, v, keyStack)
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
			newValue, err := walk(ctx, callback, v, keyStack)
			if err != nil {
				return nil, fmt.Errorf("walk array: %w", err)
			}
			typed[k] = newValue
		}
		return typed, nil
	case []any:
		for i, j := range typed {
			keyStack := append(keyStack, strconv.Itoa(i))
			newValue, err := walk(ctx, callback, j, keyStack)
			if err != nil {
				return nil, fmt.Errorf("walk array: %w", err)
			}
			typed[i] = newValue
		}
		return typed, nil
	default:
		v, err := callback.Execute(ctx, fmt.Sprintf("/%s", strings.Join(keyStack, "/")), node)
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
func YamlValueTemplateUnmarshal(executor Executor) UnmarshalFunc {
	return func(ctx context.Context, b []byte, doc any) error {
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
			value, err := walk(ctx, exec, *typed, []string{})
			if err != nil {
				return fmt.Errorf("yamlunmarshal walk valueMap: %w", err)
			}
			*typed = value
			return nil

		case *map[any]any:
			err := Walk(ctx, exec, *typed)
			if err != nil {
				return fmt.Errorf("yamlunmarshal walk valueMap: %w", err)
			}
			return nil

		case *map[string]any:
			err := Walk(ctx, exec, *typed)
			if err != nil {
				return fmt.Errorf("yamlunmarshal walk valueMap: %w", err)
			}
			return nil

		case *[]any:
			err := Walk(ctx, exec, *typed)
			if err != nil {
				return fmt.Errorf("yamlunmarshal walk valueMap: %w", err)
			}
			return nil

		default:
			return fmt.Errorf("yamlunmarshal unsupported destination type %T", doc)
		}
	}
}
