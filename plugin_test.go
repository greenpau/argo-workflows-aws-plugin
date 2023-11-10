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
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	wfclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/client-go/rest"
)

type testHTTPRequest struct {
	id          string
	method      string
	path        string
	headers     map[string]string
	query       map[string]string
	data        map[string]interface{}
	contentType string
	token       string
}

func newTestPluginHTTPClient(t *testing.T) http.Client {
	cj, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("failed adding cookie jar: %v", err)
	}

	return http.Client{
		Jar:     cj,
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
		},
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			// Do not follow redirects.
			return http.ErrUseLastResponse
		},
	}
}

func newTestKubeHTTPClient(t *testing.T, ts *httptest.Server) http.Client {
	cert, err := x509.ParseCertificate(ts.TLS.Certificates[0].Certificate[0])
	if err != nil {
		t.Fatalf("failed extracting server certs: %v", err)
	}
	cp := x509.NewCertPool()
	cp.AddCert(cert)

	cj, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("failed adding cookie jar: %v", err)
	}

	return http.Client{
		Jar:     cj,
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 5 * time.Second,
			TLSClientConfig: &tls.Config{
				RootCAs: cp,
			},
		},
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			// Do not follow redirects.
			return http.ErrUseLastResponse
		},
	}
}

func newTestHTTPRequest(t *testing.T, testName string, url string, req *testHTTPRequest) *http.Request {
	var r *http.Request
	var err error

	switch req.method {
	case "GET":
		r, err = http.NewRequest(req.method, url+req.path, nil)
	case "POST":
		if req.data == nil {
			t.Fatalf("test name %s: error=POST has no data", testName)
		}
		jsonStr, err := json.Marshal(req.data)
		if err != nil {
			t.Fatal(err)
		}
		r, err = http.NewRequest(req.method, url+req.path, bytes.NewBuffer(jsonStr))
	default:
		t.Fatalf("test name %s: error=detected unsupported method, method=%s", testName, req.method)
	}

	if err != nil {
		t.Fatal(err)
	}

	if len(req.headers) > 0 {
		for k, v := range req.headers {
			r.Header.Add(k, v)
		}
	}

	if len(req.query) > 0 {
		q := r.URL.Query()
		for k, v := range req.query {
			q.Set(k, v)
		}
		r.URL.RawQuery = q.Encode()
	}
	return r
}

