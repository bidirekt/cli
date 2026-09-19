package internal

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionIsPrintedByTheFlagAndTheCommand(t *testing.T) {
	rootCommand.AddCommand(versionCommand)

	for _, arg := range []string{"--version", "version"} {
		t.Run(arg, func(t *testing.T) {
			var out, errOut bytes.Buffer
			rootCommand.SetOut(&out)
			rootCommand.SetErr(&errOut)
			rootCommand.SetArgs([]string{arg})

			require.NoError(t, rootCommand.Execute())
			assert.Equal(t, "ctio version dev\n", out.String())
			assert.Empty(t, errOut.String())
		})
	}
}
