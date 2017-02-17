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

package protocol

// MarkedString is a string of a specified language
type MarkedString struct {
	// The language of the string.
	Language string `json:"language"`

	// The string value.
	Value string `json:"value"`
}

// ErrorCode represents the list of errors returned in a ResponseError.
type ErrorCode int

const (
	ParseError       = ErrorCode(-32700)
	InvalidRequest   = ErrorCode(-32600)
	MethodNotFound   = ErrorCode(-32601)
	InvalidParams    = ErrorCode(-32602)
	InternalError    = ErrorCode(-32603)
	ServerErrorStart = ErrorCode(-32099)
	ServerErrorEnd   = ErrorCode(-32000)
)

// Position is a position in a text document expressed as zero-based line and character offset.
type Position struct {
	// Line position in a document (zero-based).
	Line int `json:"line"`

	// Column offset on a line in a document (zero-based).
	Column int `json:"character"`
}

// Range is a range in a text document expressed as start and end positions.
type Range struct {
	// The position of the first character in the range.
	Start Position `json:"start"`

	// One past the last character in the range.
	End Position `json:"end"`
}

// Location represents a location inside a resource, such as a line inside a text file.
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// Diagnostic represents a diagnostic, such as a compiler error or warning. Diagnostic objects are only valid in the scope of a resource.
type Diagnostic struct {
	// The range at which the message applies
	Range Range `json:"range"`

	// The diagnostic's severity. Can be omitted. If omitted it is up to the
	// client to interpret diagnostics as error, warning, info or hint.
	Severity DiagnosticSeverity `json:"severity"`

	// The diagnostic's code. Can be omitted.
	Code interface{} `json:"code"` // number | string;

	// A human-readable string describing the source of this
	// diagnostic, e.g. 'typescript' or 'super lint'.
	Source string `json:"source"`

	// The diagnostic's message.
	Message string `json:"message"`
}

// DiagnosticSeverity is an enumerator of severity values.
type DiagnosticSeverity int

const (
	// SeverityError reports an error.
	SeverityError = DiagnosticSeverity(1)

	// SeverityWarning reports a warning.
	SeverityWarning = DiagnosticSeverity(2)

	// SeverityInformation reports an information.
	SeverityInformation = DiagnosticSeverity(3)

	// SeverityHint reports a hint.
	SeverityHint = DiagnosticSeverity(4)
)

// Command represents a reference to a command. Provides a title which will be used to represent a command in the UI and, optionally, an array of arguments which will be passed to the command handler function when invoked.
type Command struct {
	// Title of the command, like `save`.
	Title string `json:"title"`

	// The identifier of the actual command handler.
	Command string `json:"command"`

	// Arguments that the command handler should be
	// invoked with.
	Arguments map[string]interface{} `json:"arguments"`
}

// TextEdit is a textual edit applicable to a text document.
type TextEdit struct {
	// The range of the text document to be manipulated. To insert
	// text into a document create a range where start === end.
	Range Range `json:"range"`

	// The string to be inserted. For delete operations use an
	// empty string.
	NewText string `json:"newText"`
}

// WorkspaceEdit represents changes to many resources managed in the workspace.
type WorkspaceEdit struct {
	// Holds changes to existing resources.
	Changes interface{} `json:"changes"` // { [uri: string]: TextEdit[]; };
}

// TextDocumentIdentifier identifies a document using an URI.
type TextDocumentIdentifier struct {
	// The text document's identifier.
	URI string `json:"uri"`
}

// TextDocumentItem is an item to transfer a text document from the client to the server.
type TextDocumentItem struct {
	// The text document's identifier.
	URI string `json:"uri"`

	// The text document's language identifier
	LanguageID string `json:"languageId"`

	// The version number of this document (it will strictly increase after each
	// change, including undo/redo).
	Version int `json:"version"`

	// The content of the opened  text document.
	Text string `json:"text"`
}

