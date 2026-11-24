package contractfiles

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

var yamlFileExtensions = []string{".yaml", ".yml"}

type ContractFile struct {
	Source  string `json:"source"`
	Content string `json:"content"`
}

func ToContractFiles(paths []string, extensions []string) ([]ContractFile, error) {
	filePaths, err := toContractFilePaths(paths, slices.Concat(yamlFileExtensions, extensions))
	if err != nil {
		return nil, err
	}

	contracts := make([]ContractFile, 0, len(filePaths))

	for _, filePath := range filePaths {
		contractFileContent, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("read contract file: %w", err)
		}

		contracts = append(contracts, ContractFile{
			Source:  filePath,
			Content: string(contractFileContent),
		})
	}

	return contracts, nil
}

func toContractFilePaths(paths []string, extensions []string) ([]string, error) {
	var uniqueFilePaths []string
	reachedCleanPaths := map[string]bool{}

	for _, path := range paths {
		filePaths, err := pathToContractFilePaths(path, extensions)
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

func pathToContractFilePaths(path string, extensions []string) ([]string, error) {
	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		return folderToContractFilePaths(path, extensions)
	}

	if !isContractFile(path, extensions) {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("contract file or folder not found: %q", path)
		}

		return nil, fmt.Errorf("unsupported contract file extension: %q", path)
	}

	return []string{path}, nil
}

func folderToContractFilePaths(folder string, extensions []string) ([]string, error) {
	var filePaths []string

	err := fs.WalkDir(os.DirFS(folder), ".", func(pathInFolder string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !entry.IsDir() && isContractFile(pathInFolder, extensions) {
			filePaths = append(filePaths, filepath.Join(folder, filepath.FromSlash(pathInFolder)))
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read contract folder %q: %w", folder, err)
	}

	if len(filePaths) == 0 {
		return nil, fmt.Errorf("no contract files (%s) in folder: %q", strings.Join(extensions, ", "), folder)
	}

	return filePaths, nil
}

func isContractFile(path string, extensions []string) bool {
	return slices.Contains(extensions, strings.ToLower(filepath.Ext(path)))
}
