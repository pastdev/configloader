package config

import (
	"context"
	"fmt"
)

type RawSource struct {
	Data []byte
	// Unmarshal is the function to unmarshal the data from supplied byte slice.
	// If not specified YamlUnmarshal will be used.
	Unmarshal UnmarshalFunc
}

func (s RawSource) Load(ctx context.Context, merge Merger) error {
	var doc any
	err := unmarshal(ctx, s.Data, &doc, s.Unmarshal)
	if err != nil {
		return fmt.Errorf("load from raw: %w", err)
	}

	merge(doc)

	return nil
}

func (s RawSource) String() string {
	return "rawsource"
}
