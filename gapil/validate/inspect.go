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

package validate

import (
	"fmt"
	"reflect"

	"github.com/google/gapid/gapil/analysis"
	"github.com/google/gapid/gapil/semantic"
)

// ErrUnreachable is the error raised for unreachable code.
// It wraps an analysis.Unreachable with an error interface.
type ErrUnreachable struct {
	analysis.Unreachable
}

func (e ErrUnreachable) Error() string {
	switch e.Node.(type) {
	case *semantic.Block:
		return "Unreachable block"
	case semantic.Expression:
		return "Unreachable expression"
	case semantic.Statement:
		return "Unreachable statement"
	}
	return fmt.Sprintf("Unreachable %v", reflect.TypeOf(e.Node).Name())
}

// Inspect checks the following:
// * There are no unreachable blocks.
// TODO: Check map indices are properly guarded by a check.
func Inspect(api *semantic.API, mappings *semantic.Mappings) Issues {
	res := analysis.Analyze(api, mappings)
	return inspect(api, mappings, res)
}

func inspect(api *semantic.API, mappings *semantic.Mappings, res *analysis.Results) Issues {
	issues := Issues{}
	for _, unreachable := range res.Unreachables {
		issues.add(unreachable.At, ErrUnreachable{unreachable})
	}
	return issues
}
