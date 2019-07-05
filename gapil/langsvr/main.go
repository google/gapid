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

// The langsvr command implements a language server for the graphics API language.
//
// See https://github.com/Microsoft/language-server-protocol for more information.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	ls "github.com/google/gapid/core/langsvr"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/text/parse/cst"
	"github.com/google/gapid/gapil/analysis"
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/format"
	"github.com/google/gapid/gapil/parser"
	"github.com/google/gapid/gapil/semantic"
	"github.com/google/gapid/gapil/semantic/printer"
)

const (
	apiExt = ".api"
	lang   = "gfxapi"
)

func main() {
	dl := &debugLogger{stdin: os.Stdin, stdout: os.Stdout, stop: func() {}}
	defer dl.stop()

	ctx := dl.bind(context.Background())
	defer handlePanic(ctx)

	run(ctx, dl)
}

func run(ctx context.Context, dl *debugLogger) {
	server := &server{
		docs:        map[string]*ls.Document{},
		analyzer:    newAnalyzer(),
		debugLogger: dl,
	}

	if err := ls.Connect(ctx, dl, server); err != nil {
		log.E(ctx, "%v", err)
	}
}

// Interface compliance checks.
var (
	_ ls.Server                   = (*server)(nil)
	_ ls.HoverProvider            = (*server)(nil)
	_ ls.DefinitionProvider       = (*server)(nil)
	_ ls.ReferencesProvider       = (*server)(nil)
	_ ls.HighlightsProvider       = (*server)(nil)
	_ ls.SymbolsProvider          = (*server)(nil)
	_ ls.WorkspaceSymbolsProvider = (*server)(nil)
	_ ls.CodeActionsProvider      = (*server)(nil)
	_ ls.FormatProvider           = (*server)(nil)
	_ ls.FormatRangeProvider      = (*server)(nil)
	_ ls.RenameProvider           = (*server)(nil)
	_ ls.CompletionProvider       = (*server)(nil)
	_ ls.SignatureProvider        = (*server)(nil)
	_ ls.CodeLensProvider         = (*server)(nil)
)

// Config is is the configuration data sent from the client, held in the
// "gfxapi" group.
type Config struct {
	Debug                 bool     `json:"debug"`
	LogToFiles            bool     `json:"logToFiles"`
	IgnorePaths           []string `json:"ignorePaths"`
	CheckUnused           bool     `json:"checkUnused"`
	IncludePossibleValues bool     `json:"includePossibleValues"`
}

type server struct {
	workspaceRoot string
	docs          map[string]*ls.Document // All documents (path -> docInfo)
	analyzer      *analyzer               // The analyzer. Do not access directly.
	debugLogger   *debugLogger
	config        *Config // Synchronised from the client
}

func (s *server) rel(path string) string {
	rel, err := filepath.Rel(s.workspaceRoot, path)
	if err != nil {
		return path
	}
	return rel
}

func (s *server) nodeDoc(fa *fullAnalysis, n ast.Node) *ls.Document {
	return s.docs[fa.mappings.AST.CST(n).Tok().Source.Filename]
}

func (s *server) nodeLocation(fa *fullAnalysis, n ast.Node) ls.Location {
	if doc := s.nodeDoc(fa, n); doc != nil {
		return ls.Location{
			URI:   doc.URI(),
			Range: fa.nodeRange(doc, n),
		}
	}
	return ls.Location{}
}

func (s *server) definition(sem semantic.Node) ast.Node {
	switch sem := partial(sem).(type) {
	case *semantic.Class:
		return sem.AST
	case *semantic.Enum:
		return sem.AST
	case *semantic.EnumEntry:
		return sem.AST
	case *semantic.Field:
		if sem.AST != nil {
			return sem.AST.Name
		}
	case *semantic.Global:
		return sem.AST
	case *semantic.Local:
		if sem.Declaration != nil {
			return sem.Declaration.AST
		}
	case *semantic.Member:
		return sem.Field.AST
	case *semantic.Parameter:
		return sem.AST
	case *semantic.Pseudonym:
		return sem.AST
	case *semantic.Function:
		return sem.AST.Generic.Name
	}
	return nil
}

