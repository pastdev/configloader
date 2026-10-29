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

// InnerSource returns the wrapped source so config dir collection for
// relativeFile/relativeDir can see FileSource and DirSource through this
// wrapper.
func (b baseSourceLoader) InnerSource() config.SourceLoader {
	return b.SourceLoader
}

func isBaseSource(src config.SourceLoader) bool {
	_, ok := src.(BaseSourceLoader)
	return ok
}