// VersionedTextDocumentIdentifier is an identifier to denote a specific version of a text document.
type VersionedTextDocumentIdentifier struct {
	// The text document's URI.
	URI string `json:"uri"`

	// The version number of this document.
	Version int `json:"version"`
}

// TextDocumentPositionParams is a parameter literal used in requests to pass a
// text document and a position inside that document.
type TextDocumentPositionParams struct {
	// The text document's URI.
	Document TextDocumentIdentifier `json:"textDocument"`

	// The position inside the text document.
	Position Position `json:"position"`
}

// ClientCapabilities are currently empty.
type ClientCapabilities struct{}

// TextDocumentSyncKind Defines how the host (editor) should sync document changes to the language server.
type TextDocumentSyncKind int

const (
	// SyncNone means documents should not be synced at all.
	SyncNone = TextDocumentSyncKind(0)

	// SyncFull means documents are synced by always sending the full content of
	// the document.
	SyncFull = TextDocumentSyncKind(1)

	// SyncIncremental means documents are synced by sending the full content on
	// open. After that only incremental updates to the document are send.
	SyncIncremental = TextDocumentSyncKind(2)
)

// CompletionOptions represent text completion options.
type CompletionOptions struct {
	// The server provides support to resolve additional information for a completion item.
	ResolveProvider bool `json:"resolveProvider"`

	// The characters that trigger completion automatically.
	TriggerCharacters []string `json:"triggerCharacters"`
}

// SignatureHelpOptions represents signature help options.
type SignatureHelpOptions struct {
	// The characters that trigger signature help automatically.
	TriggerCharacters []string `json:"triggerCharacters"`
}

// CodeLensOptions represent code Lens options.
type CodeLensOptions struct {
	// Code lens has a resolve provider as well.
	ResolveProvider bool `json:"resolveProvider"`
}

// DocumentOnTypeFormattingOptions represents formatting on type options
type DocumentOnTypeFormattingOptions struct {
	// A character on which formatting should be triggered, like `}`.
	FirstTriggerCharacter string `json:"firstTriggerCharacter"`

	// More trigger characters.
	MoreTriggerCharacter []string `json:"moreTriggerCharacter"`
}

// ServerCapabilities represents the capabilities of the language server.
type ServerCapabilities struct {
	// Defines how text documents are synced.
	TextDocumentSync TextDocumentSyncKind `json:"textDocumentSync"`

	// The server provides hover support.
	HoverProvider bool `json:"hoverProvider"`

	// The server provides completion support.
	CompletionProvider CompletionOptions `json:"completionProvider"`

	// The server provides signature help support.
	SignatureHelpProvider SignatureHelpOptions `json:"signatureHelpProvider"`

	// The server provides goto definition support.
	DefinitionProvider bool `json:"definitionProvider"`

	// The server provides find references support.
	ReferencesProvider bool `json:"referencesProvider"`

	// The server provides document highlight support.
	DocumentHighlightProvider bool `json:"documentHighlightProvider"`

	// The server provides document symbol support.
	DocumentSymbolProvider bool `json:"documentSymbolProvider"`

	// The server provides workspace symbol support.
	WorkspaceSymbolProvider bool `json:"workspaceSymbolProvider"`

	// The server provides code actions.
	CodeActionProvider bool `json:"codeActionProvider"`

	// The server provides code lens.
	CodeLensProvider *CodeLensOptions `json:"codeLensProvider"`

	// The server provides document formatting.
	DocumentFormattingProvider bool `json:"documentFormattingProvider"`

	// The server provides document range formatting.
	DocumentRangeFormattingProvider bool `json:"documentRangeFormattingProvider"`

	// The server provides document formatting on typing.
	DocumentOnTypeFormattingProvider *DocumentOnTypeFormattingOptions `json:"documentOnTypeFormattingProvider"`

	// The server provides rename support.
	RenameProvider bool `json:"renameProvider"`
}

// MessageType is an enumerator of message types that can be shown to the user.
type MessageType int