func TestExecutorPlugin(t *testing.T) {
	log := NewLogger(zapcore.DebugLevel)
	defer log.Sync()

	// Initialize mock Kubernetes HTTP server.
	kubeSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Info(
			"received request",
			zap.String("source", "mock k8s server"),
			zap.String("url", r.URL.String()),
			zap.String("method", r.Method),
		)
		resp := make(map[string]interface{})
		resp["response"] = "foo"
		b, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("failed to marshal %T: %v", resp, err)
		}
		fmt.Fprintln(w, string(b))
	}))
	defer kubeSrv.Close()

	u, err := url.ParseRequestURI(kubeSrv.URL)
	if err != nil {
		t.Fatal(err)
	}
	kubeHost := u.Hostname()
	kubePort := u.Port()
	log.Debug(
		"mock server started",
		zap.String("kube_url", kubeSrv.URL),
		zap.String("kube_host", kubeHost),
		zap.String("kube_port", kubePort),
	)

	os.Setenv("KUBERNETES_SERVICE_HOST", kubeHost)
	os.Setenv("KUBERNETES_SERVICE_PORT", kubePort)
	t.Cleanup(func() {
		os.Unsetenv("KUBERNETES_SERVICE_HOST")
		os.Unsetenv("KUBERNETES_SERVICE_PORT")
	})

	certpool := x509.NewCertPool()
	certpool.AddCert(kubeSrv.Certificate())
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: kubeSrv.Certificate().Raw,
	})

	rootTmpDirPath := os.TempDir() + "/unittest-" + app.Name
	if err := os.Mkdir(rootTmpDirPath, 0755); err != nil {
		if !os.IsExist(err) {
			t.Fatal(err)
		}
	}

	tmpDir, err := os.MkdirTemp(rootTmpDirPath, "test-"+app.Name+"-*")
	if err != nil {
		t.Fatal(err)
	}

	rootCAFilePath := tmpDir + "/ca.crt"
	if err := os.WriteFile(rootCAFilePath, certPEM, 0644); err != nil {
		t.Fatal(err)
	}

	t.Logf("wrote CA file to %s", rootCAFilePath)

	tokenFilePath := tmpDir + "/token"
	token := "foo"

	config := &rest.Config{
		Host: "https://" + net.JoinHostPort(kubeHost, kubePort),
		TLSClientConfig: rest.TLSClientConfig{
			CAFile: rootCAFilePath,
		},
		BearerToken:     string(token),
		BearerTokenFile: tokenFilePath,
	}

	client, err := wfclientset.NewForConfig(config)
	if err != nil {
		t.Fatal(err)
	}

	ex := &ExecutorPlugin{
		Port:         7493,
		Mock:         true,
		ClientConfig: config,
		Client:       client,
		DebugEnabled: true,
	}

	cmd := BuildCommand(ex)

	startPlugin := func() {
		if err := cmd.Execute(); err != nil {
			t.Error(err)
		}
	}

	go startPlugin()

	var testcases = []struct {
		name      string
		req       *testHTTPRequest
		path      string
		method    string
		data      map[string]interface{}
		shouldErr bool
		err       error
		want      map[string]interface{}
	}{
		{
			name: "test validate amazon sagemaker pipeline",
			req: &testHTTPRequest{
				method: "POST",
				headers: map[string]string{
					"Content-Type": "application/json",
				},
				path: "/api/v1/template.execute",
				data: map[string]interface{}{
					"workflow": map[string]interface{}{
						"metadata": map[string]interface{}{
							"name":      "sm-pipelines-r58tg",
							"namespace": "argo",
							"uid":       "27c01e7c-9d93-450f-a001-c64d649aac99",
						},
					},
					"template": map[string]interface{}{
						"name":     "validate_pipeline",
						"inputs":   map[string]interface{}{},
						"outputs":  map[string]interface{}{},
						"metadata": map[string]interface{}{},
						"plugin": map[string]interface{}{
							"awf-aws-plugin": map[string]interface{}{
								"account_id":    "100000000002",
								"action":        "validate",
								"service":       "amazon_sagemaker_pipelines",
								"pipeline_name": "MyPipeline",
								"region_name":   "us-west-2",
								"mock":          true,
								"mock_state":    "success",
							},
						},
					},
				},
			},
			want: map[string]interface{}{
				"content_type": "text/plain; charset=utf-8",
				"status_code":  200,
				"node": map[string]interface{}{
					"message": "success",
					"phase":   "Succeeded",
				},
			},
		},
		{
			name: "test execute amazon sagemaker pipeline",
			req: &testHTTPRequest{
				method: "POST",
				headers: map[string]string{
					"Content-Type": "application/json",
				},
				path: "/api/v1/template.execute",
				data: map[string]interface{}{
					"workflow": map[string]interface{}{
						"metadata": map[string]interface{}{
							"name":      "sm-pipelines-b863f",
							"namespace": "argo",
							"uid":       "1018894b-ede2-4b38-b258-e707e133b839",
						},
					},
					"template": map[string]interface{}{
						"name":     "execute_pipeline",
						"inputs":   map[string]interface{}{},
						"outputs":  map[string]interface{}{},
						"metadata": map[string]interface{}{},
						"plugin": map[string]interface{}{
							"awf-aws-plugin": map[string]interface{}{
								"account_id":    "100000000002",
								"action":        "execute",
								"service":       "amazon_sagemaker_pipelines",
								"pipeline_name": "MyPipeline",
								"region_name":   "us-west-2",
								"mock":          true,
								"mock_state":    "running",
							},
						},
					},
				},
			},
			want: map[string]interface{}{
				"content_type": "text/plain; charset=utf-8",
				"status_code":  200,
				"node": map[string]interface{}{
					"message": "running",
					"phase":   "Running",
				},
				"requeue": "1m0s",
			},
		},
		{
			name: "test validate aws glue job",
			req: &testHTTPRequest{
				method: "POST",
				headers: map[string]string{
					"Content-Type": "application/json",
				},
				path: "/api/v1/template.execute",
				data: map[string]interface{}{
					"workflow": map[string]interface{}{
						"metadata": map[string]interface{}{
							"name":      "aws-glue-job-t7c34",
							"namespace": "argo",
							"uid":       "ff4d1c6b-7c7d-46ad-b36e-18afd3b6d3cb",
						},
					},
					"template": map[string]interface{}{
						"name":     "validate_glue_job",
						"inputs":   map[string]interface{}{},
						"outputs":  map[string]interface{}{},
						"metadata": map[string]interface{}{},
						"plugin": map[string]interface{}{
							"awf-aws-plugin": map[string]interface{}{
								"account_id":  "100000000002",
								"action":      "validate",
								"service":     "aws_glue",
								"job_name":    "MyGlueJob",
								"region_name": "us-west-2",
								"mock":        true,
								"mock_state":  "success",
							},
						},
					},
				},
			},
			want: map[string]interface{}{
				"content_type": "text/plain; charset=utf-8",
				"status_code":  200,
				"node": map[string]interface{}{
					"message": "success",
					"phase":   "Succeeded",
				},
			},
		},
		{
			name: "test execute aws glue job",
			req: &testHTTPRequest{
				method: "POST",
				headers: map[string]string{
					"Content-Type": "application/json",
				},
				path: "/api/v1/template.execute",
				data: map[string]interface{}{
					"workflow": map[string]interface{}{
						"metadata": map[string]interface{}{
							"name":      "aws-glue-job-t7c34",
							"namespace": "argo",
							"uid":       "c4525afe-971d-491c-bc95-9624268119c3",
						},
					},
					"template": map[string]interface{}{
						"name":     "execute_glue_job",
						"inputs":   map[string]interface{}{},
						"outputs":  map[string]interface{}{},
						"metadata": map[string]interface{}{},
						"plugin": map[string]interface{}{
							"awf-aws-plugin": map[string]interface{}{
								"account_id":  "100000000002",
								"action":      "execute",
								"service":     "aws_glue",
								"job_name":    "MyGlueJob",
								"region_name": "us-west-2",
								"mock":        true,
								"mock_state":  "running",
							},
						},
					},
				},
			},
			want: map[string]interface{}{
				"content_type": "text/plain; charset=utf-8",
				"status_code":  200,
				"node": map[string]interface{}{
					"message": "running",
					"phase":   "Running",
				},
				"requeue": "1m0s",
			},
		},
		{
			name: "test validate aws step function",
			req: &testHTTPRequest{
				method: "POST",
				headers: map[string]string{
					"Content-Type": "application/json",
				},
				path: "/api/v1/template.execute",
				data: map[string]interface{}{
					"workflow": map[string]interface{}{
						"metadata": map[string]interface{}{
							"name":      "aws-step-function-w62b2",
							"namespace": "argo",
							"uid":       "b4fbe449-3b30-41ab-8b3d-0bdc727d3b64",
						},
					},
					"template": map[string]interface{}{
						"name":     "validate_step_function",
						"inputs":   map[string]interface{}{},
						"outputs":  map[string]interface{}{},
						"metadata": map[string]interface{}{},
						"plugin": map[string]interface{}{
							"awf-aws-plugin": map[string]interface{}{
								"account_id":         "100000000002",
								"action":             "validate",
								"service":            "aws_step_functions",
								"step_function_name": "MyStepFunction",
								"region_name":        "us-west-2",
								"mock":               true,
								"mock_state":         "success",
							},
						},
					},
				},
			},
			want: map[string]interface{}{
				"content_type": "text/plain; charset=utf-8",
				"status_code":  200,
				"node": map[string]interface{}{
					"message": "success",
					"phase":   "Succeeded",
				},
			},
		},
		{
			name: "test execute aws step function",
			req: &testHTTPRequest{
				method: "POST",
				headers: map[string]string{
					"Content-Type": "application/json",
				},
				path: "/api/v1/template.execute",
				data: map[string]interface{}{
					"workflow": map[string]interface{}{
						"metadata": map[string]interface{}{
							"name":      "aws-step-function-w62b2",
							"namespace": "argo",
							"uid":       "0c3eda84-d431-4b9e-8ef4-71f17d8678ee",
						},
					},
					"template": map[string]interface{}{
						"name":     "execute_step_function",
						"inputs":   map[string]interface{}{},
						"outputs":  map[string]interface{}{},
						"metadata": map[string]interface{}{},
						"plugin": map[string]interface{}{
							"awf-aws-plugin": map[string]interface{}{
								"account_id":         "100000000002",
								"action":             "execute",
								"service":            "aws_step_functions",
								"step_function_name": "MyStepFunction",
								"region_name":        "us-west-2",
								"mock":               true,
								"mock_state":         "running",
							},
						},
					},
				},
			},
			want: map[string]interface{}{
				"content_type": "text/plain; charset=utf-8",
				"status_code":  200,
				"node": map[string]interface{}{
					"message": "running",
					"phase":   "Running",
				},
				"requeue": "1m0s",
			},
		},
		{
			name: "test validate aws lambda function",
			req: &testHTTPRequest{
				method: "POST",
				headers: map[string]string{
					"Content-Type": "application/json",
				},
				path: "/api/v1/template.execute",
				data: map[string]interface{}{
					"workflow": map[string]interface{}{
						"metadata": map[string]interface{}{
							"name":      "aws-lambda-function-d42f4",
							"namespace": "argo",
							"uid":       "3d8bc315-e477-4215-8da4-18a4a649d30b",
						},
					},
					"template": map[string]interface{}{
						"name":     "validate_lambda_function",
						"inputs":   map[string]interface{}{},
						"outputs":  map[string]interface{}{},
						"metadata": map[string]interface{}{},
						"plugin": map[string]interface{}{
							"awf-aws-plugin": map[string]interface{}{
								"account_id":           "100000000002",
								"action":               "validate",
								"service":              "aws_lambda",
								"lambda_function_name": "MyLambdaFunction",
								"region_name":          "us-west-2",
								"mock":                 true,
								"mock_state":           "success",
							},
						},
					},
				},
			},
			want: map[string]interface{}{
				"content_type": "text/plain; charset=utf-8",
				"status_code":  200,
				"node": map[string]interface{}{
					"message": "success",
					"phase":   "Succeeded",
				},
			},
		},
		{
			name: "test execute aws lambda function",
			req: &testHTTPRequest{
				method: "POST",
				headers: map[string]string{
					"Content-Type": "application/json",
				},
				path: "/api/v1/template.execute",
				data: map[string]interface{}{
					"workflow": map[string]interface{}{
						"metadata": map[string]interface{}{
							"name":      "aws-lambda-function-d42f4",
							"namespace": "argo",
							"uid":       "dd5282c5-7703-49dc-b2a4-878ec2df2c1b",
						},
					},
					"template": map[string]interface{}{
						"name":     "execute_lambda_function",
						"inputs":   map[string]interface{}{},
						"outputs":  map[string]interface{}{},
						"metadata": map[string]interface{}{},
						"plugin": map[string]interface{}{
							"awf-aws-plugin": map[string]interface{}{
								"account_id":           "100000000002",
								"action":               "execute",
								"service":              "aws_lambda",
								"lambda_function_name": "MyLambdaFunction",
								"region_name":          "us-west-2",
								"mock":                 true,
								"mock_state":           "running",
							},
						},
					},
				},
			},
			want: map[string]interface{}{
				"content_type": "text/plain; charset=utf-8",
				"status_code":  200,
				"node": map[string]interface{}{
					"message": "running",
					"phase":   "Running",
				},
				"requeue": "1m0s",
			},
		},
		{
			name: "test GET healthz",
			req: &testHTTPRequest{
				method: "GET",
				path:   "/healthz",
			},
			want: map[string]interface{}{
				"content_type": "text/plain; charset=utf-8",
				"status_code":  float64(200),
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("test name %s: started", tc.name)
			got := make(map[string]interface{})
			// kubeClient := newTestKubeHTTPClient(t, kubeSrv)
			// req := newTestHTTPRequest(t, tc.name, kubeSrv.URL, tc.req)

			pluginClient := newTestPluginHTTPClient(t)
			pluginURL := fmt.Sprintf("http://localhost:%d", ex.Port)
			req := newTestHTTPRequest(t, tc.name, pluginURL, tc.req)
			resp, err := pluginClient.Do(req)
			if err != nil {
				t.Fatalf("test name %s: error=%v", tc.name, err)
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				t.Fatal(err)
			}

			got["status_code"] = resp.StatusCode
			if resp.Header.Get("Content-Type") != "" {
				got["content_type"] = resp.Header.Get("Content-Type")
			}

			switch {
			case bytes.HasPrefix(body, []byte(`{`)):
				var decodedResponse map[string]interface{}
				json.Unmarshal(body, &decodedResponse)
				for k, v := range decodedResponse {
					got[k] = v
				}
			default:
				t.Logf("test name %s: error=detected non-JSON body, body=%s", tc.name, body)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("test name: %s, unexpected error (-want +got):\n%s", tc.name, diff)
			}

			t.Logf("test name %s: finished", tc.name)
		})
	}
}
