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

// RequestMessageHeader is the common part of all request messages.
type RequestMessageHeader struct {
	Message

	// The request id.
	ID interface{} `json:"id"` // number | string

	// The method to be invoked.
	Method string `json:"method"`
}

// Request is the interface implemented by all request types.
type Request interface {
	// RequestID returns the identifier of the request.
	RequestID() interface{}
}

// RequestID returns the identifier of the request.
func (h RequestMessageHeader) RequestID() interface{} {
	return h.ID
}

// ResponseMessageHeader is the common part of all response messages.
type ResponseMessageHeader struct {
	Message

	// The request identifier that this is a response to.
	ID interface{} `json:"id"` // number | string
}

// ResponseErrorHeader is the common part of all errors that can be returned as
// part of a response.
type ResponseErrorHeader struct {
	// A number indicating the error type that occured.
	Code ErrorCode `json:"code"`

	// A string providing a short decription of the error.
	Message string `json:"message"`
}

// InitializeRequest represents an 'initialize' request.
// The initialize request is send from the client to the server.
type InitializeRequest struct {
	RequestMessageHeader

	Params struct {
		// The process Id of the parent process that started the server.
		ProcessID int `json:"processId"`

		// The rootPath of the workspace. Is null if no folder is open.
		RootPath string `json:"rootPath"`

		// The capabilities provided by the client (editor)
		Capabilities ClientCapabilities `json:"capabilities"`
	} `json:"params"`
}

// InitializeResponse is the result of an 'initialize' request.
type InitializeResponse struct {
	ResponseMessageHeader

	// The result of the request.
	Result *struct {
		// The capabilities the language server provides.
		Capabilities ServerCapabilities `json:"capabilities"`
	} `json:"result,omitempty"`

	Error *struct {
		ResponseErrorHeader

		Data struct {
			// Indicates whether the client should retry to send the
			// initilize request after showing the message provided
			// in the ResponseError.
			Retry bool `json:"retry"`
		} `json:"data"`
	} `json:"error,omitempty"`
}

// ShutdownRequest represents the 'shutdown' request.
// The 'shutdown' request is sent from the client to the server.
// It asks the server to shutdown, but to not exit (otherwise the response
// might not be delivered correctly to the client). There is a separate exit
// notification that asks the server to exit.
type ShutdownRequest struct {
	RequestMessageHeader
}

// ShutdownResponse is the response to a shutdown request.
type ShutdownResponse struct {
	ResponseMessageHeader

	// Always nil, but needs to be present.
	Result *struct{} `json:"result"`

	// Code and message set in case an exception happens during the request.
	Error *ResponseErrorHeader `json:"error,omitempty"`
}

// ShowMessageRequest is a request sent from the server to client to ask the
// client to display a particular message in the user interface.
// In addition to the show message notification the request allows to pass
// actions and to wait for an answer from the client.
type ShowMessageRequest struct {
	RequestMessageHeader

	Params struct {
		// The message type.
		Type MessageType `json:"type"`

		// The actual message
		Message string `json:"message"`

		// The message action items to present.
		Actions []MessageActionItem `json:"actions"`
	} `json:"params"`
}

// ShowMessageResponse is the response to a show message request.
type ShowMessageResponse struct {
	ResponseMessageHeader

	// Always nil, but needs to be present.
	Result *struct{} `json:"result"`

	// Code and message set in case an exception happens during the request.
	Error *ResponseErrorHeader `json:"error,omitempty"`
}

// CompletionRequest is a request sent from the client to the server to compute
// completion items at a given cursor position. Completion items are presented
// in the IntelliSense user interface. If computing complete completion items is
// expensive, servers can additional provide a handler for the resolve
// completion item request. This request is send when a completion item is
// selected in the user interface.
type CompletionRequest struct {
	RequestMessageHeader

	Params TextDocumentPositionParams `json:"params"`
}

// CompletionResponse is the response to a completion request.
type CompletionResponse struct {
	ResponseMessageHeader

	Result interface{} `json:"result,omitempty"` // CompletionItem[] | CompletionList

	// Code and message set in case an exception happens during the request.
	Error *ResponseErrorHeader `json:"error,omitempty"`
}

// CompletionItemResolveRequest is a request sent from the client to the server
// to resolve additional information for a given completion item.
type CompletionItemResolveRequest struct {
	RequestMessageHeader

	Params CompletionItem `json:"params"`
}

// CompletionItemResolveResponse is the response to a completion item resolve
// request.
type CompletionItemResolveResponse struct {
	ResponseMessageHeader

	Result *CompletionItem `json:"result,omitempty"`

	// Code and message set in case an exception happens during the request.
	Error *ResponseErrorHeader `json:"error,omitempty"`
}

