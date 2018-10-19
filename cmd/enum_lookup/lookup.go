// Copyright (C) 2018 Google Inc.
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

package main

import (
	"strings"
)

type enum struct {
	API  string
	Type string
	Name string
}

var enums = map[int64][]enum{}
var bitfields = map[string]map[int64][]enum{}

func registerEnum(api, ty, name string, value int64) {
	enums[value] = append(enums[value], enum{api, ty, name})
}

func registerBitfield(api, ty, name string, value int64) {
	registerEnum(api, ty, name, value)

	m, ok := bitfields[ty]
	if !ok {
		m = map[int64][]enum{}
		bitfields[ty] = m
	}
	m[value] = append(m[value], enum{api, ty, name})
}

func LookupEnum(value int64) []enum {
	return enums[value]
}

func LookupBitfields(value int64) []enum {
	r := []enum{}
	if value != 0 {
		for ty, m := range bitfields {
			names := []string{}
			mask := int64(0)
			api := ""
			for bit, vs := range m {
				if (value & bit) == bit {
					mask |= bit
					for _, v := range vs {
						names = append(names, v.Name)
					}
					api = vs[0].API
				}
			}
			if mask == value {
				r = append(r, enum{api, ty, strings.Join(names, " | ")})
			}
		}
	}

	return r
}
