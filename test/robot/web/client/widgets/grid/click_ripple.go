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

package grid

import "github.com/google/gapid/test/robot/web/client/dom"

const (
	maxRippleAge = 1
)

type clickRipple struct {
	center *dom.Point
	age    float64
}

func (r *clickRipple) radius() float64 { return r.age * (1 + r.age) * 300 }
func (r *clickRipple) alpha() float64  { return 1.0 - (r.age / maxRippleAge) }
func (r *clickRipple) alive() bool     { return r.age < maxRippleAge }

type clickRipples []*clickRipple

func (l *clickRipples) update(dt float64) {
	c := 0
	for _, r := range *l {
		r.age += dt
		if r.alive() {
			(*l)[c] = r
			c++
		}
	}
	(*l) = (*l)[:c]
}

func (l *clickRipples) add(p *dom.Point) {
	*l = append(*l, &clickRipple{center: p})
}
