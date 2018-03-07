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

package jdwp

// VariableTable returns all of the variables that are present in the given
// Method.
func (c *Connection) VariableTable(classTy ReferenceTypeID, method MethodID) (VariableTable, error) {
	req := struct {
		Class  ReferenceTypeID
		Method MethodID
	}{classTy, method}
	var res VariableTable
	err := c.get(cmdMethodTypeVariableTable, req, &res)
	return res, err
}
