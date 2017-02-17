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

// CancelNotification is a special notification sent to cancel a request.
// A request that got canceled still needs to return from the server and send a
// response back. It can not be left open / hanging. This is in line with the
// JSON RPC protocol that requires that every request sends a reponse back. In
// addition it allows for returning paritial results on cancel.
type CancelNotification struct {
	Message

	Params struct {
		// The request to cancel.
		ID interface{} `json:"id"`
	} `json:"params"`
}

// NotificationMessageHeader is the common part to all notifications.
// A processed notification message must not send a response back. They work like events.
type NotificationMessageHeader struct {
	Message

	// The method to be invoked.
	Method string `json:"method"`
}

// ExitNotification is a notification sent from the server to the client to ask
// the server to exit its process.
type ExitNotification struct {
	NotificationMessageHeader
}

// ShowMessageNotification is a notification sent from the server to the client
// to ask the client to display a particular message in the user interface.
type ShowMessageNotification struct {
	NotificationMessageHeader

	Params struct {
		// The message type.
		Type MessageType `json:"type"`

		// The actual message
		Message string `json:"message"`
	} `json:"params"`
}

// LogMessageNotification is a notification sent from the server to the client
// to ask the client to log a particular message.
type LogMessageNotification struct {
	NotificationMessageHeader

	Params struct {
		// The message type.
		Type MessageType `json:"type"`

		// The actual message.
		Message string `json:"message"`
	} `json:"params"`
}

// DidChangeConfigurationNotification is a notification sent from the client to
// the server to signal the change of configuration settings.
type DidChangeConfigurationNotification struct {
	NotificationMessageHeader

	Params struct {
		// The actual changed settings
		Settings map[string]interface{} `json:"settings"`
	} `json:"params"`
}

// DidOpenTextDocumentNotification is a notification sent from the client to the
// server to signal newly opened text documents. The document's truth is now
// managed by the client and the server must not try to read the document's
// truth using the document's URI.
type DidOpenTextDocumentNotification struct {
	NotificationMessageHeader

	Params struct {
		// The document that was opened.
		TextDocument TextDocumentItem `json:"textDocument"`
	} `json:"params"`
}

// DidChangeTextDocumentNotification is a notification sent from the client to
// the server to signal changes to a text document. In 2.0 the shape of the
// params has changed to include proper version numbers and language ids.
type DidChangeTextDocumentNotification struct {
	NotificationMessageHeader

	Params struct {
		// The document that did change. The version number points
		// to the version after all provided content changes have
		// been applied.
		TextDocument VersionedTextDocumentIdentifier `json:"textDocument"`

		// The actual content changes.
		ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
	} `json:"params"`
}

// DidCloseTextDocumentNotification is a notification sent from the client to
// the server when the document got closed in the client. The document's truth
// now exists where the document's URI points to (e.g. if the document's URI is
// a file URI the truth now exists on disk).
type DidCloseTextDocumentNotification struct {
	NotificationMessageHeader

	Params struct {
		// The document that was closed.
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	} `json:"params"`
}

// DidSaveTextDocumentNotification is a notification sent from the client to the
// server when the document for saved in the clinet.
type DidSaveTextDocumentNotification struct {
	NotificationMessageHeader

	Params struct {
		// The document that was saved.
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	} `json:"params"`
}

// DidChangeWatchedFilesNotification is a notification sent from the client to
// the server when the client detects changes to file watched by the lanaguage
// client.
type DidChangeWatchedFilesNotification struct {
	NotificationMessageHeader

	Params struct {
		// The actual file events.
		Changes []FileEvent `json:"changes"`
	} `json:"params"`
}

// PublishDiagnosticsNotification is a notification sent from the server to the
// client to signal results of validation runs.
type PublishDiagnosticsNotification struct {
	NotificationMessageHeader

	Params struct {
		// The URI for which diagnostic information is reported.
		URI string `json:"uri"`

		// An array of diagnostic information items.
		Diagnostics []Diagnostic `json:"diagnostics"`
	} `json:"params"`
}