// HoverRequest is a request sent from the client to the server to request hover
// information at a given text document position.
type HoverRequest struct {
	RequestMessageHeader

	Params TextDocumentPositionParams `json:"params"`
}

// HoverResponse is the response to a hover request.
type HoverResponse struct {
	ResponseMessageHeader

	Result struct {
		// The hover's content
		Contents interface{} `json:"contents"` // MarkedString | []MarkedString

		// An optional range
		Range *Range `json:"range"`
	} `json:"result"`

	// Code and message set in case an exception happens during the request.
	Error *ResponseErrorHeader `json:"error,omitempty"`
}

// SignatureHelpRequest is the request is sent from the client to the server to
// request signature information at a given cursor position.
type SignatureHelpRequest struct {
	RequestMessageHeader

	Params TextDocumentPositionParams `json:"params"`
}

// SignatureHelpResponse is the response to a signature help request.
type SignatureHelpResponse struct {
	ResponseMessageHeader

	// Represents the signature of something callable. There can be multiple
	// signatures but only one active and only one active parameter.
	Result *struct {
		// One or more signatures.
		Signatures []SignatureInformation `json:"signatures"`

		// The active signature.
		ActiveSignature *int `json:"activeSignature,omitempty"`

		// The active parameter of the active signature.
		ActiveParameter *int `json:"activeParameter,omitempty"`
	} `json:"result,omitempty"`

	// Code and message set in case an exception happens during the request.
	Error *ResponseErrorHeader `json:"error,omitempty"`
}

// GotoDefinitionRequest is a request sent from the client to the server to
// resolve the defintion location of a symbol at a given text document position.
type GotoDefinitionRequest struct {
	RequestMessageHeader

	Params TextDocumentPositionParams `json:"params"`
}

// GotoDefinitionResponse is the response to a goto defintion request.
type GotoDefinitionResponse struct {
	ResponseMessageHeader

	Result interface{} `json:"result,omitempty"` //  Location | Location[]

	// Code and message set in case an exception happens during the request.
	Error *ResponseErrorHeader `json:"error,omitempty"`
}

// FindReferencesRequest is a request sent from the client to the server to
// resolve project-wide references for the symbol denoted by the given text
// document position.
type FindReferencesRequest struct {
	RequestMessageHeader

	Params struct {
		TextDocumentPositionParams

		Context struct {
			// Include the declaration of the current symbol.
			IncludeDeclaration bool `json:"includeDeclaration"`
		} `json:"context"`
	} `json:"params"`
}

// FindReferencesResponse is the response to a find references request.
type FindReferencesResponse struct {
	ResponseMessageHeader

	// TODO: Is this correct?
	Result interface{} `json:"result,omitempty"` //  Location | Location[]

	// Code and message set in case an exception happens during the request.
	Error *ResponseErrorHeader `json:"error,omitempty"`
}

// DocumentHighlightRequest is a request sent from the client to the server to
// to resolve document highlights for a given text document position.
type DocumentHighlightRequest struct {
	RequestMessageHeader

	Params TextDocumentPositionParams `json:"params"`
}

// DocumentHighlightResponse is the response to a document highlight request.
type DocumentHighlightResponse struct {
	ResponseMessageHeader

	// The list of document highlights.
	Result []DocumentHighlight `json:"result"`

	// Code and message set in case an exception happens during the request.
	Error *ResponseErrorHeader `json:"error,omitempty"`
}

// DocumentSymbolRequest is a request sent from the client to the server to
// list all symbols found in a given text document.
type DocumentSymbolRequest struct {
	RequestMessageHeader

	Params struct {
		// The text document.
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	} `json:"params"`
}

// DocumentSymbolResponse is the response to a document symbol request.
type DocumentSymbolResponse struct {
	ResponseMessageHeader

	Result []SymbolInformation `json:"result"`

	// Code and message set in case an exception happens during the request.
	Error *ResponseErrorHeader `json:"error,omitempty"`
}

// WorkspaceSymbolRequest is a request sent from the client to the server to
// list project-wide symbols matching the query string.
type WorkspaceSymbolRequest struct {
	RequestMessageHeader

	Params struct {
		// A non-empty query string
		Query string `json:"query"`
	} `json:"params"`
}

// WorkspaceSymbolResponse is the response to a workspace symbol request.
type WorkspaceSymbolResponse struct {
	ResponseMessageHeader

	Result []SymbolInformation `json:"result"`

	// Code and message set in case an exception happens during the request.
	Error *ResponseErrorHeader `json:"error,omitempty"`
}

