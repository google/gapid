// Copyright (C) 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jdbg

import "fmt"

// methodSignature represents a method signature
type methodSignature struct {
	Parameters []Type
	Return     Type
}

// accepts returns true if the signature accepts the list of arguments.
func (j *JDbg) accepts(s methodSignature, args []interface{}) bool {
	if len(args) != len(s.Parameters) {
		return false
	}
	for i := range args {
		if !j.assignable(s.Parameters[i], args[i]) {
			return false
		}
	}
	return true
}

// parseMethodSignature returns the signature from the string str.
func (j *JDbg) parseMethodSignature(str string) (methodSignature, error) {
	s := methodSignature{}
	if str[0] != '(' {
		return methodSignature{}, fmt.Errorf("Method signature doesn't start with '('")
	}
	i := 1
	for str[i] != ')' {
		ty, err := j.parseSignature(str, &i)
		if err != nil {
			return methodSignature{}, err
		}
		s.Parameters = append(s.Parameters, ty)
	}
	i++
	ty, err := j.parseSignature(str, &i)
	if err != nil {
		return methodSignature{}, err
	}
	s.Return = ty
	return s, nil
}
