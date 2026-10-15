package components

import (
	"os"

	"golang.org/x/term"
)

func IsTerminal(stream any) bool {
	file, ok := stream.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(file.Fd()))
}
