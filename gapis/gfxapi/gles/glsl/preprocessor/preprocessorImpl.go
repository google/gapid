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

package preprocessor

import (
	"bytes"
	"fmt"

	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/gapis/gfxapi/gles/glsl/ast"
)

// ifEntry is a structure containing the data necessary for proper evaluation of #if*
// preprocessor directives.
type ifEntry struct {
	HadElse  bool // Whether we already encountered an #else block
	Skipping bool // Whether the current block should be skipped
	SkipElse bool // Whether all else and elif blocks should be skipped
}

type macroDefinition struct {
	name       string          // macro name
	function   bool            // Whether this is a function macro.
	argCount   int             // The number of arguments of the macro.
	definition []macroExpander // The macro definition as a list of macroExpanders.
}

// preprocessorImpl stores the internal state of a preprocessor instance.
type preprocessorImpl struct {
	err   ast.ErrorCollector
	lexer *lexer

	macros       map[string]macroDefinition // All currently defined macros.
	version      string                     // The shader version declared with #version.
	extensions   []Extension                // All encountered #extension directives.
	ifStack      []ifEntry                  // The stack of all encountered #if directives.
	line         int                        // The current line.
	currentToken *tokenExpansion
	evaluator    ExpressionEvaluator
}

func (p *preprocessorImpl) Version() string {
	return p.version
}

func (p *preprocessorImpl) Extensions() []Extension {
	return p.extensions
}

func (p *preprocessorImpl) Errors() []error {
	return ast.ConcatErrors(p.lexer.err.GetErrors(), p.err.GetErrors())
}

// skipping returnes true if we should skip this token. We skip if any of the #if directives in
// the stack says we should skip.
func (p *preprocessorImpl) skipping() (skip bool) {
	c := len(p.ifStack)
	return c > 0 && p.ifStack[c-1].Skipping
}

// tokenReader is an internal interface encapsulating a stream of tokens.
type tokenReader interface {
	Next() tokenExpansion
	Peek() tokenExpansion
}

// listReader is an implementation of tokenReader which reads tokens from a list. It is used to
// rescan a macro expansion to expand macros recursively. It contains a nested tokenReader, which
// is read from after the own token list. This happens in case of recursive function macros with
// unbalanced parenthesis.
type listReader struct {
	list []tokenExpansion
	next tokenReader
}

func (r *listReader) Next() (t tokenExpansion) {
	if len(r.list) > 0 {
		t = r.list[0]
		r.list = r.list[1:]
	} else if r.next != nil {
		t = r.next.Next()
	}
	return
}
func (r *listReader) Peek() (t tokenExpansion) {
	if len(r.list) > 0 {
		t = r.list[0]
	} else if r.next != nil {
		t = r.next.Peek()
	}
	return
}

// processList is a helper function for processMacro. It calls processMacro on all tokens in the
// list.
func (p *preprocessorImpl) processList(r *listReader) (result []tokenExpansion) {
	for len(r.list) > 0 {
		token := r.Next()
		result = append(result, p.processMacro(token, r)...)
	}
	return
}

// readMacroArgs reads macro arguments. It returns the arguments as a list of lists of tokens.
// Failure is reported by the second return value.
func (p *preprocessorImpl) readMacroArgs(reader tokenReader) (args [][]tokenExpansion, ok bool) {
	var arg []tokenExpansion // currently processed argument
	level := 0               // number of nested parenthesis
	for {
		if reader.Peek().Info.Token == nil {
			p.err.Errorf("Unexpected end of file while processing a macro.")
			return args, false
		}
		if level == 0 {
			switch reader.Peek().Info.Token {
			case OpRParen:
				args = append(args, arg)
				return args, true
			case OpLParen:
				level++
				arg = append(arg, reader.Next())
				continue
			case ast.BoComma:
				reader.Next()
				args = append(args, arg)
				arg = nil
				continue
			}
		}
		switch reader.Peek().Info.Token {
		case OpRParen:
			level--
			arg = append(arg, reader.Next())
		case OpLParen:
			level++
			arg = append(arg, reader.Next())
		default:
			arg = append(arg, reader.Next())
		}
	}
}

