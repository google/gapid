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

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sync"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/event/task"
	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
)

const (
	ErrCaughtPanic = fault.Const("Panic caught")
	ErrStopped     = fault.Const("stopped")
)

var methods = map[string]reflect.Type{
	// requests
	"initialize":                     reflect.TypeOf(InitializeRequest{}),
	"shutdown":                       reflect.TypeOf(ShutdownRequest{}),
	"window/showMessageRequest":      reflect.TypeOf(ShowMessageRequest{}),
	"textDocument/completion":        reflect.TypeOf(CompletionRequest{}),
	"completionItem/resolve":         reflect.TypeOf(CompletionItemResolveRequest{}),
	"textDocument/hover":             reflect.TypeOf(HoverRequest{}),
	"textDocument/signatureHelp":     reflect.TypeOf(SignatureHelpRequest{}),
	"textDocument/definition":        reflect.TypeOf(GotoDefinitionRequest{}),
	"textDocument/references":        reflect.TypeOf(FindReferencesRequest{}),
	"textDocument/documentHighlight": reflect.TypeOf(DocumentHighlightRequest{}),
	"textDocument/documentSymbol":    reflect.TypeOf(DocumentSymbolRequest{}),
	"workspace/symbol":               reflect.TypeOf(WorkspaceSymbolRequest{}),
	"textDocument/codeAction":        reflect.TypeOf(CodeActionRequest{}),
	"textDocument/codeLens":          reflect.TypeOf(CodeLensRequest{}),
	"codeLens/resolve":               reflect.TypeOf(CodeLensResolveRequest{}),
	"textDocument/formatting":        reflect.TypeOf(DocumentFormattingRequest{}),
	"textDocument/rangeFormatting":   reflect.TypeOf(DocumentRangeFormattingRequest{}),
	"textDocument/onTypeFormatting":  reflect.TypeOf(DocumentOnTypeFormattingRequest{}),
	"textDocument/rename":            reflect.TypeOf(RenameRequest{}),

	// notifications
	"exit":                             reflect.TypeOf(ExitNotification{}),
	"window/showMessage":               reflect.TypeOf(ShowMessageNotification{}),
	"window/logMessage":                reflect.TypeOf(LogMessageNotification{}),
	"workspace/didChangeConfiguration": reflect.TypeOf(DidChangeConfigurationNotification{}),
	"textDocument/didOpen":             reflect.TypeOf(DidOpenTextDocumentNotification{}),
	"textDocument/didChange":           reflect.TypeOf(DidChangeTextDocumentNotification{}),
	"textDocument/didClose":            reflect.TypeOf(DidCloseTextDocumentNotification{}),
	"textDocument/didSave":             reflect.TypeOf(DidSaveTextDocumentNotification{}),
	"workspace/didChangeWatchedFiles":  reflect.TypeOf(DidChangeWatchedFilesNotification{}),
	"textDocument/publishDiagnostics":  reflect.TypeOf(PublishDiagnosticsNotification{}),

	// special
	"$/cancelRequest": reflect.TypeOf(CancelNotification{}),
}

var methodsRev = map[reflect.Type]string{}

func init() {
	for method, ty := range methods {
		methodsRev[ty] = method
	}
}

// ErrUnknownMethod is an error returned when decoding an unknown method type.
type ErrUnknownMethod struct {
	Method string
}

func (e ErrUnknownMethod) Error() string { return fmt.Sprintf("Unknown method '%s", e.Method) }

// Error is an error that can be returned by any of the Server methods.
// It can contain additional metadata that should be sent to the client.
type Error struct {
	Code    ErrorCode
	Message string
}

func (e Error) Error() string { return e.Message }

func translateError(e error) ResponseErrorHeader {
	switch e := e.(type) {
	case Error:
		return ResponseErrorHeader{Code: e.Code, Message: e.Message}
	default:
		return ResponseErrorHeader{Code: InternalError, Message: e.Error()}
	}
}

