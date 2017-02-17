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
	"io"
	"io/ioutil"
	"strings"
	"unicode/utf8"

	"github.com/google/gapid/core/context/jot"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/fault/severity"
	"github.com/google/gapid/core/langsvr/protocol"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/text/note"
)

// InitConfig is returned by Server.Initialize().
type InitConfig struct {
	// The language identifier for this language server.
	LanguageID string

	// CompletionTriggerCharacters is the list of characters that will trigger a
	// completion suggestion.
	CompletionTriggerCharacters []rune

	// CompletionTriggerCharacters is the list of characters that will trigger a
	// signature hint.
	SignatureTriggerCharacters []rune

	// Documents is a list of all the document paths that should be watched from
	// initialization.
	WorkspaceDocuments []string
}

// Server is the interface implemented by language servers.
type Server interface {
	// Initialize is called when the server is first initialized by the client.
	Initialize(ctx log.Context, rootPath string) (InitConfig, error)

	// Shutdown is called to shutdown the server.
	Shutdown(log.Context) error

	// OnConfigChange is called when the configuration settings change.
	OnConfigChange(log.Context, map[string]interface{}) error

	// OnDocumentsAdded is called when new documents of interest are discovered.
	OnDocumentsAdded(log.Context, []*Document) error

	// OnDocumentsChanged is called when documents are changed.
	OnDocumentsChanged(log.Context, []*Document) error

	// OnDocumentsRemoved is called when documents are no longer monitored.
	OnDocumentsRemoved(log.Context, []*Document) error

	// OnDocumentSaved is called when an open document is saved.
	OnDocumentSaved(log.Context, *Document) error
}

// HoverProvider is the interface implemented by servers that support hover
// information.
type HoverProvider interface {
	// Hover returns a list of source code snippets and range for the given
	// symbol at the specified position.
	Hover(log.Context, *Document, Position) (SourceCodeList, Range, error)
}

// CompletionProvider is the interface implemented by servers that support
// completion information.
type CompletionProvider interface {
	// Completions returns completion items at a given cursor position.
	// Completion items are presented in the IntelliSense user interface.
	Completions(log.Context, *Document, Position) (CompletionList, error)
}

// SignatureProvider is the interface implemented by servers that support
// callable signature information.
type SignatureProvider interface {
	// Signatures returns the list of function signatures that are candidates
	// at the given cursor position. The activeSig and activeParam are indices
	// of the signature and parameter to highlight for the given cursor
	// position.
	Signatures(log.Context, *Document, Position) (sigs SignatureList, activeSig, activeParam int, err error)
}

// DefinitionProvider is the interface implemented by servers that support
// symbol definition information.
type DefinitionProvider interface {
	// Definitions returns the list of definition locations for the given symbol
	// in the specified document at position.
	Definitions(log.Context, *Document, Position) ([]Location, error)
}

// ReferencesProvider is the interface implemented by servers that support
// symbol reference information.
type ReferencesProvider interface {
	// References returns a list of references for the given symbol in the
	// specified document at position.
	References(log.Context, *Document, Position) ([]Location, error)
}

// HighlightsProvider is the interface implemented by servers that support
// symbol highlight information.
type HighlightsProvider interface {
	// Highlights returns a list of highlights for the given symbol in the
	// specified document at position.
	Highlights(log.Context, *Document, Position) (HighlightList, error)
}

// SymbolsProvider is the interface implemented by servers that support
// document symbol information.
type SymbolsProvider interface {
	// Symbols returns symbolic information about the specified document.
	Symbols(log.Context, *Document) (SymbolList, error)
}

// WorkspaceSymbolsProvider is the interface implemented by servers that support
// whole-workspace symbol information.
type WorkspaceSymbolsProvider interface {
	// WorkspaceSymbols returns all project-wide symbols.
	WorkspaceSymbols(ctx log.Context) (SymbolList, error)
}

// CodeActionsProvider is the interface implemented by servers that support
// code actions.
type CodeActionsProvider interface {
	// CodeActions compute commands for a given document and range.
	// The request is triggered when the user moves the cursor into an problem
	// marker in the editor or presses the lightbulb associated with a marker.
	CodeActions(log.Context, *Document, Range, []Diagnostic) ([]Command, error)
}