// parseMacroCallArgs reads arguments to a function macro, pre-expands them and computes the
// intersection of their hide sets. It reads the argument from the specified token reader. In
// case of errors the hide set is nil.
func (p *preprocessorImpl) parseMacroCallArgs(reader tokenReader, macro tokenExpansion,
	argCount int) ([][]tokenExpansion, hideSet) {

	if reader.Peek().Info.Token != OpLParen {
		// Function macros are not expanded if the next token is not '('.
		return nil, nil
	}

	reader.Next()
	args, ok := p.readMacroArgs(reader)
	if !ok {
		return nil, nil
	}

	lastTok := reader.Next()
	if len(args) != argCount {
		p.err.Errorf("Incorrect number of arguments to macro '%v': expected %d, got %d.",
			macro.Info.Token, argCount, len(args))
		// Try to recover by padding args
		for len(args) < argCount {
			args = append(args, nil)
		}
	}

	// Macro argument pre-expansion
	for i := range args {
		args[i] = p.processList(&listReader{args[i], nil})
	}

	set := intersect(macro.HideSet, lastTok.HideSet)

	return args, set
}

// processMacro checks t for macro definitions and fully expands it. reader is an interface to
// the following tokens, needed for processing function macro invocations.
func (p *preprocessorImpl) processMacro(t tokenExpansion, reader tokenReader) []tokenExpansion {
	// eof pseudo-token
	if t.Info.Token == nil {
		return []tokenExpansion{t}
	}

	name := t.Info.Token.String()
	def, present := p.macros[name]
	if !present {
		// no expansion needed
		return []tokenExpansion{t}
	}

	set := t.HideSet

	if _, present := set[name]; present {
		// This macro should not be expanded.
		return []tokenExpansion{t}
	}

	var args [][]tokenExpansion
	if def.function {
		args, set = p.parseMacroCallArgs(reader, t, def.argCount)
		if set == nil {
			return []tokenExpansion{t}
		}
	}

	list := make([]tokenExpansion, 0, len(def.definition))
	// Substitute arguments into macro definition
	for _, expander := range def.definition {
		list = append(list, expander(args)...)
	}

	// Extend the hide sets
	for _, e := range list {
		e.HideSet.AddAll(set)
		e.HideSet[name] = struct{}{}
	}

	// Token pasting
	for i := 0; i < len(list); i++ {
		for i+2 < len(list) && list[i+1].Info.Token.String() == "##" {
			newIdentifier := list[i].Info.Token.String() + list[i+2].Info.Token.String()
			list[i] = newTokenExpansion(TokenInfo{Token: Identifier(newIdentifier)})
			list = append(list[:i+1], list[i+3:]...) // Remove the ## and following token
		}
	}

	// Expand macros in the definition recursively
	return p.processList(&listReader{list, reader})
}

// getDirectiveArguments is a helper function used to read the arguments of a preprocessor
// directive. It consumes all tokens until the newline and returns them. If emptyOk is false, it
// will raise an error in the case of an empty argument list.
func (p *preprocessorImpl) getDirectiveArguments(info TokenInfo, emptyOk bool) []TokenInfo {
	dir := info.Token
	var ret []TokenInfo
	for info = p.lexer.Peek(); info.Token != nil && !info.Newline; info = p.lexer.Peek() {
		ret = append(ret, p.lexer.Next())
	}

	if len(ret) == 0 && !emptyOk {
		p.err.Errorf("%s needs an argument.", dir)
	}

	return ret
}

func isIdentOrKeyword(t TokenInfo) bool {
	switch t.Token.(type) {
	case Identifier, Keyword, ast.BareType:
		return true
	default:
		return false
	}
}

// Given a list of tokens following `#define FOO(`, consume the tokens that make up the macro
// argument list. Returns the list of unconsumed tokens and a `macro_name->position` map.
func (p *preprocessorImpl) parseDefMacroArgs(macro Token,
	args []TokenInfo) (rest []TokenInfo, argMap map[string]int) {

	argMap = make(map[string]int)
	for {
		if len(args) <= 1 {
			p.err.Errorf("Macro definition ended unexpectedly.")
			return nil, nil
		}

		if !isIdentOrKeyword(args[0]) {
			p.err.Errorf("Invalid function macro definition. "+
				"Expected an identifier, got '%s'.", args[0].Token)
			return
		}
		name := args[0].Token.String()

		if _, ok := argMap[name]; ok {
			p.err.Errorf("Macro '%s' contains two arguments named '%s'.", macro, name)
		}

		argMap[name] = len(argMap)

		switch args[1].Token {
		case OpRParen:
			return args[2:], argMap
		case ast.BoComma:
			args = args[2:]
			continue
		default:
			p.err.Errorf("Invalid function macro definition. "+
				"Expected ',', ')', got '%s'.", args[1].Token)
			return nil, nil
		}
	}
}

