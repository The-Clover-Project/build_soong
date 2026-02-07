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
	"sync"

	"github.com/google/blueprint"
)

func init() {
	RegisterParallelSingletonType("ninja_phony_manager", ninjaPhonyManagerFactory)
}

var ninjaPhonyManagerOnceKey = NewOnceKey("ninja_phony_manager")

func getNinjaPhonyData(config Config) *ninjaPhonyData {
	return config.Once(ninjaPhonyManagerOnceKey, func() interface{} {
		return &ninjaPhonyData{
			m:        make(map[string]Paths),
			finished: false,
		}
	}).(*ninjaPhonyData)
}

// CreateNinjaPhonyOnce creates a ninja phony with the given name and the dependencies
// returned from getDeps. It will only create the phony once, and getDeps will only be called
// once, even if this function is called multiple times. Thus you have to ensure that the mapping
// from name -> deps remains consistent.
//
// This can be used to make the ninja file smaller, by creating aliases for many dependency files.
// Then many different modules can reuse the same alias.
//
// This function returns the PathForPhony(name). However, if getDeps() returns nothing, this
// function will return nil instead, and you should not add the nil to any deps. This is because
// ninja phonies must have at least one dependency.
func CreateNinjaPhonyOnce(ctx PathContext, name string, getDeps func() Paths) Path {
	data := getNinjaPhonyData(ctx.Config())

	phonyPath := PathForPhony(ctx, name)

	data.mLock.RLock()

	if toReturn, shouldReturn := func() (Path, bool) {
		defer data.mLock.RUnlock()

		if data.finished {
			panic("Cannot use CreateNinjaPhonyOnce after the phonies have been written out")
		}

		if deps, ok := data.m[name]; ok {
			if len(deps) > 0 {
				return phonyPath, true
			} else {
				return nil, true
			}
		}
		return nil, false
	}(); shouldReturn {
		return toReturn
	}

	data.mLock.Lock()
	defer data.mLock.Unlock()

	if deps, ok := data.m[name]; ok {
		if len(deps) > 0 {
			return phonyPath
		} else {
			return nil
		}
	}

	deps := getDeps()
	data.m[name] = deps

	if len(deps) > 0 {
		return phonyPath
	} else {
		return nil
	}
}

func ninjaPhonyManagerFactory() Singleton {
	return &ninjaPhonyManager{}
}

type ninjaPhonyManager struct{}

type ninjaPhonyData struct {
	mLock    sync.RWMutex
	m        map[string]Paths
	finished bool
}

// We use a singleton to actually create the phonies, because in builds with
// AllowMissingDependencies enabled, a module missing dependencies will have all its ctx.Build()
// calls replaced with error rules. So a module with missing dependencies may create a phony,
// and then a module without missing dependencies tries to use it if we create the phonies
// directly in CreateNinjaPhonyOnce().
func (n *ninjaPhonyManager) GenerateBuildActions(ctx SingletonContext) {
	data := getNinjaPhonyData(ctx.Config())

	data.mLock.Lock()
	defer data.mLock.Unlock()
	data.finished = true

	for _, phonyName := range SortedKeys(data.m) {
		deps := data.m[phonyName]
		if len(deps) > 0 {
			ctx.Build(pctx, BuildParams{
				Rule:   blueprint.Phony,
				Output: PathForPhony(ctx, phonyName),
				Inputs: deps,
			})
		}
	}
}
