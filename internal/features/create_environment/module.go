package create_environment

import (
	"github.com/bidirekt/cli/internal/components"
	"github.com/spf13/cobra"
)

func Register(rootCommand *cobra.Command, dependencies *components.Components) {
	client := NewCreateEnvironmentClient(dependencies.HTTPClient)
	command := NewCreateEnvironmentCommand(client)
	command.Annotations = map[string]string{components.TalksToBrokerAnnotation: "true"}
	rootCommand.AddCommand(command)
}
