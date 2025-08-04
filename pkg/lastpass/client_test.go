package lastpass_test

import (
	"encoding/json"
	"testing"

	"github.com/pastdev/configloader/pkg/lastpass"
	"github.com/stretchr/testify/require"
)

func TestFormat(t *testing.T) {
	test := func(t *testing.T, data string, args []string, expected string) {
		var entry []lastpass.Entry
		err := json.Unmarshal([]byte(data), &entry)
		require.NoError(t, err)

		actual := entry[0].Format(args[0], args[1:]...)
		require.Equal(t, expected, actual)
	}

	t.Run("simple", func(t *testing.T) {
		test(t,
			`[
  {
    "id": "3818021426",
    "name": "foo",
    "fullname": "bar/baz",
    "username": "user",
    "password": "pass",
    "last_modified_gmt": "1750178348",
    "last_touch": "1753994145",
    "share": "hip",
    "group": "hop",
    "url": "https://dip.dap.org",
    "note": "20250407: empty\n20241004: value"
  }
]`,
			[]string{"%s/%s", "username", "password"},
			"user/pass")
	})
}
