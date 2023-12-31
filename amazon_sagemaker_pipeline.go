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
	"github.com/aws/aws-sdk-go/service/sagemaker"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CheckIfSageMakerPipelineExists checks whether a particular SageMaker Pipelines instance exists.
func (ex *ExecutorPlugin) CheckIfSageMakerPipelineExists(req *PluginRequest) *PluginResponse {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(req.RegionName),
	})
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to create aws session: %s", err),
			Status:         2,
		}
	}

	sm := sagemaker.New(sess)

	params := &sagemaker.DescribePipelineInput{
		PipelineName: &req.ResourceArn,
	}

	output, err := sm.DescribePipeline(params)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to describe amazon sagemaker pipeline: %s", err),
			Status:         2,
		}
	}

	b, err := json.Marshal(output)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to pack amazon sagemaker pipeline check response: %s", err),
			Status:         2,
		}
	}

	return &PluginResponse{
		Message: string(b),
		Status:  1,
	}
}

// StartSageMakerPipelineExecution starts SageMaker Pipelines instance.
func (ex *ExecutorPlugin) StartSageMakerPipelineExecution(req *PluginRequest, workflowID string) *PluginResponse {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(req.RegionName),
	})
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to create aws session: %s", err),
		}
	}

	sm := sagemaker.New(sess)

	params := &sagemaker.StartPipelineExecutionInput{
		PipelineName: &req.ResourceArn,
	}

	output, err := sm.StartPipelineExecution(params)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to start amazon sagemaker pipeline: %s", err),
			Status:         2,
		}
	}

	b, err := json.Marshal(output)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to pack amazon sagemaker pipeline start response: %s", err),
			Status:         2,
		}
	}

	executionArn := *output.PipelineExecutionArn
	if executionArn == "" {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("amazon sagemaker pipeline start response has no execution ARN"),
			Status:         2,
		}
	}

	ex.Logger.Info("started sagemaker pipeline instance",
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

// CheckSageMakerPipelineExecution checks the status of SageMaker Pipelines execution.
func (ex *ExecutorPlugin) CheckSageMakerPipelineExecution(req *PluginRequest, executionID string) *PluginResponse {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(req.RegionName),
	})
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to create aws session: %s", err),
			Status:         2,
		}
	}

	sm := sagemaker.New(sess)

	params := &sagemaker.DescribePipelineExecutionInput{
		PipelineExecutionArn: aws.String(executionID),
	}

	output, err := sm.DescribePipelineExecution(params)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to describe amazon sagemaker pipeline execution: %s", err),
			Status:         2,
		}
	}

	b, err := json.Marshal(output)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to pack amazon sagemaker pipeline execution response: %s", err),
			Status:         2,
		}
	}

	ex.Logger.Info("checking sagemaker pipeline instance",
		zap.String("plugin_name", app.Name),
		zap.String("execution_arn", executionID),
		zap.String("execution_status", *output.PipelineExecutionStatus),
	)

	switch *output.PipelineExecutionStatus {
	case "Succeeded":
		return &PluginResponse{
			Message: string(b),
			Status:  1,
		}
	case "Stopped", "Failed":
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