// CodeLensProvider is the interface implemented by servers that support
// code lenes.
type CodeLensProvider interface {
	// CodeLenses returns a list of CodeLens for the specified document.
	CodeLenses(log.Context, *Document) ([]CodeLens, error)
}

// FormatProvider is the interface implemented by servers that support
// whole-document reformatting.
type FormatProvider interface {
	// Format returns a list of edits required to format the entire document.
	Format(ctx log.Context, doc *Document, opts FormattingOptions) (TextEditList, error)
}

// FormatRangeProvider is the interface implemented by servers that support
// document-range reformatting.
type FormatRangeProvider interface {
	// FormatRange returns a list of edits required to format the specified
	// range in the specified document.
	FormatRange(ctx log.Context, doc *Document, rng Range, opts FormattingOptions) (TextEditList, error)
}

// FormatOnTypeProvider is the interface implemented by servers that support
// reformatting as-you-type.
type FormatOnTypeProvider interface {
	// FormatOnType returns a list of edits required to format the code
	// currently being written at pos, after char was typed.
	FormatOnType(doc *Document, pos Position, char rune, opts FormattingOptions) (TextEditList, error)
}

// RenameProvider is the interface implemented by servers that support renaming
// symbols.
type RenameProvider interface {
	// Rename is called to rename the symbol at pos with newName.
	Rename(ctx log.Context, doc *Document, pos Position, newName string) (WorkspaceEdit, error)
}

// Connect creates a connection to between server and the client (code editor)
// communicating on stream.
func Connect(ctx log.Context, stream io.ReadWriter, server Server) error {
	ctx, terminate := task.WithCancel(ctx)
	conn := protocol.NewConnection(stream)
	ls := &langsvr{
		conn,
		server,
		make(map[string]*Document),
		"",
		terminate,
	}
	handler := log.GetHandler(ctx)
	ctx = ctx.Handler(func(page note.Page) error {
		msg := note.Raw.Print(page)
		if ls.languageID != "" {
			fmt.Sprintf("%v: %v", ls.languageID, msg)
		}
		switch severity.FindLevel(page) {
		case severity.Emergency, severity.Alert, severity.Critical, severity.Error:
			conn.ShowMessage(protocol.ErrorType, msg)
		case severity.Warning:
			conn.LogMessage(protocol.WarningType, msg)
		case severity.Notice, severity.Info:
			conn.LogMessage(protocol.InfoType, msg)
		case severity.Debug:
			conn.LogMessage(protocol.LogType, msg)
		}
		return handler(page)
	})
	return conn.Serve(ctx, ls)
}

// langsvr is an implementation of protocol.Server.
type langsvr struct {
	conn       *protocol.Connection
	server     Server
	documents  map[string]*Document // uri -> Document
	languageID string
	terminate  func()
}

func runesToStrings(in []rune) []string {
	out := make([]string, len(in))
	for i, r := range in {
		out[i] = string(r)
	}
	return out
}

func (s *langsvr) Initialize(ctx log.Context, processID int, rootPath string) (protocol.ServerCapabilities, error) {
	ctx = ctx.Enter("Initialize")
	cfg, err := s.server.Initialize(ctx, rootPath)
	if err != nil {
		return protocol.ServerCapabilities{}, err
	}

	s.languageID = cfg.LanguageID

	added := make([]*Document, 0, len(cfg.WorkspaceDocuments))
	for _, path := range cfg.WorkspaceDocuments {
		ctx := ctx.S("path", path)
		body, err := ioutil.ReadFile(path)
		if err != nil {
			ctx.Error().Log("Couldn't open workspace document")
			continue
		}
		doc := s.newDocument(PathToURI(path), cfg.LanguageID, 0, NewBody(string(body)))
		doc.watched = true
		added = append(added, doc)
	}
	s.server.OnDocumentsAdded(ctx, added)

	caps := protocol.ServerCapabilities{
		TextDocumentSync: protocol.SyncIncremental,
	}
	_, caps.HoverProvider = s.server.(HoverProvider)
	_, caps.DefinitionProvider = s.server.(DefinitionProvider)
	_, caps.ReferencesProvider = s.server.(ReferencesProvider)
	_, caps.DocumentHighlightProvider = s.server.(HighlightsProvider)
	_, caps.DocumentSymbolProvider = s.server.(SymbolsProvider)
	_, caps.WorkspaceSymbolProvider = s.server.(WorkspaceSymbolsProvider)
	_, caps.CodeActionProvider = s.server.(CodeActionsProvider)
	_, caps.DocumentFormattingProvider = s.server.(FormatProvider)
	_, caps.DocumentRangeFormattingProvider = s.server.(FormatRangeProvider)
	_, caps.RenameProvider = s.server.(RenameProvider)
	if _, ok := s.server.(CompletionProvider); ok {
		caps.CompletionProvider = protocol.CompletionOptions{
			ResolveProvider:   true,
			TriggerCharacters: runesToStrings(cfg.CompletionTriggerCharacters),
		}
	}
	if _, ok := s.server.(SignatureProvider); ok {
		caps.SignatureHelpProvider = protocol.SignatureHelpOptions{
			TriggerCharacters: runesToStrings(cfg.SignatureTriggerCharacters),
		}
	}
	if _, ok := s.server.(CodeLensProvider); ok {
		caps.CodeLensProvider = &protocol.CodeLensOptions{
			ResolveProvider: true,
		}
	}
	if _, ok := s.server.(FormatOnTypeProvider); ok {
		caps.DocumentOnTypeFormattingProvider = &protocol.DocumentOnTypeFormattingOptions{
			FirstTriggerCharacter: "",
			MoreTriggerCharacter:  []string{},
		}
	}
	return caps, nil

}

