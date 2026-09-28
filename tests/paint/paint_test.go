package paint_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/bidirekt/cli/internal/paint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrushPaintsOnlyWhenColorIsWanted(t *testing.T) {
	buffer := func(*testing.T) io.Writer { return &bytes.Buffer{} }
	regularFile := func(t *testing.T) io.Writer {
		file, err := os.Create(filepath.Join(t.TempDir(), "out.txt"))
		require.NoError(t, err)
		t.Cleanup(func() { assert.NoError(t, file.Close()) })
		return file
	}

	tests := []struct {
		name          string
		noColor       string
		cliColorForce string
		writer        func(t *testing.T) io.Writer
		green         string
		red           string
	}{
		{name: "buffer without any variable stays plain", writer: buffer, green: "x", red: "x"},
		{name: "regular file without any variable stays plain", writer: regularFile, green: "x", red: "x"},
		{name: "CLICOLOR_FORCE=1 paints even a buffer", cliColorForce: "1", writer: buffer, green: "\x1b[32mx\x1b[0m", red: "\x1b[31mx\x1b[0m"},
		{name: "CLICOLOR_FORCE=0 does not force", cliColorForce: "0", writer: buffer, green: "x", red: "x"},
		{name: "NO_COLOR wins over CLICOLOR_FORCE", noColor: "1", cliColorForce: "1", writer: buffer, green: "x", red: "x"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("NO_COLOR", test.noColor)
			t.Setenv("CLICOLOR_FORCE", test.cliColorForce)

			brush := paint.For(test.writer(t))

			assert.Equal(t, test.green, brush.Green("x"))
			assert.Equal(t, test.red, brush.Red("x"))
		})
	}
}
