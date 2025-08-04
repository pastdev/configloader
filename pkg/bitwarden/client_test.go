package bitwarden_test

import (
	"testing"

	"github.com/pastdev/configloader/pkg/bitwarden"
	"github.com/stretchr/testify/require"
)

func staticLookupClient(data string) bitwarden.Client {
	return bitwarden.Client{
		Lookup: func(_ string) ([]byte, error) { return []byte(data), nil },
	}
}

func TestField(t *testing.T) {
	test := func(t *testing.T, data string, name string, expected string) {
		actual, err := staticLookupClient(data).GetField("", name)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}

	t.Run("normal", func(t *testing.T) {
		test(t,
			`{
  "id": "d7213953-c6bf-468a-b220-b32c00fc75a0",
  "folder": "folder",
  "name": "example.org",
  "data": {
    "username": "user",
    "password": "newpwd",
    "totp": null,
    "uris": [
      {
        "uri": "https://info.example.org/home/",
        "match_type": null
      },
      {
        "uri": "https://example-identity.okta.com/login",
        "match_type": null
      }
    ]
  },
  "fields": [
    {
      "name": "FooField",
      "value": "FooValue",
      "type": "text"
    },
    {
      "name": "BarField",
      "value": "BarValue",
      "type": "text"
    }
  ],
  "notes": "These are some notes",
  "history": [
    {
      "last_used_date": "2025-08-01T17:07:43.855Z",
      "password": "midpwd"
    },
    {
      "last_used_date": "2025-08-01T17:07:23.424Z",
      "password": "origpwd"
    }
  ]
}`,
			"BarField",
			"BarValue")
	})
}

func TestFormat(t *testing.T) {
	test := func(t *testing.T, data string, args []string, expected string) {
		actual, err := staticLookupClient(data).GetFormat("", args[0], args[1:]...)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	}

	t.Run("simple", func(t *testing.T) {
		test(t,
			`{
  "id": "d7213953-c6bf-468a-b220-b32c00fc75a0",
  "name": "example.org",
  "data": {
    "username": "user",
    "password": "pass"
  },
  "notes": "These are some notes"
}`,
			[]string{"%s/%s", "username", "password"},
			"user/pass")
	})

	t.Run("normal", func(t *testing.T) {
		test(t,
			`{
  "id": "d7213953-c6bf-468a-b220-b32c00fc75a0",
  "folder": "folder",
  "name": "example.org",
  "data": {
    "username": "user",
    "password": "newpwd",
    "totp": null,
    "uris": [
      {
        "uri": "https://info.example.org/home/",
        "match_type": null
      },
      {
        "uri": "https://example-identity.okta.com/login",
        "match_type": null
      }
    ]
  },
  "fields": [
    {
      "name": "FooField",
      "value": "FooValue",
      "type": "text"
    },
    {
      "name": "BarField",
      "value": "BarValue",
      "type": "text"
    }
  ],
  "notes": "These are some notes",
  "history": [
    {
      "last_used_date": "2025-08-01T17:07:43.855Z",
      "password": "midpwd"
    },
    {
      "last_used_date": "2025-08-01T17:07:23.424Z",
      "password": "origpwd"
    }
  ]
}`,
			[]string{"%s/%s", "username", "password"},
			"user/newpwd")
	})
}