func (s langsvr) Shutdown(ctx log.Context) error {
	ctx = ctx.Enter("Shutdown")
	return s.server.Shutdown(ctx)
}

func (s langsvr) Completion(ctx log.Context, uri protocol.TextDocumentIdentifier, position protocol.Position) (protocol.CompletionList, error) {
	ctx = ctx.Enter("Completion")
	cp, ok := s.server.(CompletionProvider)
	if !ok {
		return protocol.CompletionList{}, nil
	}
	doc, err := s.getDoc(uri.URI)
	if err != nil {
		return protocol.CompletionList{}, err
	}
	list, err := cp.Completions(ctx, doc, pos(position))
	if err != nil {
		return protocol.CompletionList{}, err
	}
	return list.toProtocol(), nil
}

func (s langsvr) CompletionItemResolve(ctx log.Context, item protocol.CompletionItem) (protocol.CompletionItem, error) {
	ctx = ctx.Enter("CompletionItemResolve")
	return item, nil
}

func (s langsvr) Hover(ctx log.Context, uri protocol.TextDocumentIdentifier, position protocol.Position) ([]protocol.MarkedString, *protocol.Range, error) {
	ctx = ctx.Enter("Hover")
	hp, ok := s.server.(HoverProvider)
	if !ok {
		return []protocol.MarkedString{}, nil, nil
	}
	doc, err := s.getDoc(uri.URI)
	if err != nil {
		return nil, nil, err
	}
	sources, rng, err := hp.Hover(ctx, doc, pos(position))
	if err != nil {
		return nil, nil, err
	}
	rngOut := rng.toProtocol()
	return sources.toProtocol(), &rngOut, nil
}

func (s langsvr) SignatureHelp(ctx log.Context, uri protocol.TextDocumentIdentifier, position protocol.Position) ([]protocol.SignatureInformation, *int, *int, error) {
	ctx = ctx.Enter("SignatureHelp")
	sp, ok := s.server.(SignatureProvider)
	if !ok {
		return []protocol.SignatureInformation{}, nil, nil, nil
	}
	doc, err := s.getDoc(uri.URI)
	if err != nil {
		return nil, nil, nil, err
	}
	sigs, activeSig, activeParam, err := sp.Signatures(ctx, doc, pos(position))
	if err != nil {
		return nil, nil, nil, err
	}
	if len(sigs) > 0 {
		return sigs.toProtocol(), &activeSig, &activeParam, nil
	}
	return []protocol.SignatureInformation{}, nil, nil, err
}

func (s langsvr) GotoDefinition(ctx log.Context, uri protocol.TextDocumentIdentifier, position protocol.Position) ([]protocol.Location, error) {
	ctx = ctx.Enter("GotoDefinition")
	dp, ok := s.server.(DefinitionProvider)
	if !ok {
		return []protocol.Location{}, nil
	}
	doc, err := s.getDoc(uri.URI)
	if err != nil {
		return nil, err
	}
	locations, err := dp.Definitions(ctx, doc, pos(position))
	if err != nil {
		return nil, err
	}
	out := make([]protocol.Location, len(locations))
	for i, l := range locations {
		out[i] = l.toProtocol()
	}
	return out, nil
}

