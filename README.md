# configloader

`configloader` is a go library for loading configuration from multiple sources.

## pkg/config

The [`github.com/pastdev/pkg/config`](./pkg/config) package provides the core configuration loading library.
The library itself is very simple in that it has a single interface ([`SourceLoader`](./pkg/config/config.go)) that can be implemented for any type of source.
These source loader instances are aggregated together in a `Sources` object that can then be used to load and merge multiple sources together.
Subsequent sources will merge their values over the top of any existing values so the latest defined wins.

```go
sources := config.Sources[AppConfig]{
    Sources: []config.SourceLoader{
        config.FileSource{Path: "~/.config/app.yml"},
        config.DirSource{Path: "~/.config/app.d"},
    },
}

var cfg AppConfig
if err := sources.Load(&cfg); err != nil {
    return err
}
```

See the [example](./pkg/config/example_test.go) or [tests](./pkg/config/config_test.go) for more use cases.

### Merge behavior

Config is merged as a generic document before conversion into your target type.
In practice this means:

* maps are merged recursively
* slices are replaced wholesale
* scalars overwrite earlier values
* later sources override earlier sources

This makes nested overlays behave as expected, including inside `map[string]struct`-like configurations.

### Unmarshaling source documents

By default, [`YamlUnmarshal`](./pkg/config/config.go) is used to decode each source into an intermediate document.

You can replace that with a custom unmarshaler if you would like:

```go
sources := config.Sources[map[string]any]{
    Sources: []config.SourceLoader{
        config.FileSource{
            Path: "~/.config/configloader.json",
            Unmarshal: func(b []byte, cfg any) error {
                return json.Unmarshal(b, cfg)
            },
        },
    },
}
```

### Converting the merged document

After all source documents are merged, the result is converted into your target config object.

By default, [`YamlConvert`](./pkg/config/config.go) is used.
You can replace it if you want different final conversion behavior, for example to respect json tags or UnmarshalJSON methods:

```go
sources := config.Sources[AppConfig]{
    Sources: []config.SourceLoader{
        config.FileSource{Path: "~/.config/app.json"},
    },
    Convert: func(merged any, cfg *AppConfig) error {
        data, err := json.Marshal(merged)
        if err != nil {
            return err
        }
        return json.Unmarshal(data, cfg)
    },
}
```

### Templating

You can also configure your source loaders to pre-process config file values with the go templating engine:

```go
sources := config.Sources[AppConfig]{
    Sources: []config.SourceLoader{
        config.FileSource{
            Path: "/etc/configloader.tmpl.yml",
            Unmarshal: config.YamlValueTemplateUnmarshal(
                config.NewTemplate(config.DefaultFuncMap())),
        },
    },
}
```

`YamlValueTemplateUnmarshal` is intended for document-style destinations such as:

* `*any`
* `*map[any]any`
* `*map[string]any`
* `*[]any`

The `config.DefaultFuncMap()` contains utility functions for accessing secrets from various password managers and XDG helpers, including:

