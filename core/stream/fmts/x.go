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

// Package fmts contains a number of standard stream formats.

package fmts

import "github.com/google/gapid/core/stream"

var (
	X_U8 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U8,
			Sampling: stream.Linear,
			Channel:  stream.Channel_X,
		}},
	}

	X_S8 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.S8,
			Sampling: stream.Linear,
			Channel:  stream.Channel_X,
		}},
	}

	X_U8_NORM = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U8,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_X,
		}},
	}

	X_S8_NORM = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.S8,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_X,
		}},
	}

	X_U16 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U16,
			Sampling: stream.Linear,
			Channel:  stream.Channel_X,
		}},
	}

	X_S16 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.S16,
			Sampling: stream.Linear,
			Channel:  stream.Channel_X,
		}},
	}

	X_U16_NORM = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U16,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_X,
		}},
	}

	X_S16_NORM = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.S16,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_X,
		}},
	}

	X_F16 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.F16,
			Sampling: stream.Linear,
			Channel:  stream.Channel_X,
		}},
	}

	X_U32 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U32,
			Sampling: stream.Linear,
			Channel:  stream.Channel_X,
		}},
	}

	X_S32 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.S32,
			Sampling: stream.Linear,
			Channel:  stream.Channel_X,
		}},
	}

	X_F32 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.F32,
			Sampling: stream.Linear,
			Channel:  stream.Channel_X,
		}},
	}
)
