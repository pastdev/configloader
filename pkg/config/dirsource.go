package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pastdev/configloader/pkg/log"
	"github.com/rs/zerolog"
)

// DirSource is a directory containing config files to load. The files within
// the directory will be processed in order, sorted by filename, with later
// values overriding existing values.
type DirSource[T any] struct {
	Path string
	// Unmarshal is the function to unmarshal the data from each file into the
	// cfg object. If not specified YamlUnmarshal will be used.
	Unmarshal func(b []byte, cfg *T) error
}

func (s DirSource[T]) Load(cfg *T) error {
	dir := normalizePath(s.Path)
	listing, err := os.ReadDir(dir)
	if err != nil {
		log.Logger.Debug().Str("dir", dir).Msg("no configs found")
		//nolint: nilerr // intentional ignore error
		return nil
	}

	files := zerolog.Arr()
	for _, entry := range listing {
		name := entry.Name()
		if !entry.Type().IsRegular() {
			if entry.IsDir() {
				log.Logger.Debug().
					Str("dir", dir).
					Str("subdir", entry.Name()).
					Msg("skipping subdir")
				continue
			}

			path, err := filepath.EvalSymlinks(filepath.Join(dir, entry.Name()))
			if err != nil {
				return fmt.Errorf("eval symlink: %w", err)
			}
			entry, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("stat: %w", err)
			}
			if entry.IsDir() {
				log.Logger.Debug().
					Str("dir", dir).
					Str("symlinkSubdir", entry.Name()).
					Msg("skipping subdir")
				continue
			}
			name = entry.Name()
		}

		file := filepath.Join(dir, name)
		b, err := os.ReadFile(file)
		if err != nil {
			log.Logger.Debug().Str("file", file).Msg("config not found")
			//nolint: nilerr // intentional ignore error
			return nil
		}

		files.Str(file)
		err = unmarshal(b, cfg, s.Unmarshal)
		if err != nil {
			return fmt.Errorf("load from dir: %w", err)
		}
	}

	log.Logger.Debug().Str("dir", s.Path).Array("files", files).Msg("loaded dirsource config")
	return nil
}

func (s DirSource[T]) String() string {
	return fmt.Sprintf("dirsource:%s", s.Path)
}
