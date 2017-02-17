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

	"github.com/google/gapid/core/langsvr/protocol"
)

// Document represents a text document file.
type Document struct {
	uri      string // The URI used by the client to identify this document.
	path     string // The absolute file path.
	language string // The language of the document.
	version  int    // The incremental version of the document.
	server   *langsvr
	body     Body // The immutable document body.
	open     bool // true if the document is currently open (visible)
	watched  bool // true if the document is watched
}

// newDocument returns a new document initialized with the uri, language,
// version and body content.
func (s *langsvr) newDocument(uri string, language string, version int, body Body) *Document {
	if d, exists := s.documents[uri]; exists {
		panic(fmt.Errorf("Attempting to create a document that already exists. %+v", d))
	}
	d := &Document{}
	d.uri = uri
	d.path, _ = URItoPath(uri)
	d.language = language
	d.version = version
	d.server = s
	d.body = body
	s.documents[uri] = d
	return d
}

// URI returns the document's URI.
func (d Document) URI() string { return d.uri }

// Path returns the document's absolute file path.
func (d Document) Path() string { return d.path }

// Language returns the document's file language.
func (d Document) Language() string { return d.language }

// Version returns the document's version.
func (d Document) Version() int { return d.version }

// Body returns the document's text body
func (d *Document) Body() Body { return d.body }

// SetDiagnostics sets the diagnostics for the document.
func (d *Document) SetDiagnostics(diagnostics Diagnostics) {
	diag := make([]protocol.Diagnostic, len(diagnostics))
	for i, d := range diagnostics {
		diag[i] = d.toProtocol()
	}
	d.server.conn.PublishDiagnostics(d.uri, diag)
}
