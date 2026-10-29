package config

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

const (
	relativeModeThis  = "this"
	relativeModeFirst = "first"
	relativeModeLast  = "last"
)

var (
	osUserHomeDir = os.UserHomeDir
	userCurrent   = user.Current
)

func home() (string, error) {
	h, err := osUserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home: %w", err)
	}
	return h, nil
}

func currentUsername() (string, error) {
	u, err := userCurrent()
	if err != nil {
		return "", fmt.Errorf("user: %w", err)
	}
	return u.Username, nil
}

func relativeFile(path string, mode ...string) (string, error) {
	return resolveRelative("relativeFile", path, false, mode...)
}

func relativeDir(path string, mode ...string) (string, error) {
	return resolveRelative("relativeDir", path, true, mode...)
}

func resolveRelative(kind, rel string, wantDir bool, modeArgs ...string) (string, error) {
	if err := validateRelativePath(rel); err != nil {
		return "", fmt.Errorf("%s: %w", kind, err)
	}

	mode, err := parseRelativeMode(modeArgs)
	if err != nil {
		return "", fmt.Errorf("%s: %w", kind, err)
	}

	switch mode {
	case relativeModeThis:
		resolved, err := resolveThis(rel)
		if err != nil {
			return "", fmt.Errorf("%s: %w", kind, err)
		}
		return checkRelativeType(rel, resolved, wantDir)
	case relativeModeFirst, relativeModeLast:
		resolved, err := resolveAlongDirs(rel, wantDir, mode == relativeModeLast)
		if err != nil {
			return "", fmt.Errorf("%s: %w", kind, err)
		}
		return resolved, nil
	default:
		return "", fmt.Errorf("%s: unknown mode %q", kind, mode)
	}
}

func parseRelativeMode(modeArgs []string) (string, error) {
	switch len(modeArgs) {
	case 0:
		return relativeModeThis, nil
	case 1:
		switch modeArgs[0] {
		case relativeModeThis, relativeModeFirst, relativeModeLast:
			return modeArgs[0], nil
		default:
			return "", fmt.Errorf("unknown mode %q", modeArgs[0])
		}
	default:
		return "", fmt.Errorf("mode accepts at most one argument")
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

func resolveThis(rel string) (string, error) {
	if currentLoadContext.file == "" {
		return "", fmt.Errorf("no current file")
	}
	return joinUnder(filepath.Dir(currentLoadContext.file), rel)
}

func resolveAlongDirs(rel string, wantDir, last bool) (string, error) {
	var match string
	found := false
	for _, dir := range currentLoadContext.dirs {
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