// Server is the interface implemented by language servers.
type Server interface {
	// Initialize is a request to initialize the server.
	// processID is the parent process that started the server.
	// rootPath is the root path of the workspace - it is null if no folder is open.
	// capabilities are the capabilities of the client.
	Initialize(ctx context.Context, processID int, rootPath string) (ServerCapabilities, error)

	// Shutdown is a request to shutdown the server, but not exit.
	Shutdown(ctx context.Context) error

	// Completion is a request for completion items as the given cursor
	// position. If computing complete completion items is expensive,
	// servers can additional provide a handler for the resolve completion item
	// request. This request is send when a completion item is selected in the
	// user interface.
	// uri is the document identifier.
	// pos is the position in the document to complete.
	Completion(ctx context.Context, uri TextDocumentIdentifier, pos Position) (CompletionList, error)

	// CompletionItemResolve is a request to resolve additional information on
	// the completion item.
	// item is the completion item to resolve additional information.
	CompletionItemResolve(ctx context.Context, item CompletionItem) (CompletionItem, error)

	// Hover is a request for hover information as the given position.
	// uri is the document identifier.
	// pos is the position in the document to get hover information for.
	Hover(ctx context.Context, uri TextDocumentIdentifier, position Position) ([]MarkedString, *Range, error)

	// SignatureHelp is a request for function signature information at the given position.
	// uri is the document identifier.
	// pos is the position in the document to get signature information for.
	SignatureHelp(ctx context.Context, uri TextDocumentIdentifier, position Position) (sigs []SignatureInformation, activeSig *int, activeParam *int, err error)

	// GotoDefinition is a request to resolve the definition location(s) of the
	// symbol at the given position.
	// uri is the document identifier.
	// pos is the position in the document of the symbol.
	GotoDefinition(ctx context.Context, uri TextDocumentIdentifier, position Position) ([]Location, error)

	// FindReferences is a request to resolve project-wide reference
	// location(s) for the symbol at the given position.
	// uri is the document identifier.
	// pos is the position in the document of the symbol.
	FindReferences(ctx context.Context, uri TextDocumentIdentifier, position Position, includeDecl bool) ([]Location, error)

	// DocumentHighlights is a request to resolve document highlights at the
	// given position.
	// uri is the document identifier.
	// pos is the position in the document to get highlight information.
	DocumentHighlights(ctx context.Context, uri TextDocumentIdentifier, position Position) ([]DocumentHighlight, error)

	// DocumentSymbols is a request to list all the symbols for the given
	// document.
	// uri is the document identifier to get symbols for.
	DocumentSymbols(ctx context.Context, doc TextDocumentIdentifier) ([]SymbolInformation, error)

	// WorkspaceSymbols is a request to list all the project-wide symbols that
	// match the query string.
	WorkspaceSymbols(ctx context.Context, query string) ([]SymbolInformation, error)

	// CodeAction is a request to compute commands for the given text document
	// and range. The request is triggered when the user moves the cursor into
	// an problem marker in the editor or presses the lightbulb associated with
	// a marker.
	// doc is the document to compute commands for.
	// rng is the range in the document.
	// context holds additional information about the request.
	CodeAction(ctx context.Context, doc TextDocumentIdentifier, rng Range, context CodeActionContext) ([]Command, error)

	// CodeLens is a request to compute code-lenses for a given text document.
	// doc is the document to compute code-lenses for.
	CodeLens(ctx context.Context, doc TextDocumentIdentifier) ([]CodeLens, error)

	// CodeLensResolve is a request to resolve the command for a given code
	// lens item.
	CodeLensResolve(ctx context.Context, codelens CodeLens) (CodeLens, error)

	// DocumentFormatting is a request to format the entire document.
	// doc is the document to format.
	// opts are the formatting options.
	DocumentFormatting(ctx context.Context, doc TextDocumentIdentifier, opts FormattingOptions) ([]TextEdit, error)

	// DocumentRangeFormatting is a request to format the given range in a
	// document.
	// doc is the document to format.
	// rng is the range to format.
	// opts are the formatting options.
	DocumentRangeFormatting(ctx context.Context, doc TextDocumentIdentifier, rng Range, opts FormattingOptions) ([]TextEdit, error)

	// DocumentOnTypeFormatting is a request to format parts of the document
	// during typing.
	// doc is the document to format.
	// pos is the position at which the character was typed.
	// char is the character that was typed.
	// ops are the formatting options.
	DocumentOnTypeFormatting(ctx context.Context, doc TextDocumentIdentifier, pos Position, char string, opts FormattingOptions) ([]TextEdit, error)

	// Rename is a request to perform a workspace-wide rename of a symbol.
	// doc is the document holding the symbol of reference.
	// pos is the position of the symbol.
	// newName is the new name of the symbol.
	Rename(ctx context.Context, doc TextDocumentIdentifier, pos Position, newName string) (WorkspaceEdit, error)

	// OnExit is a request for the server to exit its process.
	OnExit(ctx context.Context) error

	// OnChangeConfiguration is a notification that the configuration settings
	// have changed.
	OnChangeConfiguration(ctx context.Context, settings map[string]interface{})

	// OnOpenTextDocument is a notification that signals when a document is
	// opened. The document's truth is now managed by the client and the
	// server must not try to read the document's truth using the document's
	// URI.
	OnOpenTextDocument(ctx context.Context, item TextDocumentItem)

	// OnChangeTextDocument is a notification that signals changes to a text
	// document.
	OnChangeTextDocument(ctx context.Context, item VersionedTextDocumentIdentifier, changes []TextDocumentContentChangeEvent)

	// OnCloseTextDocument is a notification that signals when a document is
	// closed.
	OnCloseTextDocument(ctx context.Context, item TextDocumentIdentifier)

	// OnSaveTextDocument is a notification that signals when a document is
	// saved.
	OnSaveTextDocument(ctx context.Context, item TextDocumentIdentifier)

	// OnChangeWatchedFiles is a notification that signals changes to files
	// watched by the lanaguage client.
	OnChangeWatchedFiles(ctx context.Context, changes []FileEvent)
}

