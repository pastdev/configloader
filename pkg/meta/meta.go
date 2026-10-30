// Package meta provides template functions that expose metadata from the
// config loader itself, such as the file currently being loaded and the
// config directories of the current sources.
package meta

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// relativeMode is how RelativeFile and RelativeDir resolve a path. See
// parseRelativeMode for the template syntax of each.
type relativeMode int

const (
	relativeModeThis relativeMode = iota
	relativeModeFirst
	relativeModeLast
)

// LoadContext is the per-load information supplied to each SourceLoader,
// UnmarshalFunc, and Executor. Template funcs such as relativeFile use it to
// resolve paths relative to config sources.
type LoadContext struct {
	// File is the file currently being unmarshaled (for "this"). It is empty
	// for sources that are not backed by a file, such as RawSource.
	File string
	// Dirs is the config directories of the current Sources list, in load
	// order, after ~/ expansion (for "first" and "last").
	Dirs []string
}

// AddFuncs adds relativeFile and relativeDir to funcs. Each is added as a
// func(LoadContext) any factory, which config.Template binds to the
// LoadContext of the load in progress.
func AddFuncs(funcs template.FuncMap) {
	funcs["relativeFile"] = func(lc LoadContext) any {
		return func(path string, mode ...string) (string, error) {
			return RelativeFile(lc, path, mode...)
		}
	}
	funcs["relativeDir"] = func(lc LoadContext) any {
		return func(path string, mode ...string) (string, error) {
			return RelativeDir(lc, path, mode...)
		}
	}
}

// RelativeFile resolves path, which must be relative and may not contain
// "..", to an existing file using lc. The optional mode is one of "this"
// (default: relative to the directory of lc.File), "first" (the first match
// along lc.Dirs), or "last" (the last match along lc.Dirs).
func RelativeFile(lc LoadContext, path string, mode ...string) (string, error) {
	return resolveRelative(lc, "relativeFile", path, false, mode...)
}

// RelativeDir is RelativeFile for a directory.
func RelativeDir(lc LoadContext, path string, mode ...string) (string, error) {
	return resolveRelative(lc, "relativeDir", path, true, mode...)
}

func resolveRelative(lc LoadContext, kind, rel string, wantDir bool, modeArgs ...string) (string, error) {
	if err := validateRelativePath(rel); err != nil {
		return "", fmt.Errorf("%s: %w", kind, err)
	}

	mode, err := parseRelativeMode(modeArgs)
	if err != nil {
		return "", fmt.Errorf("%s: %w", kind, err)
	}

	switch mode {
	case relativeModeThis:
		resolved, err := resolveThis(lc, rel)
		if err != nil {
			return "", fmt.Errorf("%s: %w", kind, err)
		}
		return checkRelativeType(rel, resolved, wantDir)
	case relativeModeFirst, relativeModeLast:
		resolved, err := resolveAlongDirs(lc, rel, wantDir, mode == relativeModeLast)
		if err != nil {
			return "", fmt.Errorf("%s: %w", kind, err)
		}
		return resolved, nil
	}

	// unreachable: parseRelativeMode only returns known modes, and the
	// exhaustive linter checks the switch above handles each of them
	return "", fmt.Errorf("%s: unhandled mode %d", kind, mode)
}

// parseRelativeMode parses the optional template mode argument: "this"
// (default), "first", or "last".
func parseRelativeMode(modeArgs []string) (relativeMode, error) {
	switch len(modeArgs) {
	case 0:
		return relativeModeThis, nil
	case 1:
		switch modeArgs[0] {
		case "this":
			return relativeModeThis, nil
		case "first":
			return relativeModeFirst, nil
		case "last":
			return relativeModeLast, nil
		default:
			return 0, fmt.Errorf("unknown mode %q", modeArgs[0])
		}
	default:
		return 0, fmt.Errorf("mode accepts at most one argument")
	}
}

func validateRelativePath(rel string) error {
	if rel == "" {
		return fmt.Errorf("path is empty")
	}
	if filepath.IsAbs(rel) {
		return fmt.Errorf("path %q is absolute", rel)
	}
	if pathHasDotDot(rel) {
		return fmt.Errorf("path %q contains a parent directory segment", rel)
	}
	return nil
}

func pathHasDotDot(path string) bool {
	for _, seg := range strings.Split(filepath.ToSlash(path), "/") {
		if seg == ".." {
			return true
		}
	}
	return false
}

func resolveThis(lc LoadContext, rel string) (string, error) {
	if lc.File == "" {
		return "", fmt.Errorf("no current file")
	}
	return joinUnder(filepath.Dir(lc.File), rel)
}

func resolveAlongDirs(lc LoadContext, rel string, wantDir, last bool) (string, error) {
	var match string
	found := false
	for _, dir := range lc.Dirs {
		candidate, err := joinUnder(dir, rel)
		if err != nil {
			return "", err
		}
		info, err := os.Stat(candidate)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("stat %q: %w", candidate, err)
		}
		if info.IsDir() != wantDir {
			continue
		}
		match = candidate
		found = true
		if !last {
			break
		}
	}
	if !found {
		return "", fmt.Errorf("%q not found", rel)
	}
	return match, nil
}

func joinUnder(base, rel string) (string, error) {
	joined := filepath.Join(base, filepath.Clean(rel))
	if err := pathEscapesBase(base, joined); err != nil {
		return "", err
	}
	return joined, nil
}

func pathEscapesBase(base, joined string) error {
	absBase, err := filepath.Abs(base)
	if err != nil {
		return fmt.Errorf("abs base %q: %w", base, err)
	}
	absJoined, err := filepath.Abs(joined)
	if err != nil {
		return fmt.Errorf("abs path %q: %w", joined, err)
	}
	rel, err := filepath.Rel(absBase, absJoined)
	if err != nil {
		return fmt.Errorf("rel %q from %q: %w", absJoined, absBase, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path escapes base %q", base)
	}
	return nil
}

func checkRelativeType(rel, resolved string, wantDir bool) (string, error) {
	info, err := os.Stat(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%q not found", rel)
		}
		return "", fmt.Errorf("stat %q: %w", resolved, err)
	}
	if wantDir {
		if !info.IsDir() {
			return "", fmt.Errorf("%q is not a directory", rel)
		}
		return resolved, nil
	}
	if info.IsDir() {
		return "", fmt.Errorf("%q is not a file", rel)
	}
	return resolved, nil
}
