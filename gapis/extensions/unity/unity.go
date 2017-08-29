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

// Package unity provides GAPIS extensions that handle Unity engine features.
package unity

import (
	"github.com/google/gapid/gapis/extensions"
	"github.com/google/gapid/gapis/resolve/cmdgrouper"
)

func init() {
	extensions.Register(extensions.Extension{
		Name: "Unity",
		CmdGroupers: func() []cmdgrouper.Grouper {
			return []cmdgrouper.Grouper{
				&stateResetGrouper{},
			}
		},
	})
}
