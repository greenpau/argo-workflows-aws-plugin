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
)

// ExecutorPlugin defines plugin pattributes.
type ExecutorPlugin struct {
	Port         int
	Logger       *zap.Logger
	Mock         bool
	ClientConfig *rest.Config
	Client       *wfclientset.Clientset
}

// Configure parses cli arguments and configures the plugin.
func (ex *ExecutorPlugin) Configure(flags *pflag.FlagSet) error {
	if ex.Logger == nil {
		newLogger := func() *zap.Logger {
			debugFlag, err := flags.GetBool("debug")
			if err != nil {
				panic(err)
			}
			if debugFlag {
				return NewLogger(zapcore.DebugLevel)
			}
			return NewLogger(zapcore.InfoLevel)
		}
		ex.Logger = newLogger()
	}

	ex.Logger.Info("executing plugin",
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
		var reqErr error
		var execErr error
		defer func() {
			if reqErr != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			var requeue *metav1.Duration
			var nodeResult *wfv1.NodeResult
			if execErr == nil {
				// nodeResult = &wfv1.NodeResult{
				// 	Phase:   wfv1.NodeSucceeded,
				// 	Message: "success",
				// }
				nodeResult = &wfv1.NodeResult{
					Phase:   wfv1.NodeRunning,
					Message: "running",
				}
			} else {
				nodeResult = &wfv1.NodeResult{
					Phase:   wfv1.NodeError,
					Message: execErr.Error(),
				}
				requeue = &metav1.Duration{
					Duration: 60 * time.Second,
				}
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
			reqErr = ErrMalformedRequest.WithArgs("method is not POST")
			return
		}

		if header := req.Header.Get("Content-Type"); header != "application/json" {
			reqErr = ErrUnsupportedContentType
			return
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			reqErr = ErrRequestReaderError.WithArgs(err)
			return
		}

		ex.Logger.Debug("received template.execute payload",
			zap.Any("body", string(body)),
		)

		args := executor.ExecuteTemplateArgs{}
		if err = json.Unmarshal(body, &args); err != nil || args.Workflow == nil || args.Template == nil {
			reqErr = ErrRequestParserError.WithArgs(err)
			return
		}

		ns := args.Workflow.ObjectMeta.Namespace
		wfName := args.Workflow.ObjectMeta.Name

		ex.Logger.Debug("received template.execute arguments",
			zap.String("namespace", ns),
			zap.String("workflow_name", wfName),
			zap.Any("template", args.Template),
			zap.Any("workflow", args.Workflow),
		)
	}
}
