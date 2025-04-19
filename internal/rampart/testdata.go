package rampart

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// RampartTestCase represents a single test case from the Rampart test data
type RampartTestCase struct {
	Start struct {
		FEN         string `json:"fen"`
		Description string `json:"description"`
	} `json:"start"`
	Expected []struct {
		Move string `json:"move"`
		FEN  string `json:"fen"`
	} `json:"expected"`
	Description string `json:"description,omitempty"`
}

// RampartTestFile represents a collection of test cases in a Rampart test file
type RampartTestFile struct {
	Description string           `json:"description"`
	TestCases   []RampartTestCase `json:"testCases"`
}

// LoadTestFile loads a Rampart test file from the given path
func LoadTestFile(path string) (*RampartTestFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read test file %s: %w", path, err)
	}

	var testFile RampartTestFile
	if err := json.Unmarshal(data, &testFile); err != nil {
		return nil, fmt.Errorf("failed to parse test file %s: %w", path, err)
	}

	return &testFile, nil
}

// LoadAllTestFiles loads all Rampart test files from the given directory
func LoadAllTestFiles(dir string) (map[string]*RampartTestFile, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to find test files in %s: %w", dir, err)
	}

	testFiles := make(map[string]*RampartTestFile)
	for _, file := range files {
		testFile, err := LoadTestFile(file)
		if err != nil {
			return nil, err
		}
		
		// Use the filename without extension as the key
		key := filepath.Base(file)
		key = key[:len(key)-len(filepath.Ext(key))]
		testFiles[key] = testFile
	}

	return testFiles, nil
}

// GetStartingPositions returns all starting positions from the test files
func GetStartingPositions(testFiles map[string]*RampartTestFile) []struct {
	Category    string
	FEN         string
	Description string
} {
	var positions []struct {
		Category    string
		FEN         string
		Description string
	}

	for category, testFile := range testFiles {
		for _, testCase := range testFile.TestCases {
			positions = append(positions, struct {
				Category    string
				FEN         string
				Description string
			}{
				Category:    category,
				FEN:         testCase.Start.FEN,
				Description: testCase.Start.Description,
			})
		}
	}

	return positions
}
