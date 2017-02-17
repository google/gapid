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

package jdbg

import "fmt"

// parseSignature returns the type for the signature string starting at offset.
// offset will be modified so that it is one byte beyond the end of the parsed
// string.
func (j *JDbg) parseSignature(sig string, offset *int) (Type, error) {
	r := sig[*offset]
	*offset++
	switch r {
	case 'V':
		return j.cache.voidTy, nil
	case 'Z':
		return j.cache.boolTy, nil
	case 'B':
		return j.cache.byteTy, nil
	case 'C':
		return j.cache.charTy, nil
	case 'S':
		return j.cache.shortTy, nil
	case 'I':
		return j.cache.intTy, nil
	case 'J':
		return j.cache.longTy, nil
	case 'F':
		return j.cache.floatTy, nil
	case 'D':
		return j.cache.doubleTy, nil
	case 'L':
		// fully-qualified-class
		start := *offset - 1 // include 'L'
		for *offset < len(sig) {
			r := sig[*offset]
			*offset++
			if r == ';' {
				return j.classFromSig(sig[start:*offset])
			}
		}
		return nil, fmt.Errorf("Fully qualified class missing terminating ';'")
	case '[':
		start := *offset - 1 // include '['
		el, err := j.parseSignature(sig, offset)
		if err != nil {
			return nil, err
		}
		sig := sig[start:*offset]
		if array, ok := j.cache.arrays[sig]; ok {
			return array, nil
		}
		class, err := j.classFromSig(sig)
		if err != nil {
			return nil, err
		}
		array := &Array{class, el}
		j.cache.arrays[sig] = array
		return array, nil
	default:
		return nil, fmt.Errorf("Unknown signature type tag '%v'", r)
	}
}
