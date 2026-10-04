package configure

import (
	"github.com/bidirekt/cli/internal/components"
	"github.com/spf13/cobra"
)

func Register(rootCommand *cobra.Command, dependencies *components.Components) {
	command := NewConfigureCommand(dependencies.Config, components.IsTerminal)
	rootCommand.AddCommand(command)
}
