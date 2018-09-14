// Copyright (C) 2018 Google Inc.
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

package compiler

func (c *C) buildExec() {
	for _, api := range c.APIs {
		c.currentAPI = api
		for _, f := range api.Externs {
			c.extern(f)
		}
		for _, f := range api.Subroutines {
			c.subroutine(f)
		}
		for _, f := range api.Functions {
			c.command(f)
		}
	}
}
