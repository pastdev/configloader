package config

import (
	"path/filepath"

	"github.com/pastdev/configloader/pkg/meta"
)

// LoadContext is the per-load information supplied to each SourceLoader,
// UnmarshalFunc, and Executor. See meta.LoadContext.
type LoadContext = meta.LoadContext

// WrappedSourceLoader is implemented by a SourceLoader that wraps another
// SourceLoader. The wrapped source is used to find the config dir that
// relativeFile and relativeDir search in "first" and "last" modes.
type WrappedSourceLoader interface {
	SourceLoader
	InnerSource() SourceLoader
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

		nested, ok := src.(WrappedSourceLoader)
		if !ok {
			return "", false
		}
		src = nested.InnerSource()
	}
	return "", false
}
