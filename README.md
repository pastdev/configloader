# configloader

`configloader` is a go library for loading configuration from multiple sources.

## pkg/config

The [`github.com/pastdev/pkg/config`](./pkg/config) package provides the core configuration loading library.
The library itself is very simple in that it has a single interface ([`SourceLoader`](./pkg/config/config.go)) that can be implemented for any type of source.
These source loader instances are aggregated together in a `Sources` object that can then be used to load and merge multiple sources together.
Subsequent sources will merge their values over the top of any existing values so the latest defined wins.

```go
    sources := config.Sources[AppConfig]{
        config.FileSource[AppConfig]{Path: "~/.config/app.yml"},
        config.DirSource[AppConfig]{Path: "~/.config/app.d"},
    }
    sources.Load(&cfg)
```

See the [example](./pkg/config/example_test.go) or [tests](./pkg/config/config_test.go) for more use cases.

### Unmarshaling

By default, `YamlUnmarshal` is used.
However, you can replace that with a custom unmarshaler if you would like:

```go
    sources := config.Sources[AppConfig]{
        config.FileSource[map[any]any]{
            Path: "~/.config/configloader.tmpl.d",
            Unmarshal: func(b []byte, cfg *map[any]any) error {
                return json.Unmarshal(b, cfg)
            },
        },
    }
```

### Templating

You can also configure your source loaders to pre-process config file values with the go templating engine:

```go
    sources := config.Sources[AppConfig]{
        config.FileSource[AppConfig]{
            Path: "/etc/configloader.tmpl.yml",
            Unmarshal: config.
                YamlValueTemplateUnmarshal[AppConfig](
                    config.NewTemplate(config.DefaultFuncMap()))
        },
    }
```

The `config.DefaultFuncMap()` contains utility functions for accessing secrets from various password managers (ie: [lastpass](#lastpass), [bitwarden](#bitwarden)).
This map can be added to, or replaced.

#### Bitwarden

To use the bitwarden template functions, you need to install the [`rbw`](https://github.com/doy/rbw) client.
The template functions assume you have an _active session_ (ie: `rbw unlock`) from which it will obtain the secrets.

#### Lastpass

To use the lastpass template functions, you need to install the [`lastpass-cli`](https://github.com/lastpass/lastpass-cli) client.
The template functions assume you have an _active session_ (ie: `lpass login <USER>`) from which it will obtain the secrets.

## pkg/log

This library uses [`zerolog`](https://github.com/rs/zerolog) for logging.
The `Logger` can be set by consumers as follows:

```go
import(
    "github.com/pastdev/configloader/pkg/log"
)

...

    log.Logger = zerolog.New(os.Stderr).Level(zerolog.TraceLevel).With().Timestamp().Logger()
```

## pkg/cobra

The [`github.com/pastdev/pkg/cobra`](./pkg/cobra) package provides CLI integration with [cobra](https://github.com/spf13/cobra).
There is 1 mandatory, and 2 optional integration points.

### ConfigLoader

First you need to define your [`ConfigLoader`](./pkg/cobra/config.go) object:

```go
    cfgldr := cobraconfig.ConfigLoader[map[any]any]{
        DefaultSources: config.Sources[map[any]any]{
            config.FileSource[map[any]any]{Path: "/etc/configloader.yml"},
            config.DirSource[map[any]any]{Path: "/etc/configloader.d"},
            config.FileSource[map[any]any]{Path: "~/.config/configloader.yml"},
            config.DirSource[map[any]any]{Path: "~/.config/configloader.d"},
        },
    }
```

Then you can pass the the configloader object to any subcommands and simply call the `.Config()` method to load and access the config object:

```go
    root.AddCommand(fooCmd(&cfgldr))
...

func fooCmd(cfgldr *cobraconfig.ConfigLoader[map[any]any]) *cobra.Command {
    return &cobra.Command{
        Use:   "foo",
        Short: `An example subcommand for how to use configloader to show the value of foo.`,
        RunE: func(_ *cobra.Command, _ []string) error {
            cfg, err := cfgldr.Config()
            if err != nil {
                return fmt.Errorf("get config: %w", err)
            }

            fmt.Printf("foo is [%s]", (*cfg)["foo"])
            return nil
        },
    }
}
```

### Config flags

You can use flags to allow your user to replace the `DefaultSources`:

```go
    cfg.PersistentFlags(&root).FileSourceVar(
        config.YamlUnmarshal[map[any]any](),
        "config",
        "location of one or more config files")
    cfg.PersistentFlags(&root).DirSourceVar(
        config.YamlUnmarshal[map[any]any](),
        "config-dir",
        "location of one or more config directories")
```

By default, if the user supplies these flags, they will replace all `DefaultSources`.
If you prefer to preserve any of the sources so that these flags are merged on top of them, you can mark the source as a `BaseSource`:

```go
cfgldr := cobraconfig.ConfigLoader[AppConfig]{
    DefaultSources: config.Sources[AppConfig]{
        cobraconfig.BaseSource(config.RawSource[AppConfig]{
            Data: []byte(`
name: my-app
port: 8080
`),
        }),
        config.DirSource[AppConfig]{Path: "/etc/my-app.d"},
        config.DirSource[AppConfig]{Path: "~/.config/my-app.d"},
    },
}
```

In this example:

* the embedded RawSource is always loaded
* the default config directories are loaded only when no explicit config flags are provided
* any explicit `--config` or `--config-dir` sources are loaded after the base source and therefore override it

### Flag value overrides

You can also bind ordinary cobra flags directly to config overrides.
This is useful when you want to load config from files first, then let explicit CLI flags overlay specific values.

```go
type AppConfig struct {
    Log struct {
        Level string `yaml:"level"`
    } `yaml:"log"`
    Port int `yaml:"port"`
}

cfgldr := cobraconfig.ConfigLoader[AppConfig]{
    DefaultSources: config.Sources[AppConfig]{
        config.FileSource[AppConfig]{Path: "/etc/my-app.yml"},
    },
}

root.PersistentFlags().StringVar(
    cobraconfig.AddOverride(&cfgldr, func(v string, cfg *AppConfig) error {
        cfg.Log.Level = v
        return nil
    }),
    "log-level",
    "info",
    "log level")
root.PersistentFlags().IntVar(
    cobraconfig.AddOverride(&cfgldr, func(v int, cfg *AppConfig) error {
        cfg.Port = v
        return nil
    }),
    "port",
    8080,
    "listen port")
```

When `cfgldr.Config()` is called:

* configuration sources are loaded and merged
* override values populated by flags are applied to the loaded config
* the final merged config is returned

### Config subcommand

You can add a `config` subcommand to your root command for printing out the configuration:

```go
    cfg.AddSubCommandTo(&root)
```

Or a use additional options when adding the subcommand:

```go
    cfgldr.AddSubCommandTo(
        &root,
        cobraconfig.WithConfigCommandOutput(
            "json",
            func(w io.Writer, cfg *map[any]any) error {
                jsonmap := map[string]any{}
                for k, v := range *cfg {
                    jsonmap[fmt.Sprintf("%s", k)] = v
                }

                err := json.NewEncoder(w).Encode(jsonmap)
                if err != nil {
                    return fmt.Errorf("format json: %w", err)
                }
                return nil
            },
        ),
        cobraconfig.WithConfigCommandSilenceUsage[map[any]any](true))
```
