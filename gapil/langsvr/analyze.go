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

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/google/gapid/core/app/crash"
	"github.com/google/gapid/core/event/task"
	ls "github.com/google/gapid/core/langsvr"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/file"
	"github.com/google/gapid/core/text/parse"
	"github.com/google/gapid/gapil"
	"github.com/google/gapid/gapil/analysis"
	"github.com/google/gapid/gapil/ast"
	"github.com/google/gapid/gapil/resolver"
	"github.com/google/gapid/gapil/semantic"
	"github.com/google/gapid/gapil/validate"
)

const (
	// Expected duration of analysis. If it does over this, warn.
	analysisWarnDuration = 5 * time.Second
)

type fullAnalysis struct {
	docs     map[string]*docAnalysis
	roots    map[string]*rootAnalysis // Root document path -> rootAnalysis
	mappings *resolver.Mappings       // AST node to semantic node map
}

type rootAnalysis struct {
	doc     *docAnalysis
	sem     *semantic.API
	results *analysis.Results
}

type docAnalysis struct {
	full   *fullAnalysis
	doc    *ls.Document
	ast    *ast.API
	errs   []parse.Error
	issues validate.Issues
}

func (da *docAnalysis) walkDown(offset int) []nodes {
	out := []nodes{}
	for n := ast.Node(da.ast); n != nil; n = astChildAt(da, n, offset) {
		var sem semantic.Node
		if sems := da.full.mappings.ASTToSemantic[n]; len(sems) > 0 {
			sem = sems[0]
		}
		out = append(out, nodes{n, sem})
	}
	return out
}

