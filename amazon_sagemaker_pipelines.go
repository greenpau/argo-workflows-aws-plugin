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
		}
	}

	b, err := json.Marshal(output)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to pack amazon sagemaker pipeline response: %s", err),
		}
	}
	return &PluginResponse{
		Message: string(b),
	}
}

// StartSageMakerPipelineExecution starts SageMaker Pipelines instance.
func (ex *ExecutorPlugin) StartSageMakerPipelineExecution(req *PluginRequest) *PluginResponse {
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
		}
	}

	b, err := json.Marshal(output)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to pack amazon sagemaker pipeline response: %s", err),
		}
	}

	return &PluginResponse{
		Message:       string(b),
		ShouldRequeue: true,
		RequeueDuration: &metav1.Duration{
			Duration: 60 * time.Second,
		},
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
		}
	}

	b, err := json.Marshal(output)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to pack amazon sagemaker pipeline response: %s", err),
		}
	}

	switch output.PipelineExecutionStatus {
	case aws.String("InProgress"), aws.String("Stopping"):
		return &PluginResponse{
			Message:       string(b),
			ShouldRequeue: true,
			RequeueDuration: &metav1.Duration{
				Duration: 60 * time.Second,
			},
			Status: 2,
		}
	case aws.String("Succeeded"):
		return &PluginResponse{
			Message: string(b),
			Status:  1,
		}
	default:
		return &PluginResponse{
			Message: string(b),
			Status:  3,
		}
	}
}
