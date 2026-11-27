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

		files, err := contractfiles.ToContractFiles([]string{"contracts"}, nil)

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

		files, err := contractfiles.ToContractFiles([]string{"./contracts/"}, nil)

		require.NoError(t, err)
		assert.Equal(t, []contractfiles.ContractFile{
			{Source: "contracts/rest/pets.yaml", Content: "# pets\n"},
		}, files)
	})

	t.Run("a symlink to a folder is read as that folder, its files named by the link", func(t *testing.T) {
		t.Chdir(t.TempDir())
		writeFiles(t, map[string]string{"shared/contracts/rest/pets.yaml": "# pets\n"})
		require.NoError(t, os.Symlink("shared/contracts", "contracts"))

		files, err := contractfiles.ToContractFiles([]string{"contracts"}, nil)

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

		files, err := contractfiles.ToContractFiles([]string{"zoo.yaml", "contracts", "./api.yml"}, nil)

		require.NoError(t, err)
		assert.Equal(t, []contractfiles.ContractFile{
			{Source: "./api.yml", Content: "# api\n"},
			{Source: "contracts/pets.yaml", Content: "# pets\n"},
			{Source: "zoo.yaml", Content: "# zoo\n"},
		}, files)
	})

	t.Run("the extensions a module adds are read beside the .yaml and .yml files, in a folder or passed directly", func(t *testing.T) {
		t.Chdir(t.TempDir())
		writeFiles(t, map[string]string{
			"contracts/pets.yaml":        "# pets\n",
			"contracts/topics/pets.csv":  "# pet topics\n",
			"contracts/topics/notes.txt": "not a contract\n",
			"orders.CSV":                 "# order topics\n",
		})

		files, err := contractfiles.ToContractFiles([]string{"contracts", "orders.CSV"}, []string{".csv"})

		require.NoError(t, err)
		assert.Equal(t, []contractfiles.ContractFile{
			{Source: "contracts/pets.yaml", Content: "# pets\n"},
			{Source: "contracts/topics/pets.csv", Content: "# pet topics\n"},
			{Source: "orders.CSV", Content: "# order topics\n"},
		}, files)
	})

	t.Run("GraphQL schema files are read beside the YAML files, in a folder or passed directly", func(t *testing.T) {
		t.Chdir(t.TempDir())
		writeFiles(t, map[string]string{
			"contracts/graph.yaml":            "provides:\n  graphql:\n    graph: petstore_graph\n",
			"contracts/graph/schema.graphqls": "type Query { pet(id: ID!): Pet }\n",
			"contracts/graph/pet.graphql":     "type Pet { name: String! }\n",
			"contracts/graph/owner.gql":       "type Owner { name: String! }\n",
			"reviews.graphql":                 "type Review { stars: Int! }\n",
			"orders.gql":                      "type Order { total: Float! }\n",
		})

		files, err := contractfiles.ToContractFiles([]string{"contracts", "reviews.graphql", "orders.gql"}, []string{".graphql", ".graphqls", ".gql"})

		require.NoError(t, err)
		assert.Equal(t, []contractfiles.ContractFile{
			{Source: "contracts/graph.yaml", Content: "provides:\n  graphql:\n    graph: petstore_graph\n"},
			{Source: "contracts/graph/owner.gql", Content: "type Owner { name: String! }\n"},
			{Source: "contracts/graph/pet.graphql", Content: "type Pet { name: String! }\n"},
			{Source: "contracts/graph/schema.graphqls", Content: "type Query { pet(id: ID!): Pet }\n"},
			{Source: "orders.gql", Content: "type Order { total: Float! }\n"},
			{Source: "reviews.graphql", Content: "type Review { stars: Int! }\n"},
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

			files, err := contractfiles.ToContractFiles(test.paths, nil)

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

		files, err := contractfiles.ToContractFiles([]string{"./contracts"}, nil)

		require.EqualError(t, err, `no contract files (.yaml, .yml) in folder: "./contracts"`)
		assert.Nil(t, files)
	})

	t.Run("a folder with files of other extensions only fails naming the folder", func(t *testing.T) {
		t.Chdir(t.TempDir())
		writeFiles(t, map[string]string{
			"contracts/pets.yaml":   "# pets\n",
			"docs/README.md":        "# docs\n",
			"docs/drafts/pets.json": "{}\n",
		})

		files, err := contractfiles.ToContractFiles([]string{"contracts", "docs/"}, nil)

		require.EqualError(t, err, `no contract files (.yaml, .yml) in folder: "docs/"`)
		assert.Nil(t, files)
	})

	t.Run("a folder without any contract file lists every extension it looked for", func(t *testing.T) {
		t.Chdir(t.TempDir())
		writeFiles(t, map[string]string{"contracts/README.md": "# contracts\n"})

		files, err := contractfiles.ToContractFiles([]string{"contracts"}, []string{".graphql", ".graphqls", ".gql"})

		require.EqualError(t, err, `no contract files (.yaml, .yml, .graphql, .graphqls, .gql) in folder: "contracts"`)
		assert.Nil(t, files)
	})

	t.Run("a folder that does not exist fails naming it", func(t *testing.T) {
		t.Chdir(t.TempDir())
		writeFiles(t, map[string]string{"contracts/spa/pets.yaml": "# pets\n"})

		files, err := contractfiles.ToContractFiles([]string{"contracts/sap"}, nil)

		require.EqualError(t, err, `contract file or folder not found: "contracts/sap"`)
		assert.Nil(t, files)
	})

	t.Run("a file passed directly still needs a .yaml or .yml extension", func(t *testing.T) {
		t.Chdir(t.TempDir())
		writeFiles(t, map[string]string{
			"contracts/pets.yaml": "# pets\n",
			"notes.txt":           "not a contract\n",
		})

		files, err := contractfiles.ToContractFiles([]string{"contracts", "notes.txt"}, nil)

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
