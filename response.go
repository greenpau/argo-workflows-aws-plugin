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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// PluginResponse contains plugin response.
type PluginResponse struct {
	Message         string               `json:"message,omitempty" xml:"message,omitempty" yaml:"message,omitempty"`
	Status          PluginWorkflowStatus `json:"status,omitempty" xml:"status,omitempty" yaml:"status,omitempty"`
	ShouldRequeue   bool                 `json:"should_requeue,omitempty" xml:"should_requeue,omitempty" yaml:"should_requeue,omitempty"`
	RequeueDuration *metav1.Duration     `json:"requeue_duration,omitempty" xml:"requeue_duration,omitempty" yaml:"requeue_duration,omitempty"`
	RequestError    error                `json:"req_error,omitempty" xml:"req_error,omitempty" yaml:"req_error,omitempty"`
	ExecutionError  error                `json:"exec_error,omitempty" xml:"exec_error,omitempty" yaml:"exec_error,omitempty"`
}
