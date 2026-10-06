package list_participants

import (
	"github.com/bidirekt/cli/internal/components"
	"github.com/spf13/cobra"
)

func Register(rootCommand *cobra.Command, dependencies *components.Components) {
	client := NewListParticipantsClient(dependencies.HTTPClient)
	command := NewListParticipantsCommand(client)
	command.Annotations = map[string]string{components.TalksToBrokerAnnotation: "true"}
	rootCommand.AddCommand(command)
}
