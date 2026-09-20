package rename_participant

import (
	"github.com/bidirekt/cli/internal/components"
	"github.com/spf13/cobra"
)

func Register(rootCommand *cobra.Command, components *components.Components) {
	client := NewRenameParticipantClient(components.HTTPClient)
	command := NewRenameParticipantCommand(client)
	rootCommand.AddCommand(command)
}
