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

package compiler

import (
	"fmt"

	"github.com/google/gapid/gapis/api"
)

func (c ErrorCode) Err() error {
	switch c {
	case ErrSuccess:
		return nil
	case ErrAborted:
		return api.ErrCmdAborted{}
	default:
		return fmt.Errorf("Unknown error code %v", c)
	}
}
