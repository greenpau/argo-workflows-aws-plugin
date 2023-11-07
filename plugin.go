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
	"io"
	"net/http"
	"time"

	wfv1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v3/pkg/plugins/executor"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

var (
	// ErrMalformedRequest indicates that the request is malformed.
	ErrMalformedRequest GenericError = "malformed request: %s"
	// ErrUnsupportedContentType indicates that the request's Content-Type header is not supported.
	ErrUnsupportedContentType GenericError = "content type header value is unsupported"
	// ErrRequestReaderError indicates that the plugin failed to read POST request's body.
	ErrRequestReaderError GenericError = "failed to read request body: %v"
	// ErrRequestParserError indicates that the plugin failed to parse the body of the request.
	ErrRequestParserError GenericError = "failed to parse request body: %v"
	// ErrRequestInputMalformedError indicates that the request to the plugin is malformed.
	ErrRequestInputMalformedError GenericError = "malformed plugin input: %v"
	// ErrExecutionError indicates that the execution of the plugin failed.
	ErrExecutionError GenericError = "failed execution: %v"
)

// ExecutorPlugin defines plugin pattributes.
type ExecutorPlugin struct {
	Port         int
	Logger       *zap.Logger
	Mock         bool
	ClientConfig *rest.Config
	Client       *wfclientset.Clientset
	DebugEnabled bool
	Workflows    map[string]*PluginWorkflow
}

// Configure parses cli arguments and configures the plugin.
func (ex *ExecutorPlugin) Configure(flags *pflag.FlagSet) error {
	if ex.Logger == nil {
		newLogger := func() *zap.Logger {
			debugFlag, err := flags.GetBool("debug")
			if err != nil {
				panic(err)
			}
			if debugFlag || ex.DebugEnabled {
				return NewLogger(zapcore.DebugLevel)
			}
			return NewLogger(zapcore.InfoLevel)
		}
		ex.Logger = newLogger()
	}

	ex.Logger.Info("configuring plugin",
		zap.String("plugin_name", app.Name),
		zap.Int("port", ex.Port),
		zap.String("log_level", ex.Logger.Level().CapitalString()),
	)

	if ex.ClientConfig == nil {
		config, err := rest.InClusterConfig()
		if err != nil {
			return err
		}
		ex.ClientConfig = config
	}

	if ex.Client == nil {
		client, err := wfclientset.NewForConfig(ex.ClientConfig)
		if err != nil {
			return err
		}
		ex.Client = client
	}

	if ex.Workflows == nil {
		ex.Workflows = make(map[string]*PluginWorkflow)
	}
	return nil
}

// Execute executes the plugin.
func (ex *ExecutorPlugin) Execute(c *cobra.Command, args []string) (err error) {
	if err := ex.Configure(c.Flags()); err != nil {
		return err
	}
	defer ex.Logger.Sync()
	http.HandleFunc("/api/v1/template.execute", handleTemplateExecute(ex))
	http.HandleFunc("/healthz", handleHealthCheck(ex))
	err = http.ListenAndServe(fmt.Sprintf(":%d", ex.Port), nil)
	return
}

func handleHealthCheck(ex *ExecutorPlugin) func(w http.ResponseWriter, req *http.Request) {
	ex.Logger.Debug("registered healthcheck handler")

	return func(w http.ResponseWriter, req *http.Request) {
		ex.Logger.Debug("received healthcheck request")
		resp := make(map[string]interface{})
		resp["status_code"] = int(http.StatusOK)
		b, err := json.Marshal(resp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, string(b))
		return
	}
}

