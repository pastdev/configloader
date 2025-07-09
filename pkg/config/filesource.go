package config

import (
	"fmt"
	"os"
)

// FileSource is a config file to load.
type FileSource[T any] struct {
	Path string
	// Unmarshal is the function to unmarshal the data from the file into the
	// cfg object. If not specified YamlUnmarshal will be used.
	Unmarshal func(b []byte, cfg *T) error
}

func (s FileSource[T]) Load(cfg *T) error {
	b, err := os.ReadFile(normalizePath(s.Path))
	if err != nil {
		Logger.Debug().Str("file", s.Path).Msg("config not found")
		//nolint: nilerr // intentional ignore error
		return nil
	}

	err = unmarshal(b, cfg, s.Unmarshal)
	if err != nil {
		return fmt.Errorf("load from dir: %w", err)
	}

	Logger.Debug().Str("file", s.Path).Msg("loaded filesource config")
	return nil
}

func (s FileSource[T]) String() string {
	return fmt.Sprintf("filesource:%s", s.Path)
}