// Connection sends and receives messages from the client.
// Construct a connection with Connect(), and then call Serve() to begin a
// communication stream.
type Connection struct {
	in          *bufio.Reader
	out         io.Writer
	stopped     task.Signal            // Signal to recvRoutine and sendRoutine to stop.
	recvChan    chan recvItem          // Message items received by recvRoutine.
	sendChan    chan sendItem          // Pending message items to send by sendRoutine.
	cancelsLock sync.Mutex             // guards cancels
	cancels     map[interface{}]func() // request ID -> cancel func
}

type recvItem struct {
	ctx context.Context
	msg interface{}
}

type sendItem struct {
	msg interface{}
	err chan<- error
}

// NewConnection returns a new connection, ready to serve.
func NewConnection(stream io.ReadWriter) *Connection {
	return &Connection{
		bufio.NewReader(stream),
		stream,
		nil,
		make(chan recvItem, 32),
		make(chan sendItem, 32),
		sync.Mutex{},
		map[interface{}]func(){},
	}
}

// Serve listens on the connection for all incomming requests and notifications
// and dispatches messages to server.
func (c *Connection) Serve(ctx context.Context, server Server) error {
	stopped, stop := task.NewSignal()
	defer stop(ctx)
	c.stopped = stopped

	sendErr, recvErr := make(chan error), make(chan error)

	// Kick a go-routine to send all messages
	crash.Go(func() { sendErr <- c.sendRoutine(ctx) })
	// Kick a go-routine to receive all the messages.
	crash.Go(func() { recvErr <- c.recvRoutine(ctx) })
	// Dispatch all messages
	for {
		select {
		case <-task.ShouldStop(ctx):
			return nil
		case err := <-sendErr:
			return log.Err(ctx, err, "sending")
		case err := <-recvErr:
			return log.Err(ctx, err, "receiving")
		case item := <-c.recvChan:
			c.dispatch(item.ctx, item.msg, server)
			if request, ok := item.msg.(Request); ok {
				c.cancelsLock.Lock()
				delete(c.cancels, request.RequestID())
				c.cancelsLock.Unlock()
			}
		}
	}
}

func (c *Connection) recvRoutine(ctx context.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = log.Errf(ctx, ErrCaughtPanic, "recvRoutine: %v", r)
		}
	}()

	for !task.Stopped(ctx) {
		packet, err := c.readPacket()
		if err != nil {
			return log.Err(ctx, err, "reading packet")
		}
		msg, err := c.decode(packet)
		switch err := err.(type) {
		case nil:
		case ErrUnknownMethod:
			continue
		default:
			return log.Err(ctx, err, "decoding packet")
		}
		switch msg := msg.(type) {
		case *CancelNotification:
			// Request to cancel a request.
			c.cancelsLock.Lock()
			stop, ok := c.cancels[msg.Params.ID]
			c.cancelsLock.Unlock()
			if ok {
				stop()
			}

		case Request:
			// Cancellable request message.
			var stop func()
			ctx, stop := task.WithCancel(ctx)
			c.cancelsLock.Lock()
			c.cancels[msg.RequestID()] = stop
			c.cancelsLock.Unlock()
			c.recvChan <- recvItem{ctx, msg}

		default:
			c.recvChan <- recvItem{ctx, msg}
		}
	}
	return log.Err(ctx, ErrStopped, "")
}