* [bitwarden](#bitwarden)
* [lastpass](#lastpass)
* [xdg](#xdg)

This map can be added to, or replaced.

#### Bitwarden

To use the bitwarden template functions, you need to install the [`rbw`](https://github.com/doy/rbw) client.
The template functions assume you have an _active session_ (ie: `rbw unlock`) from which it will obtain the secrets.

#### Lastpass

To use the lastpass template functions, you need to install the [`lastpass-cli`](https://github.com/lastpass/lastpass-cli) client.
The template functions assume you have an _active session_ (ie: `lpass login <USER>`) from which it will obtain the secrets.

#### XDG

The default template function map also includes XDG helpers:

* `xdgBinHome`
* `xdgCacheHome`
* `xdgConfigDirs`
* `xdgConfigHome`
* `xdgDataDirs`
* `xdgDataHome`
* `xdgStateHome`
* `xdgRuntimeDir`

These functions follow the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir/latest/) behavior:

* if the corresponding `XDG_*` environment variable is set and non-empty, that value is used
* otherwise the spec-defined default is used
* for values whose default depends on the user's home directory or current user, an optional fallback may be supplied and is used only if that default cannot be determined

Examples:

```gotemplate
{{ xdgConfigHome }}
{{ xdgConfigHome "/tmp/my-app-config" }}
{{ xdgRuntimeDir "/tmp/my-app-runtime" }}
```

For example, `xdgConfigHome` behaves like this:

* use `XDG_CONFIG_HOME` if it is set and non-empty
* otherwise use `$HOME/.config`
* if `$HOME` cannot be determined, use the supplied fallback if one was provided
* otherwise return an error

For `xdgConfigDirs` and `xdgDataDirs`, the spec defaults are fixed values, so fallbacks are generally unnecessary.

## pkg/log

This library uses [`zerolog`](https://github.com/rs/zerolog) for logging.
The `Logger` can be set by consumers as follows:

```go
import (
    "os"

    "github.com/pastdev/configloader/pkg/log"
    "github.com/rs/zerolog"
)

...

    log.Logger = zerolog.New(os.Stderr).
        Level(zerolog.TraceLevel).
        With().
        Timestamp().
        Logger()
```

## pkg/cobra

The [`github.com/pastdev/pkg/cobra`](./pkg/cobra) package provides CLI integration with [cobra](https://github.com/spf13/cobra).
There is 1 mandatory, and 2 optional integration points.

### ConfigLoader

First you need to define your [`ConfigLoader`](./pkg/cobra/config.go) object:

```go
    cfgldr := cobraconfig.ConfigLoader[map[string]any]{
        DefaultSources: config.Sources[map[string]any]{
            Sources: []config.SourceLoader{
                config.FileSource{Path: "/etc/configloader.yml"},
                config.DirSource{Path: "/etc/configloader.d"},
                config.FileSource{Path: "~/.config/configloader.yml"},
                config.DirSource{Path: "~/.config/configloader.d"},
            },
        },
    }
```

You can then pass the config loader to subcommands and call `.Config()` to load and access the config object:

```go
    root.AddCommand(fooCmd(&cfgldr))

...

func fooCmd(cfgldr *cobraconfig.ConfigLoader[map[string]any]) *cobra.Command {
    return &cobra.Command{
        Use:   "foo",
        Short: `Show the value of foo.`,
        RunE: func(_ *cobra.Command, _ []string) error {
            cfg, err := cfgldr.Config()
            if err != nil {
                return fmt.Errorf("get config: %w", err)
            }

            fmt.Printf("foo is [%v]\n", (*cfg)["foo"])
            return nil
        },
    }
}
```

### Config flags

You can use flags to allow your user to replace the `DefaultSources`:

```go
    cfgldr.PersistentFlags(&root).FileSourceVar(
        config.YamlUnmarshal(),
        "config",
        "location of one or more config files")
    
    cfgldr.PersistentFlags(&root).DirSourceVar(
        config.YamlUnmarshal(),
        "config-dir",
        "location of one or more config directories")
```

By default, if the user supplies these flags, they will replace all `DefaultSources`.
If you want to preserve some defaults and overlay user-supplied sources on top of them, mark those defaults as `BaseSource`:

```go
    cfgldr := cobraconfig.ConfigLoader[AppConfig]{
        DefaultSources: config.Sources[AppConfig]{
            Sources: []config.SourceLoader{
                cobraconfig.BaseSource(config.RawSource{
                    Data: []byte(`
name: my-app
port: 8080
`),
                }),
                config.DirSource{Path: "/etc/my-app.d"},
                config.DirSource{Path: "~/.config/my-app.d"},
            },
        },
    }
```

In this example:

* the embedded `RawSource` is always loaded
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

...

    cfgldr := cobraconfig.ConfigLoader[AppConfig]{
        DefaultSources: config.Sources[AppConfig]{
            Sources: []config.SourceLoader{
                config.FileSource{Path: "/etc/my-app.yml"},
            },
        },
    }
    
    cfgldr.OverrideFlags(&root).String(
        func(v string, cfg *AppConfig) error {
            cfg.Log.Level = v
            return nil
        },
        "log-level",
        "",
        "log level",
    )
    
    cfgldr.OverrideFlags(&root).Int(
        func(v int, cfg *AppConfig) error {
            cfg.Port = v
            return nil
        },
        "port",
        0,
        "listen port",
    )
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
            func(w io.Writer, cfg *map[string]any) error {
                err := json.NewEncoder(w).Encode(cfg)
                if err != nil {
                    return fmt.Errorf("format json: %w", err)
                }
                return nil
            },
        ),
        cobraconfig.WithConfigCommandSilenceUsage[map[string]any](true),
    )
```
