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

package pod

import (
	"fmt"
)

func (v Value) Format(f fmt.State, c rune) {
	switch v := v.Val.(type) {
	case *Value_Float32:
		fmt.Fprintf(f, "%v", v.Float32)
	case *Value_Float64:
		fmt.Fprintf(f, "%v", v.Float64)
	case *Value_Uint:
		fmt.Fprintf(f, "%v", v.Uint)
	case *Value_Sint:
		fmt.Fprintf(f, "%v", v.Sint)
	case *Value_Uint8:
		fmt.Fprintf(f, "%v", v.Uint8)
	case *Value_Sint8:
		fmt.Fprintf(f, "%v", v.Sint8)
	case *Value_Uint16:
		fmt.Fprintf(f, "%v", v.Uint16)
	case *Value_Sint16:
		fmt.Fprintf(f, "%v", v.Sint16)
	case *Value_Uint32:
		fmt.Fprintf(f, "%v", v.Uint32)
	case *Value_Sint32:
		fmt.Fprintf(f, "%v", v.Sint32)
	case *Value_Uint64:
		fmt.Fprintf(f, "%v", v.Uint64)
	case *Value_Sint64:
		fmt.Fprintf(f, "%v", v.Sint64)
	case *Value_Bool:
		fmt.Fprintf(f, "%v", v.Bool)
	case *Value_String_:
		fmt.Fprintf(f, "%v", v.String_)
	default:
		fmt.Fprintf(f, "%v", v)
	}
}
