package components_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadSecretRefusesWhatIsNotATerminal(t *testing.T) {
	regularFile, err := os.Create(filepath.Join(t.TempDir(), "stdin.txt"))
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, regularFile.Close()) })
	nullDevice, err := os.Open(os.DevNull)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, nullDevice.Close()) })

	for name, stream := range map[string]any{
		"a reader":  strings.NewReader("bdk_acme\n"),
		"a file":    regularFile,
		"/dev/null": nullDevice,
		"nothing":   nil,
	} {
		t.Run(name, func(t *testing.T) {
			secret, err := components.ReadSecret(stream)

			assert.EqualError(t, err, "read secret: stdin is not a terminal")
			assert.Empty(t, secret)
		})
	}
}
