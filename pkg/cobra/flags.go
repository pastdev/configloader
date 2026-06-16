package cobra

import (
	"github.com/pastdev/configloader/pkg/config"
	"github.com/pastdev/configloader/pkg/log"
	"github.com/spf13/cobra"
)

type flags[T any] struct {
	config *ConfigLoader[T]
	root   *cobra.Command
}

type sourcesValue struct {
	sources *[]config.SourceLoader
	factory func(string) config.SourceLoader
}

// DirSourceVar calls DirSourceVarP without a shorthand flag.
func (f *flags[T]) DirSourceVar(
	unmarshal func(b []byte, cfg any) error,
	name string,
	usage string,
) {
	f.DirSourceVarP(unmarshal, name, "", usage)
}

// DirSourceVarP will add a source loader that will read all files in the
// specified folder. This file iteration is not recursive. The supplied
// unmarshal func will be used to parse the files.
func (f *flags[T]) DirSourceVarP(
	unmarshal func(b []byte, cfg any) error,
	name string,
	shorthand string,
	usage string,
) {
	f.SourceVarP(
		func(path string) config.SourceLoader {
			return config.DirSource{
				Path:      path,
				Unmarshal: unmarshal,
			}
		},
		name,
		shorthand,
		usage)
}

// FileSourceVar calls FileSourceVarP without a shorthand flag.
func (f *flags[T]) FileSourceVar(
	unmarshal func(b []byte, cfg any) error,
	name string,
	usage string,
) {
	f.FileSourceVarP(unmarshal, name, "", usage)
}

// FileSourceVarP will add a source loader that will read the specified file.
// The supplied unmarshal func will be used to parse the file.
func (f *flags[T]) FileSourceVarP(
	unmarshal func(b []byte, cfg any) error,
	name string,
	shorthand string,
	usage string,
) {
	f.SourceVarP(
		func(path string) config.SourceLoader {
			return config.FileSource{
				Path:      path,
				Unmarshal: unmarshal,
			}
		},
		name,
		shorthand,
		usage)
}

// SourceVar calls SourceVarP without a shorthand flag.
func (f *flags[T]) SourceVar(
	factory func(string) config.SourceLoader,
	name string,
	usage string,
) {
	f.SourceVarP(factory, name, "", usage)
}

// SourceVarP will add a source loader defined by the supplied factory function.
func (f *flags[T]) SourceVarP(
	factory func(string) config.SourceLoader,
	name string,
	shorthand string,
	usage string,
) {
	f.root.PersistentFlags().VarP(
		newSourcesValue(nil, &f.config.sources, factory),
		name,
		shorthand,
		usage)
}

// String implements [pflag.Value]. This method is only used for defaults, but
// defautls are complicated in this scenario and are often mingled from multiple
// command line options so we cant just have one of them, or all of them, list
// defaults that are not specific to the option. Instead we use empty. It will
// be the responsibility of consumers of this library to decide how they want to
// provide defaults documentation to their users.
//
// [pflag.Value]: https://github.com/spf13/pflag/blob/1c62fb2813da5f1d1b893a49180a41b3f6be3262/flag.go#L200-L204
func (m *sourcesValue) String() string {
	return ""
}

// Set implements [pflag.Value].
//
// [pflag.Value]: https://github.com/spf13/pflag/blob/1c62fb2813da5f1d1b893a49180a41b3f6be3262/flag.go#L200-L204
func (m *sourcesValue) Set(v string) error {
	src := m.factory(v)

	if len(*m.sources) > 0 {
		log.Logger.Trace().Stringer("source", src).Msg("adding config source")
		*m.sources = append(*m.sources, src)
	} else {
		log.Logger.Trace().Stringer("source", src).Msg("initial config source")
		*m.sources = []config.SourceLoader{src}
	}

	return nil
}

// Set implements [pflag.Value].
//
// [pflag.Value]: https://github.com/spf13/pflag/blob/1c62fb2813da5f1d1b893a49180a41b3f6be3262/flag.go#L200-L204
func (*sourcesValue) Type() string {
	return "sources"
}

func newSourcesValue(
	val []config.SourceLoader,
	p *[]config.SourceLoader,
	factory func(string) config.SourceLoader,
) *sourcesValue {
	sv := new(sourcesValue)
	sv.sources = p
	*sv.sources = val
	sv.factory = factory
	return sv
}
