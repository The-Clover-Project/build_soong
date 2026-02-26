// Copyright 2025 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package android

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/blueprint"
)

func init() {
	RegisterParallelSingletonType("soong_api_db", soongApiSingletonFactory)
}

var soongApiPctx = NewPackageContext("android/soong/android/soong_api")

// SoongApiModuleRecord represents a single entry in the Soong API database.
// The term "Record" is used to clarify that this is a data snapshot of a
// module's properties intended for database storage (soong_api.db),
// rather than a functional Soong module object.
type SoongApiModuleRecord struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Path         string   `json:"path"`
	Enabled      bool     `json:"enabled"`
	InstallFiles []string `json:"install_files,omitempty"`
}

func soongApiSingletonFactory() Singleton {
	return &soongApiSingleton{}
}

type soongApiSingleton struct{}

func (c *soongApiSingleton) GenerateBuildActions(ctx SingletonContext) {
	var records []SoongApiModuleRecord

	ctx.VisitAllModuleProxies(func(m ModuleProxy) {
		commonInfo, ok := OtherModuleProvider(ctx, m, CommonModuleInfoProvider)

		if !ok {
			return
		}

		record := SoongApiModuleRecord{
			Name:    ctx.ModuleName(m),
			Type:    ctx.ModuleType(m),
			Path:    ctx.ModuleDir(m),
			Enabled: commonInfo.Enabled,
		}

		if commonInfo.InstallFiles != nil {
			for _, p := range commonInfo.InstallFiles.InstallFiles {
				record.InstallFiles = append(record.InstallFiles, p.String())
			}
		}

		records = append(records, record)
	})

	// Serialize the records into JSON format in memory.
	jsonData, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		ctx.Errorf("Failed to marshal soong api records: %s", err)
		return
	}

	// Create the ZIP content directly in memory.
	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)

	// Create a file entry within the ZIP named "soong_api.json".
	f, err := zipWriter.Create("soong_api.json")
	if err != nil {
		ctx.Errorf("Failed to create zip entry: %s", err)
		return
	}
	if _, err := f.Write(jsonData); err != nil {
		ctx.Errorf("Failed to write json to zip: %s", err)
		return
	}
	zipWriter.Close()

	// Use TARGET_PRODUCT (DeviceProduct) to partition the output directory.
	product := "generic" // Fallback for safety
	if ctx.Config().HasDeviceProduct() {
		product = ctx.Config().DeviceProduct()
	}

	// Output path for soong_api.zip
	// Path: out/soong/soong_api/<product>/soong_api.zip
	zipPath := PathForOutput(ctx, "soong_api", product, "soong_api.zip")
	WriteContentToFile(zipPath, zipBuf.String())

	ctx.DistForGoal("droid", zipPath)

	// Generate the soong_api.db using the ZIP file as the input source.
	// Path: out/soong/soong_api/<product>/soong_api.db
	soongApiDbPath := PathForOutput(ctx, "soong_api", product, "soong_api.db")

	dbRb := NewRuleBuilder(soongApiPctx, ctx)

	loaderPath := ctx.Config().HostToolPath(ctx, "soong_api_db_loader")

	// Build the command: <loader> -i <zip_file_input> -o <db_output>
	dbRb.Command().
		Tool(loaderPath).
		FlagWithInput("-i ", zipPath).
		FlagWithOutput("-o ", soongApiDbPath)

	dbRb.Build("build_soong_api_db", "Building soong_api.db from soong_api.zip")

	// Phony target for 'm soong_api.db'
	ctx.Build(soongApiPctx, BuildParams{
		Rule:   blueprint.Phony,
		Input:  soongApiDbPath,
		Output: PathForPhony(ctx, "soong_api.db"),
	})
}

// WriteContentToFile writes content to the given Path no matter what the file exist.
func WriteContentToFile(path Path, content string) {
	// 1. Convert Path to an absolute path string (e.g., "/usr/local/xxx/git_main/out/soong/soong_api/...")
	filePath := absolutePath(path.String())

	// 2. Get the directory path
	dir := filepath.Dir(filePath)

	// 3. Create the directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		panic(fmt.Errorf("failed to create directory %q: %w", dir, err))
	}

	// 4. Create the file
	f, err := os.Create(filePath)
	if err != nil {
		panic(fmt.Errorf("failed to create file %q: %w", filePath, err))
	}
	defer f.Close()

	// 5. Write content
	if _, err := io.WriteString(f, content); err != nil {
		panic(fmt.Errorf("failed to write content to %q: %w", filePath, err))
	}
}
