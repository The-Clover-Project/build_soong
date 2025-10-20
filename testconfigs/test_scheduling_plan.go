// Copyright 2025 The Android Open Source Project
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testconfigs

import (
	"android/soong/android"

	"github.com/google/blueprint"
)

type TestSchedulingPlan struct {
	android.ModuleBase

	configProperties TestSchedulingPlanProperties
}

//go:generate go run ../../blueprint/gobtools/codegen

// @auto-generate: gob
type TestSchedulingPlanProperties struct{}

func (plan *TestSchedulingPlanProperties) Validate(ctx android.ModuleContext) {}

func (plan *TestSchedulingPlanProperties) IsEmpty() bool {
	return true
}

// @auto-generate: gob
type TestSchedulingPlanInlinable struct {
	TestSchedulingPlanProperties
	Name string
}

var TestSchedulingPlanProvider = blueprint.NewProvider[TestSchedulingPlanProperties]()

func (plan *TestSchedulingPlan) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	plan.configProperties.Validate(ctx)

	// Create provider for TestSchedulingPlan information.
	android.SetProvider(ctx, TestSchedulingPlanProvider, plan.configProperties)
}
func TestSchedulingPlanFactory() android.Module {
	module := &TestSchedulingPlan{}
	module.AddProperties(&module.configProperties)
	android.InitAndroidModule(module)
	return module
}
