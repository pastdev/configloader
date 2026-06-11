package cobra

import "github.com/pastdev/configloader/pkg/config"

type BaseSourceLoader[T any] interface {
	config.SourceLoader[T]
	BaseSourceLoader()
}

type baseSourceLoader[T any] struct {
	config.SourceLoader[T]
}

// BaseSourceLoader is simply a marker function allowing the isBaseSource to
// determine if this is should be treated as a _base_ source.
func (baseSourceLoader[T]) BaseSourceLoader() {}

func BaseSource[T any](src config.SourceLoader[T]) config.SourceLoader[T] {
	return baseSourceLoader[T]{SourceLoader: src}
}

func isBaseSource[T any](src config.SourceLoader[T]) bool {
	_, ok := src.(BaseSourceLoader[T])
	return ok
}
