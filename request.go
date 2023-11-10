// Copyright 2023 Paul Greenberg greenpau@outlook.com
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

import "fmt"

var (
	allowedServiceNames = map[string]bool{
		"amazon_sagemaker_pipelines": true,
		"aws_glue":                   true,
		"aws_step_functions":         true,
		"aws_lambda":                 true,
	}
	allowedMockStates = map[string]bool{
		"running": true,
		"success": true,
	}
	allowedActions = map[string]bool{
		"validate": true,
		"execute":  true,
	}
)

// PluginRequest represent Plugin input arguments.
type PluginRequest struct {
	Kind               string                 `json:"kind,omitempty" xml:"kind,omitempty" yaml:"kind,omitempty"`
	AccountID          string                 `json:"account_id,omitempty" xml:"account_id,omitempty" yaml:"account_id,omitempty"`
	ServiceName        string                 `json:"service,omitempty" xml:"service,omitempty" yaml:"service,omitempty"`
	Action             string                 `json:"action,omitempty" xml:"action,omitempty" yaml:"action,omitempty"`
	PipelineName       string                 `json:"pipeline_name,omitempty" xml:"pipeline_name,omitempty" yaml:"pipeline_name,omitempty"`
	JobName            string                 `json:"job_name,omitempty" xml:"job_name,omitempty" yaml:"job_name,omitempty"`
	StepFunctionName   string                 `json:"step_function_name,omitempty" xml:"step_function_name,omitempty" yaml:"step_function_name,omitempty"`
	LambdaFunctionName string                 `json:"lambda_function_name,omitempty" xml:"lambda_function_name,omitempty" yaml:"lambda_function_name,omitempty"`
	Parameters         map[string]interface{} `json:"parameters,omitempty" xml:"parameters,omitempty" yaml:"parameters,omitempty"`
	ResourceArn        string                 `json:"resource_arn,omitempty" xml:"resource_arn,omitempty" yaml:"resource_arn,omitempty"`
	RegionName         string                 `json:"region_name,omitempty" xml:"region_name,omitempty" yaml:"region_name,omitempty"`
	Mock               bool                   `json:"mock,omitempty" xml:"mock,omitempty" yaml:"mock,omitempty"`
	MockState          string                 `json:"mock_state,omitempty" xml:"mock_state,omitempty" yaml:"mock_state,omitempty"`
}

// Validate validates Plugin input arguments.
func (req *PluginRequest) Validate() error {
	if req.AccountID == "" {
		return fmt.Errorf("account_id is empty")
	}
	if req.ServiceName == "" {
		return fmt.Errorf("service is empty")
	}
	if req.Action == "" {
		return fmt.Errorf("action is empty")
	}
	if req.RegionName == "" {
		return fmt.Errorf("region name is empty")
	}

	if _, exists := allowedServiceNames[req.ServiceName]; !exists {
		return fmt.Errorf("service '%s' is not supported", req.ServiceName)
	}

	if _, exists := allowedActions[req.Action]; !exists {
		return fmt.Errorf("action '%s' is not supported", req.Action)
	}

	switch req.ServiceName {
	case "amazon_sagemaker_pipelines":
		if req.PipelineName == "" {
			return fmt.Errorf("pipeline_name is empty")
		}
		req.ResourceArn = fmt.Sprintf("arn:aws:sagemaker:%s:%s:pipeline/%s", req.RegionName, req.AccountID, req.PipelineName)
	case "aws_glue":
		if req.JobName == "" {
			return fmt.Errorf("job_name is empty")
		}
		req.ResourceArn = fmt.Sprintf("arn:aws:glue:%s:%s:job/%s", req.RegionName, req.AccountID, req.JobName)
	case "aws_step_functions":
		if req.StepFunctionName == "" {
			return fmt.Errorf("step_function_name is empty")
		}
		req.ResourceArn = fmt.Sprintf("arn:aws:states:%s:%s:stateMachine:%s", req.RegionName, req.AccountID, req.StepFunctionName)
	case "aws_lambda":
		if req.LambdaFunctionName == "" {
			return fmt.Errorf("lambda_function_name is empty")
		}
		req.ResourceArn = fmt.Sprintf("arn:aws:lambda:%s:%s:function:%s", req.RegionName, req.AccountID, req.LambdaFunctionName)
	}

	if req.Mock {
		if req.MockState == "" {
			return fmt.Errorf("mock state is empty")
		}
		if _, exists := allowedMockStates[req.MockState]; !exists {
			return fmt.Errorf("mock state '%s' is not supported", req.MockState)
		}
	}
	return nil
}