func (da *docAnalysis) walkUp(offset int) []nodes {
	out := da.walkDown(offset)
	for i, c, m := 0, len(out), len(out)/2; i < m; i++ {
		j := c - i - 1
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func (da *docAnalysis) contains(n ast.Node) bool {
	return da.doc.Path() == da.full.mappings.CST(n).Token().Source.Filename
}

func (s *server) docAnalysis(ctx context.Context, doc *ls.Document) (*docAnalysis, error) {
	analysis := s.analyzer.results(ctx, s)
	if analysis == nil {
		return nil, nil
	}
	da, ok := analysis.docs[doc.Path()]
	if !ok {
		return nil, log.Err(ctx, nil, "Document not found")
	}
	if da.ast == nil {
		return nil, log.Err(ctx, nil, "Parsing failed")
	}
	return da, nil
}

type dirList []string

func (l dirList) contains(path string) bool {
	for _, d := range l {
		if strings.HasPrefix(path, d) {
			return true
		}
	}
	return false
}

type importInfo struct {
	node     *ast.Import
	importee *docAnalysis
}

// analyzer performs API file analysis on a separate goroutine.
// When using this type, none of the fields should be directly accessed. Only
// the following method should be called externally: results(), begin().
type analyzer struct {
	cancel      func()                  // Cancels any pending analysis.
	done        task.Signal             // Signal for analysis to finish.
	lastResults *fullAnalysis           // Last analysis result.
	diagnostics map[string]*ls.Document // Last diagnostics set on each document.
}

func newAnalyzer() *analyzer {
	return &analyzer{
		cancel:      func() {},
		done:        task.FiredSignal,
		diagnostics: map[string]*ls.Document{},
	}
}

// results returns the last analysis results, starting a new analysis if there
// were no last results.
func (a *analyzer) results(ctx context.Context, s *server) *fullAnalysis {
	if a.lastResults != nil {
		return a.lastResults
	}
	a.begin(ctx, s)
	a.done.Wait(ctx)
	return a.lastResults
}

// begin starts a new analysis of the API documents.
func (a *analyzer) begin(ctx context.Context, s *server) error {
	if s.config == nil {
		// We're still waiting for the configuration. Don't do anything yet,
		// we'll restart analysis when this comes through.
		return nil
	}

	// Ensure that any previous analysis is finished.
	a.cancel()
	if !a.done.Wait(ctx) {
		return task.StopReason(ctx)
	}

	// Figure out what paths we should be ignoring
	ignorePaths := make(dirList, 0, len(s.config.IgnorePaths))
	for _, rel := range s.config.IgnorePaths {
		if path, err := filepath.Abs(filepath.Join(s.workspaceRoot, rel)); err == nil {
			ignorePaths = append(ignorePaths, path)
		}
	}

	// Copy the document map and flags - these may be mutated while we're processing.
	docs := make(map[string]*ls.Document, len(s.docs))
	for path, doc := range s.docs {
		if !ignorePaths.contains(path) {
			docs[path] = doc
		}
	}
	va := validate.Options{
		CheckUnused: s.config.CheckUnused,
	}

	// Setup the new done signal and cancellation function.
	ctx, cancel := task.WithCancel(ctx)
	signal, done := task.NewSignal()
	a.done = signal
	a.cancel = func() {
		cancel()
		<-signal
		a.lastResults = nil
	}

	// Start the go-routine to perform the analysis.
	crash.Go(func() { a.doAnalysis(ctx, docs, va, done) })

	return nil
}

type docsLoader struct{ docs map[string]*ls.Document }

func (l docsLoader) Find(path file.Path) file.Path { return path }
func (l docsLoader) Load(path file.Path) ([]byte, error) {
	if doc, ok := l.docs[path.System()]; ok {
		return []byte(doc.Body().Text()), nil
	}
	return ioutil.ReadFile(path.System())
}

// doAnalysis is the internal analysis function.
// Must only be called from analyzer.begin().
func (a *analyzer) doAnalysis(
	ctx context.Context,
	docs map[string]*ls.Document,
	va validate.Options,
	done task.Task) {
	defer handlePanic(ctx)
	defer done(ctx)

	ctx, start := log.Enter(ctx, "analyse"), time.Now()
	var parseDuration, resolveDuration time.Duration

	res := &fullAnalysis{}

	// Construct a docAnalysis for each document.
	das := make(map[string]*docAnalysis, len(docs))
	for path, doc := range docs {
		das[path] = &docAnalysis{
			doc:  doc,
			full: res,
		}
	}

	// Build a processor that will 'load' from the in-memory docs, falling back
	// to disk loads.
	processor := gapil.Processor{
		Mappings:            resolver.NewMappings(),
		Loader:              docsLoader{docs: docs},
		Parsed:              map[string]gapil.ParseResult{},
		Resolved:            map[string]gapil.ResolveResult{},
		ResolveOnParseError: true,
	}

	if task.Stopped(ctx) {
		return
	}

	// Parse all files, append errors to analysis.
	{
		parseStart := time.Now()
		pool, shutdown := task.Pool(len(docs), len(docs))
		defer shutdown(ctx)
		events := &task.Events{}
		executor := task.Batch(pool, events)

		for path, da := range das {
			path, da, ctx := path, da, log.V{"file": path}.Bind(ctx)
			executor(ctx, func(ctx context.Context) error {
				defer handlePanic(ctx)
				ast, errs := processor.Parse(path)
				da.ast = ast
				if errs != nil {
					da.errs = append(da.errs, errs...)
					return nil
				}
				return nil
			})
		}

		events.Wait(ctx)
		parseDuration = time.Since(parseStart)
	}

	if task.Stopped(ctx) {
		return
	}

	// Build import graph, find roots.
	roots := map[string]*rootAnalysis{}
	for path, da := range das {
		roots[path] = &rootAnalysis{doc: da}
	}
	imports := map[*docAnalysis][]importInfo{}
	for importerPath, importerDA := range das {
		importerWD, _ := filepath.Split(importerPath)
		if importerAST := importerDA.ast; importerAST != nil {
			for _, i := range importerAST.Imports {
				importeePath, _ := filepath.Abs(filepath.Join(importerWD, i.Path.Value))
				if importeeDA, ok := das[importeePath]; ok {
					delete(roots, importeePath)
					imports[importerDA] = append(imports[importerDA], importInfo{i, importeeDA})
				}
			}
		}
	}

	if task.Stopped(ctx) {
		return
	}

	// Resolve all the roots.
	resolveStart := time.Now()
	for rootPath, rootDA := range roots {
		if task.Stopped(ctx) {
			return
		}

		sem, errs := processor.Resolve(rootPath)
		rootDA.sem = sem
		for _, err := range errs {
			if at := err.At; at != nil {
				if source := at.Token().Source; source != nil {
					if da, ok := das[source.Filename]; ok {
						da.errs = append(da.errs, err)
					}
				}
			}
		}

		if len(errs) != 0 {
			continue
		}

		a := analysis.Analyze(sem, processor.Mappings)
		rootDA.results = a
		issues := validate.WithAnalysis(sem, processor.Mappings, &va, a)
		for _, issue := range issues {
			if at := issue.At; at != nil {
				if source := at.Token().Source; source != nil {
					if da, ok := das[source.Filename]; ok {
						da.issues = append(da.issues, issue)
					}
				}
			}
		}
	}
	resolveDuration = time.Since(resolveStart)

	// depth-first traversal of di's imports
	var traverseImports func(importer *docAnalysis, f func(importer, importee *docAnalysis, node *ast.Import))
	traverseImports = func(importer *docAnalysis, f func(importer, importee *docAnalysis, node *ast.Import)) {
		for _, i := range imports[importer] {
			traverseImports(i.importee, f)
			f(importer, i.importee, i.node)
		}
	}

	// Mark imports with errors.
	for _, rootDA := range roots {
		traverseImports(rootDA.doc, func(importer, importee *docAnalysis, node *ast.Import) {
			if len(importee.errs) > 0 {
				msg := fmt.Sprintf("Import contains %d errors", len(importee.errs))
				err := parse.Error{At: processor.Mappings.CST(node), Message: msg}
				importer.errs = append(importer.errs, err)
			}
		})
	}

	// Set all the diagnostics on the open docs.
	for path, da := range das {
		doc := docs[path]
		diags := ls.Diagnostics{}
		for _, err := range da.errs {
			diags.Error(fragRange(doc, err.At), err.Message)
		}
		for _, issue := range da.issues {
			diags.Warning(fragRange(doc, issue.At), issue.String())
		}
		doc.SetDiagnostics(diags)
		delete(a.diagnostics, path)
	}

	// Clear any documents we had previously reported diagnostics for but didn't
	// analyse this time.
	for _, doc := range a.diagnostics {
		doc.SetDiagnostics(ls.Diagnostics{})
	}

	// Add all document that have had analysis this pass so they can be
	// cleared next analysis.
	for path, da := range das {
		if _, ok := docs[path]; ok {
			a.diagnostics[path] = da.doc
		}
	}

	// Check we didn't take too long.
	if d := time.Since(start); d > analysisWarnDuration {
		log.W(ctx, "Full analysis took %v (parse: %v, resolve: %v)", d, parseDuration, resolveDuration)
	}

	res.docs = das
	res.roots = roots
	res.mappings = processor.Mappings
	a.lastResults = res
}

// stackdumpTimebomb prints the entire stack of all executing goroutines if it
// isn't defused within timeout duration.
func stackdumpTimebomb(ctx context.Context, timeout time.Duration) (defuse func()) {
	stop := make(chan struct{})
	go func() {
		select {
		case <-time.After(timeout):
			buf := make([]byte, 64<<10)
			buf = buf[:runtime.Stack(buf[:], true)]
			log.E(ctx, "Stack dump:\n%v", string(buf[:]))
		case <-stop:
		}
	}()
	return func() { close(stop) }
}
