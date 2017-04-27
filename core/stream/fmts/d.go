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
	D_U16_NORM = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U16,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Depth,
		}},
	}

	D_U24_NORM = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U24,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Depth,
		}},
	}

	D_F32 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.F32,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Depth,
		}},
	}

	ЖD_U8U24_NORM = &stream.Format{
		// According to Vulkan Spec, the first 8 bit is not used,
		// the following 24 bits are used for depth.
		Components: []*stream.Component{{
			DataType: &stream.U8,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Undefined,
		}, {
			DataType: &stream.U24,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Depth,
		}},
	}
)
