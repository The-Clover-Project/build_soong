package main

import (
	"android/soong/kernel/common"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var (
	soongZip         = flag.String("soong_zip", "", "path to soong_zip executable")
	zipSync          = flag.String("zipsync", "", "path to zipsync executable")
	mergeZips        = flag.String("merge_zips", "", "path to merge_zips executable")
	depmod           = flag.String("depmod", "", "path to depmod executable")
	llvmStrip        = flag.String("llvm-strip", "", "path to llvm-strip executable")
	outDir           string
	installPartition string
)

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: kernel_modules_builder <props.json> <install partition> <temp dir> <out load file> <out zip>")
		flag.PrintDefaults()
	}

	flag.Parse()

	if len(flag.Args()) != 5 {
		flag.Usage()
		os.Exit(1)
	}

	propsFile := flag.Arg(0)
	installPartition = flag.Arg(1)
	outDir = flag.Arg(2)
	outLoadFile := flag.Arg(3)
	outInstallZip := flag.Arg(4)

	must(os.RemoveAll(outDir))
	must(os.MkdirAll(outDir, 0o777))

	var props common.PrebuiltKernelModulesPropertiesJSON
	must(json.Unmarshal(must2(os.ReadFile(propsFile)), &props))

	modules := props.Srcs

	var loadFile = filepath.Join(outDir, "modules.load")
	createLoadFile(&props, loadFile)

	modulesZip := filepath.Join(outDir, "modules.zip")
	args := []string{"-o", modulesZip, "-j"}
	for _, mod := range modules {
		args = append(args, "-f", mod)
	}
	cmd(*soongZip, args...)

	zipsToMerge := []string{modulesZip}
	if len(props.Srcs_16k) > 0 {
		zip := filepath.Join(outDir, "modules_16k.zip")
		args := []string{"-o", zip, "-j", "-P", "16k-mode"}
		for _, mod := range props.Srcs_16k {
			args = append(args, "-f", mod)
		}
		cmd(*soongZip, args...)

		zipsToMerge = append(zipsToMerge, zip)
	}

	runDepmod(modulesZip, loadFile, String(props.System_dep), &zipsToMerge)
	installBlocklistFile(&props, &zipsToMerge)
	installOptionsFile(&props, &zipsToMerge)

	allInstallsZip := filepath.Join(outDir, "installs.zip")
	cmd(*mergeZips, slices.Concat([]string{allInstallsZip}, zipsToMerge)...)

	if BoolDefault(props.Strip_debug_symbols, true) {
		allInstallsZip = stripDebugSymbols(allInstallsZip)
	}

	// Move to output locations
	must(os.Rename(loadFile, outLoadFile))
	must(os.Rename(allInstallsZip, outInstallZip))
}

func runDepmod(modulesZip string, loadFile string, systemModulesZip string, zipsToMerge *[]string) {
	baseDir := filepath.Join(outDir, "depmod_tmp")
	fakeVer := "0.0" // depmod requires this
	modulesDir := filepath.Join(baseDir, "lib", "modules", fakeVer)
	modulesCpDir := modulesDirForAndroidDlkm(modulesDir, false)
	must(os.MkdirAll(modulesCpDir, 0o777))

	cmd(*zipSync, "-d", modulesCpDir, modulesZip)

	if systemModulesZip != "" {
		modulesDirForSystemDlkm := modulesDirForAndroidDlkm(modulesDir, true)
		must(os.MkdirAll(modulesDirForSystemDlkm, 0o777))
		cmd(*zipSync, "-d", modulesDirForSystemDlkm, systemModulesZip)

		// https://source.corp.google.com/h/googleplex-android/platform/build/+/71d79d0a58e112f76ee2c2dfdebd331971145b4c:core/Makefile;l=480-487;bpv=1;bpt=0;drc=567ee7be9833ea96a65e36fb21d4bd783ff74f1c
		// When there is a duplicate module present in both directories, we want modules in PRIVATE_MODULES to take
		// precedence. Since depmod does not provide any guarantee about ordering of
		// dependency resolution, we achieve this by maually removing any duplicate
		// modules with lower priority.
		for _, dirEnt := range must2(os.ReadDir(modulesCpDir)) {
			if strings.HasSuffix(dirEnt.Name(), ".ko") {
				err := os.Remove(filepath.Join(modulesDirForSystemDlkm, dirEnt.Name()))
				if !errors.Is(err, os.ErrNotExist) {
					must(err)
				}
			}
		}
	}

	modulesLoad := filepath.Join(modulesDir, "modules.load")
	cp(loadFile, modulesLoad)

	cmd(*depmod, "-b", baseDir, fakeVer)

	depFile := filepath.Join(modulesDir, "modules.dep")

	// Add a leading slash to paths in modules.dep of android dlkm and vendor ramdisk
	switch installPartition {
	case "system_dlkm", "vendor_dlkm", "odm_dlkm", "vendor_ramdisk", "vendor_kernel_ramdisk":
		r := regexp.MustCompile(`[^:\s]*lib/modules/[^:\s]*`)
		contents := must2(os.ReadFile(depFile))
		contents = r.ReplaceAll(contents, []byte("/$0"))
		must(os.WriteFile(depFile, contents, 0o777))
	}

	installZip := filepath.Join(baseDir, "depmod_installs.zip")
	cmd(*soongZip, "-o", installZip, "-C", modulesDir,
		"-f", depFile,
		"-f", filepath.Join(modulesDir, "modules.softdep"),
		"-f", filepath.Join(modulesDir, "modules.alias"))

	*zipsToMerge = append(*zipsToMerge, installZip)
}

