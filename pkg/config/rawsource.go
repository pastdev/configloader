package config

import (
	"fmt"
)

type RawSource[T any] struct {
	Data []byte
	// Unmarshal is the function to unmarshal the data from the file into the
	// cfg object. If not specified YamlUnmarshal will be used.
	Unmarshal func(b []byte, cfg *T) error
}

func (s RawSource[T]) Load(cfg *T) error {
	err := unmarshal(s.Data, cfg, s.Unmarshal)
	if err != nil {
		return fmt.Errorf("load from raw: %w", err)
	}

	return nil
}

func (s RawSource[T]) String() string {
	return "rawsource"
}
