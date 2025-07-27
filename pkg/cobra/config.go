package cobra

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pastdev/configloader/pkg/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type ConfigCommandOption[T any] func(*ConfigCommandOptions[T])

type ConfigCommandOptions[T any] struct {
	Output       map[string]func(w io.Writer, cfg *T) error
	SilenceUsage bool
}

// ConfigLoader provides utilities to integrate configuration loading into cobra
// CLI commands.
type ConfigLoader[T any] struct {
	config T
	// DefaultSources are sources that you can configure in the code and allow
	// for the flags to replace at runtime.
	DefaultSources config.Sources[T]
	loaded         bool
	sources        config.Sources[T]
}

// Config returns the generated configuration object that will be loaded by the
// Load method.
func (c *ConfigLoader[T]) Config() (*T, error) {
	if !c.loaded {
		err := c.load()
		if err != nil {
			return nil, err
		}
		c.loaded = true
	}
	return &c.config, nil
}

// Load loads the configuration. If sources were set using the persistent flags,
// then the DefaultSources will be ignored. Otherwise, configurationis loaded
// from the DefaultSources.
func (c *ConfigLoader[T]) load() error {
	sources := c.sources
	if len(sources) == 0 {
		sources = c.DefaultSources
	}

	err := sources.Load(&c.config)
	if err != nil {
		return fmt.Errorf("configloader load sources: %w", err)
	}
	return nil
}

// PersistentFlags returns a factory for adding configuration source flags to
// the supplied root command.
//
//nolint:revive // want to limit what can be done to the returned object
func (c *ConfigLoader[T]) PersistentFlags(root *cobra.Command) *flags[T] {
	return &flags[T]{
		config: c,
		root:   root,
	}
}

// AddSubCommandTo will add a config subcommand to the supplied root command.
// This subcommand will print out the configuration.
func (c *ConfigLoader[T]) AddSubCommandTo(root *cobra.Command, opts ...ConfigCommandOption[T]) {
	options := ConfigCommandOptions[T]{
		Output: map[string]func(w io.Writer, cfg *T) error{
			"yaml": func(w io.Writer, cfg *T) error {
				err := yaml.NewEncoder(w).Encode(cfg)
				if err != nil {
					return fmt.Errorf("serialize config: %w", err)
				}
				return nil
			},
		},
	}
	for _, opt := range opts {
		opt(&options)
	}

	var output string

	cmd := cobra.Command{
		Use:          "config",
		Short:        `Print out the config data.`,
		Args:         cobra.NoArgs,
		SilenceUsage: options.SilenceUsage,
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := c.Config()
			if err != nil {
				return fmt.Errorf("get config: %w", err)
			}

			formatter, ok := options.Output[output]
			if !ok {
				return fmt.Errorf("undefined formatter: %s", output)
			}

			err = formatter(os.Stdout, cfg)
			if err != nil {
				return fmt.Errorf("format config: %w", err)
			}
			return nil
		},
	}

	formatters := make([]string, 0, len(options.Output))
	for formatter := range options.Output {
		formatters = append(formatters, formatter)
	}

	if len(formatters) > 1 {
		cmd.Flags().StringVar(
			&output,
			"output",
			"yaml",
			fmt.Sprintf("Format of output, one of: %s", strings.Join(formatters, ", ")))
	} else {
		// no option was added, so we set output to the default formatter
		output = "yaml"
	}

	root.AddCommand(&cmd)
}

func WithConfigCommandOutput[T any](
	name string,
	formatter func(w io.Writer, cfg *T) error,
) ConfigCommandOption[T] {
	return func(cco *ConfigCommandOptions[T]) {
		cco.Output[name] = formatter
	}
}

func WithConfigCommandSilenceUsage[T any](s bool) ConfigCommandOption[T] {
	return func(cco *ConfigCommandOptions[T]) {
		cco.SilenceUsage = s
	}
}
