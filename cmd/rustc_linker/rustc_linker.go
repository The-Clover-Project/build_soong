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

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	androidClangBin, newClangArgs, err := rewriteArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}

	err = runCommand(androidClangBin, newClangArgs...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "running %q failed: %s\n", androidClangBin, err)
		os.Exit(-1)
	}
}

func rewriteArgs(oldClangArgs []string) (androidClangBin string, newClangArgs []string, err error) {
	var replacementVersionScript string

	const androidClangBinFlag = "--android-clang-bin="
	const androidVersionScriptFlag = "-Wl,--android-version-script="

	for _, arg := range oldClangArgs {
		if strings.HasPrefix(arg, androidClangBinFlag) {
			androidClangBin = strings.TrimPrefix(arg, androidClangBinFlag)
		} else if strings.HasPrefix(arg, androidVersionScriptFlag) {
			// Record and remove the custom android-version-script arg
			replacementVersionScript = strings.TrimPrefix(arg, androidVersionScriptFlag)
		} else if arg == "rsend.o" || arg == "rsbegin.o" {
			// Remove object files rustc emits for Windows target
			// We provide these as an rlib.
		} else {
			// Keep the arg
			newClangArgs = append(newClangArgs, arg)
		}
	}

	if androidClangBin == "" {
		err = fmt.Errorf("--android-clang-bin= argument is required")
		return
	}

	// Modify args
	if replacementVersionScript != "" {
		var versionScriptFound bool
		for i, arg := range newClangArgs {
			if strings.HasPrefix(arg, "-Wl,--version-script=") {
				newClangArgs[i] = "-Wl,--version-script=" + replacementVersionScript
				versionScriptFound = true
				break
			}
		}

		if !versionScriptFound {
			// If rustc did not emit a version script, just append the arg
			newClangArgs = append(newClangArgs, "-Wl,--version-script="+replacementVersionScript)
		}
	}

	return
}

// runCommand runs a command, connecting stdin, stdout and stderr to the corresponding descriptors in this process,
// and waiting for it to exit.
func runCommand(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
