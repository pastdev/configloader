// Pachage xdg provides xdg env var with default fallback support see the
// [XDG Base Directory Specification] for details
//
// [XDG Base Directory Specification]: https://specifications.freedesktop.org/basedir/latest/
package xdg

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"text/template"
)

func homePath(subPath string, envVar string) (string, error) {
	if v, ok := os.LookupEnv(envVar); ok {
		return v, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("user home dir: %w", err)
	}

	return filepath.Join(home, subPath), nil
}

// BinHome returns the conventional dir for user binaries. This is not
// technically part of the spec, but the spec does say:
//
//	There is a single base directory relative to which user-specific executable
//	files may be written.
//
// And later says:
//
//	User-specific executable files may be stored in $HOME/.local/bin.
//	Distributions should ensure this directory shows up in the UNIX $PATH
//	environment variable, at an appropriate place.
func BinHome() (string, error) {
	return homePath(".local/bin", "XDG_BIN_HOME")
}

// CacheHome implements the spec for:
//
//	There is a single base directory relative to which user-specific
//	non-essential (cached) data should be written. This directory is defined by
//	the environment variable $XDG_CACHE_HOME.
func CacheHome() (string, error) {
	return homePath(".cache", "XDG_CACHE_HOME")
}

// ConfigDirs implements the spec for:
//
//	There is a set of preference ordered base directories relative to which
//	configuration files should be searched. This set of directories is defined
//	by the environment variable $XDG_CONFIG_DIRS.
func ConfigDirs() (string, error) {
	if v, ok := os.LookupEnv("XDG_CONFIG_DIRS"); ok {
		return v, nil
	}

	return "/etc/xdg", nil
}

// ConfigHome implements the spec for:
//
//	There is a single base directory relative to which user-specific
//	configuration files should be written. This directory is defined by the
//	environment variable $XDG_CONFIG_HOME.
func ConfigHome() (string, error) {
	return homePath(".config", "XDG_CONFIG_HOME")
}

// DataDirs implements the spec for:
//
//	There is a set of preference ordered base directories relative to which
//	data files should be searched. This set of directories is defined by the
//	environment variable $XDG_DATA_DIRS.
func DataDirs() (string, error) {
	if v, ok := os.LookupEnv("XDG_DATA_DIRS"); ok {
		return v, nil
	}

	return "/usr/local/share/:/usr/share/", nil
}

// DataHome implements the spec for:
//
//	There is a single base directory relative to which user-specific data files
//	should be written. This directory is defined by the environment variable
//	$XDG_DATA_HOME.
func DataHome() (string, error) {
	return homePath(".local/share", "XDG_DATA_HOME")
}

// StateHome implements the spec for:
//
//	There is a single base directory relative to which user-specific state data
//	should be written. This directory is defined by the environment variable
//	$XDG_STATE_HOME.
func StateHome() (string, error) {
	return homePath(".local/state", "XDG_STATE_HOME")
}

// RuntimeDir implements the spec for:
//
//	There is a single base directory relative to which user-specific runtime
//	files and other file objects should be placed. This directory is defined by
//	the environment variable $XDG_RUNTIME_DIR.
func RuntimeDir() (string, error) {
	if v, ok := os.LookupEnv("XDG_RUNTIME_DIR"); ok {
		return v, nil
	}

	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("current user: %w", err)
	}

	// this is not specified in the spec but seems to be the most common location
	return fmt.Sprintf("/run/user/%s", u.Uid), nil
}

func AddFuncs(funcs template.FuncMap) {
	funcs["xdgBinHome"] = BinHome
	funcs["xdgCacheHome"] = CacheHome
	funcs["xdgConfigDirs"] = ConfigDirs
	funcs["xdgConfigHome"] = ConfigHome
	funcs["xdgDataDirs"] = DataDirs
	funcs["xdgDataHome"] = DataHome
	funcs["xdgStateHome"] = StateHome
	funcs["xdgRuntimeDir"] = RuntimeDir
}
