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
)

type TestConfigZipper struct {
	testModulesTestSuiteInfo map[string]*android.TestSuiteInfo
	testExecutionPlans       map[string]*TestExecutionPlanProperties
	testSchedulingPlans      map[string]*TestSchedulingPlanProperties
	testWorkflows            map[string]*TestWorkflowProperties
	testTriggers             map[string]*TestTriggerInfo
}

func TestConfigZipperFactory() android.Singleton {
	singleton := &TestConfigZipper{
		testModulesTestSuiteInfo: make(map[string]*android.TestSuiteInfo),
		testExecutionPlans:       make(map[string]*TestExecutionPlanProperties),
		testSchedulingPlans:      make(map[string]*TestSchedulingPlanProperties),
		testWorkflows:            make(map[string]*TestWorkflowProperties),
		testTriggers:             make(map[string]*TestTriggerInfo),
	}
	return singleton
}

func (zipper *TestConfigZipper) GenerateBuildActions(ctx android.SingletonContext) {
	// Find and store information from all test configuration related modules.
	zipper.gatherRelatedModuleInfos(ctx)
	zipper.gatherInlinedModuleInfos(ctx)

	// Validate test suites.
	zipper.validateTestSuites(ctx)

	// Write the gathered information to a zipped file for downstream consumption.
	zipper.writeZip(ctx)
}

func (zipper *TestConfigZipper) gatherRelatedModuleInfos(ctx android.SingletonContext) {
	ctx.VisitAllModuleProxies(func(proxy android.ModuleProxy) {
		if testSuiteInfo, ok := android.OtherModuleProvider(ctx, proxy, android.TestSuiteInfoProvider); ok {
			zipper.testModulesTestSuiteInfo[proxy.Name()] = &testSuiteInfo
		}
		if testExecutionPlan, ok := android.OtherModuleProvider(ctx, proxy, TestExecutionPlanProvider); ok {
			zipper.testExecutionPlans[proxy.Name()] = &testExecutionPlan
		}
		if testSchedulingPlan, ok := android.OtherModuleProvider(ctx, proxy, TestSchedulingPlanProvider); ok {
			zipper.testSchedulingPlans[proxy.Name()] = &testSchedulingPlan
		}
		if testWorkflow, ok := android.OtherModuleProvider(ctx, proxy, TestWorkflowProvider); ok {
			zipper.testWorkflows[proxy.Name()] = &testWorkflow
		}
		if testTrigger, ok := android.OtherModuleProvider(ctx, proxy, TestTriggerProvider); ok {
			zipper.testTriggers[proxy.Name()] = &testTrigger
		}
	})
}

func (zipper *TestConfigZipper) gatherInlinedModuleInfos(ctx android.SingletonContext) {
	// Modules containing inlinables:
	//	* test_workflow
	// 	* test_triggers (scheduling_plan requires no validation yet)
	for name, workflow := range zipper.testWorkflows {
		if !workflow.Execution_plan.IsEmpty() {
			if _, found := zipper.testExecutionPlans[workflow.Execution_plan.Name]; found {
				ctx.Errorf("test_workflow \"%s\": contained inline execution_plan \"%s\" which already exists", name, workflow.Execution_plan.Name)
			} else {
				zipper.testExecutionPlans[workflow.Execution_plan.Name] = &workflow.Execution_plan.TestExecutionPlanProperties
			}

			zipper.testSchedulingPlans[workflow.Scheduling_plan.Name] = &workflow.Scheduling_plan.TestSchedulingPlanProperties
		}
	}

	for _, trigger := range zipper.testTriggers {
		if !trigger.TestTriggerInlineProperties.IsEmpty() {
			zipper.testSchedulingPlans[trigger.Scheduling_plan.Name] = &trigger.Scheduling_plan.TestSchedulingPlanProperties
		}
	}
}
