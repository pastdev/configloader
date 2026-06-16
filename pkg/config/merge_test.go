//nolint:goconst // explicit strings have explanatory value in tests
package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeDocument(t *testing.T) {
	tester := func(t *testing.T, expected any, dst any, src any) {
		actual := mergeDocument(
			normalizeDocument(dst),
			normalizeDocument(src),
		)
		require.Equal(t, expected, actual)
	}

	t.Run("nested map override", func(t *testing.T) {
		tester(t,
			map[string]any{
				"app": map[string]any{
					"user": map[string]any{
						"name":  "not_me",
						"shell": "/bin/bash",
					},
				},
			},
			map[string]any{
				"app": map[string]any{
					"user": map[string]any{
						"name":  "me",
						"shell": "/bin/bash",
					},
				},
			},
			map[string]any{
				"app": map[string]any{
					"user": map[string]any{
						"name": "not_me",
					},
				},
			},
		)
	})

	t.Run("null clears nested value", func(t *testing.T) {
		tester(t,
			map[string]any{
				"app": map[string]any{
					"user": nil,
				},
			},
			map[string]any{
				"app": map[string]any{
					"user": map[string]any{
						"name":  "me",
						"shell": "/bin/bash",
					},
				},
			},
			map[string]any{
				"app": map[string]any{
					"user": nil,
				},
			},
		)
	})

	t.Run("false overrides true", func(t *testing.T) {
		tester(t,
			map[string]any{
				"enabled": false,
			},
			map[string]any{
				"enabled": true,
			},
			map[string]any{
				"enabled": false,
			},
		)
	})

	t.Run("zero overrides non zero", func(t *testing.T) {
		tester(t,
			map[string]any{
				"count": 0,
			},
			map[string]any{
				"count": 7,
			},
			map[string]any{
				"count": 0,
			},
		)
	})

	t.Run("empty string overrides non empty", func(t *testing.T) {
		tester(t,
			map[string]any{
				"name": "",
			},
			map[string]any{
				"name": "me",
			},
			map[string]any{
				"name": "",
			},
		)
	})

	t.Run("slice replaces earlier slice", func(t *testing.T) {
		tester(t,
			map[string]any{
				"items": []any{"c"},
			},
			map[string]any{
				"items": []any{"a", "b"},
			},
			map[string]any{
				"items": []any{"c"},
			},
		)
	})
}
