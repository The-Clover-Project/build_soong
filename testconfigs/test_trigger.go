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

func (inline *TestTriggerInlineProperties) IsEmpty() bool {
	if len(inline.Tests) > 0 ||
		inline.Scheduling_plan.Name != "" ||
		!inline.Scheduling_plan.IsEmpty() {
		return false
	}
	return true
}

// @auto-generate: gob
type TestTriggerInfo struct {
	TestTriggerProperties

	modulePath string
}

func (info *TestTriggerInfo) Validate(ctx android.ModuleContext) {
	if info.TestTriggerInlineProperties.IsEmpty() {
		// Reference mode.
		for _, testWorkflow := range info.Test_workflows {
			if !ctx.OtherModuleExists(testWorkflow) {
				ctx.ModuleErrorf("failed to find referenced test_workflow %s", testWorkflow)
			}
		}
	} else {
		// Inline mode.
		info.TestTriggerInlineProperties.Scheduling_plan.Validate(ctx)
		testExecutionPlan := &TestExecutionPlanProperties{
			Tests: info.TestTriggerInlineProperties.Tests,
		}
		testExecutionPlan.Validate(ctx)
	}

	for _, filePattern := range info.File_patterns {
		locatedFiles := android.PathsForModuleSrc(ctx, []string{filePattern})
		if len(locatedFiles) == 0 {
			ctx.ModuleErrorf("test_trigger %s could not find any matches for file pattern '%s'", ctx.ModuleName(), filePattern)
		}
	}
}

var TestTriggerProvider = blueprint.NewProvider[TestTriggerInfo]()

func (trigger *TestTrigger) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	info := TestTriggerInfo{
		TestTriggerProperties: trigger.configProperties,
		modulePath:            ctx.ModuleDir(),
	}
	info.Validate(ctx)

	// Create provider for TestTrigger information.
	android.SetProvider(ctx, TestTriggerProvider, info)
}

func TestTriggerFactory() android.Module {
	module := &TestTrigger{}
	module.AddProperties(&module.configProperties)
	android.InitAndroidModule(module)
	return module
}
