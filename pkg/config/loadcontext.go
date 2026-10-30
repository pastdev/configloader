package config

import (
	"context"
	"path/filepath"
)

// loadFileKey and loadDirsKey are the context keys for the per-load
// information template functions need to resolve paths relative to config
// sources.
type (
	loadFileKey struct{}
	loadDirsKey struct{}
)

// withLoadFile returns ctx carrying file, the YAML file currently being
// unmarshaled (for "this").
func withLoadFile(ctx context.Context, file string) context.Context {
	return context.WithValue(ctx, loadFileKey{}, file)
}

func loadFile(ctx context.Context) string {
	file, _ := ctx.Value(loadFileKey{}).(string)
	return file
}

// withLoadDirs returns ctx carrying dirs, the config directories of the
// current Sources list, in load order, after normalizePath (for "first" and
// "last").
func withLoadDirs(ctx context.Context, dirs []string) context.Context {
	return context.WithValue(ctx, loadDirsKey{}, dirs)
}

func loadDirs(ctx context.Context) []string {
	dirs, _ := ctx.Value(loadDirsKey{}).([]string)
	return dirs
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