func (c *Connection) sendRoutine(ctx context.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = log.Errf(ctx, ErrCaughtPanic, "sendRoutine: %v", r)
		}
	}()
	for {
		select {
		case <-task.ShouldStop(ctx):
			return log.Err(ctx, ErrStopped, "")
		case item := <-c.sendChan:
			packet, err := json.Marshal(item.msg)
			if err != nil {
				item.err <- err
				continue
			}
			fmt.Fprintf(c.out, "Content-Length: %d\r\n\r\n", len(packet))
			_, err = c.out.Write(packet)
			item.err <- err
		}
	}
}

// ShowMessage asks the client to display a particular message in the user interface.
func (c *Connection) ShowMessage(ty MessageType, message string) error {
	msg := ShowMessageNotification{}
	initNotificationHeader(&msg)
	msg.Params.Type = ty
	msg.Params.Message = message
	return c.send(msg)
}

// LogMessage asks the client to log a particular message.
func (c *Connection) LogMessage(ty MessageType, message string) error {
	msg := LogMessageNotification{}
	initNotificationHeader(&msg)
	msg.Params.Type = ty
	msg.Params.Message = message
	return c.send(msg)
}

// PublishDiagnostics sends the list of diagnostics for the document with the
// specified URI to the client.
func (c *Connection) PublishDiagnostics(uri string, diagnostics []Diagnostic) error {
	msg := PublishDiagnosticsNotification{}
	initNotificationHeader(&msg)
	msg.Params.URI = uri
	msg.Params.Diagnostics = diagnostics
	return c.send(msg)
}

func isEOL(r byte, err error) bool {
	if err != nil {
		return false
	}
	switch r {
	case '\r', '\n':
		return true
	default:
		return false
	}
}

// readPacket reads a single packet of data.
func (c *Connection) readPacket() ([]byte, error) {
	line, err := c.in.ReadString('\n')
	if err != nil {
		return nil, err
	}
	length := 0
	if _, err := fmt.Sscanf(line, "Content-Length: %d", &length); err != nil {
		return nil, err
	}

	for isEOL(c.in.ReadByte()) { /* skip EOL bytes */
	}
	c.in.UnreadByte()

	packet := make([]byte, length)
	if _, err := io.ReadFull(c.in, packet); err != nil {
		return nil, err
	}
	return packet, nil
}

// decode decodes a message from buf.
func (c *Connection) decode(buf []byte) (interface{}, error) {
	var msg map[string]interface{}
	if err := json.Unmarshal(buf, &msg); err != nil {
		return nil, err
	}
	method, ok := msg["method"].(string)
	if !ok {
		return nil, fmt.Errorf("Message was missing method key.")
	}
	ty, ok := methods[method]
	if !ok {
		return nil, ErrUnknownMethod{method}
	}
	obj := reflect.New(ty).Interface()
	if err := json.Unmarshal(buf, obj); err != nil {
		return nil, err
	}
	return obj, nil
}

