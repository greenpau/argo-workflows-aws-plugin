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
	"testing"

	"github.com/google/go-cmp/cmp"
)

var errUnitTestGeneric GenericError = "this is generic error"
var errUnitTestDetailed GenericError = "this is detailed error: %v"

func TestNewGenericError(t *testing.T) {

	var testcases = []struct {
		name      string
		input     error
		shouldErr bool
		err       error
	}{
		{
			name:  "test generic error",
			input: errUnitTestGeneric,
			err:   fmt.Errorf("this is generic error"),
		},
		{
			name:  "test generic error with foo arg",
			input: errUnitTestDetailed.WithArgs("foo"),
			err:   fmt.Errorf("this is detailed error: foo"),
		},
		{
			name:  "test generic error with nil arg",
			input: errUnitTestDetailed.WithArgs(nil),
			err:   fmt.Errorf("this is detailed error: foo"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.input != nil {
				if diff := cmp.Diff(tc.err.Error(), tc.input.Error()); diff != "" {
					t.Fatalf("test name: %s, unexpected error (-want +got):\n%s", tc.name, diff)
				}
			}
		})
	}
}

func TestNewDetailedError(t *testing.T) {

	var testcases = []struct {
		name      string
		input     DetailedError
		shouldErr bool
		err       error
	}{
		{
			name: "test detailed error",
			input: DetailedError{
				err: fmt.Errorf("foo"),
				v:   []interface{}{"bar"},
			},
			err: DetailedError{
				err: fmt.Errorf("foo"),
				v:   []interface{}{"bar"},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if diff := cmp.Diff(tc.err.Error(), tc.input.Error()); diff != "" {
				t.Fatalf("test name: %s, unexpected error (-want +got):\n%s", tc.name, diff)
			}
			tc.input.Unwrap()
		})
	}
}
