package publish_contract

import (
	"github.com/bidirekt/cli/internal/components"
	"github.com/bidirekt/cli/internal/reports"
	"github.com/spf13/cobra"
)

func Register(rootCommand *cobra.Command, dependencies *components.Components, modules []reports.Module) {
	publishContractClient := NewPublishContractClient(dependencies.HTTPClient)
	publishCommand := NewPublishCommand(publishContractClient, modules)
	publishCommand.Annotations = map[string]string{components.TalksToBrokerAnnotation: "true"}
	rootCommand.AddCommand(publishCommand)
}
