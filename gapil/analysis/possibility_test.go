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

package analysis_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/gapil/analysis"
)

var (
	T = analysis.True
	F = analysis.False
	M = analysis.Maybe
	I = analysis.Impossible
)

func TestPossibilityEquals(t *testing.T) {
	assert := assert.To(t)

	for _, test := range []struct {
		a, b     analysis.Possibility
		expected analysis.Possibility
	}{
		{T, T, T},
		{T, F, F},
		{T, M, M},
		{F, T, F},
		{F, F, T},
		{F, M, M},
		{M, T, M},
		{M, F, M},
		{M, M, M},
	} {
		got := test.a.Equals(test.b)
		assert.For("got").That(got).Equals(test.expected)
	}
}

func TestPossibilityBinaryOps(t *testing.T) {
	assert := assert.To(t)

	for _, test := range []struct {
		a, b     analysis.Possibility
		f        func(analysis.Possibility, analysis.Possibility) analysis.Possibility
		expected analysis.Possibility
	}{
		{T, T, analysis.Possibility.And, T},
		{T, F, analysis.Possibility.And, F},
		{T, M, analysis.Possibility.And, M},
		{F, T, analysis.Possibility.And, F},
		{F, F, analysis.Possibility.And, F},
		{F, M, analysis.Possibility.And, F},
		{M, T, analysis.Possibility.And, M},
		{M, F, analysis.Possibility.And, F},
		{M, M, analysis.Possibility.And, M},

		{T, T, analysis.Possibility.Or, T},
		{T, F, analysis.Possibility.Or, T},
		{T, M, analysis.Possibility.Or, T},
		{F, T, analysis.Possibility.Or, T},
		{F, F, analysis.Possibility.Or, F},
		{F, M, analysis.Possibility.Or, M},
		{M, T, analysis.Possibility.Or, T},
		{M, F, analysis.Possibility.Or, M},
		{M, M, analysis.Possibility.Or, M},
	} {
		got := test.f(test.a, test.b)
		assert.For("got").That(got).Equals(test.expected)
	}
}

func TestBoolUnion(t *testing.T) {
	assert := assert.To(t)

	for _, test := range []struct {
		a, b     analysis.Possibility
		expected analysis.Possibility
	}{
		{T, T, T},
		{T, F, M},
		{T, M, M},
		{F, T, M},
		{F, F, F},
		{F, M, M},
		{M, T, M},
		{M, F, M},
		{M, M, M},
	} {
		got := test.a.Union(test.b)
		assert.For("got").That(got).Equals(test.expected)
	}
}

func TestBoolIntersect(t *testing.T) {
	assert := assert.To(t)

	for _, test := range []struct {
		a, b     analysis.Possibility
		expected analysis.Possibility
	}{
		{T, T, T},
		{T, F, I},
		{T, M, T},
		{F, T, I},
		{F, F, F},
		{F, M, F},
		{M, T, T},
		{M, F, F},
		{M, M, M},
	} {
		got := test.a.Intersect(test.b)
		assert.For("got").That(got).Equals(test.expected)
	}
}

func TestBoolDifference(t *testing.T) {
	assert := assert.To(t)

	for _, test := range []struct {
		a, b     analysis.Possibility
		expected analysis.Possibility
	}{
		{T, T, I},
		{T, F, T},
		{T, M, T},
		{F, T, F},
		{F, F, I},
		{F, M, F},
		{M, T, F},
		{M, F, T},
		{M, M, M},
	} {
		got := test.a.Difference(test.b)
		assert.For("got").That(got).Equals(test.expected)
	}
}
