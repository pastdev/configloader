package lastpass_test

import (
	"testing"

	"github.com/pastdev/configloader/pkg/lastpass"
	"github.com/stretchr/testify/require"
)

func TestShow(t *testing.T) {
	entry := `[
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
]`
	require.NotEmpty(t, entry)

	actual, err := lastpass.Show("3818021426", "%s/%s", "username", "password")
	require.NoError(t, err)
	require.Equal(t, "user/pass", actual)
}
