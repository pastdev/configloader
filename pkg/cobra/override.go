package cobra

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type configOverride[C any] interface {
	apply(*C) error
}

type override[T any, C any] struct {
	v       T
	f       func(T, *C) error
	changed func() bool
}

type overrideFlags[C any] struct {
	config *ConfigLoader[C]
	flags  *pflag.FlagSet
}

//nolint:revive // intentionally forcing explicit pattern of invocation similar to cobra.Command
func (c *ConfigLoader[C]) OverrideFlags(cmd *cobra.Command) *overrideFlags[C] {
	return &overrideFlags[C]{
		config: c,
		flags:  cmd.Flags(),
	}
}

//nolint:revive // intentionally forcing explicit pattern of invocation similar to cobra.Command
func (c *ConfigLoader[C]) PersistentOverrideFlags(cmd *cobra.Command) *overrideFlags[C] {
	return &overrideFlags[C]{
		config: c,
		flags:  cmd.PersistentFlags(),
	}
}

func (o *overrideFlags[C]) String(
	f func(string, *C) error,
	name string,
	value string,
	usage string,
) {
	addOverrideFlag(o.config, o.flags, name, func(p *string) {
		o.flags.StringVar(p, name, value, usage)
	}, f)
}

//nolint:unused // invoked via configOverride interface
func (o *override[T, C]) apply(cfg *C) error {
	if o.changed != nil && !o.changed() {
		return nil
	}
	return o.f(o.v, cfg)
}

func (o *overrideFlags[C]) StringP(
	f func(string, *C) error,
	name string,
	shorthand string,
	value string,
	usage string,
) {
	addOverrideFlag(o.config, o.flags, name, func(p *string) {
		o.flags.StringVarP(p, name, shorthand, value, usage)
	}, f)
}

func (o *overrideFlags[C]) Int(
	f func(int, *C) error,
	name string,
	value int,
	usage string,
) {
	addOverrideFlag(o.config, o.flags, name, func(p *int) {
		o.flags.IntVar(p, name, value, usage)
	}, f)
}

func (o *overrideFlags[C]) IntP(
	f func(int, *C) error,
	name string,
	shorthand string,
	value int,
	usage string,
) {
	addOverrideFlag(o.config, o.flags, name, func(p *int) {
		o.flags.IntVarP(p, name, shorthand, value, usage)
	}, f)
}

func (o *overrideFlags[C]) Bool(
	f func(bool, *C) error,
	name string,
	value bool,
	usage string,
) {
	addOverrideFlag(o.config, o.flags, name, func(p *bool) {
		o.flags.BoolVar(p, name, value, usage)
	}, f)
}

func (o *overrideFlags[C]) BoolP(
	f func(bool, *C) error,
	name string,
	shorthand string,
	value bool,
	usage string,
) {
	addOverrideFlag(o.config, o.flags, name, func(p *bool) {
		o.flags.BoolVarP(p, name, shorthand, value, usage)
	}, f)
}

func addOverrideFlag[T any, C any](
	c *ConfigLoader[C],
	flags *pflag.FlagSet,
	name string,
	register func(*T),
	f func(T, *C) error,
) {
	o := &override[T, C]{f: f}
	register(&o.v)

	flag := flags.Lookup(name)
	o.changed = func() bool {
		return flag != nil && flag.Changed
	}

	c.overrides = append(c.overrides, o)
}
