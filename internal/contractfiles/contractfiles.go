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

const contractsFolderName = "contracts"

var yamlFileExtensions = []string{".yaml", ".yml"}

type ContractFile struct {
	Source  string `json:"source"`
	Content string `json:"content"`
}

func FileTypesHelp(extensions []string) string {
	return fmt.Sprintf("A contract file ends in one of %s. "+
		"A folder must be named contracts, like contracts/ or api/contracts/, and every contract file inside it is read, subfolders included. "+
		"With no argument, the contracts/ folder here is read.",
		strings.Join(slices.Concat(yamlFileExtensions, extensions), ", "))
}

func ToContractFiles(paths []string, extensions []string) ([]ContractFile, error) {
	if len(paths) == 0 {
		info, err := os.Stat(contractsFolderName)
		if err != nil || !info.IsDir() {
			return nil, errors.New("no contract files given and no contracts/ folder here")
		}

		paths = []string{contractsFolderName}
	}

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
		if err := requireContractsFolder(path); err != nil {
			return nil, err
		}

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

func requireContractsFolder(folder string) error {
	cleanFolder := filepath.Clean(folder)
	if filepath.Base(cleanFolder) == contractsFolderName {
		return nil
	}

	for parent := filepath.Dir(cleanFolder); parent != filepath.Dir(parent); parent = filepath.Dir(parent) {
		if filepath.Base(parent) == contractsFolderName {
			return fmt.Errorf("%s/ is inside the contracts folder: pass %s/ instead", cleanFolder, parent)
		}
	}

	return fmt.Errorf("%s/ is not the contracts folder: pass its files by name, or move them to contracts/", cleanFolder)
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
