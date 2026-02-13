// Copyright 2020 Google Inc. All rights reserved.
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
	"slices"

	"github.com/google/blueprint"
	"github.com/google/blueprint/depset"
)

func init() {
	RegisterParallelSingletonType("ninja_phony_manager", ninjaPhonyManagerFactory)
}

type CreateNinjaPhonyOnceContext interface {
	CreateNinjaPhonyOnce(name string, deps Paths) Path
}

// CreateNinjaPhonyOnce creates a ninja phony with the given name and the dependencies
// returned from getDeps. It will only create the phony once even if this function is called
// multiple times. Thus you have to ensure that the mapping
// from name -> deps remains consistent.
//
// This can be used to make the ninja file smaller, by creating aliases for many dependency files.
// Then many different modules can reuse the same alias.
//
// This function returns the PathForPhony(name). However, if deps is empty, this
// function will return nil instead, and you should not add the nil to any deps. This is because
// ninja phonies must have at least one dependency.
func (m *moduleContext) CreateNinjaPhonyOnce(name string, deps Paths) Path {
	if len(deps) == 0 {
		return nil
	}
	depsUnique := depset.New(depset.POSTORDER, deps, nil)
	if m.ninjaPhonies == nil {
		m.ninjaPhonies = make(map[string]NinjaPhoniesDepsInfo)
	} else if oldDeps, ok := m.ninjaPhonies[name]; ok {
		if !slices.Equal(Paths(oldDeps.Deps.ToList()).Strings(), deps.Strings()) {
			m.ModuleErrorf("CreateNinjaPhonyOnce called twice with a different list of deps for phony %q:\n  %s\nand:\n  %s\n",
				name, Paths(oldDeps.Deps.ToList()).Strings(), deps.Strings())
		}
		return PathForPhony(m, name)
	}
	m.ninjaPhonies[name] = NinjaPhoniesDepsInfo{depsUnique}
	return PathForPhony(m, name)
}

func ninjaPhonyManagerFactory() Singleton {
	return &ninjaPhonyManager{}
}

type ninjaPhonyManager struct{}

// We use a singleton to actually create the phonies, because in builds with
// AllowMissingDependencies enabled, a module missing dependencies will have all its ctx.Build()
// calls replaced with error rules. So a module with missing dependencies may create a phony,
// and then a module without missing dependencies tries to use it if we create the phonies
// directly in CreateNinjaPhonyOnce().
func (n *ninjaPhonyManager) GenerateBuildActions(ctx SingletonContext) {
	phonies := make(map[string]depset.DepSet[Path])

	ctx.VisitAllModuleProxies(func(proxy ModuleProxy) {
		commoninfo, ok := OtherModuleProvider(ctx, proxy, CommonModuleInfoProvider)
		if !ok {
			return
		}
		for name, deps := range commoninfo.NinjaPhonies {
			if oldDeps, ok := phonies[name]; ok {
				if !slices.Equal(Paths(oldDeps.ToList()).Strings(), Paths(deps.Deps.ToList()).Strings()) {
					ctx.Errorf("CreateNinjaPhonyOnce called twice with a different list of deps for phony %q:\n  %s\nand:\n  %s\n",
						name, Paths(oldDeps.ToList()).Strings(), Paths(deps.Deps.ToList()).Strings())
				}
				continue
			}
			phonies[name] = deps.Deps
		}
	})

	for _, phonyName := range SortedKeys(phonies) {
		deps := phonies[phonyName].ToList()
		if len(deps) > 0 {
			ctx.Build(pctx, BuildParams{
				Rule:   blueprint.Phony,
				Output: PathForPhony(ctx, phonyName),
				Inputs: deps,
			})
		}
	}
}
