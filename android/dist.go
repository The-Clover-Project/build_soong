// Copyright 2025 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package android

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/blueprint/proptools"
)

//go:generate go run ../../blueprint/gobtools/codegen

// The contributions to the dist.
type distContributions struct {
	// Path to license metadata file.
	licenseMetadataFile Path
	// List of goals and the dist copy instructions.
	copiesForGoals []*copiesForGoals
}

// getCopiesForGoals returns a copiesForGoals into which copy instructions that
// must be processed when building one or more of those goals can be added.
func (d *distContributions) getCopiesForGoals(goals []string) *copiesForGoals {
	copiesForGoals := &copiesForGoals{goals: goals}
	d.copiesForGoals = append(d.copiesForGoals, copiesForGoals)
	return copiesForGoals
}

// Associates a list of dist copy instructions with a set of goals for which they
// should be run.
type copiesForGoals struct {
	// goals are build targets that will trigger the copy instructions.
	goals []string

	// A list of instructions to copy a module's output files to somewhere in the
	// dist directory.
	copies []distCopy
}

// Adds a copy instruction.
func (d *copiesForGoals) addCopyInstruction(from Path, dest string) {
	d.copies = append(d.copies, distCopy{from, dest})
}

// Instruction on a path that must be copied into the dist.
// @auto-generate: gob
type distCopy struct {
	// The path to copy from.
	from Path

	// The destination within the dist directory to copy to.
	dest string
}

func (d *distCopy) String() string {
	if len(d.dest) == 0 {
		return d.from.String()
	}
	return fmt.Sprintf("%s:%s", d.from.String(), d.dest)
}

type distCopies []distCopy

func (d *distCopies) Strings() (ret []string) {
	if d == nil {
		return
	}
	for _, dist := range *d {
		ret = append(ret, dist.String())
	}
	return
}

// This gets the dist contributions from the given module that were specified in the Android.bp
// file using the dist: property. It does not include contributions that the module's
// implementation may have defined with ctx.DistForGoals(), for that, see DistProvider.
func getDistContributions(ctx ConfigAndOtherModuleProviderContext, mod ModuleOrProxy) *distContributions {
	name := mod.Name()

	commonInfo := OtherModulePointerProviderOrDefault(ctx, mod, CommonModuleInfoProvider)

	info := GetInstallFilesCommon(commonInfo)
	availableTaggedDists := info.DistFiles

	if len(availableTaggedDists) == 0 {
		// Nothing dist-able for this module.
		return nil
	}

	// Collate the contributions this module makes to the dist.
	distContributions := &distContributions{}

	if !exemptFromRequiredApplicableLicensesProperty(mod) {
		distContributions.licenseMetadataFile = info.LicenseMetadataFile
	}

	// Iterate over this module's dist structs, merged from the dist and dists properties.
	for _, dist := range commonInfo.Dists {
		// Get the list of goals this dist should be enabled for. e.g. sdk, droidcore
		goals := dist.Targets

		// Get the tag representing the output files to be dist'd. e.g. ".jar", ".proguard_map"
		var tag string
		if dist.Tag == nil {
			// If the dist struct does not specify a tag, use the default output files tag.
			tag = DefaultDistTag
		} else {
			tag = *dist.Tag
		}

		// Get the paths of the output files to be dist'd, represented by the tag.
		// Can be an empty list.
		tagPaths := availableTaggedDists[tag]
		if len(tagPaths) == 0 {
			// Nothing to dist for this tag, continue to the next dist.
			continue
		}

		if len(tagPaths) > 1 && (dist.Dest != nil || dist.Suffix != nil) {
			errorMessage := "%s: Cannot apply dest/suffix for more than one dist " +
				"file for %q goals tag %q in module %s. The list of dist files, " +
				"which should have a single element, is:\n%s"
			panic(fmt.Errorf(errorMessage, mod, goals, tag, name, tagPaths))
		}

		copiesForGoals := distContributions.getCopiesForGoals(goals)

		// Iterate over each path adding a copy instruction to copiesForGoals
		for _, path := range tagPaths {
			// It's possible that the Path is nil from errant modules. Be defensive here.
			if path == nil {
				tagName := "default" // for error message readability
				if dist.Tag != nil {
					tagName = *dist.Tag
				}
				panic(fmt.Errorf("Dist file should not be nil for the %s tag in %s", tagName, name))
			}

			dest := filepath.Base(path.String())

			if dist.Dest != nil {
				var err error
				if dest, err = validateSafePath(*dist.Dest); err != nil {
					// This was checked in ModuleBase.GenerateBuildActions
					panic(err)
				}
			}

			ext := filepath.Ext(dest)
			suffix := ""
			if dist.Suffix != nil {
				suffix = *dist.Suffix
			}

			prependProductString := ""
			if proptools.Bool(dist.Prepend_artifact_with_product) {
				prependProductString = fmt.Sprintf("%s-", ctx.Config().DeviceProduct())
			}

			appendProductString := ""
			if proptools.Bool(dist.Append_artifact_with_product) {
				appendProductString = fmt.Sprintf("_%s", ctx.Config().DeviceProduct())
			}

			if suffix != "" || appendProductString != "" || prependProductString != "" {
				dest = prependProductString + strings.TrimSuffix(dest, ext) + suffix + appendProductString + ext
			}

			if dist.Dir != nil {
				var err error
				if dest, err = validateSafePath(*dist.Dir, dest); err != nil {
					// This was checked in ModuleBase.GenerateBuildActions
					panic(err)
				}
			}

			copiesForGoals.addCopyInstruction(path, dest)
		}
	}

	return distContributions
}
