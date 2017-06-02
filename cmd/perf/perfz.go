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
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"time"

	"github.com/google/gapid/core/app/benchmark"
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
)

const (
	PerfzVersion = "1.3"
	perfzRoot    = "index.json"
)

var supportedVersions = regexp.MustCompile(`1\.(\d+)(\.(\d+))?`)

// Perfz is a collection of benchmark results and associated data.
type Perfz struct {
	PerfzVersion string                `diff:"2"` // To enforce version checks later.
	Benchmarks   map[string]*Benchmark // Collection of Benchnarks.
	Files        map[string]*DataEntry `diff:"3"` // Additional data (e.g. gapis binaries, traces).
}

// Benchmark is a set of results after running a benchmark.
type Benchmark struct {
	root *Perfz `diff:"ignore"`

	Input          BenchmarkInput          `diff:"2"` // Relevant input and configuration to run this.
	Meta           Metadata                `diff:"2"` // Additional information about the run.
	TotalTimeTaken Sample                  // How long it took to complete the full run of this
	Fit            string                  // String representation of the best Complexity.Fit() for the Samples.
	FitByFrame     string                  `json:",omitempty"` // String representation of the best fit, with frames on the x-axis.
	Links          map[string]*Link        `diff:"3"`          // Links to additional data (e.g. cpu or heap profiles).
	Samples        KeyedSamples            `diff:"2"`          // Per-atom-index duration data.
	Metrics        map[string]*Multisample // Additional per-benchmark metrics.
	Counters       *benchmark.Counters     // Benchmark counters.

	AtomIndicesToFrames map[int64]int `json:"-"`
}

// BenchmarkInput represents all the relevant input
// defining a benchmark.
type BenchmarkInput struct {
	Name              string
	Comment           string `json:",omitempty"`
	Trace             *Link
	Gapis             *Link
	Runs              int
	MaxSamples        int
	Seed              int64
	BenchmarkType     string
	SampleOrder       string
	EnableCPUProfile  bool
	EnableHeapProfile bool
	MaxFrameWidth     int
	MaxFrameHeight    int
	Timeout           time.Duration
}

// Metadata represents additional information about a benchmark.
type Metadata struct {
	HostName    string    `json:",omitempty"` // Machine where this was run.
	DateStarted time.Time // Time the benchmark was started.
}

// DataSource provides access to a blob of data.
type DataSource interface {
	// Writes the underlying data to the given io.Writer.
	WriteTo(io.Writer) error

	// Returns a potentially-temporary file containing this data.
	DiskFile() (filename string, isTemporary bool, err error)
}

// DataEntry represents a blob of data in the context of a .perfz archive.
// It stores information about the original source of the data, as well as
// instructions on where/whether to include this blob when saving the archive.
type DataEntry struct {
	DataSource `json:"-" diff:"ignore"` // Represents and allows access to the data.
	Name       string                   // Zip archive entry name if bundling this in the .perfz
	Source     string                   // Original filename, will be used on deserialization
	// to create a DataSource if the entry isn't bundled.
	Bundle bool // Whether to include this file in the .perfz on save
}

// Link references a DataEntry by ID (SHA) in a Perfz instance.
// We can have multiple benchmarks referencing e.g. the same trace
// or Gapis instance. Links are a lightweight approach to being able
// to get to these files from multiple places.
type Link struct {
	Key  string
	root *Perfz `diff:"ignore"`
}

// NewPerfz instantiates a new Perfz object.
func NewPerfz() *Perfz {
	return &Perfz{
		PerfzVersion: PerfzVersion,
		Benchmarks:   make(map[string]*Benchmark),
		Files:        make(map[string]*DataEntry),
	}
}

// NewBenchmarkWithName creates a new Benchmark and associates it with the Perfz.
func (p *Perfz) NewBenchmarkWithName(name string) *Benchmark {
	b := &Benchmark{
		Samples:             NewKeyedSamples(),
		Links:               make(map[string]*Link),
		Metrics:             make(map[string]*Multisample),
		root:                p,
		Counters:            benchmark.NewCounters(),
		AtomIndicesToFrames: make(map[int64]int),
	}
	p.Benchmarks[name] = b
	return b
}

