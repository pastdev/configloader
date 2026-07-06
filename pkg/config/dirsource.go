package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pastdev/configloader/pkg/log"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

// configloaderFile is the name of the optional control file, read from a
// DirSource's directory, that governs which files in that directory are
// eligible for loading. It is always skipped as a config file itself.
const configloaderFile = ".configloader"

// defaultIncludes is the glob pattern set used to select files when a
// directory has no .configloader file, or has one that doesn't specify
// Include.
var defaultIncludes = []string{"*.yml", "*.yaml"}

// DirSourceControl is the schema for the optional .configloader file. When
// present in a DirSource's directory, it governs which files in that
// directory are eligible for loading, in addition to the built-in defaults.
type DirSourceControl struct {
	// Include is the set of glob patterns (matched with filepath.Match
	// against a file's base name) a file must satisfy at least one of to be
	// eligible for loading. If nil, the built-in default of "*.yml" and
	// "*.yaml" is used. If non-nil (including an empty list), it entirely
	// replaces the built-in default.
	Include []string `yaml:"include"`
	// Exclude is a set of glob patterns, matched the same way as Include,
	// that remove files from the eligible set even if they matched Include.
	Exclude []string `yaml:"exclude"`
}

// DirSource is a directory containing config files to load. The files within
// the directory will be processed in order, sorted by filename, with later
// values overriding existing values.
//
// Which files are considered is governed by an optional .configloader file
// in the directory (see DirSourceControl). Without one, only *.yml and
// *.yaml files are considered.
type DirSource struct {
	Path string
	// Unmarshal is the function to unmarshal the data from each file in the
	// directory. If not specified YamlUnmarshal will be used.
	Unmarshal func(b []byte, cfg any) error
}

func (s DirSource) Load(merge Merger) error {
	dir := normalizePath(s.Path)
	listing, err := os.ReadDir(dir)
	if err != nil {
		log.Logger.Debug().Str("dir", dir).Msg("no configs found")
		//nolint: nilerr // intentional ignore error
		return nil
	}

	control, err := loadDirSourceControl(dir)
	if err != nil {
		return fmt.Errorf("load control: %w", err)
	}

	includes := control.Include
	if includes == nil {
		includes = defaultIncludes
	}

	files := zerolog.Arr()
	for _, entry := range listing {
		entryName := entry.Name()
		if entryName == configloaderFile {
			continue
		}

		name := entryName
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

		eligible, err := dirSourceEligible(entryName, includes, control.Exclude)
		if err != nil {
			return fmt.Errorf("filter: %w", err)
		}
		if !eligible {
			log.Logger.Debug().
				Str("dir", dir).
				Str("file", entryName).
				Msg("skipping filtered file")
			continue
		}

		file := filepath.Join(dir, name)
		//nolint:gosec // intent is to allow user specified config directory/file
		b, err := os.ReadFile(file)
		if err != nil {
			log.Logger.Debug().Str("file", file).Msg("config not found")
			//nolint: nilerr // intentional ignore error
			return nil
		}

		files.Str(file)
		var doc any
		err = unmarshal(b, &doc, s.Unmarshal)
		if err != nil {
			return fmt.Errorf("load from dir: %w", err)
		}

		merge(doc)
	}

	log.Logger.Debug().Str("dir", s.Path).Array("files", files).Msg("loaded dirsource config")
	return nil
}

func (s DirSource) String() string {
	return fmt.Sprintf("dirsource:%s", s.Path)
}

// loadDirSourceControl reads and parses the .configloader file from dir, if
// present. A missing file is not an error; it results in a zero-value
// DirSourceControl, which selects the built-in default include patterns.
func loadDirSourceControl(dir string) (DirSourceControl, error) {
	//nolint:gosec // intent is to allow user specified config directory/file
	b, err := os.ReadFile(filepath.Join(dir, configloaderFile))
	if err != nil {
		if os.IsNotExist(err) {
			return DirSourceControl{}, nil
		}
		return DirSourceControl{}, fmt.Errorf("read %s: %w", configloaderFile, err)
	}

	var control DirSourceControl
	err = yaml.Unmarshal(b, &control)
	if err != nil {
		return DirSourceControl{}, fmt.Errorf("unmarshal %s: %w", configloaderFile, err)
	}
	return control, nil
}

// dirSourceEligible reports whether name is eligible for loading: it must
// match at least one include pattern, and must not match any exclude
// pattern.
func dirSourceEligible(name string, includes, excludes []string) (bool, error) {
	included, err := dirSourceMatchAny(name, includes)
	if err != nil {
		return false, err
	}
	if !included {
		return false, nil
	}

	excluded, err := dirSourceMatchAny(name, excludes)
	if err != nil {
		return false, err
	}
	return !excluded, nil
}

func dirSourceMatchAny(name string, patterns []string) (bool, error) {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, name)
		if err != nil {
			return false, fmt.Errorf("match pattern %q: %w", pattern, err)
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}
