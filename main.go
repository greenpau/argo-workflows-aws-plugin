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

	"github.com/greenpau/versioned"
	"github.com/spf13/cobra"
)

var (
	app        *versioned.PackageManager
	appVersion string
	gitBranch  string
	gitCommit  string
	buildUser  string
	buildDate  string
)

func init() {
	app = versioned.NewPackageManager("argo-workflows-aws-plugin")
	app.Description = "Argo Workflows Executor Plugin for AWS Services, e.g. SageMaker Pipelines, Glue, etc."
	app.Documentation = "https://github.com/greenpau/argo-workflows-aws-plugin/"
	app.SetVersion(appVersion, "1.0.7")
	app.SetGitBranch(gitBranch, "main")
	app.SetGitCommit(gitCommit, "v1.0.6-28-ga974fa3")
	app.SetBuildUser(buildUser, "")
	app.SetBuildDate(buildDate, "")
}

func main() {
	ex := &ExecutorPlugin{}
	cmd := BuildCommand(ex)
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}

// BuildCommand builds CLI command.
func BuildCommand(ex *ExecutorPlugin) *cobra.Command {
	usage := fmt.Sprintf("%s\n", app.Banner())
	usage += fmt.Sprintf("\n%s\n", app.Description)
	usage += fmt.Sprintf("\nDocumentation: %s\n\n", app.Documentation)
	examples := fmt.Sprintf("\n  %s --port 6789\n", app.Name)
	examples += fmt.Sprintf("  %s --debug", app.Name)
	cmd := &cobra.Command{
		Use:     app.Name,
		Long:    usage,
		RunE:    ex.Execute,
		Version: app.Version,
		Example: examples,
	}
	ConfigureFlags(cmd, ex)
	return cmd
}

// ConfigureFlags configures flags for CLI.
func ConfigureFlags(cmd *cobra.Command, ex *ExecutorPlugin) {
	flags := cmd.Flags()
	port := 7492
	if ex.Port > 0 {
		port = ex.Port
	}
	flags.IntVarP(&ex.Port, "port", "", port, "listening port of HTTP server")
	flags.Bool("debug", false, "enable debug level logging")
}