// This is the path in soong intermediates where the .ko files will be copied.
// The layout should match the layout on device so that depmod can create meaningful modules.* files.
func modulesDirForAndroidDlkm(modulesDir string, system bool) string {
	if installPartition == "system_dlkm" || system {
		// The first component can be either system or system_dlkm
		// system works because /system/lib/modules is a symlink to /system_dlkm/lib/modules.
		// system was chosen to match the contents of the kati built modules.dep
		return filepath.Join(modulesDir, "system", "lib", "modules")
	} else if installPartition == "vendor_dlkm" {
		return filepath.Join(modulesDir, "vendor", "lib", "modules")
	} else if installPartition == "odm_dlkm" {
		return filepath.Join(modulesDir, "odm", "lib", "modules")
	} else if installPartition == "vendor_ramdisk" || installPartition == "vendor_kernel_ramdisk" {
		return filepath.Join(modulesDir, "lib", "modules")
	} else {
		// not an android dlkm module.
		return modulesDir
	}
}

func createLoadFile(props *common.PrebuiltKernelModulesPropertiesJSON, loadFile string) {
	if !BoolDefault(props.Load_by_default, true) {
		must(os.WriteFile(loadFile, nil, 0o777))
	} else if len(props.Src_filenames_to_load) > 0 {
		validNames := make(map[string]bool, len(props.Srcs))
		for _, src := range props.Srcs {
			validNames[filepath.Base(src)] = true
		}
		var contents strings.Builder
		for _, filenameToLoad := range props.Src_filenames_to_load {
			if _, ok := validNames[filenameToLoad]; !ok {
				fmt.Printf("%s in src_filenames_to_load not present in srcs\n", filenameToLoad)
				os.Exit(1)
			}
			contents.WriteString(filenameToLoad)
			contents.WriteString("\n")
		}
		must(os.WriteFile(loadFile, []byte(contents.String()), 0o777))
	} else {
		var contents strings.Builder
		for _, filenameToLoad := range props.Srcs {
			contents.WriteString(filepath.Base(filenameToLoad))
			contents.WriteString("\n")
		}
		must(os.WriteFile(loadFile, []byte(contents.String()), 0o777))
	}
}

func stripDebugSymbols(modulesZip string) string {
	output := filepath.Join(outDir, "stripped.zip")
	tmpDir := filepath.Join(outDir, "stripped")
	cmd(*zipSync, "-d", tmpDir, modulesZip)

	for _, dirEnt := range must2(os.ReadDir(tmpDir)) {
		if strings.HasSuffix(dirEnt.Name(), ".ko") {
			f := filepath.Join(tmpDir, dirEnt.Name())
			cmd(*llvmStrip, f, "--strip-debug", f)
		}
	}

	cmd(*soongZip, "-o", output, "-C", tmpDir, "-D", tmpDir)

	return output
}

// Install the blocklist file, removing empty lines and raising an error if a line is
// not formatted as `blocklist $name.ko`
func installBlocklistFile(props *common.PrebuiltKernelModulesPropertiesJSON, zipsToMerge *[]string) {
	if String(props.Blocklist_file) == "" {
		return
	}

	lines := strings.Split(string(must2(os.ReadFile(*props.Blocklist_file))), "\n")
	var contents strings.Builder
	failed := false
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "#") {
			fields := strings.Fields(line)
			if len(fields) != 2 || fields[0] != "blocklist" {
				fmt.Printf("Invalid blocklist line %d: %q\n", i+1, line)
				failed = true
			}
			line = strings.Join(fields, " ")
		}
		contents.WriteString(line)
		contents.WriteString("\n")
	}
	if failed {
		os.Exit(1)
	}

	out := filepath.Join(outDir, "modules.blocklist")
	must(os.WriteFile(out, []byte(contents.String()), 0o777))
	installZip := filepath.Join(outDir, "blocklist_install.zip")
	cmd(*soongZip, "-o", installZip, "-e", "modules.blocklist", "-f", out)

	*zipsToMerge = append(*zipsToMerge, installZip)
}

// Install the optinos file, removing empty lines and raising an error if a line is
// not formatted as `options $name.ko`
func installOptionsFile(props *common.PrebuiltKernelModulesPropertiesJSON, zipsToMerge *[]string) {
	if String(props.Options_file) == "" {
		return
	}

	lines := strings.Split(string(must2(os.ReadFile(*props.Options_file))), "\n")
	var contents strings.Builder
	failed := false
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "#") {
			fields := strings.Fields(line)
			if len(fields) < 2 || fields[0] != "options" {
				fmt.Printf("Invalid options line %d: %q\n", i+1, line)
				failed = true
			}
			line = strings.Join(fields, " ")
		}
		contents.WriteString(line)
		contents.WriteString("\n")
	}
	if failed {
		os.Exit(1)
	}

	out := filepath.Join(outDir, "modules.options")
	must(os.WriteFile(out, []byte(contents.String()), 0o777))
	installZip := filepath.Join(outDir, "options_install.zip")
	cmd(*soongZip, "-o", installZip, "-e", "modules.options", "-f", out)

	*zipsToMerge = append(*zipsToMerge, installZip)
}

func cp(from string, to string) {
	cmd("cp", from, to)
}

func cmd(tool string, args ...string) {
	cmd := exec.Command(tool, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	must(cmd.Run())
}

func must(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func must2[T any](x T, err error) T {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	return x
}

func String(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func BoolDefault(b *bool, def bool) bool {
	if b == nil {
		return def
	}
	return *b
}
