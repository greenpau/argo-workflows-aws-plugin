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
	"errors"
	"fmt"
)

// GenericError is a standard error.
type GenericError string

func (e GenericError) Error() string {
	return string(e)
}

// WithArgs accepts errors with parameters.
func (e GenericError) WithArgs(v ...interface{}) error {
	var hasErr, hasNil bool
	for _, vv := range v {
		switch err := vv.(type) {
		case error:
			if err == nil {
				return nil
			}
			hasErr = true
		case nil:
			hasNil = true
		}
	}

	if hasNil && !hasErr {
		return nil
	}

	err := DetailedError{
		err: fmt.Errorf("%w", e),
		v:   v,
	}

	return err
}

// DetailedError is an error with parameters.
type DetailedError struct {
	err error
	v   []interface{}
}

// Error returns error string.
func (e DetailedError) Error() string {
	return fmt.Sprintf(e.err.Error(), e.v...)
}

// Unwrap returns unwrapped error.
func (e DetailedError) Unwrap() error {
	return errors.Unwrap(e.err)
}
