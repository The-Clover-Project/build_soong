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

type TestTrigger struct {
	android.ModuleBase

	configProperties TestTriggerProperties
}

//go:generate go run ../../blueprint/gobtools/codegen

// @auto-generate: gob
type TestTriggerProperties struct {
	Imports        []string
	File_patterns  []string
	Test_workflows []string

	// If Test_workflows is unset, check for simple scheduling plan and included test use case.
	TestTriggerInlineProperties
}

// @auto-generate: gob
type TestTriggerInlineProperties struct {
	Scheduling_plan TestSchedulingPlanInlinable
	Tests           []ModuleProperties
}

// @auto-generate: gob
type TestTriggerInfo struct {
	TestTriggerProperties

	modulePath string
}

var TestTriggerProvider = blueprint.NewProvider[TestTriggerInfo]()

func (trigger *TestTrigger) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// Create provider for TestTrigger information.
	android.SetProvider(ctx, TestTriggerProvider, TestTriggerInfo{
		TestTriggerProperties: trigger.configProperties,
		modulePath:            ctx.ModuleDir(),
	})
}

func TestTriggerFactory() android.Module {
	module := &TestTrigger{}
	module.AddProperties(&module.configProperties)
	android.InitAndroidModule(module)
	return module
}
