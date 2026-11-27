package can_i_deploy

import (
	"github.com/bidirekt/cli/internal/components"
	"github.com/bidirekt/cli/internal/reports"
	"github.com/spf13/cobra"
)

func Register(rootCommand *cobra.Command, dependencies *components.Components, modules []reports.Module) {
	client := NewCanIDeployClient(dependencies.HTTPClient)
	command := NewCanIDeployCommand(client, modules)
	command.Annotations = map[string]string{components.TalksToBrokerAnnotation: "true"}
	rootCommand.AddCommand(command)
}