func (s langsvr) FindReferences(ctx log.Context, uri protocol.TextDocumentIdentifier, position protocol.Position, includeDecl bool) ([]protocol.Location, error) {
	ctx = ctx.Enter("FindReferences")
	rp, ok := s.server.(ReferencesProvider)
	if !ok {
		return []protocol.Location{}, nil
	}
	doc, err := s.getDoc(uri.URI)
	if err != nil {
		return nil, err
	}
	references, err := rp.References(ctx, doc, pos(position))
	if err != nil {
		return nil, err
	}
	out := make([]protocol.Location, len(references))
	for i, r := range references {
		out[i] = r.toProtocol()
	}
	return out, nil
}

func (s langsvr) DocumentHighlights(ctx log.Context, uri protocol.TextDocumentIdentifier, position protocol.Position) ([]protocol.DocumentHighlight, error) {
	ctx = ctx.Enter("DocumentHighlights")
	hp, ok := s.server.(HighlightsProvider)
	if !ok {
		return []protocol.DocumentHighlight{}, nil
	}
	doc, err := s.getDoc(uri.URI)
	if err != nil {
		return nil, err
	}
	highlights, err := hp.Highlights(ctx, doc, pos(position))
	return highlights.toProtocol(), err
}

func (s langsvr) DocumentSymbols(ctx log.Context, docID protocol.TextDocumentIdentifier) ([]protocol.SymbolInformation, error) {
	ctx = ctx.Enter("DocumentSymbols")
	sp, ok := s.server.(SymbolsProvider)
	if !ok {
		return []protocol.SymbolInformation{}, nil
	}
	doc, err := s.getDoc(docID.URI)
	if err != nil {
		return nil, err
	}
	symbols, err := sp.Symbols(ctx, doc)
	if err != nil {
		return nil, err
	}
	return symbols.toProtocol(), nil
}

func (s langsvr) WorkspaceSymbols(ctx log.Context, query string) ([]protocol.SymbolInformation, error) {
	ctx = ctx.Enter("WorkspaceSymbols")
	sp, ok := s.server.(WorkspaceSymbolsProvider)
	if !ok {
		return []protocol.SymbolInformation{}, nil
	}
	symbols, err := sp.WorkspaceSymbols(ctx)
	if err != nil {
		return nil, err
	}
	query = strings.ToLower(query)
	symbols = symbols.Filter(func(s Symbol) bool { return strings.Contains(strings.ToLower(s.Name), query) })
	return symbols.toProtocol(), nil
}

func (s langsvr) CodeAction(ctx log.Context, docID protocol.TextDocumentIdentifier, inRng protocol.Range, context protocol.CodeActionContext) ([]protocol.Command, error) {
	ctx = ctx.Enter("CodeAction")
	doc, err := s.getDoc(docID.URI)
	if err != nil {
		return nil, err
	}
	cp, ok := s.server.(CodeActionsProvider)
	if !ok {
		return []protocol.Command{}, nil
	}
	diagnostics := make([]Diagnostic, len(context.Diagnostics))
	for i, diag := range context.Diagnostics {
		diagnostics[i] = diagnostic(diag)
	}
	commands, err := cp.CodeActions(ctx, doc, rng(inRng), diagnostics)
	if err != nil {
		return nil, err
	}
	out := make([]protocol.Command, len(commands))
	for i, cmd := range commands {
		out[i] = cmd.toProtocol()
	}
	return out, nil
}

func (s langsvr) CodeLens(ctx log.Context, docID protocol.TextDocumentIdentifier) ([]protocol.CodeLens, error) {
	ctx = ctx.Enter("CodeLens")
	doc, err := s.getDoc(docID.URI)
	if err != nil {
		return nil, err
	}
	cp, ok := s.server.(CodeLensProvider)
	if !ok {
		return []protocol.CodeLens{}, nil
	}
	cls, err := cp.CodeLenses(ctx, doc)
	if err != nil {
		return nil, err
	}
	out := make([]protocol.CodeLens, len(cls))
	for i, cl := range cls {
		out[i] = protocol.CodeLens{
			Range: cl.Range.toProtocol(),
			Data:  i,
		}
	}
	return out, nil
}

