package contractfiles

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var supportedFileExtensions = map[string]bool{".yaml": true, ".yml": true}

type ContractFragment struct {
	Source  string `json:"source"`
	Content string `json:"content"`
}

func ToContractFragments(paths []string) ([]ContractFragment, error) {
	filePaths, err := toContractFilePaths(paths)
	if err != nil {
		return nil, err
	}

	contracts := make([]ContractFragment, 0, len(filePaths))

	for _, filePath := range filePaths {
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

func toContractFilePaths(paths []string) ([]string, error) {
	var uniqueFilePaths []string
	reachedCleanPaths := map[string]bool{}

	for _, path := range paths {
		filePaths, err := pathToContractFilePaths(path)
		if err != nil {
			return nil, err
		}

		for _, filePath := range filePaths {
			cleanPath := filepath.Clean(filePath)
			if reachedCleanPaths[cleanPath] {
				continue
			}

			reachedCleanPaths[cleanPath] = true
			uniqueFilePaths = append(uniqueFilePaths, filePath)
		}
	}

	slices.Sort(uniqueFilePaths)

	return uniqueFilePaths, nil
}

func pathToContractFilePaths(path string) ([]string, error) {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return folderToContractFilePaths(path)
	}

	if !isContractFile(path) {
		return nil, fmt.Errorf("unsupported contract file extension: %q", path)
	}

	return []string{path}, nil
}

func folderToContractFilePaths(folder string) ([]string, error) {
	var filePaths []string

	err := fs.WalkDir(os.DirFS(folder), ".", func(pathInFolder string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !entry.IsDir() && isContractFile(pathInFolder) {
			filePaths = append(filePaths, filepath.Join(folder, filepath.FromSlash(pathInFolder)))
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read contract folder %q: %w", folder, err)
	}

	if len(filePaths) == 0 {
		return nil, fmt.Errorf("no contract YAML files in folder: %q", folder)
	}

	return filePaths, nil
}

func isContractFile(path string) bool {
	return supportedFileExtensions[strings.ToLower(filepath.Ext(path))]
}