// LoadPerfz deserializes and prepares a Perfz from a .perfz file.
func LoadPerfz(ctx context.Context, zipfile string, verify bool) (*Perfz, error) {
	log.I(ctx, "Loading .perfz from %s", zipfile)

	p := &Perfz{}

	if err := readZipEntry(zipfile, perfzRoot, func(r io.Reader) error {
		data, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}
		log.D(ctx, "Reading perfz root")
		return json.Unmarshal(data, p)
	}); err != nil {
		return nil, err
	}

	if !supportedVersions.MatchString(p.PerfzVersion) {
		return nil, fmt.Errorf("Unsupported .perfz version: %s", p.PerfzVersion)
	}

	// Wire up and maybe verify data entries.
	for key, ref := range p.Files {
		if ref.Bundle {
			// Data was included in this archive, so reference the archive entry.
			ref.DataSource = zipEntryDataSource(zipfile, ref.Name)
		} else if ref.Source != "" {
			// Data was not included, but we know the original filename, reference
			// the external file.
			ref.DataSource = FileDataSource(ref.Source)
		} else {
			// We only know the hash.
			panic("TODO: support finding files by hash, also, how did we get here?")
		}

		if verify {
			log.D(ctx, "Verifying entry %s", ref.Name)
			gotID, err := ref.ToID()
			if err != nil {
				return nil, err
			}

			if key != gotID.String() {
				log.W(ctx,
					"Hash mismatch for entry '%s', (source '%s', expected %s, got %v)",
					ref.Name, ref.Source, key, gotID)
			}
		}
	}

	// Wire up links to the Perfz.
	for _, b := range p.Benchmarks {
		b.root = p
	}
	for _, l := range p.Links() {
		l.root = p
	}

	return p, nil
}

// Links returns all the Link instances across all benchmarks in the Perfz.
func (p *Perfz) Links() []*Link {
	links := []*Link{}
	for _, b := range p.Benchmarks {
		links = append(links, b.Input.Trace, b.Input.Gapis)
		for _, l := range b.Links {
			links = append(links, l)
		}
	}
	return links
}

// RemoveUnreferencedFiles drops DataEntries not referenced by any Links.
// Applicable e.g. when using run or redo and overriding old Gapis links,
// or when removing a benchmark from the archive.
func (p *Perfz) RemoveUnreferencedFiles() {
	m := map[string]bool{}
	for _, l := range p.Links() {
		m[l.Key] = true
	}
	for k := range p.Files {
		_, found := m[k]
		if !found {
			delete(p.Files, k)
		}
	}
}

// WriteTo archives the Perfz and bundled data entries to a .perfz (zip) file.
func (p *Perfz) WriteTo(ctx context.Context, zipfile string) error {
	p.RemoveUnreferencedFiles()
	p.PerfzVersion = PerfzVersion
	log.I(ctx, "Creating .perfz archive: %s", zipfile)

	// Use a temporary file because we might need to copy
	// data entries from the archive we're about to overwrite.
	tmpFile, err := ioutil.TempFile("", "zip")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()
	zipWriter := zip.NewWriter(tmpFile)

	log.D(ctx, "Writing %s", perfzRoot)
	w, err := zipWriter.Create(perfzRoot)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	if err != nil {
		return err
	}

	for _, entry := range p.Files {
		if entry.Bundle {
			log.D(ctx, "Writing %s", entry.Name)
			w, err := zipWriter.Create(entry.Name)
			if err != nil {
				return err
			}
			if err = entry.WriteTo(w); err != nil {
				return err
			}
		}
	}

	err = zipWriter.Close()
	if err != nil {
		log.E(ctx, "%s", err.Error())
	}
	_, err = tmpFile.Seek(0, 0)
	if err != nil {
		return err
	}
	outFile, err := os.OpenFile(zipfile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(0644))
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, tmpFile)
	if err != nil {
		return err
	}

	log.I(ctx, "Archive created.")
	return nil
}

// NewLink calculates the hash of the DataEntry's contents and
// adds it to the list of daeta entries in the Perfz.
func (p *Perfz) NewLink(a *DataEntry) (*Link, error) {
	id, err := a.ToID()
	if err != nil {
		return nil, err
	}
	p.Files[id.String()] = a
	return &Link{
		Key:  id.String(),
		root: p,
	}, nil
}

