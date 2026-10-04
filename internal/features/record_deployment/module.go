package record_deployment

import (
	"github.com/bidirekt/cli/internal/components"
	"github.com/spf13/cobra"
)

func Register(rootCommand *cobra.Command, dependencies *components.Components) {
	client := NewRecordDeploymentClient(dependencies.HTTPClient)
	command := NewRecordDeploymentCommand(client)
	command.Annotations = map[string]string{components.TalksToBrokerAnnotation: "true"}
	rootCommand.AddCommand(command)
}
