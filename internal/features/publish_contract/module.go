package publish_contract

import (
	"github.com/bidirekt/cli/internal/components"
	"github.com/spf13/cobra"
)

func Register(rootCommand *cobra.Command, dependencies *components.Components) {
	publishContractClient := NewPublishContractClient(dependencies.HTTPClient)
	publishCommand := NewPublishCommand(publishContractClient)
	publishCommand.Annotations = map[string]string{components.TalksToBrokerAnnotation: "true"}
	rootCommand.AddCommand(publishCommand)
}
