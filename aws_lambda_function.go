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
	"github.com/aws/aws-sdk-go/service/lambda"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CheckIfLambdaFunctionExists checks whether a particular AWS Lambda Function instance exists.
func (ex *ExecutorPlugin) CheckIfLambdaFunctionExists(req *PluginRequest) *PluginResponse {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(req.RegionName),
	})
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to create aws session: %s", err),
			Status:         2,
		}
	}

	cli := lambda.New(sess)

	params := &lambda.GetFunctionInput{
		FunctionName: &req.ResourceArn,
	}

	output, err := cli.GetFunction(params)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to describe aws lambda function: %s", err),
			Status:         2,
		}
	}

	b, err := json.Marshal(output)
	if err != nil {
		return &PluginResponse{
			ExecutionError: fmt.Errorf("failed to pack aws lambda function check response: %s", err),
			Status:         2,
		}
	}

	return &PluginResponse{
		Message: string(b),
		Status:  1,
	}
}

// InvokeLambdaFunctionAsync invokes AWS Lambda function asynchroniously.
func InvokeLambdaFunctionAsync(ex *ExecutorPlugin, req *PluginRequest, wf *PluginWorkflow) {
	defer func() {
		if r := recover(); r != nil {
			err := r.(error)
			wf.Lock()
			wf.Status = "FAILED"
			wf.Message = err.Error()
			wf.Unlock()
		}
	}()

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(req.RegionName),
	})
	if err != nil {
		wf.Lock()
		wf.Status = "FAILED"
		wf.Message = fmt.Sprintf("failed to create aws session: %s", err)
		wf.Unlock()
		return
	}

	payload := []byte("{}")

	if req.Parameters != nil {
		payload, err = json.Marshal(req.Parameters)
		if err != nil {
			wf.Lock()
			wf.Status = "FAILED"
			wf.Message = fmt.Sprintf("failed to build aws lambda invocation payload: %s", err)
			wf.Unlock()
			return
		}
	}

	cli := lambda.New(sess)
	params := &lambda.InvokeInput{
		FunctionName:   &req.LambdaFunctionName,
		InvocationType: aws.String(lambda.InvocationTypeEvent),
		LogType:        aws.String(lambda.LogTypeNone),
		Payload:        payload,
	}

	output, err := cli.Invoke(params)
	if err != nil {
		wf.Lock()
		wf.Status = "FAILED"
		wf.Message = fmt.Sprintf("aws lambda invocation failed: %s", err)
		wf.Unlock()
		return
	}

	ex.Logger.Info("completed aws lambda invocation",
		zap.String("plugin_name", app.Name),
		zap.Int64("status_code", *output.StatusCode),
	)

	b, err := json.Marshal(output)
	if err != nil {
		wf.Lock()
		wf.Status = "FAILED"
		wf.Message = fmt.Sprintf("failed to pack aws lambda invocation response: %s", err)
		wf.Unlock()
		return
	}

	wf.Lock()
	wf.Status = "SUCCEEDED"
	wf.Message = string(b)
	wf.Unlock()
	return
}

// StartLambdaFunctionExecution starts AWS Lambda Function run.
func (ex *ExecutorPlugin) StartLambdaFunctionExecution(req *PluginRequest, workflowID string) *PluginResponse {
	wf := &PluginWorkflow{
		Status:  "RUNNING",
		Message: "running aws lambda function async execution",
	}
	ex.Workflows[workflowID] = wf

	go InvokeLambdaFunctionAsync(ex, req, wf)

	ex.Logger.Info("started aws lambda function async execution",
		zap.String("plugin_name", app.Name),
	)

	return &PluginResponse{
		Message:       "started aws lambda function async execution",
		ShouldRequeue: true,
		RequeueDuration: &metav1.Duration{
			Duration: 5 * time.Second,
		},
		Status: 3,
	}
}

// CheckLambdaFunctionExecution checks the status of AWS Glue job run.
func (ex *ExecutorPlugin) CheckLambdaFunctionExecution(req *PluginRequest, wf *PluginWorkflow) *PluginResponse {
	ex.Logger.Info("checking aws lambda function async execution",
		zap.String("plugin_name", app.Name),
	)

	wf.Lock()
	defer wf.Unlock()

	switch wf.Status {
	case "SUCCEEDED":
		return &PluginResponse{
			Message: wf.Message,
			Status:  1,
		}
	case "FAILED":
		return &PluginResponse{
			Message: wf.Message,
			Status:  2,
		}
	default:
		// RUNNING
		return &PluginResponse{
			Message:       wf.Message,
			ShouldRequeue: true,
			RequeueDuration: &metav1.Duration{
				Duration: 5 * time.Second,
			},
			Status: 3,
		}
	}
}
