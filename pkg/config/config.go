package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// YamlUnmarshal is an Unmarshal function that unmarshals from yaml.
func YamlUnmarshal[T any](b []byte, cfg *T) error {
	err := yaml.Unmarshal(b, cfg)
	if err != nil {
		return fmt.Errorf("yamlunmarshal: %w", err)
	}
	return nil
}
