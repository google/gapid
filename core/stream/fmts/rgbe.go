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

package fmts

import "github.com/google/gapid/core/stream"

var (
	RGBE_U9U9U9U5 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U9,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.U9,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.U9,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Blue,
		}, {
			DataType: &stream.U5,
			Sampling: stream.Linear,
			Channel:  stream.Channel_SharedExponent,
		}},
	}
)
