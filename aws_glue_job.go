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
	"github.com/aws/aws-sdk-go/service/glue"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CheckIfGlueJobExists checks whether a particular AWS Glue job instance exists.
func (ex *ExecutorPlugin) CheckIfGlueJobExists(req *PluginRequest) *PluginResponse {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(req.RegionName),
	})
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to create aws session: %s", err),
			Status:         2,
		}
	}

	g := glue.New(sess)

	params := &glue.GetJobInput{
		JobName: &req.ResourceArn,
	}

	output, err := g.GetJob(params)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to describe aws glue job: %s", err),
			Status:         2,
		}
	}

	b, err := json.Marshal(output)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to pack aws glue job check response: %s", err),
			Status:         2,
		}
	}

	return &PluginResponse{
		Message: string(b),
		Status:  1,
	}
}

// StartGlueJobExecution starts AWS Glue job run.
func (ex *ExecutorPlugin) StartGlueJobExecution(req *PluginRequest, workflowID string) *PluginResponse {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(req.RegionName),
	})
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to create aws session: %s", err),
		}
	}

	g := glue.New(sess)

	params := &glue.StartJobRunInput{
		JobName: &req.ResourceArn,
	}

	output, err := g.StartJobRun(params)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to start aws glue job: %s", err),
			Status:         2,
		}
	}

	b, err := json.Marshal(output)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to pack aws glue job start response: %s", err),
			Status:         2,
		}
	}

	jobRunID := *output.JobRunId
	if jobRunID == "" {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("aws glue job start response has no job run id"),
			Status:         2,
		}
	}

	ex.Logger.Info("started aws glue job run",
		zap.String("plugin_name", app.Name),
		zap.String("job_run_id", jobRunID),
	)

	ex.Workflows[workflowID] = &PluginWorkflow{
		ID: jobRunID,
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

// CheckGlueJobExecution checks the status of AWS Glue job run.
func (ex *ExecutorPlugin) CheckGlueJobExecution(req *PluginRequest, jobRunID string) *PluginResponse {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(req.RegionName),
	})
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to create aws session: %s", err),
			Status:         2,
		}
	}

	g := glue.New(sess)

	params := &glue.GetJobRunInput{
		RunId:   aws.String(jobRunID),
		JobName: aws.String(req.JobName),
	}

	output, err := g.GetJobRun(params)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to get aws glue job run: %s", err),
			Status:         2,
		}
	}

	b, err := json.Marshal(output)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to pack aws glue job execution response: %s", err),
			Status:         2,
		}
	}

	ex.Logger.Info("checking aws glue job run",
		zap.String("plugin_name", app.Name),
		zap.String("job_run_id", jobRunID),
		zap.String("job_status", *output.JobRun.JobRunState),
	)

	// STARTING, RUNNING, STOPPING, STOPPED, SUCCEEDED, FAILED, ERROR, WAITING and TIMEOUT

	switch *output.JobRun.JobRunState {
	case "SUCCEEDED":
		return &PluginResponse{
			Message: string(b),
			Status:  1,
		}
	case "STOPPED", "FAILED", "ERROR", "TIMEOUT":
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
