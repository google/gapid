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

// GetArrayLength returns the length of the specified array.
func (c *Connection) GetArrayLength(id ArrayID) (int, error) {
	var res int
	err := c.get(cmdArrayReferenceLength, id, &res)
	return res, err
}

// GetArrayValues the values of the specified array.
func (c *Connection) GetArrayValues(id ArrayID, first, length int) ([]Value, error) {
	req := struct {
		ID     ArrayID
		First  int
		Length int
	}{id, first, length}
	var res []Value
	err := c.get(cmdArrayReferenceGetValues, req, &res)
	return res, err
}

// SetArrayValues the values of the specified array.
func (c *Connection) SetArrayValues(id ArrayID, first int, values interface{}) error {
	req := struct {
		ID     ArrayID
		First  int
		Values interface{}
	}{id, first, values}
	return c.get(cmdArrayReferenceSetValues, req, nil)
}