// Initialize is called when the server is first initialized by the client.
func (s *server) Initialize(ctx context.Context, rootPath string) (ls.InitConfig, error) {
	s.workspaceRoot = rootPath
	return ls.InitConfig{
		LanguageID:                  lang,
		CompletionTriggerCharacters: []rune{'.'},
		SignatureTriggerCharacters:  []rune{'('},
		WorkspaceDocuments:          findAPIs(rootPath),
	}, nil
}

// Shutdown is called to shutdown the server.
func (s *server) Shutdown(context.Context) error {
	return nil
}

// OnConfigChange is called when the configuration settings change.
func (s *server) OnConfigChange(ctx context.Context, cfgMap map[string]interface{}) error {
	data, err := json.Marshal(cfgMap)
	if err != nil {
		return err
	}
	var payload struct {
		Cfg Config `json:"gfxapi"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	s.config = &payload.Cfg
	if err := s.debugLogger.setEnabled(s.config.LogToFiles); err != nil {
		return err
	}
	s.analyzer.begin(ctx, s)
	return nil
}

// OnDocumentsAdded is called when new documents of interest are discovered.
func (s *server) OnDocumentsAdded(ctx context.Context, docs []*ls.Document) error {
	for _, doc := range docs {
		s.docs[doc.Path()] = doc
	}
	s.analyzer.begin(ctx, s)
	return nil
}

// OnDocumentsChanged is called when documents are changed.
func (s *server) OnDocumentsChanged(ctx context.Context, docs []*ls.Document) error {
	for _, doc := range docs {
		s.docs[doc.Path()] = doc
	}
	s.analyzer.begin(ctx, s)
	return nil
}

// OnDocumentsRemoved is called when documents are no longer monitored.
func (s *server) OnDocumentsRemoved(ctx context.Context, docs []*ls.Document) error {
	for _, doc := range docs {
		delete(s.docs, doc.Path())
	}
	s.analyzer.begin(ctx, s)
	return nil
}

// OnDocumentSaved is called when an open document is saved.
func (s *server) OnDocumentSaved(ctx context.Context, doc *ls.Document) error {
	s.analyzer.begin(ctx, s)
	return nil
}

// CodeLenses returns a list of CodeLens for the specified document.
func (s *server) CodeLenses(ctx context.Context, doc *ls.Document) ([]ls.CodeLens, error) {
	return nil, nil
}

// Completions returns completion items at a given cursor position.
// Completion items are presented in the IntelliSense user interface.
func (s *server) Completions(ctx context.Context, doc *ls.Document, pos ls.Position) (ls.CompletionList, error) {
	da, err := s.docAnalysis(ctx, doc)
	if da == nil || err != nil {
		return ls.CompletionList{}, err
	}

	offset := doc.Body().Offset(pos)
	list := ls.CompletionList{}
	for _, n := range da.walkUp(offset) {
		switch sem := partial(n.sem).(type) {
		case *semantic.API:
			for _, f := range sem.Subroutines {
				list.Add(f.Name(), ls.Function, "subroutine")
			}
			for _, f := range sem.Externs {
				list.Add(f.Name(), ls.Function, "extern")
			}
			for _, f := range sem.Enums {
				list.Add(f.Name(), ls.Enum, "enum")
				for _, e := range f.Entries {
					list.Add(e.Name(), ls.Enum, fmt.Sprintf("%v(%v)", f.Name(), da.full.mappings.AST.CST(e.AST.Value).Tok().String()))
				}
			}
			for _, c := range sem.Classes {
				list.Add(c.Name(), ls.Class, "class")
			}
			return list, nil

		case *semantic.Block:
			for _, sem := range sem.Statements {
				switch sem := sem.(type) {
				case *semantic.DeclareLocal:
					list.Add(sem.Local.Name(), ls.Variable, typename(sem.Local.Type))
				}
			}

		case *semantic.Member:
			switch obj := underlying(partial(sem.Object)).(type) {
			case *semantic.Local, *semantic.Global, *semantic.Field, *semantic.Member:
				switch ty := underlying(typeof(obj)).(type) {
				case *semantic.Class:
					for _, f := range ty.Fields {
						list.Add(f.Name(), ls.Field, typename(f.Type))
					}
				}
			}
			return list, nil
		case *semantic.Function:
			for _, param := range sem.CallParameters() {
				list.Add(param.Name(), ls.Variable, typename(param.Type))
			}
		}
	}
	return list, nil
}

// Signatures returns the list of function signatures that are candidates
// at the given cursor position. The activeSig and activeParam are indices
// of the signature and parameter to highlight for the given cursor
// position.
func (s *server) Signatures(ctx context.Context, doc *ls.Document, pos ls.Position) (sigs ls.SignatureList, activeSig, activeParam int, err error) {
	da, err := s.docAnalysis(ctx, doc)
	if da == nil || err != nil {
		return nil, 0, 0, err
	}
	offset := doc.Body().Offset(pos)
	for _, n := range da.walkUp(offset) {
		call, ok := partial(n.sem).(*semantic.Call)
		if !ok {
			continue
		}
		function := call.Target.Function
		if function == nil {
			continue
		}
		params := ls.ParameterList{}
		paramIndex := 0
		for _, p := range function.CallParameters() {
			params.Add(p.Name(), da.full.documentation(p.AST))
		}
		for i, a := range call.AST.Arguments {
			if tok := da.full.mappings.AST.CST(a).Tok(); offset >= tok.Start {
				paramIndex = i
			}
		}
		sigs.Add(function.Name(), da.full.documentation(function.AST), params)
		return sigs, 0, paramIndex, nil
	}
	return nil, 0, 0, nil
}

// Definitions returns the list of definition locations for the given symbol in
// the specified document at position.
func (s *server) Definitions(ctx context.Context, doc *ls.Document, pos ls.Position) ([]ls.Location, error) {
	da, err := s.docAnalysis(ctx, doc)
	if da == nil || err != nil {
		return nil, err
	}
	for _, n := range da.walkUp(doc.Body().Offset(pos)) {
		if _, isIdent := n.ast.(*ast.Identifier); !isIdent || n.sem == nil {
			continue
		}
		if n := s.definition(n.sem); n != nil {
			return []ls.Location{s.nodeLocation(da.full, n)}, nil
		}
	}
	return nil, nil
}

// References returns a list of references for the given symbol in the
// specified document at position.
func (s *server) References(ctx context.Context, doc *ls.Document, pos ls.Position) ([]ls.Location, error) {
	da, err := s.docAnalysis(ctx, doc)
	if da == nil || err != nil {
		return nil, err
	}
	for _, n := range da.walkUp(doc.Body().Offset(pos)) {
		if _, isIdent := n.ast.(*ast.Identifier); !isIdent {
			continue
		}
		locations := []ls.Location{}
		for _, n := range da.full.mappings.SemanticToAST[n.sem] {
			if _, isIdent := n.(*ast.Identifier); isIdent {
				locations = append(locations, s.nodeLocation(da.full, n))
			}
		}
		return locations, nil
	}
	return nil, nil
}

// Highlights returns a list of highlights for the given symbol in the
// specified document at position.
func (s *server) Highlights(ctx context.Context, doc *ls.Document, pos ls.Position) (ls.HighlightList, error) {
	da, err := s.docAnalysis(ctx, doc)
	if da == nil || err != nil {
		return nil, err
	}
	for _, n := range da.walkUp(doc.Body().Offset(pos)) {
		if _, isIdent := n.ast.(*ast.Identifier); !isIdent {
			continue
		}
		highlights := ls.HighlightList{}
		for _, n := range da.full.mappings.SemanticToAST[n.sem] {
			if _, isIdent := n.(*ast.Identifier); isIdent && da.contains(n) {
				highlights.Add(da.full.nodeRange(doc, n), ls.TextHighlight)
			}
		}
		return highlights, nil
	}
	return nil, nil
}

// Format returns a list of edits required to format the the entire document.
func (s *server) Format(ctx context.Context, doc *ls.Document, opts ls.FormattingOptions) (ls.TextEditList, error) {
	m := &ast.Mappings{}
	ast, err := parser.Parse("", doc.Body().Text(), m)
	if err != nil {
		// Reformatting ASTs with parse errors?
		// You're going to have a bad time.
		return ls.TextEditList{}, nil
	}
	formatted := &bytes.Buffer{}
	format.Format(ast, m, formatted)
	edits := ls.TextEditList{}
	edits.Add(doc.Body().FullRange(), formatted.String())
	return edits, nil
}

func (s *server) FormatRange(ctx context.Context, doc *ls.Document, rng ls.Range, opts ls.FormattingOptions) (ls.TextEditList, error) {
	if rng == doc.Body().FullRange() {
		return s.Format(ctx, doc, opts)
	}
	return ls.TextEditList{}, nil
}

// Rename is called to rename the symbol at pos with newName.
func (s *server) Rename(ctx context.Context, doc *ls.Document, pos ls.Position, newName string) (ls.WorkspaceEdit, error) {
	da, err := s.docAnalysis(ctx, doc)
	if da == nil || err != nil {
		return ls.WorkspaceEdit{}, err
	}
	offset := doc.Body().Offset(pos)
	edits := ls.WorkspaceEdit{}
	for _, n := range da.walkUp(offset) {
		if _, ok := n.ast.(*ast.Identifier); !ok {
			continue // We can only sensibly rename identifiers.
		}
		for _, n := range da.full.mappings.SemanticToAST[n.sem] {
			if _, ok := n.(*ast.Identifier); ok {
				edits.Add(s.nodeLocation(da.full, n), newName)
			}
		}
	}
	return edits, nil
}

// Hover returns a list of source code snippets and range for the given
// symbol at the specified position.
func (s *server) Hover(ctx context.Context, doc *ls.Document, pos ls.Position) (ls.SourceCodeList, ls.Range, error) {
	da, err := s.docAnalysis(ctx, doc)
	if da == nil || err != nil {
		if s.config.Debug {
			log.W(ctx, "No analysis results: %v", err)
		}
		return nil, ls.Range{}, nil
	}
	offset := doc.Body().Offset(pos)
	code := ls.SourceCodeList{}

	if s.config.IncludePossibleValues {
		if val, res := possibleValues(da, pos); val != nil {
			code.Add("plain", "Possible values: "+val.Print(res))
		}
	}

	if s.config != nil && s.config.Debug {
		astBreadcrumbs, semBreadcrumbs := []string{}, []string{}
		for _, n := range da.walkDown(offset) {
			astBreadcrumbs = append(astBreadcrumbs, goTypename(n.ast))
			semBreadcrumbs = append(semBreadcrumbs, goTypename(n.sem))
		}
		code.Add("plain", fmt.Sprintf("offset: %v", offset))
		code.Add("plain", "AST: "+strings.Join(astBreadcrumbs, "->"))
		code.Add("plain", "SEM: "+strings.Join(semBreadcrumbs, "->"))

		for _, n := range da.walkUp(doc.Body().Offset(pos)) {
			if n.ast == nil {
				continue
			}
			branch, ok := da.full.mappings.AST.CST(n.ast).(*cst.Branch)
			if !ok {
				continue
			}
			if len(branch.Children) < 5 {
				for _, n := range branch.Children {
					tok := n.Tok()
					lines := strings.Split(tok.String(), "\n")
					cnt := len(lines)
					if cnt == 1 {
						code.Add("plain", fmt.Sprintf("%v-%v: %v", tok.Start, tok.End, lines[0]))
					} else {
						code.Add("plain", fmt.Sprintf("%v-%v: %v ... %v", tok.Start, tok.End, lines[0], lines[cnt-1]))
					}
				}
			}
			return code, da.full.nodeRange(doc, n.ast), nil
		}
		return code, ls.Range{}, nil
	}

	for _, n := range da.walkUp(doc.Body().Offset(pos)) {
		if _, isIdent := n.ast.(*ast.Identifier); !isIdent || n.sem == nil {
			continue
		}
		if def := s.definition(n.sem); def != nil {
			if doc := s.nodeDoc(da.full, def); doc != nil {
				rng := da.full.nodeRange(doc, def)
				rng.Start.Column = 1
				rng.End.Column = 500000
				source := doc.Body().GetRange(rng)
				code.Add(lang, source)
			}
		}
		return code, da.full.nodeRange(doc, n.ast), nil
	}
	return nil, ls.Range{}, nil
}

// Symbols returns symbolic information about the specified document.
func (s *server) Symbols(ctx context.Context, doc *ls.Document) (ls.SymbolList, error) {
	syms, err := s.WorkspaceSymbols(ctx)
	syms = syms.Filter(func(s ls.Symbol) bool { return s.Location.URI == doc.URI() })
	return syms, err
}

// WorkspaceSymbols returns project-wide symbols.
func (s *server) WorkspaceSymbols(ctx context.Context) (ls.SymbolList, error) {
	a := s.analyzer.results(ctx, s)
	if a == nil {
		return nil, log.Err(ctx, nil, "Analyser config not ready yet")
	}

	syms := ls.SymbolList{}
	for _, root := range a.roots {
		full := root.doc.full
		for _, sem := range root.sem.Classes {
			classSym := syms.Add(sem.Name(), ls.KindClass, s.nodeLocation(full, sem.AST), nil)
			for _, f := range sem.Fields {
				syms.Add(f.Name(), ls.KindField, s.nodeLocation(full, f.AST), &classSym)
			}
		}
		for _, sem := range root.sem.Enums {
			enumSym := syms.Add(sem.Name(), ls.KindEnum, s.nodeLocation(full, sem.AST), nil)
			for _, e := range sem.Entries {
				syms.Add(e.Name(), ls.KindEnum, s.nodeLocation(full, e.AST), &enumSym)
			}
		}
		for _, sem := range root.sem.Externs {
			syms.Add(sem.Name(), ls.KindFunction, s.nodeLocation(full, sem.AST), nil)
		}
		for _, sem := range root.sem.Functions {
			syms.Add(sem.Name(), ls.KindFunction, s.nodeLocation(full, sem.AST), nil)
		}
		for _, sem := range root.sem.Globals {
			syms.Add(sem.Name(), ls.KindVariable, s.nodeLocation(full, sem.AST), nil)
		}
		for _, sem := range root.sem.Methods {
			syms.Add(sem.Name(), ls.KindMethod, s.nodeLocation(full, sem.AST), nil)
		}
		for _, sem := range root.sem.Subroutines {
			syms.Add(sem.Name(), ls.KindFunction, s.nodeLocation(full, sem.AST), nil)
		}
		for _, sem := range root.sem.Pseudonyms {
			switch underlying(sem).(type) {
			case *semantic.StaticArray:
				syms.Add(sem.Name(), ls.KindArray, s.nodeLocation(full, sem.AST), nil)
			case *semantic.Map, *semantic.Class:
				syms.Add(sem.Name(), ls.KindClass, s.nodeLocation(full, sem.AST), nil)
			case *semantic.Builtin:
				syms.Add(sem.Name(), ls.KindNumber, s.nodeLocation(full, sem.AST), nil)
			case *semantic.Enum:
				syms.Add(sem.Name(), ls.KindEnum, s.nodeLocation(full, sem.AST), nil)
			default:
				syms.Add(sem.Name(), ls.KindClass, s.nodeLocation(full, sem.AST), nil)
			}
		}
	}
	return syms, nil
}

// CodeActions compute commands for a given document and range.
// The request is triggered when the user moves the cursor into an problem
// marker in the editor or presses the lightbulb associated with a marker.
func (s *server) CodeActions(context.Context, *ls.Document, ls.Range, []ls.Diagnostic) ([]ls.Command, error) {
	return []ls.Command{}, nil
}

func findAPIs(root string) []string {
	apis := []string{}
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == apiExt {
			if path, err := filepath.Abs(path); err == nil {
				apis = append(apis, path)
			}
		}
		return nil
	})
	return apis
}

// possibleValues returns the possible values for the token at pos.
func possibleValues(da *docAnalysis, pos ls.Position) (analysis.Value, *analysis.Results) {
	for _, n := range da.walkUp(da.doc.Body().Offset(pos)) {
		for _, root := range da.full.roots {
			if root.results == nil {
				continue
			}
			switch sem := n.sem.(type) {
			case *semantic.Global:
				if val, ok := root.results.Globals[sem]; ok {
					return val, root.results
				}
			case *semantic.Parameter:
				if val, ok := root.results.Parameters[sem]; ok {
					return val, root.results
				}
			case *semantic.Field:
				var out analysis.Value
				for create, val := range root.results.Instances {
					if create.Type.To == sem.Owner() {
						val := val.(*analysis.ClassValue)
						if val, ok := val.Fields[sem.Name()]; ok {
							out = analysis.UnionOf(val, out)
						}
					}
				}
				if out != nil {
					return out, root.results
				}
			}
		}
	}
	return nil, nil
}

// partial returns the deepest partial if sem is a (or a chain of)
// semantic.Invalid, otherwise it just returns sem.
func partial(sem semantic.Node) semantic.Node {
	if invalid, ok := sem.(semantic.Invalid); ok {
		return partial(invalid.Partial)
	}
	return sem
}

// typeof returns the type of sem.
func typeof(sem semantic.Node) semantic.Type {
	switch sem := sem.(type) {
	case semantic.Type:
		return sem
	case semantic.Expression:
		return sem.ExpressionType()
	default:
		return nil
	}
}

// underlying returns the underlying type of ty.
func underlying(ty semantic.Node) semantic.Node {
	switch ty := ty.(type) {
	case *semantic.Reference:
		return underlying(ty.To)
	case *semantic.Pseudonym:
		return underlying(ty.To)
	default:
		return ty
	}
}

func astChildAt(da *docAnalysis, parent ast.Node, offset int) ast.Node {
	var child ast.Node
	ast.Visit(parent, func(n ast.Node) {
		cst := da.full.mappings.AST.CST(n)
		if cst == nil {
			return
		}
		if tok := cst.Tok(); offset >= tok.Start && offset <= tok.End {
			child = n
		}
	})
	return child
}

type nodes struct {
	ast ast.Node
	sem semantic.Node
}

func goTypename(o interface{}) string {
	if invalid, ok := o.(semantic.Invalid); ok && invalid.Partial != nil {
		return fmt.Sprintf("Invalid<%s>", goTypename(invalid.Partial))
	}
	if ty := reflect.TypeOf(o); ty != nil {
		for ty.Kind() == reflect.Ptr {
			ty = ty.Elem()
		}
		if ty.Kind() == reflect.Struct {
			return ty.Name()
		}
	}
	return fmt.Sprintf("%T", o)
}

func typename(t semantic.Type) string {
	return printer.New().WriteType(t).String()
}

func (fa *fullAnalysis) nodeLocation(doc *ls.Document, n ast.Node) ls.Location {
	return ls.Location{URI: doc.URI(), Range: fa.nodeRange(doc, n)}
}

func (fa *fullAnalysis) nodeRange(doc *ls.Document, n ast.Node) ls.Range {
	return tokRange(doc, fa.mappings.AST.CST(n).Tok())
}

func (fa *fullAnalysis) nodePosition(doc *ls.Document, n ast.Node) ls.Position {
	return tokRange(doc, fa.mappings.AST.CST(n).Tok()).Start
}

func tokRange(doc *ls.Document, tok cst.Token) ls.Range {
	return doc.Body().Range(tok.Start, tok.End)
}

func fragRange(doc *ls.Document, f cst.Fragment) ls.Range {
	if f == nil {
		return doc.Body().Range(0, 0)
	}
	return tokRange(doc, f.Tok())
}

func (fa *fullAnalysis) documentation(n ast.Node) string {
	buf := &bytes.Buffer{}
	cst := fa.mappings.AST.CST(n)
	cst.Prefix().Write(buf)
	cst.Suffix().Write(buf)
	lines := strings.Split(buf.String(), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, `//`) || strings.HasPrefix(line, `/*`) {
			line = line[2:]
		}
		if strings.HasSuffix(line, `*/`) {
			line = line[:len(line)-2]
		}
		lines[i] = line
	}
	return strings.Join(lines, " ")
}

func handlePanic(ctx context.Context) {
	if r := recover(); r != nil {
		log.E(ctx, "\n\n\n------------------------ PANIC ------------------------\n\n")
		buf := make([]byte, 64<<10)
		buf = buf[:runtime.Stack(buf, true)]
		log.E(ctx, "%v\n%s", r, string(buf))
	}
}
