// SPDX-FileCopyrightText: Copyright 2023 Prasad Tengse
// SPDX-License-Identifier: MIT

package apitestdata

import (
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// Testdata App Owner.
const AppOwner = "gh-integration-tests"

// Testdata installation owner.
const InstallationOwner = "gh-integration-tests"

// Installation repository.
const InstallationRepository = "go-githubapp-repo-one"

// Testdata installation ID.
const InstallationID = 42101303

// Test App ID.
const AppID = 145695471

// Test App slug.
const AppSlug = "gh-integration-tests-app"

// Read api data once.
var once sync.Once

// API dat storage.
var apiDataMap map[string][]byte

// Get returns API test data which is map of test data to JSON responses
// From API endpoint.
func Get(t *testing.T) map[string][]byte {
	once.Do(func() {
		apiDataMap = make(map[string][]byte)
		dir := filepath.Join("internal", "testdata", "apitestdata")
		items, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("failed to read dir %s: %s", dir, err)
		}

		dataFiles := make([]fs.DirEntry, 0, len(items))
		for _, item := range items {
			if filepath.Ext(item.Name()) == ".json" && item.Type().IsRegular() {
				dataFiles = append(dataFiles, item)
			}
		}

		if len(dataFiles) == 0 {
			t.Fatalf("no api response data found in %s", dir)
		}

		for _, item := range dataFiles {
			slurp, err := os.ReadFile(filepath.Join(dir, item.Name()))
			if err != nil {
				t.Fatalf("Failed to read file %s: %s", item, err)
			}

			apiDataMap[item.Name()] = slurp
			apiDataMap[strings.TrimSuffix(item.Name(), ".json")] = slurp
		}
	})

	if apiDataMap == nil {
		t.Fatalf("failed to populate api data")
	}

	// Return clone of the map, as some callers may mutate map keys.
	return maps.Clone(apiDataMap)
}
