package cobra

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pastdev/configloader/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestBaseSource(t *testing.T) {
	t.Run("inner source dir is searched by relativeFile", func(t *testing.T) {
		root := t.TempDir()
		system := filepath.Join(root, "system")
		userDir := filepath.Join(root, "user")
		userFile := filepath.Join(userDir, "keys", "gitlab.token")

		files := map[string]string{
			"system/config.yml":        `token: '{{ relativeFile "keys/gitlab.token" "last" }}'` + "\n",
			"system/keys/gitlab.token": "system",
			"user/keys/gitlab.token":   "user",
		}
		for name, content := range files {
			path := filepath.Join(root, filepath.FromSlash(name))
			err := os.MkdirAll(filepath.Dir(path), 0o700)
			require.NoError(t, err)
			err = os.WriteFile(path, []byte(content), 0o600)
			require.NoError(t, err)
		}

		unmarshal := config.YamlValueTemplateUnmarshal(nil)
		var actual map[any]any
		err := (config.Sources[map[any]any]{
			Sources: []config.SourceLoader{
				config.FileSource{
					Path:      filepath.Join(system, "config.yml"),
					Unmarshal: unmarshal,
				},
				BaseSource(config.DirSource{Path: userDir}),
			},
		}).Load(&actual)
		require.NoError(t, err)
		require.Equal(t, userFile, actual["token"])
	})
}
