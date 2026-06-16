package cobra

import "github.com/pastdev/configloader/pkg/config"

type BaseSourceLoader interface {
	config.SourceLoader
	BaseSourceLoader()
}

type baseSourceLoader struct {
	config.SourceLoader
}

// BaseSourceLoader is simply a marker function allowing the isBaseSource to
// determine if this is should be treated as a _base_ source.
func (baseSourceLoader) BaseSourceLoader() {}

func BaseSource(src config.SourceLoader) config.SourceLoader {
	return baseSourceLoader{SourceLoader: src}
}

func isBaseSource(src config.SourceLoader) bool {
	_, ok := src.(BaseSourceLoader)
	return ok
}
