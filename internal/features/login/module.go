package login

import (
	"github.com/bidirekt/cli/internal/components"
	"github.com/spf13/cobra"
)

func Register(rootCommand *cobra.Command, dependencies *components.Components) {
	client := NewLoginClient(dependencies.HTTPClient)
	command := NewLoginCommand(dependencies.Config, client, components.IsTerminal, components.ReadSecret)
	rootCommand.AddCommand(command)
}
