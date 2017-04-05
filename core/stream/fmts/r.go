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
	R_U8 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U8,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Red,
		}},
	}

	R_U8_NORM = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U8,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Red,
		}},
	}

	R_F16 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.F16,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Red,
		}},
	}

	R_U16_NORM = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U16,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Red,
		}},
	}

	R_S16_NORM = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.S16,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Red,
		}},
	}

	R_F32 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.F32,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Red,
		}},
	}
)
