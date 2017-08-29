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
	"bytes"
	"fmt"
	"strings"
)

var aliases = map[string]string{
	"RGBA_N_sRGBU8N_sRGBU8N_sRGBU8NU8": "SRGBA_U8_NORM",
	"RGB_U8_NORM_sRGB":                 "SRGB_U8_NORM",
}

// Format prints the Format to w.
func (f Format) Format(w fmt.State, r rune) {
	buf := &bytes.Buffer{}
	samplings := map[Sampling]struct{}{}
	datatypes := map[DataType]struct{}{}
	for _, c := range f.Components {
		fmt.Fprint(buf, c.Channel)
		datatypes[*c.DataType] = struct{}{}
		if s := c.Sampling; s != nil {
			samplings[*s] = struct{}{}
		}
	}
	fmt.Fprint(buf, "_")

	datatypesCommon := len(datatypes) < 2
	samplingsCommon := len(samplings) < 2
	defaultSampling := Sampling{}

	if !datatypesCommon || !samplingsCommon {
		for _, c := range f.Components {
			if !samplingsCommon && *c.Sampling != defaultSampling {
				fmt.Fprintf(buf, "%c", c.Sampling)
			}
			fmt.Fprint(buf, c.DataType)
		}
		if samplingsCommon {
			for sampling := range samplings {
				if sampling != defaultSampling {
					fmt.Fprint(buf, "_", sampling)
				}
			}
		}
	} else {
		if datatypesCommon {
			for datatype := range datatypes {
				fmt.Fprint(buf, datatype)
			}
		}
		if samplingsCommon {
			for sampling := range samplings {
				if sampling != defaultSampling {
					fmt.Fprint(buf, "_", sampling)
				}
			}
		}
	}
	name := buf.String()
	if alias, ok := aliases[name]; ok {
		w.Write([]byte(alias))
	} else {
		w.Write([]byte(name))
	}
}

// Clone returns a copy of this format.
func (f *Format) Clone() *Format {
	out := &Format{Components: make([]*Component, len(f.Components))}
	copy(out.Components, f.Components)
	return out
}

// Size returns the size in bytes of the full stream.
func (f *Format) Size(count int) int {
	return count * f.Stride()
}

// Stride returns the number of bytes between each element.
func (f *Format) Stride() int {
	bits := 0
	for _, c := range f.Components {
		bits += int(c.DataType.Bits())
	}
	bytes := (bits + 7) / 8
	return bytes
}

// Component returns the component in f that matches any of c.
// If no matching component is found then (nil, nil) is returned.
// If multiple matching components are found then an error is returned.
func (f *Format) Component(c ...Channel) (*Component, error) {
	channels := map[Channel]struct{}{}
	for _, c := range c {
		channels[c] = struct{}{}
	}

	matches := []*Component{}
	for _, t := range f.Components {
		if _, found := channels[t.Channel]; found {
			matches = append(matches, t)
		}
	}
	switch len(matches) {
	case 0:
		return nil, nil
	case 1:
		return matches[0], nil
	default:
		list := make([]string, len(matches))
		for i, c := range matches {
			list[i] = fmt.Sprint(c)
		}
		return nil, fmt.Errorf("%d components found matching: %v\n •%v",
			len(list), c, strings.Join(list, "\n •"))
	}
}

// HasComponent returns true if the Format contains a component with the channel
// c.
func (f *Format) HasComponent(c Channel) bool {
	for _, t := range f.Components {
		if t.Channel == c {
			return true
		}
	}
	return false
}

// HasColorComponent returns true if the format contains a color component.
// See ColorChannels for channels considered colors.
func (f *Format) HasColorComponent() bool {
	for _, t := range f.Components {
		if t.Channel.IsColor() {
			return true
		}
	}
	return false
}

// GetSingleColorComponent returns a component if the format contains a single
// color component, otherwise nil. See ColorChannels for channels considered colors.
func (f *Format) GetSingleColorComponent() *Component {
	var c *Component
	for _, t := range f.Components {
		if t.Channel.IsColor() {
			if c != nil {
				return nil
			}
			c = t
		}
	}
	return c
}

// HasDepthComponent returns true if the format contains a depth component.
// See DepthChannels for channels considered depth.
func (f *Format) HasDepthComponent() bool {
	for _, t := range f.Components {
		if t.Channel.IsDepth() {
			return true
		}
	}
	return false
}

// HasVectorComponent returns true if the format contains a vector component.
// See VectorChannels for channels considered vectors.
func (f *Format) HasVectorComponent() bool {
	for _, t := range f.Components {
		if t.Channel.IsVector() {
			return true
		}
	}
	return false
}

// BitOffsets returns the bit-offsets for the components of the format.
func (f *Format) BitOffsets() map[*Component]uint32 {
	out := make(map[*Component]uint32, len(f.Components))
	offset := uint32(0)
	for _, c := range f.Components {
		out[c] = offset
		offset += c.DataType.Bits()
	}
	return out
}

// Swizzle returns a new format with the components channels rearragned into the
// parameter order.
// Swizzle will return an error if c contains a channel that does not match any
// in f, or if the format has duplicate channels.
func (f *Format) Swizzle(c ...Channel) (*Format, error) {
	m := make(map[Channel]*Component, len(f.Components))
	for _, c := range f.Components {
		if _, dup := m[c.Channel]; dup {
			return nil, fmt.Errorf("Format has duplicate components of channel %v", c.Channel)
		}
		m[c.Channel] = c
	}
	out := &Format{Components: make([]*Component, len(c))}
	for i, channel := range c {
		component, ok := m[channel]
		if !ok {
			return nil, fmt.Errorf("Format missing channel %v", c)
		}
		out.Components[i] = component
	}
	return out, nil
}
