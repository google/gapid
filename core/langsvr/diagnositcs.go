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

import "github.com/google/gapid/core/langsvr/protocol"

// Diagnostics is a list of Diagnostic
type Diagnostics []Diagnostic

// Error appends a error diagnostic at rng.
func (l *Diagnostics) Error(rng Range, msg string) {
	*l = append(*l, Diagnostic{rng, SeverityError, msg, "", ""})
}

// Warning appends a warning diagnostic at rng.
func (l *Diagnostics) Warning(rng Range, msg string) {
	*l = append(*l, Diagnostic{rng, SeverityWarning, msg, "", ""})
}

// Info appends an info diagnostic at rng.
func (l *Diagnostics) Info(rng Range, msg string) {
	*l = append(*l, Diagnostic{rng, SeverityInformation, msg, "", ""})
}

// Hint appends a hint diagnostic at rng.
func (l *Diagnostics) Hint(rng Range, msg string) {
	*l = append(*l, Diagnostic{rng, SeverityHint, msg, "", ""})
}

// Diagnostic represents a compiler diagnostic, such as a warning or error.
type Diagnostic struct {
	// The range at which the message applies
	Range Range

	// The diagnostic's severity.
	Severity Severity

	// The diagnostic's message.
	Message string

	// The diagnostic's code (optional)
	Code string

	// The source of the diagnostic
	Source string
}

func diagnostic(d protocol.Diagnostic) Diagnostic {
	return Diagnostic{
		Range:    rng(d.Range),
		Severity: Severity(d.Severity),
		Code:     d.Code.(string),
		Message:  d.Message,
		Source:   d.Source,
	}
}

func (d Diagnostic) toProtocol() protocol.Diagnostic {
	return protocol.Diagnostic{
		Range:    d.Range.toProtocol(),
		Severity: d.Severity.toProtocol(),
		Code:     d.Code,
		Message:  d.Message,
		Source:   d.Source,
	}
}

// Severity represents the severity level of a diagnostic.
type Severity int

const (
	// SeverityError reports an error.
	SeverityError = Severity(protocol.SeverityError)

	// SeverityWarning reports a warning.
	SeverityWarning = Severity(protocol.SeverityWarning)

	// SeverityInformation reports an information.
	SeverityInformation = Severity(protocol.SeverityInformation)

	// SeverityHint reports a hint.
	SeverityHint = Severity(protocol.SeverityHint)
)

func (s Severity) toProtocol() protocol.DiagnosticSeverity { return protocol.DiagnosticSeverity(s) }