// process a #define directive
func (p *preprocessorImpl) processDefine(args []TokenInfo) {
	macro := args[0]

	if _, ok := p.macros[macro.Token.String()]; ok {
		delete(p.macros, macro.Token.String())
	}

	args = args[1:]

	if len(args) == 0 || args[0].Whitespace || args[0].Token != OpLParen {
		// Just an object macro, we're done.
		expansion := make([]macroExpander, len(args))
		for i := range args {
			expansion[i] = args[i].expand
		}

		name := macro.Token.String()
		p.macros[name] = macroDefinition{name, false, 0, expansion}
		return
	}

	args, argMap := p.parseDefMacroArgs(macro.Token, args[1:])
	if argMap == nil {
		return
	}

	expansion := make([]macroExpander, len(args))
	for i := range args {
		if arg, ok := argMap[args[i].Token.String()]; ok {
			expansion[i] = argumentExpander(arg).expand
		} else {
			expansion[i] = args[i].expand
		}
	}

	name := macro.Token.String()
	p.macros[name] = macroDefinition{name, true, len(argMap), expansion}
}

// processDirectives reads any preprocessor directives from the input stream and processes them.
func (p *preprocessorImpl) processDirectives() {
	for {
		if _, ok := p.lexer.Peek().Token.(ppKeyword); !ok {
			break
		}
		p.processDirective(p.lexer.Next())
	}
}

func (p *preprocessorImpl) evaluateDefined(arg TokenInfo) tokenExpansion {
	var ic ast.IntValue
	if _, present := p.macros[arg.Token.String()]; present {
		ic = ast.IntValue(1)
	} else {
		ic = ast.IntValue(0)
	}
	return newTokenExpansion(TokenInfo{Token: ic})
}

func (p *preprocessorImpl) evaluateIf(args []TokenInfo) bool {
	// append fake EOF
	lastToken := args[len(args)-1].Cst.Token()
	eof := &parse.Leaf{}
	eof.SetToken(parse.Token{Source: lastToken.Source, Start: lastToken.End, End: lastToken.End})
	args = append(args, TokenInfo{Token: nil, Cst: eof})

	var list []tokenExpansion
	// convert args to tokenExpansions and evaluate defined(X)
	for i := 0; i < len(args); i++ {
		if args[i].Token == Identifier("defined") {
			if i+1 < len(args) && isIdentOrKeyword(args[i+1]) {
				list = append(list, p.evaluateDefined(args[i+1]))
				i++
			} else if i+3 < len(args) && args[i+1].Token == OpLParen &&
				isIdentOrKeyword(args[i+2]) && args[i+3].Token == OpRParen {

				list = append(list, p.evaluateDefined(args[i+2]))
				i += 3
			} else {
				p.err.Errorf("Operator 'defined' used incorrectly.")
			}
		} else {
			list = append(list, newTokenExpansion(args[i]))
		}
	}

	reader := &listReader{list: list} // reader will read the arguments
	worker := &listWorker{reader, p}  // worker will expand them
	pp := &Preprocessor{impl: worker} // pp will provide the lookahead
	val, err := p.evaluator(pp)       // and evaluator will evalate them

	p.err.Error(err...)
	return val != 0
}