const (
	// ErrorType represents an error message.
	ErrorType = MessageType(1)

	// WarningType represents a warning message.
	WarningType = MessageType(2)

	// InfoType represents an information message.
	InfoType = MessageType(3)

	// LogType represents a log message.
	LogType = MessageType(4)
)

// MessageActionItem represents a single action that can be performed in a
// ShowMessageRequest.
type MessageActionItem struct {
	// A short title like 'Retry', 'Open Log' etc.
	Title string `json:"title"`
}

// TextDocumentContentChangeEvent is an event describing a change to a text
// document. If range and rangeLength are omitted the new text is considered to
// be the full content of the document.
type TextDocumentContentChangeEvent struct {
	// The range of the document that changed.
	Range *Range `json:"range"`

	// The length of the range that got replaced.
	RangeLength *int `json:"rangeLength"`

	// The new text of the document.
	Text string `json:"text"`
}

// FileChangeType is an enumerator of file events.
type FileChangeType int

const (
	// Created represents a file that created.
	Created = FileChangeType(1)

	// Changed represents a file that got changed.
	Changed = FileChangeType(2)

	// Deleted represents a file that got deleted.
	Deleted = FileChangeType(3)
)

// FileEvent describes a file change event.
type FileEvent struct {
	// The file's uri.
	URI string `json:"uri"`

	// The change type.
	Type FileChangeType `json:"type"`
}

// CompletionList represents a collection of CompletionItems to be presented in
// the editor.
type CompletionList struct {
	// This list it not complete. Further typing should result in recomputing
	// this list.
	IsIncomplete bool `json:"isIncomplete"`

	// The completion items.
	Items []CompletionItem `json:"items"`
}

// CompletionItem represents an item to be presented in the editor.
type CompletionItem struct {
	// The label of this completion item. By default also the text that is
	// inserted when selecting this completion.
	Label string `json:"label"`

	// The kind of this completion item. Based of the kind an icon is chosen by
	// the editor.
	Kind *CompletionItemKind `json:"kind,omitempty"`

	// A human-readable string with additional information about this item, like
	// type or symbol information.
	Detail *string `json:"detail,omitempty"`

	// A human-readable string that represents a doc-comment.
	Documentation *string `json:"documentation,omitempty"`

	// A string that should be used when comparing this item with other items.
	// When `falsy` the label is used.
	SortText *string `json:"sortText,omitempty"`

	// A string that should be used when filtering a set of completion items.
	// When `falsy` the label is used.
	FilterText *string `json:"filterText,omitempty"`

	// A string that should be inserted a document when selecting this
	// completion. When `falsy` the label is used.
	InsertText *string `json:"insertText,omitempty"`

	// An edit which is applied to a document when selecting
	// this completion. When an edit is provided the value of
	// insertText is ignored.
	TextEdit *TextEdit `json:"textEdit,omitempty"`

	// An data entry field that is preserved on a completion item between
	// a completion and a completion resolve request.
	Data interface{} `json:"data,omitempty"`
}

// CompletionItemKind is the kind of a completion entry.
type CompletionItemKind int

const (
	Text        = CompletionItemKind(1)
	Method      = CompletionItemKind(2)
	Function    = CompletionItemKind(3)
	Constructor = CompletionItemKind(4)
	Field       = CompletionItemKind(5)
	Variable    = CompletionItemKind(6)
	Class       = CompletionItemKind(7)
	Interface   = CompletionItemKind(8)
	Module      = CompletionItemKind(9)
	Property    = CompletionItemKind(10)
	Unit        = CompletionItemKind(11)
	Value       = CompletionItemKind(12)
	Enum        = CompletionItemKind(13)
	Keyword     = CompletionItemKind(14)
	Snippet     = CompletionItemKind(15)
	Color       = CompletionItemKind(16)
	File        = CompletionItemKind(17)
	Reference   = CompletionItemKind(18)
)

