// Copyright 2026 Google Inc. All rights reserved.
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

package cc

import (
	"strings"

	"github.com/google/blueprint"

	"android/soong/android"
)

func init() {
	RegisterBuildComponents(android.InitRegistrationContext)
}

func RegisterBuildComponents(ctx android.RegistrationContext) {
	ctx.RegisterModuleType("artless_denylist_stub", ArtlessDenylistFactory)
	ctx.RegisterModuleType("all_artless_denylists", AllArtlessDenylistsFactory)
}

var (
	// ndkStubGenerator is defined in ndk_library.go.
	genNativeStubSrc = pctx.AndroidStaticRule("genNativeStubSrc",
		blueprint.RuleParams{
			Command: "$ndkStubGenerator --arch $arch --api current " +
				"--api-map $apiMap --artless-denylist $flags $in $out",
			CommandDeps:     []string{"$ndkStubGenerator"},
			SandboxDisabled: true,
		}, "arch", "apiMap", "flags")
)

// Creates a stub static library that denies access to APIs incompatible with
// native-only processes, based on the provided version file.
//
// Example:
//
// artless_denylist_stub {
//
//	name: "libfoo_denylist",
//	symbol_file: "libfoo.map.txt",
//
// }
type artlessDenylistLibraryProperties struct {
	// Relative path to the symbol map.
	// See build/soong/docs/map_files.md.
	Symbol_file *string `android:"path"`
}

type artlessDenylistDecorator struct {
	*libraryDecorator

	properties artlessDenylistLibraryProperties

	versionScriptPath android.ModuleGenPath
}

func AddArtlessDenylistLibraryCompilerFlags(flags Flags) Flags {
	// All symbols in the stubs library should be visible.
	if inList("-fvisibility=hidden", flags.Local.CFlags) {
		flags.Local.CFlags = append(flags.Local.CFlags, "-fvisibility=default")
	}
	return flags
}

func (stub *artlessDenylistDecorator) compilerFlags(ctx ModuleContext, flags Flags, deps PathDeps) Flags {
	flags = stub.baseCompiler.compilerFlags(ctx, flags, deps)
	return AddArtlessDenylistLibraryCompilerFlags(flags)
}

func genNativeStubs(ctx android.ModuleContext, symbolFile string, genstubFlags string) NdkApiOutputs {
	stubSrcPath := android.PathForModuleGen(ctx, "stub.c")
	versionScriptPath := android.PathForModuleGen(ctx, "stub.map")
	symbolFilePath := android.PathForModuleSrc(ctx, symbolFile)
	symbolListPath := android.PathForModuleGen(ctx, "abi_symbol_list.txt")
	apiLevelsJson := android.GetApiLevelsJson(ctx)
	ctx.Build(pctx, android.BuildParams{
		Rule:        genNativeStubSrc,
		Description: "generate native-only denylist " + symbolFilePath.Rel(),
		Outputs: []android.WritablePath{stubSrcPath, versionScriptPath,
			symbolListPath},
		Input:     symbolFilePath,
		Implicits: []android.Path{apiLevelsJson},
		Args: map[string]string{
			"arch":   ctx.Arch().ArchType.String(),
			"apiMap": apiLevelsJson.String(),
			"flags":  genstubFlags,
		},
	})

	return NdkApiOutputs{
		StubSrc:       stubSrcPath,
		VersionScript: versionScriptPath,
		symbolList:    symbolListPath,
	}
}

func (c *artlessDenylistDecorator) compile(ctx ModuleContext, flags Flags, deps PathDeps) Objects {
	if !strings.HasSuffix(String(c.properties.Symbol_file), ".map.txt") {
		ctx.PropertyErrorf("symbol_file", "must end with .map.txt")
	}

	symbolFile := String(c.properties.Symbol_file)
	nativeAbiResult := genNativeStubs(ctx, symbolFile, "")
	objs := CompileStubLibrary(ctx, flags, nativeAbiResult.StubSrc, ctx.getSharedFlags())
	c.versionScriptPath = nativeAbiResult.VersionScript
	return objs
}

func (c *artlessDenylistDecorator) linkerDeps(ctx DepsContext, deps Deps) Deps {
	deps = c.libraryDecorator.linkerDeps(ctx, deps)
	deps.HeaderLibs = append(deps.HeaderLibs, "liblog_headers")
	return deps
}

// artless_denylist_stub creates a static library that redirects functions
// incompatible with native-only app processes to an aborting implementation.
func ArtlessDenylistFactory() android.Module {
	module, library := NewLibrary(android.DeviceSupported)
	library.BuildOnlyStatic()
	module.stl = nil
	module.sanitize = nil

	stub := &artlessDenylistDecorator{
		libraryDecorator: library,
	}
	module.compiler = stub
	module.linker = stub
	module.installer = stub
	module.library = stub

	module.AddProperties(&stub.properties)

	return module.Init()
}

type allArtlessDenylistsDecorator struct {
	*libraryDecorator
}

func (c *allArtlessDenylistsDecorator) linkerDeps(ctx DepsContext, deps Deps) Deps {
	deps = c.libraryDecorator.linkerDeps(ctx, deps)
	for _, lib := range *getNDKKnownLibs(ctx.Config()) {
		libName := strings.TrimSuffix(lib, ndkLibrarySuffix)
		deps.WholeStaticLibs = append(deps.WholeStaticLibs, libName+"_denylist")
	}
	deps.SharedLibs = append(deps.SharedLibs, "liblog")
	return deps
}

func (c *allArtlessDenylistsDecorator) linkerFlags(ctx ModuleContext, flags Flags) Flags {
	flags = c.libraryDecorator.linkerFlags(ctx, flags)
	flags.Local.LdFlags = append(flags.Local.LdFlags, "-Wl,-z,global")
	return flags
}

func AllArtlessDenylistsFactory() android.Module {
	module, library := NewLibrary(android.DeviceSupported)
	library.BuildOnlyShared()
	module.stl = nil
	module.sanitize = nil

	decorator := &allArtlessDenylistsDecorator{
		libraryDecorator: library,
	}
	module.compiler = decorator
	module.linker = decorator
	module.installer = decorator
	module.library = decorator

	return module.Init()
}
