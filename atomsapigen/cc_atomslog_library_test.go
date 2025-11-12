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
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func DirectDepsList(ctx *android.TestResult, module android.Module) []string {
	deps := []string{}
	ctx.VisitDirectDeps(module, func(dep android.Module) {
		deps = append(deps, dep.Name())
	})
	return deps
}

func TestMain(m *testing.M) {
	m.Run()
}

// Common Android.bp content needed for all tests
const commonBp = `
	cc_library_static {
		name: "stats-log-api-gen-cc-lib",
		// srcs: ["dummy.cpp"], // Not strictly necessary for dependency resolution
	}

	cc_library_shared {
		name: "libstatssocket",
	}

	cc_library_shared {
		name: "libstatspull",
	}
`

// Prepare the test environment for cc_atomslog_library
func testCcAtomslogLibraryFixturePreparers(t *testing.T) android.FixturePreparer {
	t.Helper()
	return android.GroupFixturePreparers(
		cc.PrepareForTestWithCcDefaultModules,
		android.FixtureRegisterWithContext(func(ctx android.RegistrationContext) {
			ctx.RegisterModuleType("cc_atomslog_library", CcAtomslogLibraryFactory)
			ctx.RegisterModuleType("cc_atomslog_library_static", CcAtomslogLibraryStaticFactory)
			ctx.RegisterModuleType("cc_atomslog_library_shared", CcAtomslogLibrarySharedFactory)
		}))
}

// Test valid cases
func TestCcAtomslogLibrary_VerifyCodeGen(t *testing.T) {
	testCases := []struct {
		name             string
		optionalParams   string
		expectedBasename string
	}{
		{
			name: "basic",
			optionalParams: `
				basename: "my_atoms_out",
			`,
			expectedBasename: "my_atoms_out",
		},
		{
			name:             "default basename",
			optionalParams:   "", // no basename specified.
			expectedBasename: "statslog_myatoms",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			bp := fmt.Sprintf(`
				cc_atomslog_library {
					name: "libmyatoms_test",
					atoms_module: "myatoms",
					namespace: "test::namespace",
					%s
				}
			`, tt.optionalParams)

			result := testCcAtomslogLibraryFixturePreparers(t).RunTestWithBp(t, commonBp+bp)
			module := result.ModuleForTests(t, "libmyatoms_test", "android_arm64_armv8-a_static")
			cppRule := module.Rule("atomslog_cc_generation")
			hdrRule := module.Rule("atomslog_h_generation")

			outputPaths := []android.Path{
				cppRule.Output,
				hdrRule.Output,
			}

			expectedRelPaths := []string{
				tt.expectedBasename + ".cpp",
				"include/" + tt.expectedBasename + ".h",
			}

			android.AssertPathsEndWith(t, "atomslog_generation_paths", expectedRelPaths, outputPaths)

			expectedCppCmd := fmt.Sprintf(
				"out/host/linux-x86/bin/stats-log-api-gen --cpp %s --namespace test::namespace --importHeader %s --omitExtraSrcs --module myatoms",
				cppRule.Output.String(), hdrRule.Output.Base())
			cppCmd, _, _ := strings.Cut(cppRule.RuleParams.Command, " # ") // Remove the hash comment
			android.AssertStringEquals(t, "wrong .cpp gen command", expectedCppCmd, cppCmd)

			includePath := filepath.Dir(hdrRule.Output.String())
			expectedHdrCmd := fmt.Sprintf(
				"rm -rf %s && mkdir -p %s && out/host/linux-x86/bin/stats-log-api-gen --header %s --namespace test::namespace --omitExtraSrcs --module myatoms",
				includePath, includePath, hdrRule.Output.String())
			hdrCmd, _, _ := strings.Cut(hdrRule.RuleParams.Command, " # ") // Remove the hash comment
			android.AssertStringEquals(t, "wrong .h gen command", expectedHdrCmd, hdrCmd)
		})
	}
}