// Metric records a new duration with the given metric, creating it
// it if it doesn't exist.
func (b *Benchmark) Metric(name string, t time.Duration) {
	ms, found := b.Metrics[name]
	if !found {
		ms = new(Multisample)
		b.Metrics[name] = ms
	}

	ms.Add(t)
}

// Returns the Perfz associated with the Benchmark.
func (b *Benchmark) Root() *Perfz {
	return b.root
}

func (a *DataEntry) UnmarshalJSON(b []byte) error {
	token, err := json.NewDecoder(bytes.NewReader(b)).Token()
	if err != nil {
		return err
	}
	entryName, ok := token.(string)
	if ok {
		*a = DataEntry{
			Name:   entryName,
			Bundle: true,
		}
	} else {
		s := struct {
			Name   string
			Source string
			Bundle bool
		}{}
		err := json.Unmarshal(b, &s)
		if err != nil {
			return err
		}
		*a = DataEntry{Name: s.Name, Source: s.Source, Bundle: s.Bundle}
	}
	return nil
}

func (a DataEntry) MarshalJSON() ([]byte, error) {
	if a.Bundle && a.Source == "" {
		return json.Marshal(a.Name)
	} else {
		return json.Marshal(struct {
			Name   string
			Source string
			Bundle bool
		}{
			Name:   a.Name,
			Source: a.Source,
			Bundle: a.Bundle,
		})
	}
}

// ToID calculates the hash of the data entry's data.
func (a *DataEntry) ToID() (id.ID, error) {
	return id.Hash(a.WriteTo)
}

func (l *Link) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.Key)
}

// Get dereferences the Link.
func (l Link) Get() *DataEntry {
	return l.root.Files[l.Key]
}

func (l *Link) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &(l.Key))
}

// FuncDataSource is a DataSource wrapper around a function
// that produces data.
type FuncDataSource func(w io.Writer) error

// WriteTo implements DataSource.
func (f FuncDataSource) WriteTo(w io.Writer) error {
	return f(w)
}

// DiskFile implements DataSource. FuncDataSource doesn't hold
// any information about already-existing files, and so this
// method will write the data to a temporary file and return
// its name.
func (f FuncDataSource) DiskFile() (string, bool, error) {
	tmpfile, err := ioutil.TempFile("", "")
	if err != nil {
		return "", false, err
	}
	err = f(tmpfile)
	tmpfile.Close()
	if err != nil {
		os.Remove(tmpfile.Name())
		return "", false, err
	} else {
		return tmpfile.Name(), true, nil
	}
}

// FileDataSource is an implementation of DataSource referencing
// a regular file.
type FileDataSource string

// WriteTo implements DataSource.
func (f FileDataSource) WriteTo(w io.Writer) error {
	r, err := os.Open(string(f))
	if err != nil {
		return err
	}
	defer r.Close()
	_, err = io.Copy(w, r)
	return err
}

// DiskFile implements DataSource, by simply returning the
// path to the file. We're hopeful the file is immutable.
func (f FileDataSource) DiskFile() (string, bool, error) {
	return string(f), false, nil
}

// ByteSliceDataSource creates a DataSource from a byte slice.
func ByteSliceDataSource(b []byte) FuncDataSource {
	return FuncDataSource(func(w io.Writer) error {
		_, err := io.Copy(w, bytes.NewReader(b))
		return err
	})
}

// zipEntryDataSource returns a DataSource referencing a zip entry.
func zipEntryDataSource(zipFile string, entry string) FuncDataSource {
	return FuncDataSource(func(w io.Writer) error {
		return readZipEntry(zipFile, entry, func(r io.Reader) error {
			_, err := io.Copy(w, r)
			return err
		})
	})
}

// readZipEntry opens a zip archive entry and passes the io.Reader to the
// provided function for consumption.
func readZipEntry(zipFile string, entry string, f func(r io.Reader) error) error {
	inZip, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer inZip.Close()
	for _, zf := range inZip.File {
		if zf.Name == entry {
			r, err := zf.Open()
			if err != nil {
				return err
			}
			return f(r)
		}
	}
	return fmt.Errorf("Entry %s not found in zip %s", entry, zipFile)
}

func (p *Perfz) String() string {
	p.RemoveUnreferencedFiles()
	jsonRoot, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(jsonRoot)
}
