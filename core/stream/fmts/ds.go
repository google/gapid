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
	DS_NU16U8 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U16,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Depth,
		}, {
			DataType: &stream.U8,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Stencil,
		}},
	}

	DS_NU16S8 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U16,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Depth,
		}, {
			DataType: &stream.S8,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Stencil,
		}},
	}

	DS_NU24U8 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U24,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Depth,
		}, {
			DataType: &stream.U8,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Stencil,
		}},
	}

	DS_NU24S8 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U24,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Depth,
		}, {
			DataType: &stream.S8,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Stencil,
		}},
	}

	DS_F32U8 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.F32,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Depth,
		}, {
			DataType: &stream.U8,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Stencil,
		}},
	}
)