func (s langsvr) CodeLensResolve(ctx log.Context, codelens protocol.CodeLens) (protocol.CodeLens, error) {
	ctx = ctx.Enter("CodeLensResolve")
	return codelens, nil
}

func (s langsvr) DocumentFormatting(ctx log.Context, item protocol.TextDocumentIdentifier, opts protocol.FormattingOptions) ([]protocol.TextEdit, error) {
	ctx = ctx.Enter("DocumentFormatting")
	fp, ok := s.server.(FormatProvider)
	if !ok {
		return []protocol.TextEdit{}, nil
	}
	doc, err := s.getDoc(item.URI)
	if err != nil {
		jot.Fail(ctx, err, "Unknown document")
		return []protocol.TextEdit{}, nil
	}
	edits, err := fp.Format(ctx, doc, FormattingOptions{
		TabSize:      opts.TabSize,
		InsertSpaces: opts.InsertSpaces,
	})
	if err != nil {
		return nil, err
	}
	return edits.toProtocol(), nil
}

func (s langsvr) DocumentRangeFormatting(ctx log.Context, item protocol.TextDocumentIdentifier, fmtrng protocol.Range, opts protocol.FormattingOptions) ([]protocol.TextEdit, error) {
	ctx = ctx.Enter("DocumentRangeFormatting")
	fp, ok := s.server.(FormatRangeProvider)
	if !ok {
		return []protocol.TextEdit{}, nil
	}
	doc, err := s.getDoc(item.URI)
	if err != nil {
		jot.Fail(ctx, err, "Unknown document")
		return []protocol.TextEdit{}, nil
	}
	edits, err := fp.FormatRange(ctx, doc, rng(fmtrng), FormattingOptions{
		TabSize:      opts.TabSize,
		InsertSpaces: opts.InsertSpaces,
	})
	if err != nil {
		return nil, err
	}
	return edits.toProtocol(), nil
}

func (s langsvr) DocumentOnTypeFormatting(ctx log.Context, item protocol.TextDocumentIdentifier, position protocol.Position, char string, opts protocol.FormattingOptions) ([]protocol.TextEdit, error) {
	ctx = ctx.Enter("DocumentOnTypeFormatting")
	fp, ok := s.server.(FormatOnTypeProvider)
	if !ok {
		return []protocol.TextEdit{}, nil
	}
	doc, err := s.getDoc(item.URI)
	if err != nil {
		jot.Fail(ctx, err, "Unknown document")
		return []protocol.TextEdit{}, nil
	}
	r, _ := utf8.DecodeRuneInString(char)
	edits, err := fp.FormatOnType(doc, pos(position), r, FormattingOptions{
		TabSize:      opts.TabSize,
		InsertSpaces: opts.InsertSpaces,
	})
	if err != nil {
		return nil, err
	}
	return edits.toProtocol(), nil
}

func (s langsvr) Rename(ctx log.Context, item protocol.TextDocumentIdentifier, position protocol.Position, newName string) (protocol.WorkspaceEdit, error) {
	ctx = ctx.Enter("Rename")
	rp, ok := s.server.(RenameProvider)
	if !ok {
		return protocol.WorkspaceEdit{Changes: []protocol.TextEdit{}}, nil
	}
	doc, err := s.getDoc(item.URI)
	if err != nil {
		jot.Fail(ctx, err, "Unknown document")
		return protocol.WorkspaceEdit{Changes: []protocol.TextEdit{}}, nil
	}
	edits, err := rp.Rename(ctx, doc, pos(position), newName)
	return edits.toProtocol(), nil
}

func (s langsvr) OnExit(ctx log.Context) error {
	ctx = ctx.Enter("OnExit")
	s.terminate()
	return nil
}

func (s langsvr) OnChangeConfiguration(ctx log.Context, config map[string]interface{}) {
	ctx = ctx.Enter("OnChangeConfiguration")
	s.server.OnConfigChange(ctx, config)
}

func (s langsvr) OnOpenTextDocument(ctx log.Context, item protocol.TextDocumentItem) {
	ctx = ctx.Enter("OnOpenTextDocument")
	doc := s.documents[item.URI]
	if doc == nil {
		doc = s.newDocument(item.URI, item.LanguageID, item.Version, NewBody(item.Text))
		s.server.OnDocumentsAdded(ctx, []*Document{doc})
	}
	doc.open = true
}

