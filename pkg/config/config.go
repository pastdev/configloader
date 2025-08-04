package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pastdev/configloader/pkg/log"
	"gopkg.in/yaml.v3"
)

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
	start := time.Now()
	for _, src := range s {
		err := src.Load(cfg)
		if err != nil {
			return fmt.Errorf("load: %w", err)
		}
	}
	log.Logger.Debug().Dur("duration", time.Since(start)).Msg("load complete")
	return nil
}

func normalizePath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Logger.Trace().Err(err).Msg("User home directory not defined")
			return path
		}
		path = filepath.Join(homeDir, path[1:])
	}
	return path
}

func unmarshal[T any](b []byte, cfg *T, unmarshal func(b []byte, cfg *T) error) error {
	if unmarshal == nil {
		unmarshal = YamlUnmarshal[T]()
	}

	err := unmarshal(b, cfg)
	if err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}

// YamlUnmarshal is an Unmarshal function that unmarshals from yaml.
func YamlUnmarshal[T any]() func(b []byte, cfg *T) error {
	return func(b []byte, cfg *T) error {
		err := yaml.Unmarshal(b, cfg)
		if err != nil {
			return fmt.Errorf("yamlunmarshal: %w", err)
		}
		return nil
	}
}