func (c *Connection) dispatch(ctx context.Context, msg interface{}, server Server) error {
	if request, ok := msg.(Request); ok {
		ctx = log.V{"id": request.RequestID()}.Bind(ctx)
	}

	switch msg := msg.(type) {
	case *InitializeRequest:
		res := InitializeResponse{}
		if caps, err := server.Initialize(ctx, msg.Params.ProcessID, msg.Params.RootPath); err != nil {
			initResponseErr(&res, msg.ID, err)
		} else {
			initResponseRes(&res, msg.ID)
			res.Result.Capabilities = caps
		}
		return c.send(res)

	case *ShutdownRequest:
		res := ShutdownResponse{}
		if err := server.Shutdown(ctx); err != nil {
			initResponseErr(&res, msg.ID, err)
		} else {
			initResponseRes(&res, msg.ID)
		}
		return c.send(res)

	case *CompletionRequest:
		res := CompletionResponse{}
		if items, err := server.Completion(ctx, msg.Params.Document, msg.Params.Position); err != nil {
			initResponseErr(&res, msg.ID, err)
		} else {
			initResponseRes(&res, msg.ID)
			res.Result = items
		}
		return c.send(res)

	case *CompletionItemResolveRequest:
		resolve, err := server.CompletionItemResolve(ctx, msg.Params)
		res := CompletionItemResolveResponse{}
		if err != nil {
			initResponseErr(&res, msg.ID, err)
		} else {
			initResponseRes(&res, msg.ID)
			res.Result = &resolve
		}
		return c.send(res)

	case *HoverRequest:
		content, rng, err := server.Hover(ctx, msg.Params.Document, msg.Params.Position)
		res := HoverResponse{}
		if err != nil {
			initResponseErr(&res, msg.ID, err)
		} else {
			initResponseRes(&res, msg.ID)
			res.Result.Contents = content
			res.Result.Range = rng
		}
		return c.send(res)

	case *SignatureHelpRequest:
		sigs, activeSig, activeParam, err := server.SignatureHelp(ctx, msg.Params.Document, msg.Params.Position)
		res := SignatureHelpResponse{}
		if err != nil {
			initResponseErr(&res, msg.ID, err)
		} else {
			initResponseRes(&res, msg.ID)
			res.Result.Signatures = sigs
			res.Result.ActiveSignature = activeSig
			res.Result.ActiveParameter = activeParam
		}
		return c.send(res)

	case *GotoDefinitionRequest:
		locs, err := server.GotoDefinition(ctx, msg.Params.Document, msg.Params.Position)
		res := GotoDefinitionResponse{}
		if err != nil {
			initResponseErr(&res, msg.ID, err)
		} else {
			initResponseRes(&res, msg.ID)
			res.Result = locs
		}
		return c.send(res)

	case *FindReferencesRequest:
		l := msg.Params.TextDocumentPositionParams
		locations, err := server.FindReferences(ctx, l.Document, l.Position, msg.Params.Context.IncludeDeclaration)
		res := FindReferencesResponse{}
		if err != nil {
			initResponseErr(&res, msg.ID, err)
		} else {
			initResponseRes(&res, msg.ID)
			res.Result = locations
		}
		return c.send(res)

	case *DocumentHighlightRequest:
		highlights, err := server.DocumentHighlights(ctx, msg.Params.Document, msg.Params.Position)
		res := DocumentHighlightResponse{}
		if err != nil {
			initResponseErr(&res, msg.ID, err)
		} else {
			initResponseRes(&res, msg.ID)
			res.Result = highlights
		}
		return c.send(res)

	case *DocumentSymbolRequest:
		symbols, err := server.DocumentSymbols(ctx, msg.Params.TextDocument)
		res := DocumentSymbolResponse{}
		if err != nil {
			initResponseErr(&res, msg.ID, err)
		} else {
			initResponseRes(&res, msg.ID)
			res.Result = symbols
		}
		return c.send(res)

	case *WorkspaceSymbolRequest:
		symbols, err := server.WorkspaceSymbols(ctx, msg.Params.Query)
		res := WorkspaceSymbolResponse{}
		if err != nil {
			initResponseErr(&res, msg.ID, err)
		} else {
			initResponseRes(&res, msg.ID)
			res.Result = symbols
		}
		return c.send(res)

	case *CodeActionRequest:
		commands, err := server.CodeAction(ctx, msg.Params.TextDocument, msg.Params.Range, msg.Params.Context)
		res := CodeActionResponse{}
		if err != nil {
			initResponseErr(&res, msg.ID, err)
		} else {
			initResponseRes(&res, msg.ID)
			res.Result = commands
		}
		return c.send(res)

	case *CodeLensRequest:
		lenses, err := server.CodeLens(ctx, msg.Params.TextDocument)
		res := CodeLensResponse{}
		if err != nil {
			initResponseErr(&res, msg.ID, err)
		} else {
			initResponseRes(&res, msg.ID)
			res.Result = lenses
		}
		return c.send(res)

	case *CodeLensResolveRequest:
		lens, err := server.CodeLensResolve(ctx, msg.Params)
		res := CodeLensResolveResponse{}
		if err != nil {
			initResponseErr(&res, msg.ID, err)
		} else {
			initResponseRes(&res, msg.ID)
			res.Result = &lens
		}
		return c.send(res)

	case *DocumentFormattingRequest:
		edits, err := server.DocumentFormatting(ctx, msg.Params.TextDocument, msg.Params.Options)
		res := DocumentRangeFormattingResponse{}
		if err != nil {
			initResponseErr(&res, msg.ID, err)
		} else {
			initResponseRes(&res, msg.ID)
			res.Result = edits
		}
		return c.send(res)

	case *DocumentRangeFormattingRequest:
		edits, err := server.DocumentRangeFormatting(ctx, msg.Params.TextDocument, msg.Params.Range, msg.Params.Options)
		res := DocumentRangeFormattingResponse{}
		if err != nil {
			initResponseErr(&res, msg.ID, err)
		} else {
			initResponseRes(&res, msg.ID)
			res.Result = edits
		}
		return c.send(res)

	case *DocumentOnTypeFormattingRequest:
		edits, err := server.DocumentOnTypeFormatting(ctx, msg.Params.TextDocument, msg.Params.Position, msg.Params.Character, msg.Params.Options)
		res := DocumentOnTypeFormattingResponse{}
		if err != nil {
			initResponseErr(&res, msg.ID, err)
		} else {
			initResponseRes(&res, msg.ID)
			res.Result = edits
		}
		return c.send(res)

	case *RenameRequest:
		edit, err := server.Rename(ctx, msg.Params.TextDocument, msg.Params.Position, msg.Params.NewName)
		res := RenameResponse{}
		if err != nil {
			initResponseErr(&res, msg.ID, err)
		} else {
			initResponseRes(&res, msg.ID)
			res.Result = &edit
		}
		return c.send(res)

	case *ExitNotification:
		server.OnExit(ctx)
		return nil

	case *DidChangeConfigurationNotification:
		server.OnChangeConfiguration(ctx, msg.Params.Settings)

	case *DidOpenTextDocumentNotification:
		server.OnOpenTextDocument(ctx, msg.Params.TextDocument)

	case *DidChangeTextDocumentNotification:
		server.OnChangeTextDocument(ctx, msg.Params.TextDocument, msg.Params.ContentChanges)

	case *DidCloseTextDocumentNotification:
		server.OnCloseTextDocument(ctx, msg.Params.TextDocument)

	case *DidSaveTextDocumentNotification:
		server.OnSaveTextDocument(ctx, msg.Params.TextDocument)

	case *DidChangeWatchedFilesNotification:
		server.OnChangeWatchedFiles(ctx, msg.Params.Changes)

	default:
		return fmt.Errorf("Unhandled message type %T", msg)
	}
	return nil
}

