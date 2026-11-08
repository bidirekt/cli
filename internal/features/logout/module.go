package logout

import (
	"github.com/bidirekt/cli/internal/components"
	"github.com/spf13/cobra"
)

func Register(rootCommand *cobra.Command, dependencies *components.Components) {
	rootCommand.AddCommand(NewLogoutCommand(dependencies.Config))
}
