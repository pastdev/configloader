package config

import (
	"path/filepath"
)

// loadContext is the per-load information template functions need to resolve
// paths relative to config sources.
type loadContext struct {
	// file is the YAML file currently being unmarshaled (for "this").
	file string
	// dirs is the config directories of the current Sources list, in load
	// order, after normalizePath (for "first" and "last").
	dirs []string
}

var currentLoadContext loadContext

func withLoadFile(file string, fn func() error) error {
	prev := currentLoadContext.file
	currentLoadContext.file = file
	defer func() { currentLoadContext.file = prev }()
	return fn()
}

func withLoadDirs(dirs []string, fn func() error) error {
	prev := currentLoadContext.dirs
	currentLoadContext.dirs = dirs
	defer func() { currentLoadContext.dirs = prev }()
	return fn()
}

func configDirs(sources []SourceLoader) []string {
	var dirs []string
	for _, src := range sources {
		if dir, ok := configDirOf(src); ok {
			dirs = append(dirs, dir)
		}
	}
	return dirs
}

func configDirOf(src SourceLoader) (string, bool) {
	for src != nil {
		switch s := src.(type) {
		case FileSource:
			return filepath.Dir(normalizePath(s.Path)), true
		case *FileSource:
			return filepath.Dir(normalizePath(s.Path)), true
		case DirSource:
			return normalizePath(s.Path), true
		case *DirSource:
			return normalizePath(s.Path), true
		}

		nested, ok := src.(interface{ InnerSource() SourceLoader })
		if !ok {
			return "", false
		}
		src = nested.InnerSource()
	}
	return "", false
}
