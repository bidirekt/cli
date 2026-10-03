package components_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsTerminalReportsCharacterDevicesOnly(t *testing.T) {
	regularFile, err := os.Create(filepath.Join(t.TempDir(), "stdin.txt"))
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, regularFile.Close()) })

	assert.False(t, components.IsTerminal(&bytes.Buffer{}))
	assert.False(t, components.IsTerminal(strings.NewReader("")))
	assert.False(t, components.IsTerminal(regularFile))
	assert.False(t, components.IsTerminal(nil))

	if runtime.GOOS != "windows" {
		characterDevice, err := os.Open(os.DevNull)
		require.NoError(t, err)
		t.Cleanup(func() { assert.NoError(t, characterDevice.Close()) })

		assert.True(t, components.IsTerminal(characterDevice))
	}
}