// CodeActionRequest is a request sent from the client to the server to compute
// commands for a given text document and range. The request is triggered when
// the user moves the cursor into an problem marker in the editor or presses
// the lightbulb associated with a marker.
type CodeActionRequest struct {
	RequestMessageHeader

	Params struct {
		// The document in which the command was invoked.
		TextDocument TextDocumentIdentifier `json:"textDocument"`

		// The range for which the command was invoked.
		Range Range `json:"range"`

		// Context carrying additional information.
		Context CodeActionContext `json:"context"`
	} `json:"params"`
}

// CodeActionResponse is the response to a code action request.
type CodeActionResponse struct {
	ResponseMessageHeader

	Result []Command `json:"result"`

	// Code and message set in case an exception happens during the request.
	Error *ResponseErrorHeader `json:"error,omitempty"`
}

// CodeLensRequest is a request sent from the client to the server to compute
// code lenses for a given text document.
type CodeLensRequest struct {
	RequestMessageHeader

	Params struct {
		// The document to request code lens for.
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	} `json:"params"`
}

// CodeLensResponse is the response to a code lens request.
type CodeLensResponse struct {
	ResponseMessageHeader

	Result []CodeLens `json:"result"`

	// Code and message set in case an exception happens during the request.
	Error *ResponseErrorHeader `json:"error,omitempty"`
}

// CodeLensResolveRequest is a request sent from the client to the server to
// resolve the command for a given code lens item.
type CodeLensResolveRequest struct {
	RequestMessageHeader

	Params CodeLens `json:"params"`
}

// CodeLensResolveResponse is the response to a code lens resolve request.
type CodeLensResolveResponse struct {
	ResponseMessageHeader

	Result *CodeLens `json:"result,omitempty"`

	// Code and message set in case an exception happens during the request.
	Error *ResponseErrorHeader `json:"error,omitempty"`
}

// DocumentFormattingRequest is a request sent from the client to the server to
// format a whole document.
type DocumentFormattingRequest struct {
	RequestMessageHeader

	Params struct {
		// The document to format.
		TextDocument TextDocumentIdentifier `json:"textDocument"`

		// The format options
		Options FormattingOptions `json:"options"`
	} `json:"params"`
}

// DocumentFormattingResponse is the response to a document formatting request.
type DocumentFormattingResponse struct {
	ResponseMessageHeader

	Result []TextEdit `json:"result"`

	// Code and message set in case an exception happens during the request.
	Error *ResponseErrorHeader `json:"error,omitempty"`
}

// DocumentRangeFormattingRequest is a request sent from the client to the
// server to format a given range in a document.
type DocumentRangeFormattingRequest struct {
	RequestMessageHeader

	Params struct {
		// The document to format.
		TextDocument TextDocumentIdentifier `json:"textDocument"`

		// The range to format
		Range Range `json:"range"`

		// The format options
		Options FormattingOptions `json:"options"`
	} `json:"params"`
}

// DocumentRangeFormattingResponse is the response to a document formatting
// range request.
type DocumentRangeFormattingResponse struct {
	ResponseMessageHeader

	Result []TextEdit `json:"result"`

	// Code and message set in case an exception happens during the request.
	Error *ResponseErrorHeader `json:"error,omitempty"`
}

// DocumentOnTypeFormattingRequest is a request sent from the client to the
// server to format parts of the document during typing.
type DocumentOnTypeFormattingRequest struct {
	RequestMessageHeader

	Params struct {
		// The document to format.
		TextDocument TextDocumentIdentifier `json:"textDocument"`

		// The position at which this request was send.
		Position Position `json:"position"`

		// The character that has been typed.
		Character string `json:"ch"`

		// The format options
		Options FormattingOptions `json:"options"`
	} `json:"params"`
}

// DocumentOnTypeFormattingResponse is the response to a document on-type
// formatting request.
type DocumentOnTypeFormattingResponse struct {
	ResponseMessageHeader

	Result []TextEdit `json:"result"`

	// Code and message set in case an exception happens during the request.
	Error *ResponseErrorHeader `json:"error,omitempty"`
}

// RenameRequest is a request sent from the client to the server to do a
// workspace wide rename of a symbol.
type RenameRequest struct {
	RequestMessageHeader

	Params struct {
		// The document holding the symbol of reference.
		TextDocument TextDocumentIdentifier `json:"textDocument"`

		// The position at which this request was send.
		Position Position `json:"position"`

		// The new name of the symbol. If the given name is not valid the
		// request must return a ResponseError with an appropriate message set.
		NewName string `json:"newName"`
	} `json:"params"`
}

// RenameResponse is the response to a rename request.
type RenameResponse struct {
	ResponseMessageHeader

	Result *WorkspaceEdit `json:"result"`

	// Code and message set in case an exception happens during the request.
	Error *ResponseErrorHeader `json:"error,omitempty"`
}
