package cobra

import "fmt"

type configOverride[C any] interface {
	apply(*C) error
}

type override[T any, C any] struct {
	v T
	f func(T, *C) error
}

func (o *override[T, C]) apply(cfg *C) error {
	err := o.f(o.v, cfg)
	if err != nil {
		return fmt.Errorf("apply override: %w", err)
	}
	return nil
}

func AddOverride[T any, C any](c *ConfigLoader[C], f func(T, *C) error) *T {
	o := &override[T, C]{f: f}
	c.overrides = append(c.overrides, o)
	return &o.v
}