func (s langsvr) OnChangeTextDocument(ctx log.Context, item protocol.VersionedTextDocumentIdentifier, changes []protocol.TextDocumentContentChangeEvent) {
	ctx = ctx.Enter("OnChangeTextDocument")
	doc, err := s.getDoc(item.URI)
	if err != nil {
		jot.Fail(ctx, err, "Unknown document")
		return
	}
	body := doc.Body()
	for _, change := range changes {
		start := body.offset(change.Range.Start)
		end := body.offset(change.Range.End)
		change := []rune(change.Text)
		runes := make([]rune, 0, len(body.Runes())-(end-start)+len(change))
		runes = append(runes, body.Runes()[:start]...)
		runes = append(runes, change...)
		runes = append(runes, body.Runes()[end:]...)
		body = NewBodyFromRunes(runes)
	}
	// Each document's body must be immutable.
	// Make a copy, and replace the existing entry.
	clone := *doc
	doc = &clone
	doc.body = body
	doc.version = item.Version
	s.documents[doc.uri] = doc
	s.server.OnDocumentsChanged(ctx, []*Document{doc})
}

func (s langsvr) OnCloseTextDocument(ctx log.Context, docID protocol.TextDocumentIdentifier) {
	ctx = ctx.Enter("OnCloseTextDocument")
	doc, err := s.getDoc(docID.URI)
	if err != nil {
		jot.Fail(ctx, err, "Unknown document")
		return
	}
	doc.open = false
	if !doc.watched {
		s.server.OnDocumentsRemoved(ctx, []*Document{doc})
		delete(s.documents, docID.URI)
	}
}

func (s langsvr) OnSaveTextDocument(ctx log.Context, docID protocol.TextDocumentIdentifier) {
	ctx = ctx.Enter("OnSaveTextDocument")
	doc, err := s.getDoc(docID.URI)
	if err != nil {
		jot.Fail(ctx, err, "Unknown document")
		return
	}
	s.server.OnDocumentSaved(ctx, doc)
}

func (s langsvr) OnChangeWatchedFiles(ctx log.Context, changes []protocol.FileEvent) {
	ctx = ctx.Enter("OnChangeWatchedFiles")

	created := make([]*Document, 0, len(changes))
	for _, change := range changes {
		if change.Type == protocol.Created {
			path, err := URItoPath(change.URI)
			if err != nil {
				jot.Fail(ctx, err, "Couldn't get path from change URI")
				continue
			}
			body, err := ioutil.ReadFile(path)
			if err != nil {
				jot.Fail(ctx, err, "Couldn't read created file")
				continue
			}
			doc := s.newDocument(change.URI, s.languageID, 0, NewBody(string(body)))
			doc.watched = true
			created = append(created, doc)
		}
	}
	if len(created) > 0 {
		s.server.OnDocumentsAdded(ctx, created)
	}

	changed := make([]*Document, 0, len(changes))
	for _, change := range changes {
		if change.Type == protocol.Changed {
			doc, err := s.getDoc(change.URI)
			if err != nil {
				jot.Fail(ctx, err, "Unknown document")
				continue
			}
			if doc.open {
				// Changes in open documents will be handled by OnChangeTextDocument.
				continue
			}
			body, err := ioutil.ReadFile(doc.path)
			if err != nil {
				jot.Fail(ctx, err, "Couldn't read changed file")
				continue
			}
			doc.body = NewBody(string(body))
			changed = append(changed, doc)
		}
	}
	if len(changed) > 0 {
		s.server.OnDocumentsChanged(ctx, changed)
	}

	deleted := make([]*Document, 0, len(changes))
	for _, change := range changes {
		if change.Type == protocol.Deleted {
			doc, err := s.getDoc(change.URI)
			if err != nil {
				jot.Fail(ctx, err, "Unknown document")
				continue
			}
			doc.watched = false
			if !doc.open {
				deleted = append(deleted, doc)
				delete(s.documents, change.URI)
			}
		}
	}
	if len(deleted) > 0 {
		s.server.OnDocumentsRemoved(ctx, deleted)
	}
}

func (s langsvr) getDoc(uri string) (*Document, error) {
	doc, ok := s.documents[uri]
	if !ok {
		return nil, protocol.Error{Code: protocol.InvalidRequest, Message: "Unknown document ID"}
	}
	return doc, nil
}
