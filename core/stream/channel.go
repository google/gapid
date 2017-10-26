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

package stream

import (
	"fmt"
)

// Format prints the Channel to w.
func (c Channel) Format(w fmt.State, r rune) {
	switch c {
	case Channel_Red:
		fmt.Fprint(w, "R")
	case Channel_Green:
		fmt.Fprint(w, "G")
	case Channel_Blue:
		fmt.Fprint(w, "B")
	case Channel_Alpha:
		fmt.Fprint(w, "A")
	case Channel_Luminance:
		fmt.Fprint(w, "L")
	case Channel_Depth:
		fmt.Fprint(w, "D")
	case Channel_Stencil:
		fmt.Fprint(w, "S")
	case Channel_ChromaU:
		fmt.Fprint(w, "ChromaU")
	case Channel_ChromaV:
		fmt.Fprint(w, "ChromaV")
	case Channel_Gray:
		fmt.Fprint(w, "Gray")
	case Channel_U:
		fmt.Fprint(w, "U")
	case Channel_V:
		fmt.Fprint(w, "V")
	case Channel_W:
		fmt.Fprint(w, "W")
	case Channel_X:
		fmt.Fprint(w, "X")
	case Channel_Y:
		fmt.Fprint(w, "Y")
	case Channel_Z:
		fmt.Fprint(w, "Z")
	case Channel_SharedExponent:
		fmt.Fprint(w, "E")
	case Channel_Undefined:
		fmt.Fprint(w, "Ж")
	default:
		fmt.Fprint(w, "?")
	}
}

// ColorChannels is the list of channels considered colors.
var ColorChannels = Channels{
	Channel_Red,
	Channel_Green,
	Channel_Blue,
	Channel_Alpha,
	Channel_Luminance,
	Channel_Gray,
	Channel_ChromaU,
	Channel_ChromaV,
}

// DepthChannels is the list of channels considered depth.
var DepthChannels = Channels{
	Channel_Depth,
}

// StencilChannels is the list of channels considered stencil.
var StencilChannels = Channels{
	Channel_Stencil,
}

// VectorChannels is the list of channels considered vectors.
var VectorChannels = Channels{
	Channel_X,
	Channel_Y,
	Channel_Z,
	Channel_W,
}

// IsColor returns true if the channel is considered a color channel.
// See ColorChannels for the list of channels considered color.
func (c Channel) IsColor() bool {
	for _, t := range ColorChannels {
		if t == c {
			return true
		}
	}
	return false
}

// IsDepth returns true if the channel is considered a depth channel.
// See DepthChannels for the list of channels considered depth.
func (c Channel) IsDepth() bool {
	for _, t := range DepthChannels {
		if t == c {
			return true
		}
	}
	return false
}

// IsStencil returns true if the channel is considered a stencil channel.
// See StencilChannels for the list of channels considered stencil.
func (c Channel) IsStencil() bool {
	for _, t := range StencilChannels {
		if t == c {
			return true
		}
	}
	return false
}

// IsVector returns true if the channel is considered a vector channel.
// See VectorChannels for the list of channels considered vector.
func (c Channel) IsVector() bool {
	for _, t := range VectorChannels {
		if t == c {
			return true
		}
	}
	return false
}

// Channels is a list of channels.
type Channels []Channel

// Contains returns true if l contains c.
func (l Channels) Contains(c Channel) bool {
	for _, t := range l {
		if t == c {
			return true
		}
	}
	return false
}

// ContainsColor returns true if l contains a color channel.
// See ColorChannels for channels considered colors.
func (l Channels) ContainsColor() bool {
	for _, t := range l {
		if t.IsColor() {
			return true
		}
	}
	return false
}

// ContainsDepth returns true if l contains a depth channel.
// See DepthChannels for channels considered depth.
func (l Channels) ContainsDepth() bool {
	for _, t := range l {
		if t.IsDepth() {
			return true
		}
	}
	return false
}

// ContainsStencil returns true if l contains a stencil channel.
// See StencilChannels for channels considered stencil.
func (l Channels) ContainsStencil() bool {
	for _, t := range l {
		if t.IsStencil() {
			return true
		}
	}
	return false
}

// ContainsVector returns true if l contains a vector channel.
// See VectorChannels for channels considered vectors.
func (l Channels) ContainsVector() bool {
	for _, t := range l {
		if t.IsVector() {
			return true
		}
	}
	return false
}
