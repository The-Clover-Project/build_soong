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
	"testing"

	"android/soong/android"
)

func TestAllArtlessDenylistDependency(t *testing.T) {
	t.Parallel()
	bp := `
		ndk_library {
			name: "libfoo",
			first_version: "29",
			symbol_file: "libfoo.map.txt",
		}

		cc_library {
			name: "libfoo",
		}

		cc_library {
		  name: "liblog",
		}

		all_artless_denylists {
			name: "all_denylists",
		}
	`

	ctx := prepareForCcTest.RunTestWithBp(t, bp)

	// Check if all_denylists depends on libfoo_denylist
	allDenylists := ctx.ModuleForTests(t, "all_denylists", "android_arm64_armv8-a_shared").Module()
	libfooDenylist := ctx.ModuleForTests(t, "libfoo_denylist", "android_arm64_armv8-a_static").Module()
	libcDenylist := ctx.ModuleForTests(t, "libc_denylist", "android_arm64_armv8-a_static").Module()
	libmDenylist := ctx.ModuleForTests(t, "libm_denylist", "android_arm64_armv8-a_static").Module()
	libdlDenylist := ctx.ModuleForTests(t, "libdl_denylist", "android_arm64_armv8-a_static").Module()

	android.AssertHasDirectDep(t, ctx, allDenylists, libfooDenylist)
	android.AssertHasDirectDep(t, ctx, allDenylists, libcDenylist)
	android.AssertHasDirectDep(t, ctx, allDenylists, libmDenylist)
	android.AssertHasDirectDep(t, ctx, allDenylists, libdlDenylist)
}