func handleTemplateExecute(ex *ExecutorPlugin) func(w http.ResponseWriter, req *http.Request) {
	ex.Logger.Debug("registered template.execute handler")

	return func(w http.ResponseWriter, req *http.Request) {
		ex.Logger.Debug("received template.execute request")
		resp := &PluginResponse{}
		defer func() {
			if resp.RequestError != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			var requeue *metav1.Duration
			var nodeResult *wfv1.NodeResult
			var phase wfv1.NodePhase
			if resp.ExecutionError == nil {
				if resp.ShouldRequeue {
					if resp.Message == "" {
						resp.Message = "running"
					}
					phase = wfv1.NodeRunning
					if requeue == nil {
						requeue = &metav1.Duration{
							Duration: 60 * time.Second,
						}
					}
				} else {
					if resp.Message == "" {
						resp.Message = "success"
					}
					phase = wfv1.NodeSucceeded
				}
			} else {
				if resp.Message == "" {
					resp.Message = resp.ExecutionError.Error()
				}
				phase = wfv1.NodeError
			}

			nodeResult = &wfv1.NodeResult{
				Phase:   phase,
				Message: resp.Message,
			}

			jsonResp, jsonErr := json.Marshal(executor.ExecuteTemplateReply{
				Node:    nodeResult,
				Requeue: requeue,
			})
			if jsonErr != nil {
				ex.Logger.Warn("failed to build JSON response", zap.Error(jsonErr))
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			} else {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(jsonResp)
			}
		}()

		if req.Method != http.MethodPost {
			resp.RequestError = ErrMalformedRequest.WithArgs("method is not POST")
			return
		}

		if header := req.Header.Get("Content-Type"); header != "application/json" {
			resp.RequestError = ErrUnsupportedContentType
			return
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			resp.RequestError = ErrRequestReaderError.WithArgs(err)
			return
		}

		ex.Logger.Debug("received template.execute payload",
			zap.Any("body", string(body)),
		)

		args := executor.ExecuteTemplateArgs{}
		if err = json.Unmarshal(body, &args); err != nil || args.Workflow == nil || args.Template == nil {
			resp.RequestError = ErrRequestParserError.WithArgs(err)
			return
		}

		ns := args.Workflow.ObjectMeta.Namespace
		wfName := args.Workflow.ObjectMeta.Name
		wfID := args.Workflow.ObjectMeta.Uid

		ex.Logger.Debug("received template.execute arguments",
			zap.String("namespace", ns),
			zap.String("workflow_name", wfName),
			zap.String("workflow_id", wfID),
		)

		pluginInputJSON, err := args.Template.Plugin.MarshalJSON()
		if err != nil {
			ex.Logger.Error("encountered error during marshaling of plugin request body", zap.Error(err))
			resp.RequestError = ErrRequestParserError.WithArgs(err)
			return
		}

		pluginInputBody := make(map[string]*PluginRequest)
		if err := json.Unmarshal(pluginInputJSON, &pluginInputBody); err != nil {
			ex.Logger.Error("encountered error during unmarshaling of plugin request", zap.Error(err))
			resp.RequestError = ErrRequestParserError.WithArgs(err)
			return
		}

		pluginInput, pluginInputFound := pluginInputBody["awf-aws-plugin"]
		if !pluginInputFound {
			ex.Logger.Error("plugin input not found")
			resp.RequestError = ErrRequestInputMalformedError.WithArgs("plugin input not found")
			return
		}

		if err := pluginInput.Validate(); err != nil {
			ex.Logger.Error("encountered error during validation of plugin request", zap.Error(err))
			resp.RequestError = ErrRequestInputMalformedError.WithArgs(err)
			return
		}

		ex.Logger.Debug("plugin input arguments",
			zap.String("action", pluginInput.Action),
			zap.String("service", pluginInput.ServiceName),
			zap.String("resource_arn", pluginInput.ResourceArn),
		)

		if pluginInput.Mock {
			switch pluginInput.MockState {
			case "success":
				return
			case "running":
				resp.ShouldRequeue = true
				return
			case "error":
				resp.ExecutionError = ErrExecutionError.WithArgs("expected mock error")
				return
			}
		}

		if pluginInput.ServiceName == "amazon_sagemaker_pipelines" {
			switch pluginInput.Action {
			case "validate":
				resp = ex.CheckIfSageMakerPipelineExists(pluginInput)
				return
			case "execute":
				pluginWorkflow, exists := ex.Workflows[wfID]
				if exists {
					resp = ex.CheckSageMakerPipelineExecution(pluginInput, pluginWorkflow.ID)
					return
				}
				resp = ex.StartSageMakerPipelineExecution(pluginInput)
				return
			}
		}
	}
}
