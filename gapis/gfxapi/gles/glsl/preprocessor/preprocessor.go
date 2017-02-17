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

// Package preprocessor defines the preprocessor for the OpenGL ES Shading Language.
//
// It can be used in two ways. The function Preprocess takes an input strings and returns the sequence
// of preprocessed tokens as a result. The function PreprocessStream returns a Preprocessor object,
// which can be used to access the result in a streaming fashion, using the Next() and Peek()
// methods, returning TokenInfo objects. The end of input is signalled by a nil Token field of
// the object.
//
// The preprocessor does not attempt to distinguish between prefix and postfix versions of the
// increment and decrement operators even though it defines the corresponding tokens. This job is
// left to the parser, which should not assume that both UoPreinc and UoPostinc can be returned
// for a "++" token, etc.
//
// Functions Preprocess and PreprocessStream are parameterized by an ExpressionEvaluator
// function, which is used for evaluating #if expressions. The third parameter is the file
// identifier, while will be returned by the __FILE__ macro.
package preprocessor

import "github.com/google/gapid/gapis/gfxapi/gles/glsl/ast"

// Extension describes a GLSL shader extension defined by #extension.
type Extension struct {
	Name, Behaviour string
}

// Internal interface which glues preprocessor wrapper class to the actual implementation.
type worker interface {
	Work() []tokenExpansion  // Performs a single unit of work, and return the resulting token sequence.
	Version() string         // Returns the shader version (if declared).
	Extensions() []Extension // Returns encountered #extension directives.
	Errors() []error         // Returns any errors encountered.
}

// Implementation of worker, which reads tokens from a list and processes them. Used for
// processing #if directives.
type listWorker struct {
	reader *listReader
	pp     *preprocessorImpl
}

func (w *listWorker) Work() []tokenExpansion  { return w.pp.processMacro(w.reader.Next(), w.reader) }
func (w *listWorker) Version() string         { return "" }
func (w *listWorker) Extensions() []Extension { return nil }
func (w *listWorker) Errors() []error         { return nil }

// ExpressionEvaluator is a function type. These functions are used to process #if expressions.
// If you want to implement your own expression evaluator, pass a function which given a
// Preprocessor, returns a value and a list of errors. Otherwise, pass
// ast.EvaluatePreprocessorExpression.
//
// This function is passed a preprocessor object, which reads the tokens following the #if
// directive. The tokens are passed as-is, with the exception of defined(MACRO) expressions,
// which are evaluated to "0" or "1".
type ExpressionEvaluator func(*Preprocessor) (val ast.IntValue, err []error)

// Preprocessor is the streaming interface to the GLES Shading Language preprocessor. It supports
// the usual Peek/Next operations. At any point it can return the list of detected preprocessor
// errors.
type Preprocessor struct {
	impl      worker
	lookahead []tokenExpansion
}

// Return the next token.
func (p *Preprocessor) Peek() TokenInfo { return p.PeekN(0) }

func eof(list []tokenExpansion) bool { l := len(list); return l > 0 && list[l-1].Info.Token == nil }

// Return the n-th token (for the next token, set n=0).
func (p *Preprocessor) PeekN(n int) TokenInfo {
	for n >= len(p.lookahead) && !eof(p.lookahead) {
		p.lookahead = append(p.lookahead, p.impl.Work()...)
	}
	if n >= len(p.lookahead) {
		n = len(p.lookahead) - 1
	}
	return p.lookahead[n].Info
}

// Return the next token and advance the stream.
func (p *Preprocessor) Next() TokenInfo {
	ret := p.Peek()
	if len(p.lookahead) > 1 || !eof(p.lookahead) {
		p.lookahead = p.lookahead[1:]
	}
	return ret
}

// Version returns the shader version (if declared).
func (p *Preprocessor) Version() string { return p.impl.Version() }

// Extensions returns encountered #extension directives.
func (p *Preprocessor) Extensions() []Extension { return p.impl.Extensions() }

// Errors returns the list of detected preprocessor errors.
func (p *Preprocessor) Errors() []error { return p.impl.Errors() }

func PreprocessStream(data string, eval ExpressionEvaluator, file int) *Preprocessor {
	return &Preprocessor{
		impl: newPreprocessorImpl(data, eval, file),
	}
}

// Preprocess preprocesses a GLES Shading language program string into a sequence of tokens. Any
// preprocessing errors are returned in the second return value.
func Preprocess(data string, eval ExpressionEvaluator, file int) (tokens []TokenInfo, version string, err []error) {
	pp := PreprocessStream(data, eval, file)
	for t := pp.Next(); t.Token != nil; t = pp.Next() {
		tokens = append(tokens, t)
	}

	version, err = pp.Version(), pp.Errors()
	return
}
