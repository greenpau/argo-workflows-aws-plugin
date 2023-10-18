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
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestNewLogger(t *testing.T) {

	var testcases = []struct {
		name      string
		input     zapcore.Level
		shouldErr bool
		err       error
	}{
		{
			name:  "test info log level",
			input: zapcore.InfoLevel,
		},
		{
			name:  "test debug log level",
			input: zapcore.DebugLevel,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			log := NewLogger(tc.input)
			log.Info("foo")
		})
	}
}