func (p *preprocessorImpl) processDirective(info TokenInfo) {
	switch info.Token {
	case ppDefine:
		args := p.getDirectiveArguments(info, false)
		if p.skipping() || args == nil {
			return
		}
		p.processDefine(args)

	case ppUndef:
		args := p.getDirectiveArguments(info, false)
		if p.skipping() || args == nil {
			return
		}

		if _, ok := p.macros[args[0].Token.String()]; !ok {
			p.err.Errorf("Macro '%s' not defined.", args[0].Token)
			return
		}
		delete(p.macros, args[0].Token.String())

	case ppIf:
		args := p.getDirectiveArguments(info, false)
		if p.skipping() || args == nil {
			// Skip both of the branches if the parent condition evaluated to false.
			// We intentionally do not evaluate the condition since it might be invalid.
			p.ifStack = append(p.ifStack, ifEntry{Skipping: true, SkipElse: true})
			return
		}

		val := p.evaluateIf(args)
		p.ifStack = append(p.ifStack, ifEntry{Skipping: !val, SkipElse: val})

	case ppElif:
		args := p.getDirectiveArguments(info, true)

		if len(p.ifStack) == 0 {
			p.err.Errorf("Unmatched #elif.")
			return
		}

		entry := &p.ifStack[len(p.ifStack)-1]
		if entry.HadElse {
			p.err.Errorf("#elif after #else.")
			entry.Skipping = true
			return
		}
		if entry.SkipElse {
			entry.Skipping = true
		} else {
			val := p.evaluateIf(args)
			entry.Skipping = !val
			entry.SkipElse = val
		}
		return

	case ppVersion:
		args := p.getDirectiveArguments(info, false)
		if len(args) > 0 {
			p.version = args[0].Token.String()
		} else {
			p.err.Errorf("expected version number after #version")
		}
		return

		// TODO: support #pragma instead of silently ignoring it.
	case ppPragma:
		_ = p.getDirectiveArguments(info, false)
		return

	case ppExtension:
		args := p.getDirectiveArguments(info, false)
		if p.skipping() {
			return
		}
		if len(args) == 3 {
			name, nameOk := args[0].Token.(Identifier)
			colonOk := args[1].Token == OpColon
			behaviour, behaviourOk := args[2].Token.(Identifier)
			if nameOk && colonOk && behaviourOk {
				extension := Extension{Name: name.String(), Behaviour: behaviour.String()}
				p.extensions = append(p.extensions, extension)
				return
			}
		}
		p.err.Errorf("#extension should have the form '#extension name : behaviour'")
		return

	case ppIfdef, ppIfndef:
		args := p.getDirectiveArguments(info, false)

		if p.skipping() {
			// Skip both of the branches if the parent condition evaluated to false.
			p.ifStack = append(p.ifStack, ifEntry{Skipping: true, SkipElse: true})
			return
		}

		var defined bool
		if args == nil {
			defined = false
		} else {
			_, defined = p.macros[args[0].Token.String()]
		}

		value := defined == (info.Token == ppIfdef)

		p.ifStack = append(p.ifStack, ifEntry{Skipping: !value, SkipElse: value})

	case ppElse:
		_ = p.getDirectiveArguments(info, true)

		if len(p.ifStack) == 0 {
			p.err.Errorf("Unmatched #else.")
			return
		}
		entry := &p.ifStack[len(p.ifStack)-1]
		if entry.HadElse {
			p.err.Errorf("#if directive has multiple #else directives.")
			entry.Skipping = true
			return
		}
		entry.HadElse = true
		entry.Skipping = entry.SkipElse

	case ppEndif:
		_ = p.getDirectiveArguments(info, true)

		if len(p.ifStack) == 0 {
			p.err.Errorf("Unmatched #endif.")
			return
		}
		p.ifStack = p.ifStack[:len(p.ifStack)-1]

	case ppLine:
		args := p.getDirectiveArguments(info, true)
		if len(args) != 1 && len(args) != 2 {
			p.err.Errorf("expected line/file number after #line")
		}

	case ppError:
		args := p.getDirectiveArguments(info, true)
		if p.skipping() {
			return
		}

		var msg bytes.Buffer
		for _, i := range args {
			i.Cst.Prefix().WriteTo(&msg)
			msg.Write([]byte(i.Token.String()))
			i.Cst.Suffix().WriteTo(&msg)
		}
		p.err.Errorf(msg.String())
	}
}

func addBuiltinMacro(macros map[string]macroDefinition, name string, expander macroExpander) {
	macros[name] = macroDefinition{
		name:       name,
		definition: []macroExpander{expander},
	}
}

func newPreprocessorImpl(data string, eval ExpressionEvaluator, file int) *preprocessorImpl {
	p := &preprocessorImpl{
		lexer:     newLexer(fmt.Sprintf("File %v", file), data),
		macros:    make(map[string]macroDefinition),
		evaluator: eval,
	}
	addBuiltinMacro(p.macros, "__LINE__", p.expandLine)
	addBuiltinMacro(p.macros, "__FILE__", TokenInfo{Token: ast.IntValue(file)}.expand)
	addBuiltinMacro(p.macros, "__VERSION__", TokenInfo{Token: ast.IntValue(300)}.expand)
	addBuiltinMacro(p.macros, "GL_ES", TokenInfo{Token: ast.IntValue(1)}.expand)
	return p
}

//////////////////////////// tokenReader interface /////////////////////////

func (p *preprocessorImpl) Peek() tokenExpansion {
	for p.currentToken == nil {
		// process any preprocessor directives
		p.processDirectives()

		tok := newTokenExpansion(p.lexer.Next())
		p.line, _ = tok.Info.Cst.Token().Cursor()

		if tok.Info.Token == nil {
			if len(p.ifStack) > 0 {
				p.err.Errorf("Unterminated #if directive at the end of file.")
			}
			p.currentToken = &tok
		} else if !p.skipping() {
			p.currentToken = &tok
		}
	}
	return *p.currentToken
}

func (p *preprocessorImpl) Next() tokenExpansion {
	ret := p.Peek()
	p.currentToken = nil
	return ret
}

//////////////////////////// worker interface /////////////////////////

func (p *preprocessorImpl) Work() []tokenExpansion {
	return p.processMacro(p.Next(), p)
}
