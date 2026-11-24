package contractfiles_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bidirekt/cli/internal/contractfiles"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToContractFiles(t *testing.T) {
	t.Run("a folder is read with its subfolders, only its .yaml and .yml files, in path order", func(t *testing.T) {
		t.Chdir(t.TempDir())
		writeFiles(t, map[string]string{
			"contracts/schemas.yaml":   "# schemas\n",
			"contracts/rest/store.yml": "# store\n",
			"contracts/rest/pets.yaml": "# pets\n",
			"contracts/rest/notes.txt": "not a contract\n",
			"contracts/README.md":      "# contracts\n",
		})

		files, err := contractfiles.ToContractFiles([]string{"contracts"})

		require.NoError(t, err)
		assert.Equal(t, []contractfiles.ContractFile{
			{Source: "contracts/rest/pets.yaml", Content: "# pets\n"},
			{Source: "contracts/rest/store.yml", Content: "# store\n"},
			{Source: "contracts/schemas.yaml", Content: "# schemas\n"},
		}, files)
	})

	t.Run("a file found in a folder is named by the folder joined with its path inside it", func(t *testing.T) {
		t.Chdir(t.TempDir())
		writeFiles(t, map[string]string{"contracts/rest/pets.yaml": "# pets\n"})

		files, err := contractfiles.ToContractFiles([]string{"./contracts/"})

		require.NoError(t, err)
		assert.Equal(t, []contractfiles.ContractFile{
			{Source: "contracts/rest/pets.yaml", Content: "# pets\n"},
		}, files)
	})

	t.Run("a symlink to a folder is read as that folder, its files named by the link", func(t *testing.T) {
		t.Chdir(t.TempDir())
		writeFiles(t, map[string]string{"shared/contracts/rest/pets.yaml": "# pets\n"})
		require.NoError(t, os.Symlink("shared/contracts", "contracts"))

		files, err := contractfiles.ToContractFiles([]string{"contracts"})

		require.NoError(t, err)
		assert.Equal(t, []contractfiles.ContractFile{
			{Source: "contracts/rest/pets.yaml", Content: "# pets\n"},
		}, files)
	})

	t.Run("folders and files mix in path order, a file passed directly named as typed", func(t *testing.T) {
		t.Chdir(t.TempDir())
		writeFiles(t, map[string]string{
			"zoo.yaml":            "# zoo\n",
			"contracts/pets.yaml": "# pets\n",
			"api.yml":             "# api\n",
		})

		files, err := contractfiles.ToContractFiles([]string{"zoo.yaml", "contracts", "./api.yml"})

		require.NoError(t, err)
		assert.Equal(t, []contractfiles.ContractFile{
			{Source: "./api.yml", Content: "# api\n"},
			{Source: "contracts/pets.yaml", Content: "# pets\n"},
			{Source: "zoo.yaml", Content: "# zoo\n"},
		}, files)
	})
}

func TestToContractFilesReadsAFileReachedTwiceOnce(t *testing.T) {
	for _, test := range []struct {
		name           string
		paths          []string
		expectedSource string
	}{
		{
			name:           "passed twice, spelled as the first argument",
			paths:          []string{"./contracts/rest/pets.yaml", "contracts//rest/pets.yaml"},
			expectedSource: "./contracts/rest/pets.yaml",
		},
		{
			name:           "passed directly, then found in its folder, spelled as passed",
			paths:          []string{"./contracts/rest/pets.yaml", "contracts"},
			expectedSource: "./contracts/rest/pets.yaml",
		},
		{
			name:           "found in its folder, then passed directly, spelled as found",
			paths:          []string{"contracts", "./contracts/rest/pets.yaml"},
			expectedSource: "contracts/rest/pets.yaml",
		},
		{
			name:           "found in its folder and in the parent folder",
			paths:          []string{"contracts/rest", "./contracts"},
			expectedSource: "contracts/rest/pets.yaml",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			writeFiles(t, map[string]string{"contracts/rest/pets.yaml": "# pets\n"})

			files, err := contractfiles.ToContractFiles(test.paths)

			require.NoError(t, err)
			assert.Equal(t, []contractfiles.ContractFile{
				{Source: test.expectedSource, Content: "# pets\n"},
			}, files)
		})
	}
}

func TestToContractFilesFails(t *testing.T) {
	t.Run("a folder without any file fails naming the folder", func(t *testing.T) {
		t.Chdir(t.TempDir())
		require.NoError(t, os.MkdirAll("contracts/rest", 0o700))

		files, err := contractfiles.ToContractFiles([]string{"./contracts"})

		require.EqualError(t, err, `no contract YAML files in folder: "./contracts"`)
		assert.Nil(t, files)
	})

	t.Run("a folder with files of other extensions only fails naming the folder", func(t *testing.T) {
		t.Chdir(t.TempDir())
		writeFiles(t, map[string]string{
			"contracts/pets.yaml":   "# pets\n",
			"docs/README.md":        "# docs\n",
			"docs/drafts/pets.json": "{}\n",
		})

		files, err := contractfiles.ToContractFiles([]string{"contracts", "docs/"})

		require.EqualError(t, err, `no contract YAML files in folder: "docs/"`)
		assert.Nil(t, files)
	})

	t.Run("a folder that does not exist fails naming it", func(t *testing.T) {
		t.Chdir(t.TempDir())
		writeFiles(t, map[string]string{"contracts/spa/pets.yaml": "# pets\n"})

		files, err := contractfiles.ToContractFiles([]string{"contracts/sap"})

		require.EqualError(t, err, `contract file or folder not found: "contracts/sap"`)
		assert.Nil(t, files)
	})

	t.Run("a file passed directly still needs a .yaml or .yml extension", func(t *testing.T) {
		t.Chdir(t.TempDir())
		writeFiles(t, map[string]string{
			"contracts/pets.yaml": "# pets\n",
			"notes.txt":           "not a contract\n",
		})

		files, err := contractfiles.ToContractFiles([]string{"contracts", "notes.txt"})

		require.EqualError(t, err, `unsupported contract file extension: "notes.txt"`)
		assert.Nil(t, files)
	})
}

func writeFiles(t *testing.T, contentByPath map[string]string) {
	t.Helper()

	for path, content := range contentByPath {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	}
}
