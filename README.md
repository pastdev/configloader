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

### pkg/config Logger

This library uses [`zerolog`](https://github.com/rs/zerolog) for logging.
The `Logger` can be set by consumers as follows:

```go
import(
    "github.com/pastdev/configloader/pkg/config"
)

...

    config.Logger = zerolog.New(os.Stderr).Level(zerolog.TraceLevel).With().Timestamp().Logger()
```

## pkg/cobra

The [`github.com/pastdev/pkg/cobra`](./pkg/cobra) package provides CLI integration with [cobra](https://github.com/spf13/cobra).
There is 1 mandatory, and 2 optional integration points.
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

Optionally, you can use flags to allow your user to replace the `DefaultSources`:

```go
    cfg.PersistentFlags(&root).FileSourceVar(
        config.YamlUnmarshal,
        "config",
        "location of one or more config files")
    cfg.PersistentFlags(&root).DirSourceVar(
        config.YamlUnmarshal,
        "config-dir",
        "location of one or more config directories")
```

And add a `config` subcommand to your root command for printing out the configuration:

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
