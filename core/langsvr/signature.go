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

package langsvr

import (
	"fmt"
	"strings"

	"github.com/google/gapid/core/langsvr/protocol"
)

// Signature represents the signature of something callable.
type Signature struct {
	// The label of this signature. Will be shown in the UI.
	Label string

	// The human-readable doc-comment of this signature.
	// Will be shown in the UI but can be omitted.
	Documentation string

	// The parameters of this signature.
	Parameters ParameterList
}

func (s Signature) toProtocol() protocol.SignatureInformation {
	params := make([]string, len(s.Parameters))
	for i, p := range s.Parameters {
		params[i] = p.Label
	}
	label := fmt.Sprintf("%s(%s)", s.Label, strings.Join(params, ", "))
	out := protocol.SignatureInformation{
		Label:         label,
		Documentation: &s.Documentation,
		Parameters:    s.Parameters.toProtocol(),
	}
	if s.Documentation == "" {
		out.Documentation = nil
	}
	return out
}

// SignatureList is a list of signatures.
type SignatureList []Signature

// Add appends the signature to the list.
func (l *SignatureList) Add(label, doc string, params ParameterList) {
	*l = append(*l, Signature{label, doc, params})
}

// Parameter represents a parameter of a callable-signature.
type Parameter struct {
	// The label of this signature. Will be shown in the UI.
	Label string

	// The human-readable doc-comment of this signature.
	// Will be shown in the UI but can be omitted.
	Documentation string
}

// ParameterList is a list of parameters.
type ParameterList []Parameter

// Add appends the signature to the list.
func (l *ParameterList) Add(label, doc string) {
	*l = append(*l, Parameter{label, doc})
}

func (s Parameter) toProtocol() protocol.ParameterInformation {
	out := protocol.ParameterInformation{
		Label:         s.Label,
		Documentation: &s.Documentation,
	}
	if s.Documentation == "" {
		out.Documentation = nil
	}
	return out
}

func (l SignatureList) toProtocol() []protocol.SignatureInformation {
	out := make([]protocol.SignatureInformation, len(l))
	for i, s := range l {
		out[i] = s.toProtocol()
	}
	return out
}

func (l ParameterList) toProtocol() []protocol.ParameterInformation {
	out := make([]protocol.ParameterInformation, len(l))
	for i, p := range l {
		out[i] = p.toProtocol()
	}
	return out
}
