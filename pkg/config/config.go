package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pastdev/configloader/pkg/log"
	"gopkg.in/yaml.v3"
)

type Merger func(doc any)

// UnmarshalFunc unmarshals b into cfg. The supplied ctx carries per-load
// information (such as the file currently being read) that template funcs
// like relativeFile need, and should be passed along to any Executor.
type UnmarshalFunc func(ctx context.Context, b []byte, cfg any) error

// WrapUnmarshal adapts a context free unmarshal func, such as json.Unmarshal
// or yaml.Unmarshal, to an UnmarshalFunc.
func WrapUnmarshal(fn func(b []byte, cfg any) error) UnmarshalFunc {
	return func(_ context.Context, b []byte, cfg any) error {
		return fn(b, cfg)
	}
}

// SourceLoader is the primary interface for loading configuration from a
// source.
type SourceLoader interface {
	// Load calls merge for each document in the source. Implementations
	// should pass ctx along to their UnmarshalFunc.
	Load(ctx context.Context, merge Merger) error
	String() string
}

// Sources is an aggregation of SourceLoaders paired with a final document to
// object converter.
type Sources[T any] struct {
	// Sources is the list of SourceLoaders that are used to load and merge
	// configuration.
	Sources []SourceLoader
	// Convert converts the final merged document into the destination configuration
	// object. If not specified YamlConvert will be used.
	Convert func(merged any, cfg *T) error
}

// Load will load the configuration from all the Sources. Each SourceLoader will
// be supplied a merge func to call with its values. Each call to the merge
// function overlays its values onto the previously merged document. The merged
// document will then be supplied to the converter to unmarshal into the
// supplied cfg object.
func (s Sources[T]) Load(cfg *T) error {
	return s.LoadContext(context.Background(), cfg)
}

// LoadContext is Load with a caller supplied ctx that is passed to each
// SourceLoader.
func (s Sources[T]) LoadContext(ctx context.Context, cfg *T) error {
	start := time.Now()
	ctx = withLoadDirs(ctx, configDirs(s.Sources))

	var merged any
	merger := func(doc any) {
		if doc == nil {
			return
		}
		merged = mergeDocument(merged, normalizeDocument(doc))
	}

	for _, src := range s.Sources {
		err := src.Load(ctx, merger)
		if err != nil {
			return fmt.Errorf("load %s: %w", src, err)
		}
	}

	if merged != nil {
		convert := s.Convert
		if convert == nil {
			convert = YamlConvert
		}
		err := convert(merged, cfg)
		if err != nil {
			return fmt.Errorf("convert: %w", err)
		}
	}

	log.Logger.Debug().Dur("duration", time.Since(start)).Msg("load complete")
	return nil
}

func normalizePath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Logger.Trace().Err(err).Msg("User home directory not defined")
			return path
		}
		path = filepath.Join(homeDir, path[1:])
	}
	return path
}

func unmarshal(ctx context.Context, b []byte, cfg any, unmarshal UnmarshalFunc) error {
	if unmarshal == nil {
		unmarshal = YamlUnmarshal()
	}

	err := unmarshal(ctx, b, cfg)
	if err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	return nil
}

// YamlConvert will Marshal, then Unmarshal doc into cfg.
func YamlConvert[T any](doc any, cfg *T) error {
	data, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal merged: %w", err)
	}

	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return fmt.Errorf("final unmarshal merged: %w", err)
	}

	return nil
}

// YamlUnmarshal is an Unmarshal function that unmarshals from yaml.
func YamlUnmarshal() UnmarshalFunc {
	return func(_ context.Context, b []byte, cfg any) error {
		err := yaml.Unmarshal(b, cfg)
		if err != nil {
			return fmt.Errorf("yamlunmarshal: %w", err)
		}
		return nil
	}
}