// Test that lib dependencies are added
func TestCcAtomslogLibrary_VerifyDeps(t *testing.T) {
	bp := `
		cc_atomslog_library {
			name: "mystatslog",
			atoms_module: "myatoms",
			namespace: "test::namespace",
		}
	`
	result := testCcAtomslogLibraryFixturePreparers(t).RunTestWithBp(t, commonBp+bp)
	module := result.ModuleForTests(t, "mystatslog", "android_arm64_armv8-a_static")
	rule := module.Rule("arWithLibs")
	android.EnsureListContainsSuffix(t, rule.Inputs.Strings(), "stats-log-api-gen-cc-lib.a")

	deps := DirectDepsList(result, module.Module())
	android.AssertStringListContains(t, "missing libstatssocket", deps, "libstatssocket")
	android.AssertStringListContains(t, "missing libstatspull", deps, "libstatspull")
}

func TestCcAtomslogLibrary_NoAtomsModule(t *testing.T) {
	bp := `
		cc_atomslog_library {
			name: "mystatslog",
			basename: "mystatslog",
			namespace: "test::namespace",
		}
	`
	result := testCcAtomslogLibraryFixturePreparers(t).RunTestWithBp(t, commonBp+bp)
	module := result.ModuleForTests(t, "mystatslog", "android_arm64_armv8-a_static")
	cppRule := module.Rule("atomslog_cc_generation")
	hdrRule := module.Rule("atomslog_h_generation")

	android.AssertStringDoesNotContain(t, "no module param", cppRule.RuleParams.Command, "--module")
	android.AssertStringDoesNotContain(t, "no module param", hdrRule.RuleParams.Command, "--module")
}

func TestCcAtomslogLibrary_VerifyExcludeDefaultSharedLibs(t *testing.T) {
	bp := `
		cc_atomslog_library {
			name: "mystatslog",
			atoms_module: "myatoms",
			namespace: "test::namespace",
			include_default_shared_libs: false,
		}
	`
	result := testCcAtomslogLibraryFixturePreparers(t).RunTestWithBp(t, commonBp+bp)
	module := result.ModuleForTests(t, "mystatslog", "android_arm64_armv8-a_static")
	deps := DirectDepsList(result, module.Module())
	android.AssertStringListDoesNotContain(t, "unexpected libstatssocket", deps, "libstatssocket")
	android.AssertStringListDoesNotContain(t, "unexpected libstatspull", deps, "libstatspull")
}

func TestCcAtomslogLibrary_VerifyStaticCannotBeLinkedAsShared(t *testing.T) {
	bp := `
		cc_atomslog_library_static {
			name: "mystatslog",
			atoms_module: "myatoms",
			namespace: "test::namespace",
		}

		cc_library_static {
			name: "myclientlib",
			shared_libs: ["mystatslog"],
		}
	`
	testCcAtomslogLibraryFixturePreparers(t).
		ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(
			"dependency \"mystatslog\" of \"myclientlib\" missing variant")).
		RunTestWithBp(t, commonBp+bp)
}

func TestCcAtomslogLibrary_VerifySharedCannotBeLinkedAsStatic(t *testing.T) {
	bp := `
		cc_atomslog_library_shared {
			name: "mystatslog",
			atoms_module: "myatoms",
			namespace: "test::namespace",
		}

		cc_library_static {
			name: "myclientlib",
			static_libs: ["mystatslog"],
		}
	`
	testCcAtomslogLibraryFixturePreparers(t).
		ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(
			"dependency \"mystatslog\" of \"myclientlib\" missing variant")).
		RunTestWithBp(t, commonBp+bp)
}

func TestCcAtomslogLibrary_MissingAtomsModuleAndBasename(t *testing.T) {
	bp := `
		cc_atomslog_library_shared {
			name: "mystatslog",
			namespace: "test::namespace",
		}
	`
	testCcAtomslogLibraryFixturePreparers(t).
		ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(
			"At least one of atoms_module or basename must be provided")).
		RunTestWithBp(t, commonBp+bp)
}

func TestCcAtomslogLibrary_MissingNamespace(t *testing.T) {
	bp := `
		cc_atomslog_library_shared {
			name: "mystatslog",
			atoms_module: "myatoms",
		}
	`
	testCcAtomslogLibraryFixturePreparers(t).
		ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(
			"namespace: can't be empty")).
		RunTestWithBp(t, commonBp+bp)
}
