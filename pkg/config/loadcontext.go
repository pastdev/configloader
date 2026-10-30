package config

import (
	"path/filepath"
)

// LoadContext is the per-load information supplied to each SourceLoader,
// UnmarshalFunc, and Executor. Template funcs such as relativeFile use it to
// resolve paths relative to config sources.
type LoadContext struct {
	// File is the file currently being unmarshaled (for "this"). It is empty
	// for sources that are not backed by a file, such as RawSource.
	File string
	// Dirs is the config directories of the current Sources list, in load
	// order, after normalizePath (for "first" and "last").
	Dirs []string
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
