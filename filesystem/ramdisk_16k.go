// Copyright (C) 2025 The Android Open Source Project
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

package filesystem

import (
	"android/soong/android"
	"android/soong/cc/config"
	_ "android/soong/cc/config"
	"android/soong/filesystem/ramdisk_16k/common"
)

type ramdisk16kImg struct {
	android.ModuleBase
	properties Ramdisk16kImgProperties
}

type Ramdisk16kImgProperties struct {
	// List or filegroup of prebuilt kernel module files. Should have .ko suffix.
	Srcs []string `android:"path,arch_variant"`

	// List or filegroup of prebuilt kernel module files that the debug symbols will be stripped.
	// Should have .ko suffix. These entries must be listed in srcs as well, otherwise an error
	// will be thrown. This is because ther order of the srcs is used for generating the
	// modules.load file if load property is not specified.
	Strip_symbol_srcs []string `android:"path,arch_variant"`

	// List specifying load order of kernel modules.
	Load []string

	// Path to the prebuilt 16KB kernel
	Kernel *string `android:"path"`
}

func (p *Ramdisk16kImgProperties) resolve(ctx android.ModuleContext) common.Ramdisk16kImgPropertiesJSON {
	return common.Ramdisk16kImgPropertiesJSON{
		Srcs:              android.PathsForModuleSrc(ctx, p.Srcs).Strings(),
		Strip_symbol_srcs: android.PathsForModuleSrc(ctx, p.Strip_symbol_srcs).Strings(),
		Load:              p.Load,
		Kernel:            p.Kernel,
	}
}

func Ramdisk16kImgFactory() android.Module {
	module := &ramdisk16kImg{}
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibFirst)
	module.AddProperties(&module.properties)
	return module
}

// Extracts version information from the kernel and packages the .ko modules in
// a version-specific subdirectory of the .img file.
func (p *ramdisk16kImg) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	if len(p.properties.Srcs) == 0 {
		return
	}
	outputDir := android.PathForModuleOut(ctx, "ramdisk_16k")
	output := outputDir.Join(ctx, "ramdisk_16k.img")
	intermediatesDir := outputDir.Join(ctx, "intermediates")

	propsFile := android.PathForModuleOut(ctx, "props.json")
	props := p.properties.resolve(ctx)
	android.WriteFileRule(ctx, propsFile, props.ToJSON())

	llvmStrip := config.ClangPath(ctx, "bin/llvm-strip")
	llvmLib := config.ClangPath(ctx, "lib/x86_64-unknown-linux-gnu/libc++.so")

	builder := android.NewRuleBuilder(pctx, ctx).SandboxDisabled().Sbox(
		outputDir,
		android.PathForModuleOut(ctx, "ramdisk_16k_intermediates.textproto"),
	)

	// Determine the kernel version during execution.
	builder.Command().
		BuiltTool("ramdisk_16k_builder").
		Flag("--extract_kernel").BuiltTool("extract_kernel").
		Flag("--depmod").BuiltTool("depmod").
		Flag("--llvm-strip").Input(llvmStrip).Implicit(llvmLib).
		Flag("--lz4").BuiltTool("lz4").
		Flag("--mkbootfs").BuiltTool("mkbootfs").
		Input(propsFile).
		Text(intermediatesDir.String()).
		Output(output).
		Implicits(android.PathsForModuleSrc(ctx, p.properties.Srcs)).
		Implicits(android.PathsForModuleSrc(ctx, p.properties.Strip_symbol_srcs))

	builder.Build("ramdisk_16k", "ramdisk_16k")

	ctx.ModulePhonyFiles(output)
	android.SetProvider(ctx, FilesystemProvider, FilesystemInfo{
		Output: output,
	})
	android.SetProvider(ctx, ramdiskFragmentInfoProvider, ramdiskFragmentInfo{
		Output:       output,
		Ramdisk_name: "16K",
	})
}
