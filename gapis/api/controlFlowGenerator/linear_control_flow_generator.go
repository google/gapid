// Copyright (C) 2020 Google Inc.
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

package controlFlowGenerator

import (
	"context"

	"github.com/google/gapid/gapis/api/transform2"
)

type linearControlFlowGenerator struct {
	chain *transform2.TransformChain
}

// NewLinearControlFlowGenerator generates a simple control flow
// that takes initial and real commands and transforms all of them
func NewLinearControlFlowGenerator(chain *transform2.TransformChain) ControlFlowGenerator {
	return &linearControlFlowGenerator{
		chain: chain,
	}
}

func (cf *linearControlFlowGenerator) TransformAll(ctx context.Context) error {
	for !cf.chain.IsEndOfCommands() {
		if err := cf.chain.GetNextTransformedCommands(ctx); err != nil {
			return err
		}
	}

	return nil
}
