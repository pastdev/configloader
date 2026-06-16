package config

import (
	"fmt"
	"os"

	"github.com/pastdev/configloader/pkg/log"
)

// FileSource is a config file to load.
type FileSource struct {
	Path string
	// Unmarshal is the function to unmarshal the data from file.
	// If not specified YamlUnmarshal will be used.
	Unmarshal func(b []byte, cfg any) error
}

func (s FileSource) Load(merge Merger) error {
	b, err := os.ReadFile(normalizePath(s.Path))
	if err != nil {
		log.Logger.Debug().Str("file", s.Path).Msg("config not found")
		//nolint: nilerr // intentional ignore error
		return nil
	}

	var doc any
	err = unmarshal(b, &doc, s.Unmarshal)
	if err != nil {
		return fmt.Errorf("load from file: %w", err)
	}

	merge(doc)

	log.Logger.Debug().Str("file", s.Path).Msg("loaded filesource config")
	return nil
}

func (s FileSource) String() string {
	return fmt.Sprintf("filesource:%s", s.Path)
}
