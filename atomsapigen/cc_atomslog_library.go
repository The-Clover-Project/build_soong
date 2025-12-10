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

package atomsapigen

import (
	"android/soong/android"
	"android/soong/cc"
	"github.com/google/blueprint/proptools"
	"path/filepath"
)

var (
	pctx = android.NewPackageContext("android/soong/atomsapigen")
)

const (
	includeDefaultLibsFull   = "full"
	includeDefaultLibsHeader = "headers_only"
	includeDefaultLibsNone   = "none"
)

func init() {
	RegisterBuildComponents(android.InitRegistrationContext)
}

func RegisterBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("cc_atomslog_library", CcAtomslogLibraryFactory)
	ctx.RegisterModuleType("cc_atomslog_library_static", CcAtomslogLibraryStaticFactory)
	ctx.RegisterModuleType("cc_atomslog_library_shared", CcAtomslogLibrarySharedFactory)
}

type CcAtomslogLibraryProperties struct {
	// Atoms module annotation.
	Atoms_module string

	// C++ namespace where the generated symbols go.
	Namespace string

	// Basename of the header and cpp files generated.
	Basename string

	// Options for including libstatssocket and libstatspull dependencies. There are 3 choices:
	// "full": include libstatssocket and libstatspull as shared_libs,
	// "headers_only": include libstatssocket_headers and libstatspull_headers as header_libs,
	// "none": Nothing is included by default.
	// The default option is "full".
	Include_default_libs *string
}

type CcAtomslogLibraryCallbacks struct {
	properties *CcAtomslogLibraryProperties

	generatedCpp android.WritablePath
	generatedHdr android.WritablePath
}

func CcAtomslogLibraryFactory() android.Module {
	callbacks := &CcAtomslogLibraryCallbacks{
		properties: &CcAtomslogLibraryProperties{},
	}
	return cc.GeneratedCcLibraryModuleFactory(callbacks)
}

func CcAtomslogLibraryStaticFactory() android.Module {
	callbacks := &CcAtomslogLibraryCallbacks{
		properties: &CcAtomslogLibraryProperties{},
	}
	return cc.GeneratedCcLibraryStaticModuleFactory(callbacks)
}

func CcAtomslogLibrarySharedFactory() android.Module {
	callbacks := &CcAtomslogLibraryCallbacks{
		properties: &CcAtomslogLibraryProperties{},
	}
	return cc.GeneratedCcLibrarySharedModuleFactory(callbacks)
}

func (this *CcAtomslogLibraryCallbacks) GeneratorInit(ctx cc.BaseModuleContext) {
}

func (this *CcAtomslogLibraryCallbacks) GeneratorProps() []interface{} {
	return []interface{}{this.properties}
}

func (this *CcAtomslogLibraryCallbacks) GeneratorDeps(ctx cc.DepsContext, deps cc.Deps) cc.Deps {
	// Add the library containing the extra C++ sources like StatsHistogram.
	deps.WholeStaticLibs = append(deps.WholeStaticLibs, "stats-log-api-gen-cc-lib")

	includeDefaultLibs := proptools.StringDefault(this.properties.Include_default_libs, includeDefaultLibsFull)
	switch includeDefaultLibs {
	case includeDefaultLibsFull:
		deps.SharedLibs = android.Concat(deps.SharedLibs, []string{"libstatssocket", "libstatspull"})
	case includeDefaultLibsHeader:
		deps.HeaderLibs = android.Concat(deps.HeaderLibs, []string{"libstatssocket_headers", "libstatspull_headers"})
	case includeDefaultLibsNone:
	default:
		ctx.PropertyErrorf(
			"include_default_libs", "must be one of \"%s\", \"%s\", or \"%s\"",
			includeDefaultLibsFull,
			includeDefaultLibsHeader,
			includeDefaultLibsNone)
	}

	return deps
}

func (this *CcAtomslogLibraryCallbacks) GeneratorSources(ctx cc.ModuleContext) cc.GeneratedSource {
	result := cc.GeneratedSource{}

	atomsModule := this.properties.Atoms_module
	basename := this.properties.Basename

	if basename == "" {
		if atomsModule == "" {
			ctx.ModuleErrorf("At least one of atoms_module or basename must be provided")
			return result
		} else {
			basename = "statslog_" + atomsModule
		}
	}

	// Figure out the generated file paths.
	generatedIncludeDir := android.PathForModuleGen(ctx, "include")
	result.IncludeDirs = []android.Path{generatedIncludeDir}
	result.ReexportedDirs = []android.Path{generatedIncludeDir}

	this.generatedCpp = android.PathForModuleGen(ctx, basename+".cpp")
	result.Sources = []android.Path{this.generatedCpp}

	// Put the header file in an include subfolder so the .cpp file can't be included by clients.
	this.generatedHdr = android.PathForModuleGen(ctx, "include", basename+".h")
	result.Headers = []android.Path{this.generatedHdr}

	return result
}

func (this *CcAtomslogLibraryCallbacks) GeneratorFlags(ctx cc.ModuleContext, flags cc.Flags, deps cc.PathDeps) cc.Flags {
	return flags
}

func (this *CcAtomslogLibraryCallbacks) GeneratorBuildActions(ctx cc.ModuleContext, flags cc.Flags, deps cc.PathDeps) {
	cppNamespace := this.properties.Namespace
	if cppNamespace == "" {
		ctx.PropertyErrorf("namespace", "can't be empty")
	}

	atomsModule := this.properties.Atoms_module

	includeDir := filepath.Dir(this.generatedHdr.String())

	builder := android.NewRuleBuilder(pctx, ctx).SandboxDisabled()
	apiGenCmd := builder.Command().BuiltTool("stats-log-api-gen").
		FlagWithOutput("--cpp ", this.generatedCpp).
		FlagWithArg("--namespace ", cppNamespace).
		FlagWithArg("--importHeader ", this.generatedHdr.Base()).
		Flag("--omitExtraSrcs")
	if atomsModule != "" {
		apiGenCmd.FlagWithArg("--module ", atomsModule)
	}
	builder.Build("atomslog_cc_generation", "generate .cpp file")

	builder = android.NewRuleBuilder(pctx, ctx).SandboxDisabled()
	builder.Command().Text("rm -rf").Flag(includeDir)
	builder.Command().Text("mkdir -p").Flag(includeDir)
	apiGenCmd = builder.Command().BuiltTool("stats-log-api-gen").
		FlagWithOutput("--header ", this.generatedHdr).
		FlagWithArg("--namespace ", cppNamespace).
		Flag("--omitExtraSrcs")
	if atomsModule != "" {
		apiGenCmd.FlagWithArg("--module ", atomsModule)
	}
	builder.Build("atomslog_h_generation", "generate .h file")
}
