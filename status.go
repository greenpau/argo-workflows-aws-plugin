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
	"fmt"
)

// PluginWorkflowStatus identifies the status of a workflow.
type PluginWorkflowStatus int

const (
	// UNKNOWN identifies unknown status.
	UNKNOWN PluginWorkflowStatus = iota
	// SUCCESS identifies successful status (1).
	SUCCESS
	// RUNNING identifies the workflow is still running (2).
	RUNNING
	// ERROR identifies failed workflow (3).
	ERROR
)

// String returns the description for IdentityProviderType enum.
func (m PluginWorkflowStatus) String() string {
	switch m {
	case SUCCESS:
		return "success"
	case RUNNING:
		return "running"
	case ERROR:
		return "error"
	}
	return fmt.Sprintf("PluginWorkflowStatus(%d)", int(m))
}
