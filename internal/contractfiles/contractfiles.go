package contractfiles

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var supportedFileExtensions = map[string]bool{".yaml": true, ".yml": true}

type ContractFragment struct {
	Source  string `json:"source"`
	Content string `json:"content"`
}

func ToContractFragments(filePaths []string) ([]ContractFragment, error) {
	contracts := make([]ContractFragment, 0, len(filePaths))

	for _, filePath := range filePaths {
		if !supportedFileExtensions[strings.ToLower(filepath.Ext(filePath))] {
			return nil, fmt.Errorf("unsupported contract file extension: %q", filePath)
		}

		contractFileContent, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("read contract file: %w", err)
		}

		contracts = append(contracts, ContractFragment{
			Source:  filePath,
			Content: string(contractFileContent),
		})
	}

	return contracts, nil
}
