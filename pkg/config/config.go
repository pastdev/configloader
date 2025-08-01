package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/pastdev/configloader/pkg/bitwarden"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// Logger is the logger used by this library
var Logger zerolog.Logger

// SourceLoader is the primary interface for loading configuration from a
// source.
type SourceLoader[T any] interface {
	Load(cfg *T) error
	String() string
}

// Sources is an aggregate of SourceLoaders that is used to load and merge
// configuration.
type Sources[T any] []SourceLoader[T]

// Load will load the configuration from all the Sources. Each SourceLoader will
// load its values over the top of the previous loaders directly into the
// supplied cfg object.
func (s Sources[T]) Load(cfg *T) error {
	for _, src := range s {
		err := src.Load(cfg)
		if err != nil {
			return fmt.Errorf("load: %w", err)
		}
	}
	return nil
}

func DefaultTemplateFunc(keystack []string, value any) (any, error) {
	str, ok := value.(string)
	if !ok || !strings.Contains(str, "{{") {
		return nil, nil
	}

	tmpl, err := template.
		New("value").
		Funcs(template.FuncMap{
			"bitwardenFormat": bitwarden.GetFormat,
			"bitwardenJSON":   bitwarden.GetJSON,
		}).
		Parse(str)
	if err != nil {
		return nil, fmt.Errorf("new template (%s): %w", strings.Join(keystack, ","), err)
	}

	var newValue bytes.Buffer
	err = tmpl.Execute(&newValue, nil)
	if err != nil {
		return nil, fmt.Errorf("execute template (%s): %w", strings.Join(keystack, ","), err)
	}

	return newValue.String(), nil
}

func normalizePath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Trace().Err(err).Msg("User home directory not defined")
			return path
		}
		path = filepath.Join(homeDir, path[1:])
	}
	return path
}

func unmarshal[T any](b []byte, cfg *T, unmarshal func(b []byte, cfg *T) error) error {
	if unmarshal == nil {
		unmarshal = YamlUnmarshal
	}

	err := unmarshal(b, cfg)
	if err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}

// Walk will recursively iterate over all the nodes of data calling callback
// for each node.
func Walk(callback func(key []string, value any) (any, error), data any) error {
	_, err := walk(callback, data, []string{})
	if err != nil {
		return err
	}
	return nil
}

func walk(callback func(key []string, value any) (any, error), node any, keyStack []string) (any, error) {
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
		return nil, nil
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
		return nil, nil
	case []any:
		for i, j := range typed {
			keyStack := append(keyStack, strconv.Itoa(i))
			newValue, err := walk(callback, j, keyStack)
			if err != nil {
				return nil, fmt.Errorf("walk array: %w", err)
			}
			typed[i] = newValue
		}
		return nil, nil
	default:
		return callback(keyStack, node)
	}
}

// YamlUnmarshal is an Unmarshal function that unmarshals from yaml.
func YamlUnmarshal[T any](b []byte, cfg *T) error {
	err := yaml.Unmarshal(b, cfg)
	if err != nil {
		return fmt.Errorf("yamlunmarshal: %w", err)
	}
	return nil
}

// YamlValueTemplateUnmarshal is an Unmarshal function that unmarshals from
// yaml, then processes each _value_ individually through the go template engine
// then reserializes the result to yaml before unmarshaling into T.
func YamlValueTemplateUnmarshal[T any](
	b []byte,
	cfg *T,
	templateFunc func(keystack []string, value any) (any, error),
) error {
	var valueMap map[any]any
	err := yaml.Unmarshal(b, &valueMap)
	if err != nil {
		return fmt.Errorf("yamlunmarshal to valueMap: %w", err)
	}

	if templateFunc == nil {
		templateFunc = DefaultTemplateFunc
	}

	// walk the map and template each value
	err = Walk(templateFunc, valueMap)
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
