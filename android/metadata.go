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
	"encoding/json"
	"path/filepath"

	"github.com/google/blueprint"
)

func init() {
	RegisterParallelSingletonType("metadata_db", metadataSingletonFactory)
}

var metadataPctx = NewPackageContext("android/soong/android/metadata")

type ModuleMetadata struct {
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Path         string   `json:"path"`
	Enabled      bool     `json:"enabled"`
	InstallFiles []string `json:"install_files,omitempty"`
}

func metadataSingletonFactory() Singleton {
	return &metadataSingleton{}
}

type metadataSingleton struct{}

func (c *metadataSingleton) GenerateBuildActions(ctx SingletonContext) {
	var modules []ModuleMetadata

	ctx.VisitAllModuleProxies(func(m ModuleProxy) {
		commonInfo, ok := OtherModuleProvider(ctx, m, CommonModuleInfoProvider)

		if !ok {
			return
		}

		info := ModuleMetadata{
			Name:    ctx.ModuleName(m),
			Type:    ctx.ModuleType(m),
			Path:    ctx.ModuleDir(m),
			Enabled: commonInfo.Enabled,
		}

		if commonInfo.InstallFiles != nil {
			for _, p := range commonInfo.InstallFiles.InstallFiles {
				info.InstallFiles = append(info.InstallFiles, p.String())
			}
		}

		modules = append(modules, info)
	})

	jsonData, err := json.MarshalIndent(modules, "", "  ")
	if err != nil {
		ctx.Errorf("Failed to marshal metadata: %s", err)
		return
	}

	jsonPath := PathForOutput(ctx, "metadata", "metadata.json")
	WriteFileRule(ctx, jsonPath, string(jsonData))

	zipPath := PathForOutput(ctx, "metadata", "metadata.zip")
	baseDir := filepath.Dir(jsonPath.String())

	// Rule to build metadata.zip
	zipRb := NewRuleBuilder(metadataPctx, ctx)
	zipRb.Command().
		BuiltTool("soong_zip").
		FlagWithOutput("-o ", zipPath).
		FlagWithArg("-C ", baseDir).
		FlagWithInput("-f ", jsonPath)
	zipRb.Build("build_metadata_zip", "Building metadata zip")

	// Phony target for 'm metadata.zip'
	ctx.Build(metadataPctx, BuildParams{
		Rule:   blueprint.Phony,
		Input:  zipPath,
		Output: PathForPhony(ctx, "metadata.zip"),
	})

	// Rule to build metadata.db from metadata.zip
	metadataDbPath := PathForOutput(ctx, "metadata", "metadata.db")
	dbRb := NewRuleBuilder(metadataPctx, ctx)

	// Get the path to the metadata_db_loader executable
	loaderPath := ctx.Config().HostToolPath(ctx, "metadata_db_loader")

	// Build the command: <path_to_loader_script> -i <input> -o <output>
	dbRb.Command().
		Tool(loaderPath).
		FlagWithInput("-i ", jsonPath).
		FlagWithOutput("-o ", metadataDbPath)

	dbRb.Build("build_metadata_db", "Building metadata.db from metadata.json")

	// Phony target for 'm metadata.db'
	ctx.Build(metadataPctx, BuildParams{
		Rule:   blueprint.Phony,
		Input:  metadataDbPath,
		Output: PathForPhony(ctx, "metadata.db"),
	})
}
