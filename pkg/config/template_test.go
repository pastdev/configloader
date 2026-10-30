//nolint:goconst // explicit strings have explanatory value in tests
package config_test

import (
	"fmt"
	"strconv"
	"testing"
	"text/template"

	"github.com/pastdev/configloader/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestDefaultFuncMap(t *testing.T) {
	tester := func(t *testing.T, name string) {
		t.Helper()
		_, ok := config.DefaultFuncMap()[name]
		require.True(t, ok, "expected %s in DefaultFuncMap", name)
	}

	t.Run("includes base funcs", func(t *testing.T) {
		for _, name := range []string{"home", "user", "userRaw"} {
			tester(t, name)
		}
	})

	t.Run("includes meta funcs", func(t *testing.T) {
		for _, name := range []string{"relativeFile", "relativeDir"} {
			tester(t, name)
		}
	})

	t.Run("includes xdg funcs", func(t *testing.T) {
		tester(t, "xdgConfigHome")
	})
}

func TestNewTemplate(t *testing.T) {
	tester := func(t *testing.T, funcs map[string]any, value string, expected any) {
		t.Helper()
		actual, err := config.NewTemplate(funcs).Execute(config.LoadContext{File: "/app/config.yml"}, "test", value)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}

	file := func(lc config.LoadContext) any {
		return func() string { return lc.File }
	}

	t.Run("plain func is called as is", func(t *testing.T) {
		tester(t,
			map[string]any{"greet": func() string { return "hello" }},
			"{{ greet }}",
			"hello")
	})

	t.Run("factory is bound to the load context", func(t *testing.T) {
		tester(t,
			map[string]any{"file": file},
			"{{ file }}",
			"/app/config.yml")
	})

	t.Run("factory added after NewTemplate is bound", func(t *testing.T) {
		funcs := map[string]any{}
		tmpl := config.NewTemplate(funcs)
		funcs["file"] = file

		actual, err := tmpl.Execute(config.LoadContext{File: "/app/config.yml"}, "test", "{{ file }}")
		require.NoError(t, err)
		require.Equal(t, "/app/config.yml", actual)
	})
}

func TestYamlValueTemplateUnmarshal(t *testing.T) {
	unmarshal := config.YamlValueTemplateUnmarshal(
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
			config.LoadContext{},
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
			config.LoadContext{},
			[]byte(`---
num: '{{number "1"}}'
`),
			&actual)
		require.NoError(t, err)
		require.Equal(t, map[any]any{"num": float64(1)}, actual)
	})

	t.Run("creds", func(t *testing.T) {
		var actual map[any]any
		err := unmarshal(
			config.LoadContext{},
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

	t.Run("root scalar", func(t *testing.T) {
		var actual any
		err := unmarshal(
			config.LoadContext{},
			[]byte(`'{{number "1"}}'`),
			&actual)
		require.NoError(t, err)
		require.Equal(t, float64(1), actual)
	})
}
