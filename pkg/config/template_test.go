package config_test

import (
	"testing"

	"github.com/pastdev/configloader/pkg/config"
	"github.com/stretchr/testify/require"
)

type FooExecutor struct {
}

func (e *FooExecutor) Execute(name string, value any) (any, error) {
	return "foo", nil
}

func TestYamlValueTemplateUnmarshal(t *testing.T) {
	var valueMap map[any]any
	unmarshal := config.YamlValueTemplateUnmarshal[map[any]any](
		&FooExecutor{})
	err := unmarshal([]byte(`{"foo":"bar"}`), &valueMap)
	require.NoError(t, err)
	require.Equal(t, map[any]any{"foo": "foo"}, valueMap)
}
