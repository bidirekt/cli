package components_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsTerminalReportsTerminalsOnly(t *testing.T) {
	regularFile, err := os.Create(filepath.Join(t.TempDir(), "stdin.txt"))
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, regularFile.Close()) })
	nullDevice, err := os.Open(os.DevNull)
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, nullDevice.Close()) })

	assert.False(t, components.IsTerminal(&bytes.Buffer{}))
	assert.False(t, components.IsTerminal(strings.NewReader("")))
	assert.False(t, components.IsTerminal(regularFile))
	assert.False(t, components.IsTerminal(nullDevice))
	assert.False(t, components.IsTerminal(nil))
}
