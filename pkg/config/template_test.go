package config_test

import (
	"fmt"
	"strconv"
	"testing"
	"text/template"

	"github.com/pastdev/configloader/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestYamlValueTemplateUnmarshal(t *testing.T) {
	unmarshal := config.YamlValueTemplateUnmarshal[map[any]any](
		config.NewTemplate(template.FuncMap{
			"number": strconv.Atoi,
			"object": func(key, val string) string {
				return fmt.Sprintf(`{"%s": "%s"}`, key, val)
			},
			"password": func(id string) string { return fmt.Sprintf("pass-for-%s", id) },
			"username": func(id string) string { return fmt.Sprintf("user-for-%s", id) },
		}))

	t.Run("object", func(t *testing.T) {
		var valueMap map[any]any
		err := unmarshal(
			[]byte(`---
obj: '{{object "foo" "bar"}}'
`),
			&valueMap)
		require.NoError(t, err)
		require.Equal(t, map[any]any{"obj": map[string]any{"foo": "bar"}}, valueMap)
	})

	t.Run("number", func(t *testing.T) {
		var actual map[any]any
		err := unmarshal(
			[]byte(`---
num: '{{number "1"}}'
`),
			&actual)
		require.NoError(t, err)
		require.Equal(t, map[any]any{"num": 1}, actual)
	})

	t.Run("creds", func(t *testing.T) {
		var actual map[any]any
		err := unmarshal(
			[]byte(`---
creds:
  password: '{{password "example.com"}}'
  username: '{{username "example.com"}}'
`),
			&actual)
		require.NoError(t, err)
		require.Equal(t,
			map[any]any{
				"creds": map[string]any{
					"password": "pass-for-example.com",
					"username": "user-for-example.com",
				},
			},
			actual)
	})
}
