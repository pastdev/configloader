package config_test

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/pastdev/configloader/pkg/config"
	"gopkg.in/yaml.v3"
)

func ExampleSources_Load() {
	type AppConfig struct {
		Foo string `yaml:"foo"`
		Hip string `yaml:"hip"`
	}

	var cfg AppConfig

	sources := config.Sources[AppConfig]{
		Sources: []config.SourceLoader{
			config.RawSource{
				Data: []byte(`{"foo":"bar","hip":"hop"}`),
			},
			config.RawSource{
				Data: []byte(`{"foo":"baz"}`),
				// can customize unmarshaler, by default its yaml...
				Unmarshal: json.Unmarshal,
			},
		},
	}

	err := sources.Load(&cfg)
	if err != nil {
		log.Fatalf("failed to load: %s", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		log.Fatalf("failed to marshal: %s", err)
	}

	fmt.Print(string(data))

	// Output:
	// foo: baz
	// hip: hop
}