// SignatureInformation represents the signature of something callable.
// A signature can have a label, like a function-name, a doc-comment, and
// a set of parameters.
type SignatureInformation struct {
	// The label of this signature. Will be shown in the UI.
	Label string `json:"label"`

	// The human-readable doc-comment of this signature. Will be shown
	// in the UI but can be omitted.
	Documentation *string `json:"documentation,omitempty"`

	// The parameters of this signature.
	Parameters []ParameterInformation `json:"parameters"`
}

// ParameterInformation represents a parameter of a callable-signature.
// A parameter can have a label and a doc-comment.
type ParameterInformation struct {
	// The label of this signature. Will be shown in the UI.
	Label string `json:"label"`

	// he human-readable doc-comment of this signature.
	// Will be shown in the UI but can be omitted.
	Documentation *string `json:"documentation,omitempty"`
}

// DocumentHighlight is a range inside a text document which deserves
// special attention. Usually a document highlight is visualized by changing
// the background color of its range.
type DocumentHighlight struct {
	// The range this highlight applies to.
	Range Range `json:"range"`

	// The highlight kind, default is DocumentHighlightKind.Text.
	Kind *DocumentHighlightKind `json:"kind"`
}

// DocumentHighlightKind is a document highlight kind enumerator.
type DocumentHighlightKind int

const (
	// TextHighlight represents a textual occurrance.
	TextHighlight = DocumentHighlightKind(1)

	// ReadHighlight represents read-access of a symbol, like reading a
	// variable.
	ReadHighlight = DocumentHighlightKind(2)

	// WriteHighlight represents write-access of a symbol, like writing to a
	// variable.
	WriteHighlight = DocumentHighlightKind(3)
)

// SymbolInformation represents information about programming constructs like variables, classes,
// interfaces etc.
type SymbolInformation struct {
	// The name of this symbol.
	Name string `json:"name"`

	// The kind of this symbol.
	Kind SymbolKind `json:"kind"`

	// The location of this symbol.
	Location Location `json:"location"`

	// The name of the symbol containing this symbol.
	ContainerName *string `json:"containerName"`
}

// SymbolKind is an enumerator of symbol kinds.
type SymbolKind int

const (
	KindFile        = SymbolKind(1)
	KindModule      = SymbolKind(2)
	KindNamespace   = SymbolKind(3)
	KindPackage     = SymbolKind(4)
	KindClass       = SymbolKind(5)
	KindMethod      = SymbolKind(6)
	KindProperty    = SymbolKind(7)
	KindField       = SymbolKind(8)
	KindConstructor = SymbolKind(9)
	KindEnum        = SymbolKind(10)
	KindInterface   = SymbolKind(11)
	KindFunction    = SymbolKind(12)
	KindVariable    = SymbolKind(13)
	KindConstant    = SymbolKind(14)
	KindString      = SymbolKind(15)
	KindNumber      = SymbolKind(16)
	KindBoolean     = SymbolKind(17)
	KindArray       = SymbolKind(18)
)

// CodeActionContext contains additional diagnostic information about the
// context in which a code action is run.
type CodeActionContext struct {
	// An array of diagnostics.
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// CodeLens represents a command that should be shown along with source text,
// like the number of references, a way to run tests, etc.
//
// A code lens is _unresolved_ when no command is associated to it. For
// performance reasons the creation of a code lens and resolving should be done
// to two stages.
type CodeLens struct {
	// The range in which this code lens is valid. Should only span a single line.
	Range Range `json:"range"`

	// The command this code lens represents.
	Command *Command `json:"command"`

	// An data entry field that is preserved on a code lens item between
	// a code lens and a code lens resolve request.
	Data interface{} `json:"data"`
}

// FormattingOptions describes what options formatting should use.
type FormattingOptions struct {
	// Size of a tab in spaces.
	TabSize int `json:"tabSize"`

	// Prefer spaces over tabs.
	InsertSpaces bool `json:"insertSpaces"`

	// Signature for further properties.
	// [key: string]: boolean | number | string;
}