func initNotificationHeader(notificationPtr interface{}) {
	r := reflect.ValueOf(notificationPtr).Elem()
	header := NotificationMessageHeader{
		Message: Message{JSONRPC: "2.0"},
		Method:  methodsRev[r.Type()],
	}
	r.FieldByName("NotificationMessageHeader").Set(reflect.ValueOf(header))
}

func initResponseErr(responsePtr interface{}, id interface{}, err error) {
	header := ResponseMessageHeader{
		Message: Message{JSONRPC: "2.0"},
		ID:      id,
	}
	r := reflect.ValueOf(responsePtr).Elem()
	r.FieldByName("ResponseMessageHeader").Set(reflect.ValueOf(header))
	fieldError := r.FieldByName("Error")
	fieldError.Set(reflect.New(fieldError.Type().Elem()))
	fieldError.Elem().Set(reflect.ValueOf(translateError(err)))
}

func initResponseRes(responsePtr interface{}, id interface{}) {
	header := ResponseMessageHeader{
		Message: Message{JSONRPC: "2.0"},
		ID:      id,
	}
	r := reflect.ValueOf(responsePtr).Elem()
	r.FieldByName("ResponseMessageHeader").Set(reflect.ValueOf(header))
	fieldResult := r.FieldByName("Result")
	switch fieldResult.Kind() {
	case reflect.Ptr:
		fieldResult.Set(reflect.New(fieldResult.Type().Elem()))
	case reflect.Slice:
		reflect.MakeSlice(reflect.SliceOf(fieldResult.Type().Elem()), 0, 0)
	}
}

func (c *Connection) send(msg interface{}) error {
	err := make(chan error)
	select {
	case <-c.stopped:
		return ErrConnectionClosed
	case c.sendChan <- sendItem{msg, err}:
		select {
		case <-c.stopped:
			return ErrConnectionClosed
		case err := <-err:
			return err
		}
	}
}

// ErrConnectionClosed is returned when attempting to send a message on a closed
// connection.
var ErrConnectionClosed = fmt.Errorf("Connection closed")
