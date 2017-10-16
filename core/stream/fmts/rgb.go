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
	RGB_U1 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U1,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.U1,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.U1,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Blue,
		}},
	}

	RGB_U1_NORM = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U1,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.U1,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.U1,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Blue,
		}},
	}

	RGB_U4 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U4,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.U4,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.U4,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Blue,
		}},
	}

	RGB_U4_NORM = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U4,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.U4,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.U4,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Blue,
		}},
	}

	RGB_U5_NORM = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U5,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.U5,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.U5,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Blue,
		}},
	}

	RGB_U5U6U5_NORM = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U5,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.U6,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.U5,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Blue,
		}},
	}

	RGB_U8 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U8,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.U8,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.U8,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Blue,
		}},
	}

	RGB_S8 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.S8,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.S8,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.S8,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Blue,
		}},
	}

	RGB_U8_NORM = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U8,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.U8,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.U8,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Blue,
		}},
	}

	RGB_S8_NORM = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.S8,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.S8,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.S8,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Blue,
		}},
	}

	SRGB_U8_NORM = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U8,
			Sampling: stream.SRGBNormalized,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.U8,
			Sampling: stream.SRGBNormalized,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.U8,
			Sampling: stream.SRGBNormalized,
			Channel:  stream.Channel_Blue,
		}},
	}

	RGB_U16 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U16,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.U16,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.U16,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Blue,
		}},
	}

	RGB_S16 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.S16,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.S16,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.S16,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Blue,
		}},
	}

	RGB_U16_NORM = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U16,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.U16,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.U16,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Blue,
		}},
	}

	RGB_S16_NORM = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.S16,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.S16,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.S16,
			Sampling: stream.LinearNormalized,
			Channel:  stream.Channel_Blue,
		}},
	}

	RGB_F16 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.F16,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.F16,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.F16,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Blue,
		}},
	}

	RGB_U32 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U32,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.U32,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.U32,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Blue,
		}},
	}

	RGB_S32 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.S32,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.S32,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.S32,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Blue,
		}},
	}

	RGB_F32 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.F32,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.F32,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.F32,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Blue,
		}},
	}

	RGB_U64 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.U64,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.U64,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.U64,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Blue,
		}},
	}

	RGB_S64 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.S64,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.S64,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.S64,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Blue,
		}},
	}

	RGB_F64 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.F64,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.F64,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.F64,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Blue,
		}},
	}

	RGB_F11F11F10 = &stream.Format{
		Components: []*stream.Component{{
			DataType: &stream.F11,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Red,
		}, {
			DataType: &stream.F11,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Green,
		}, {
			DataType: &stream.F10,
			Sampling: stream.Linear,
			Channel:  stream.Channel_Blue,
		}},
	}
)
