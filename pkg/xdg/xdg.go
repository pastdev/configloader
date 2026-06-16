// Package xdg provides xdg env var with default fallback support see the
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

var (
	osUserHomeDir = os.UserHomeDir
	userCurrent   = user.Current
)

func nonEmptyEnv(name string) (string, bool) {
	v, ok := os.LookupEnv(name)
	return v, ok && v != ""
}

func firstFallback(fallback ...string) (string, bool) {
	if len(fallback) == 0 || fallback[0] == "" {
		return "", false
	}
	return fallback[0], true
}

func homePath(subPath string, envVar string, fallback ...string) (string, error) {
	if v, ok := nonEmptyEnv(envVar); ok {
		return v, nil
	}

	home, err := osUserHomeDir()
	if err != nil {
		if v, ok := firstFallback(fallback...); ok {
			return v, nil
		}
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
//
// If XDG_BIN_HOME is set and non-empty, it is used. Otherwise $HOME/.local/bin
// is used. If the home directory cannot be determined, the optional fallback is
// used if supplied.
func BinHome(fallback ...string) (string, error) {
	return homePath(".local/bin", "XDG_BIN_HOME", fallback...)
}

// CacheHome implements the spec for:
//
//	There is a single base directory relative to which user-specific
//	non-essential (cached) data should be written. This directory is defined by
//	the environment variable $XDG_CACHE_HOME.
//
// If XDG_CACHE_HOME is set and non-empty, it is used. Otherwise $HOME/.cache
// is used. If the home directory cannot be determined, the optional fallback is
// used if supplied.
func CacheHome(fallback ...string) (string, error) {
	return homePath(".cache", "XDG_CACHE_HOME", fallback...)
}

// ConfigDirs implements the spec for:
//
//	There is a set of preference ordered base directories relative to which
//	configuration files should be searched. This set of directories is defined
//	by the environment variable $XDG_CONFIG_DIRS.
//
// If XDG_CONFIG_DIRS is set and non-empty, it is used. Otherwise /etc/xdg is
// used.
func ConfigDirs(_ ...string) (string, error) {
	if v, ok := nonEmptyEnv("XDG_CONFIG_DIRS"); ok {
		return v, nil
	}

	return "/etc/xdg", nil
}

// ConfigHome implements the spec for:
//
//	There is a single base directory relative to which user-specific
//	configuration files should be written. This directory is defined by the
//	environment variable $XDG_CONFIG_HOME.
//
// If XDG_CONFIG_HOME is set and non-empty, it is used. Otherwise $HOME/.config
// is used. If the home directory cannot be determined, the optional fallback is
// used if supplied.
func ConfigHome(fallback ...string) (string, error) {
	return homePath(".config", "XDG_CONFIG_HOME", fallback...)
}

// DataDirs implements the spec for:
//
//	There is a set of preference ordered base directories relative to which
//	data files should be searched. This set of directories is defined by the
//	environment variable $XDG_DATA_DIRS.
//
// If XDG_DATA_DIRS is set and non-empty, it is used. Otherwise
// /usr/local/share/:/usr/share/ is used.
func DataDirs(_ ...string) (string, error) {
	if v, ok := nonEmptyEnv("XDG_DATA_DIRS"); ok {
		return v, nil
	}

	return "/usr/local/share/:/usr/share/", nil
}

// DataHome implements the spec for:
//
//	There is a single base directory relative to which user-specific data files
//	should be written. This directory is defined by the environment variable
//	$XDG_DATA_HOME.
//
// If XDG_DATA_HOME is set and non-empty, it is used. Otherwise
// $HOME/.local/share is used. If the home directory cannot be determined, the
// optional fallback is used if supplied.
func DataHome(fallback ...string) (string, error) {
	return homePath(".local/share", "XDG_DATA_HOME", fallback...)
}

// StateHome implements the spec for:
//
//	There is a single base directory relative to which user-specific state data
//	should be written. This directory is defined by the environment variable
//	$XDG_STATE_HOME.
//
// If XDG_STATE_HOME is set and non-empty, it is used. Otherwise
// $HOME/.local/state is used. If the home directory cannot be determined, the
// optional fallback is used if supplied.
func StateHome(fallback ...string) (string, error) {
	return homePath(".local/state", "XDG_STATE_HOME", fallback...)
}

// RuntimeDir implements the spec for:
//
//	There is a single base directory relative to which user-specific runtime
//	files and other file objects should be placed. This directory is defined by
//	the environment variable $XDG_RUNTIME_DIR.
//
// If XDG_RUNTIME_DIR is set and non-empty, it is used. Otherwise a best-effort
// replacement directory is returned. If that cannot be determined, the optional
// fallback is used if supplied.
func RuntimeDir(fallback ...string) (string, error) {
	if v, ok := nonEmptyEnv("XDG_RUNTIME_DIR"); ok {
		return v, nil
	}

	u, err := userCurrent()
	if err != nil {
		if v, ok := firstFallback(fallback...); ok {
			return v, nil
		}
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
