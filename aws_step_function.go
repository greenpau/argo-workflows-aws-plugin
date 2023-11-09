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

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sfn"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CheckIfStepFunctionExists checks whether a particular SageMaker Pipelines instance exists.
func (ex *ExecutorPlugin) CheckIfStepFunctionExists(req *PluginRequest) *PluginResponse {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(req.RegionName),
	})
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to create aws session: %s", err),
			Status:         2,
		}
	}

	sf := sfn.New(sess)

	params := &sfn.DescribeStateMachineInput{
		StateMachineArn: &req.ResourceArn,
	}

	output, err := sf.DescribeStateMachine(params)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to describe aws step function: %s", err),
			Status:         2,
		}
	}

	b, err := json.Marshal(output)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to pack aws step function check response: %s", err),
			Status:         2,
		}
	}

	return &PluginResponse{
		Message: string(b),
		Status:  1,
	}
}

// StartStepFunctionExecution starts SageMaker Pipelines instance.
func (ex *ExecutorPlugin) StartStepFunctionExecution(req *PluginRequest, workflowID string) *PluginResponse {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(req.RegionName),
	})
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to create aws session: %s", err),
		}
	}

	sf := sfn.New(sess)

	params := &sfn.StartExecutionInput{
		StateMachineArn: &req.ResourceArn,
	}

	output, err := sf.StartExecution(params)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to start aws step function: %s", err),
			Status:         2,
		}
	}

	b, err := json.Marshal(output)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to pack aws step function start response: %s", err),
			Status:         2,
		}
	}

	executionArn := *output.ExecutionArn
	if executionArn == "" {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("aws step function start response has no execution ARN"),
			Status:         2,
		}
	}

	ex.Logger.Info("started aws step function execution",
		zap.String("plugin_name", app.Name),
		zap.String("execution_arn", executionArn),
	)

	ex.Workflows[workflowID] = &PluginWorkflow{
		ID: executionArn,
	}

	return &PluginResponse{
		Message:       string(b),
		ShouldRequeue: true,
		RequeueDuration: &metav1.Duration{
			Duration: 60 * time.Second,
		},
		Status: 3,
	}
}

// CheckStepFunctionExecution checks the status of SageMaker Pipelines execution.
func (ex *ExecutorPlugin) CheckStepFunctionExecution(req *PluginRequest, executionID string) *PluginResponse {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(req.RegionName),
	})
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to create aws session: %s", err),
			Status:         2,
		}
	}

	sf := sfn.New(sess)

	params := &sfn.DescribeExecutionInput{
		ExecutionArn: aws.String(executionID),
	}

	output, err := sf.DescribeExecution(params)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to describe aws step function execution: %s", err),
			Status:         2,
		}
	}

	b, err := json.Marshal(output)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to pack aws step function execution response: %s", err),
			Status:         2,
		}
	}

	ex.Logger.Info("checking aws step function execution status",
		zap.String("plugin_name", app.Name),
		zap.String("execution_arn", executionID),
		zap.String("execution_status", *output.Status),
	)

	// RUNNING | SUCCEEDED | FAILED | TIMED_OUT | ABORTED

	switch *output.Status {
	case "SUCCEEDED":
		return &PluginResponse{
			Message: string(b),
			Status:  1,
		}
	case "TIMED_OUT", "FAILED", "ABORTED":
		return &PluginResponse{
			Message: string(b),
			Status:  2,
		}
	default:
		// Covers Stopping and Executing
		return &PluginResponse{
			Message:       string(b),
			ShouldRequeue: true,
			RequeueDuration: &metav1.Duration{
				Duration: 60 * time.Second,
			},
			Status: 3,
		}
	}
}
