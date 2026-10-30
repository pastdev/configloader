package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"strconv"
	"strings"
	"text/template"

	"github.com/pastdev/configloader/internal/fallback"
	"github.com/pastdev/configloader/pkg/bitwarden"
	"github.com/pastdev/configloader/pkg/lastpass"
	"github.com/pastdev/configloader/pkg/meta"
	"github.com/pastdev/configloader/pkg/xdg"
	"gopkg.in/yaml.v3"
)

func DefaultFuncMap() map[string]any {
	funcs := map[string]any{}
	bitwarden.New().AddFuncs(funcs)
	lastpass.New().AddFuncs(funcs)
	xdg.AddFuncs(funcs)
	AddBaseFuncs(funcs)
	meta.AddFuncs(funcs)
	return funcs
}

// AddBaseFuncs adds home, user, and userRaw to funcs.
func AddBaseFuncs(funcs template.FuncMap) {
	funcs["home"] = home
	funcs["user"] = currentUsername
	funcs["userRaw"] = currentUsernameRaw
}

var (
	osUserHomeDir = os.UserHomeDir
	userCurrent   = user.Current
)

// home returns os.UserHomeDir. If that fails, the optional fallback is used if
// supplied.
func home(fallbacks ...string) (string, error) {
	h, err := osUserHomeDir()
	if err != nil {
		if v, ok := fallback.First(fallbacks...); ok {
			return v, nil
		}
		return "", fmt.Errorf("home: %w", err)
	}
	return h, nil
}

// currentUsername returns the current username lowercased and with any
// windows domain prefix (DOMAIN\) removed. If the lookup fails, the optional
// fallback is used if supplied.
func currentUsername(fallbacks ...string) (string, error) {
	username, err := currentUsernameRaw(fallbacks...)
	if err != nil {
		return "", err
	}

	username = strings.ToLower(username)
	if idx := strings.Index(username, "\\"); idx != -1 {
		username = username[idx+1:]
	}
	return username, nil
}

// currentUsernameRaw returns the current username as reported by the OS. If
// the lookup fails, the optional fallback is used if supplied.
func currentUsernameRaw(fallbacks ...string) (string, error) {
	u, err := userCurrent()
	if err != nil {
		if v, ok := fallback.First(fallbacks...); ok {
			return v, nil
		}
		return "", fmt.Errorf("user: %w", err)
	}
	return u.Username, nil
}

// Template executes values as go templates using its func map. A func map
// entry of type func(LoadContext) any is treated as a factory for a
// LoadContext bound func: on each Execute it is called with the supplied lc
// and its result is used as the template func of that name.
type Template struct {
	funcMap map[string]any
}

type Executor interface {
	// Executes the template stored in value and returns the result. The name
	// value is intended for error reporting only to provide context as to
	// which template failed. The lc is the one supplied to the
	// UnmarshalFunc.
	Execute(lc LoadContext, name string, value any) (any, error)
}

func (t *Template) Execute(lc LoadContext, name string, value any) (any, error) {
	str, ok := value.(string)
	if !ok || !strings.Contains(str, "{{") {
		return value, nil
	}

	tmpl, err := template.New(name).Funcs(t.funcs(lc)).Parse(str)
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

// funcs returns the func map with each func(LoadContext) any factory bound to
// lc.
func (t *Template) funcs(lc LoadContext) template.FuncMap {
	funcs := make(template.FuncMap, len(t.funcMap))
	for k, v := range t.funcMap {
		switch typed := v.(type) {
		case func(LoadContext) any:
			funcs[k] = typed(lc)
		default:
			funcs[k] = v
		}
	}
	return funcs
}

func NewTemplate(funcMap template.FuncMap) *Template {
	return &Template{
		funcMap: funcMap,
	}
}

// Walk will recursively iterate over all the nodes of data calling callback
// for each node.
func Walk(lc LoadContext, callback Executor, data any) error {
	_, err := walk(lc, callback, data, []string{})
	if err != nil {
		return err
	}
	return nil
}

func walk(lc LoadContext, callback Executor, node any, keyStack []string) (any, error) {
	switch typed := node.(type) {
	case map[string]any:
		// json deserialized
		for k, v := range typed {
			keyStack := append(keyStack, fmt.Sprintf("%v", k))
			newValue, err := walk(lc, callback, v, keyStack)
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
			newValue, err := walk(lc, callback, v, keyStack)
			if err != nil {
				return nil, fmt.Errorf("walk array: %w", err)
			}
			typed[k] = newValue
		}
		return typed, nil
	case []any:
		for i, j := range typed {
			keyStack := append(keyStack, strconv.Itoa(i))
			newValue, err := walk(lc, callback, j, keyStack)
			if err != nil {
				return nil, fmt.Errorf("walk array: %w", err)
			}
			typed[i] = newValue
		}
		return typed, nil
	default:
		v, err := callback.Execute(lc, fmt.Sprintf("/%s", strings.Join(keyStack, "/")), node)
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
	return func(lc LoadContext, b []byte, doc any) error {
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
			value, err := walk(lc, exec, *typed, []string{})
			if err != nil {
				return fmt.Errorf("yamlunmarshal walk valueMap: %w", err)
			}
			*typed = value
			return nil

		case *map[any]any:
			err := Walk(lc, exec, *typed)
			if err != nil {
				return fmt.Errorf("yamlunmarshal walk valueMap: %w", err)
			}
			return nil

		case *map[string]any:
			err := Walk(lc, exec, *typed)
			if err != nil {
				return fmt.Errorf("yamlunmarshal walk valueMap: %w", err)
			}
			return nil

		case *[]any:
			err := Walk(lc, exec, *typed)
			if err != nil {
				return fmt.Errorf("yamlunmarshal walk valueMap: %w", err)
			}
			return nil

		default:
			return fmt.Errorf("yamlunmarshal unsupported destination type %T", doc)
		}
	}
}
