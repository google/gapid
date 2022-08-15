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

package benchmark

import (
	"fmt"
	"math"
	"time"
)

// Complexity represents the complexity of an algorithm.
type Complexity interface {
	Fit(samples Samples) (fit Fit, err float64)
}

// Fit represents a complexity fit for the samples.
type Fit interface {
	String() string
}

// LinearTime represents an algorithmic complexity of O(n).
var LinearTime linearTime

type linearTime struct{}

func (linearTime) String() string { return "O(n)" }

// Fit calculates simple linear regression
// See https://en.wikipedia.org/wiki/Simple_linear_regression
//
//	https://en.wikipedia.org/wiki/Covariance
//	https://en.wikipedia.org/wiki/Variance
func (linearTime) Fit(samples Samples) (fit Fit, err float64) {
	if len(samples) < 2 {
		return nil, math.MaxFloat64
	}

	n := len(samples)
	Sqr := func(x float64) float64 { return x * x }
	E := func(value func(i int) float64) float64 {
		// Calculate average of values ranging over 'n'
		sum := float64(0)
		for i := 0; i < n; i++ {
			sum += value(i)
		}
		return sum / float64(n)
	}
	x := func(i int) float64 { return float64(samples[i].Index) }
	y := func(i int) float64 { return float64(samples[i].Time) }
	E_x := E(x)
	E_y := E(y)
	Cov := E(func(i int) float64 { return (x(i) - E_x) * (y(i) - E_y) })
	Var := E(func(i int) float64 { return Sqr(x(i) - E_x) })
	β := Cov / Var
	α := E_y - β*E_x
	err = E(func(i int) float64 { return Sqr(α + β*x(i) - y(i)) })

	return LinearFit{time.Duration(α), time.Duration(β)}, 0
}

// LinearFit is a linear time fitting (y = α + βx).
type LinearFit struct {
	α time.Duration // Fixed systemic cost
	β time.Duration // Per sample cost
}

func NewLinearFit(α, β time.Duration) LinearFit {
	return LinearFit{time.Duration(α), time.Duration(β)}
}

func (f LinearFit) String() string {
	return fmt.Sprintf("%v + %v per sample", f.α, f.β)
}
