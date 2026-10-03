package create_participant

import (
	"github.com/bidirekt/cli/internal/components"
	"github.com/spf13/cobra"
)

func Register(rootCommand *cobra.Command, dependencies *components.Components) {
	client := NewCreateParticipantClient(dependencies.HTTPClient)
	command := NewCreateParticipantCommand(client)
	command.Annotations = map[string]string{components.TalksToBrokerAnnotation: "true"}
	rootCommand.AddCommand(command)
}
