package config

import (
	"fmt"
)

type RawSource struct {
	Data []byte
	// Unmarshal is the function to unmarshal the data from supplied byte slice.
	// If not specified YamlUnmarshal will be used.
	Unmarshal func(b []byte, cfg any) error
}

func (s RawSource) Load(merge Merger) error {
	var doc any
	err := unmarshal(s.Data, &doc, s.Unmarshal)
	if err != nil {
		return fmt.Errorf("load from raw: %w", err)
	}

	merge(doc)

	return nil
}

func (s RawSource) String() string {
	return "rawsource"
}
