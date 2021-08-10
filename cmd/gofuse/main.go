// Copyright (C) 2018 Google Inc.
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

// gofuse is a utility program to help go tools work with bazel generated files.
//
// gofuse will create a new 'fused' directory in the project root which
// contains:
//  • Symlinks to authored files in the GAPID source tree.
//  • Symlinks to bazel-generated files (bazel-out/[config]/{bin,genfiles}).
//  • Symlinks to external 3rd-party .go files.
// These symlinks are 'fused' into a single, common directory structure that
// is expected by the typical GOPATH rules used by go tooling.
//
// Note: the extensive use of symlinks makes Windows support unlikely.
//
// Examples:
//   bazel run //cmd/gofuse
//   bazel run //cmd/gofuse -- --bazelout=k8-fastbuild
//   bazel run //cmd/gofuse -- --bazelout=k8-dbg
//   bazel run //cmd/gofuse -- --bazelout=darwin-fastbuild
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Map of bazel external package names to the expected import names.
var externals = map[string]string{
	"com_github_golang_protobuf":       filepath.Join("github.com", "golang", "protobuf"),
	"com_github_google_go_github":      filepath.Join("github.com", "google", "go-github"),
	"com_github_google_go_querystring": filepath.Join("github.com", "google", "go-querystring"),
	"com_github_grpc_grpc":             filepath.Join("github.com", "grpc", "grpc"),
	"com_github_pkg_errors":            filepath.Join("github.com", "pkg", "errors"),
	"org_golang_google_grpc":           filepath.Join("google.golang.org", "grpc"),
	"org_golang_x_crypto":              filepath.Join("golang.org", "x", "crypto"),
	"org_golang_x_net":                 filepath.Join("golang.org", "x", "net"),
	"org_golang_x_text":                filepath.Join("golang.org", "x", "text"),
	"org_golang_x_tools":               filepath.Join("golang.org", "x", "tools"),
	"org_golang_x_sys":                 filepath.Join("golang.org", "x", "sys"),
	"llvm":                             "llvm",
}

var (
	fuseDir = flag.String("dir", "", "directory to use as the fuse root")

	bazelOutDirectory = flag.String("bazelout", "",
		"The bazel-out/X directory name from which to include .go files. E.g. k8-fastbuild, darwin-fastbuild, k8-dbg, etc.")

	runOnWindows = flag.Bool("force-run-on-windows", false, "Don't fail to run on Windows, even though it's unsupported.")
)

func main() {
	flag.Parse()

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run() error {
	if runtime.GOOS == "windows" && !*runOnWindows {
		return fmt.Errorf("cmd/gofuse is not supported on Windows")
	}

	if len(*bazelOutDirectory) == 0 {
		switch runtime.GOOS {
		case "linux":
			*bazelOutDirectory = "k8-fastbuild"
			break
		case "darwin":
			*bazelOutDirectory = "darwin-fastbuild"
		default:
		}

		if len(*bazelOutDirectory) == 0 {
			fmt.Println("\nFailed to guess bazel-out/X directory. Please pass --bazelout=X")
			return fmt.Errorf("need bazelout flag")
		}

		fmt.Printf("\nWARNING: guessed bazel-out/X directory as: %s. Please pass --bazelout=X to be more explicit.\n", *bazelOutDirectory)
		fmt.Println()
	}

	// If run via 'bazel run', use the workspace directory.
	if cwd := os.Getenv("BUILD_WORKSPACE_DIRECTORY"); cwd != "" {
		os.Chdir(cwd)
	}

	// Locate the root of the GAPID project.
	projectRoot, err := projectRoot()
	if err != nil {
		return err
	}

	// fusedRoot is the root of the generated directory.
	fusedRoot := filepath.Join(projectRoot, "fused")
	if *fuseDir != "" {
		fusedRoot = *fuseDir
	}

	fmt.Println("Updating fused directory at:", fusedRoot)

	// Collect all the existing symlinks under the fused root.
	fusedFiles := collect(fusedRoot, always).ifTrue(and(isFile, isSymlink))

	fmt.Print("Collecting files from:", projectRoot)
	srcMapping := collect(projectRoot,
		and(
			// Don't traverse the fused root
			hasPrefix(fusedRoot).not(),
			// Don't traverse the bazel directories
			hasPrefix(filepath.Join(projectRoot, "bazel-")).not(),
			// Don't traverse directories starting with "."; don't use filepath.Join as it simplifies "."
			hasPrefix(projectRoot+string(filepath.Separator)+".").not(),
		)).
		// Only consider files; ignore those in root starting with "."; don't use filepath.Join as it simplifies "."
		ifTrue(and(isFile, hasPrefix(projectRoot+string(filepath.Separator)+".").not())).
		mapping(func(path string) string {
			return filepath.Join(fusedRoot, "src", "github.com", "google", "gapid", rel(projectRoot, path))
		})

	// E.g. bazel-out/k8-dbg/genfiles
	genfilesOut := filepath.Join(projectRoot, "bazel-out", *bazelOutDirectory, "genfiles")
	fmt.Println("Collecting generated .go files from:", genfilesOut)
	genfilesMappingOut := collect(genfilesOut, always).
		ifTrue(and(isFile, hasSuffix(".go"))). // Only consider .go files
		mapping(func(path string) string {
			return filepath.Join(fusedRoot, "src", "github.com", "google", "gapid", rel(genfilesOut, path))
		})

	// E.g. bazel-out/k8-dbg/bin
	// Currently just gets generated gapid files.
	binOut := filepath.Join(projectRoot, "bazel-out", *bazelOutDirectory, "bin")
	fmt.Println("Collecting generated .go files from:", binOut)
	binMappingOut := collect(binOut, always).ifTrue(and(
		isFile,
		contains(filepath.Join("github.com", "google", "gapid")),
		hasSuffix(".go"),
	)).mapping(func(path string) string {
		return filepath.Join(fusedRoot, "src", trimUpTo(rel(projectRoot, path), "github.com"))
	})

	// Get ".go" files generated from templates.
	templateGenedGofiles := collect(binOut, always).ifTrue(and(
		isFile,
		contains(filepath.Join("github.com", "google", "gapid")).not(),
		hasSuffix(".go"),
	)).mapping(func(path string) string {
		return filepath.Join(fusedRoot, "src", "github.com", "google", "gapid", rel(binOut, path))
	})

	// Get ".cpp" and ".h" files generated from templates.
	templateGenedCppfiles := collect(binOut, always).ifTrue(and(
		isFile,
		contains(filepath.Join("github.com", "google", "gapid")).not(),
		or(hasSuffix(".cpp"), hasSuffix(".h")),
	)).mapping(func(path string) string {
		return filepath.Join(fusedRoot, "src", "github.com", "google", "gapid", rel(binOut, path))
	})

	// Collect all the external package file mappings.

	// After resolving symlinks, bazel-out points to:
	// /home/paulthomson/.cache/bazel/_bazel_paulthomson/1234/execroot/gapid/bazel-out
	// We edit the path to get:
	// /home/paulthomson/.cache/bazel/_bazel_paulthomson/1234/external
	// ...which includes all externals

	// E.g. /home/paulthomson/.cache/bazel/_bazel_paulthomson/1234/execroot/gapid/bazel-out
	bazelGapidResolved, err := filepath.EvalSymlinks(filepath.Join(projectRoot, "bazel-out"))
	if err != nil {
		return err
	}
	bazelGapidResolved, err = filepath.Abs(bazelGapidResolved)
	if err != nil {
		return err
	}

	// E.g. /home/paulthomson/.cache/bazel/_bazel_paulthomson/1234/
	bazelCacheDir := filepath.Dir(filepath.Dir(filepath.Dir(bazelGapidResolved)))

	// E.g. /home/paulthomson/.cache/bazel/_bazel_paulthomson/1234/external
	bazelExternals := filepath.Join(bazelCacheDir, "external")

	extMapping := mappings{}
	for pkg, imp := range externals {
		src := filepath.Join(bazelExternals, pkg)
		fmt.Println("Collecting .go, .h and .hpp files from:", src)
		dst := filepath.Join(fusedRoot, "src", imp)
		m := collect(src, always).ifTrue(and(isFile, or(hasSuffix(".go"), hasSuffix(".h"), hasSuffix(".hpp")))).
			mapping(func(path string) string {
				return filepath.Join(dst, rel(src, path))
			})
		extMapping = append(extMapping, m...)
	}

	thirdPartiesOut := filepath.Join(projectRoot, "bazel-out", *bazelOutDirectory, "bin", "tools", "build", "third_party")
	fmt.Println("Collecting generated .go from:", thirdPartiesOut)
	perfettoProtosMappingOut := collect(thirdPartiesOut, always).ifTrue(and(
		isFile,
		contains(filepath.Join("protos", "perfetto")),
		hasSuffix(".go")),
	).mapping(func(path string) string {
		return filepath.Join(fusedRoot, "src", trimUpTo(rel(projectRoot, path), "protos"))
	})
	// Every mapping we're going to deal with.
	allMappings := join(srcMapping, genfilesMappingOut, binMappingOut, templateGenedGofiles, templateGenedCppfiles, extMapping, perfettoProtosMappingOut)

	// Remove all existing symlinks in the fused directory that are not part of the
	// mappings. This may never happen if the OS automatically deletes deleted
	// symlink targets.
	if err := fusedFiles.ifFalse(allMappings.dsts().set().contains).foreach(remove); err != nil {
		return err
	}

	// Create symlinks for all of the missing mappings.
	if err := allMappings.ifDstFalse(fusedFiles.set().contains).foreach(mapping.symlink); err != nil {
		return err
	}

	// Finally remove any empty directories
	dirs := collect(fusedRoot, isDir).
		ifFalse(contains(".git")) // In case you go-get your go tools into the fused dir
	for len(dirs) > 0 { // Reverse loop to delete child directories first
		dir := dirs[len(dirs)-1]
		dirs = dirs[:len(dirs)-1]
		if isEmpty(dir) {
			fmt.Println("Removing empty directory", dir)
			os.Remove(dir)
		}
	}

	return nil
}

// A predicate function.
type pred func(string) bool

// A string transform.
type transform func(string) string

// A collection of file paths.
type paths []string

// A mapping from source to destination path.
type mapping struct{ src, dst string }

// A collection of mappings.
type mappings []mapping

// A unique set of strings.
type set map[string]struct{}

// mapping returns a new list of mappings by transforming the source paths in
// l to destination paths using the transform t.
func (l paths) mapping(t transform) mappings {
	out := make(mappings, len(l))
	for i, p := range l {
		out[i] = mapping{p, t(p)}
	}
	return out
}

// ifTrue returns all the paths in l where the predicate p returns true.
func (l paths) ifTrue(p pred) paths {
	out := make(paths, 0, len(l))
	for _, s := range l {
		if p(s) {
			out = append(out, s)
		}
	}
	return out
}

// ifTrue returns all the paths in l where the predicate p returns false.
func (l paths) ifFalse(p pred) paths {
	return l.ifTrue(p.not())
}

// set returns a new set from the list of paths.
func (l paths) set() set {
	out := set{}
	for _, s := range l {
		out[s] = struct{}{}
	}
	return out
}

// foreach calls f for each path in l.
func (l paths) foreach(f func(string) error) error {
	for _, m := range l {
		if err := f(m); err != nil {
			return err
		}
	}
	return nil
}

// join returns the concatenated list of mappings in l.
func join(l ...mappings) mappings {
	out := mappings{}
	for _, m := range l {
		out = append(out, m...)
	}
	return out
}

// dsts returns the destination paths of all the mappings in l.
func (l mappings) dsts() paths {
	out := make(paths, len(l))
	for i, p := range l {
		out[i] = p.dst
	}
	return out
}

// ifDstTrue returns all the mappings in l where the predicate p returns true
// for the destination path.
func (l mappings) ifDstTrue(p pred) mappings {
	out := make(mappings, 0, len(l))
	for _, m := range l {
		if p(m.dst) {
			out = append(out, m)
		}
	}
	return out
}

// ifDstTrue returns all the mappings in l where the predicate p returns false
// for the destination path.
func (l mappings) ifDstFalse(p pred) mappings {
	return l.ifDstTrue(p.not())
}

// foreach calls f for each mapping in l.
func (l mappings) foreach(f func(mapping) error) error {
	for _, m := range l {
		if err := f(m); err != nil {
			return err
		}
	}
	return nil
}

// symlink creates a symlink from the mapping source to the mapping destination.
func (m mapping) symlink() error {
	fmt.Println("--- Symlinking source file:\n", m.src, "->", m.dst)
	dir, _ := filepath.Split(m.dst)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.Symlink(m.src, m.dst)
}

// contains returns true if the set s contains str.
func (s set) contains(str string) bool {
	_, ok := s[str]
	return ok
}

// remove deletes the file or directory at p.
func remove(p string) error {
	fmt.Println("--- Removing symlink:\n", p)
	return os.Remove(p)
}

// hasPrefix returns a prediacte that returns true iff the string begins with
// str.
func hasPrefix(str string) pred {
	return func(p string) bool { return strings.HasPrefix(p, str) }
}

// hasSuffix returns a prediacte that returns true iff the string ends with str.
func hasSuffix(str string) pred {
	return func(p string) bool { return strings.HasSuffix(p, str) }
}

// contains returns a predicate that returns true iff the string contains the
// substring s.
func contains(s string) pred {
	return func(p string) bool { return strings.Contains(p, s) }
}

// not inverses the predicate.
func (f pred) not() pred {
	return func(p string) bool { return !f(p) }
}

// always is a predicate that always returns true.
func always(string) bool { return true }

// or returns a predicate that returns true if any of the predicates in l
// return true.
func or(l ...pred) pred {
	return func(path string) bool {
		for _, f := range l {
			if f(path) {
				return true
			}
		}
		return false
	}
}

// or returns a predicate that returns true if all of the predicates in l
// return true.
func and(list ...pred) pred {
	return func(path string) bool {
		for _, f := range list {
			if !f(path) {
				return false
			}
		}
		return true
	}
}

func fromBool(b bool) pred {
	return func(path string) bool {
		return b
	}
}

// trimUpTo returns a string with all runes in str omitted up to (but not
// including) the first occurance of pat. If str does not contain pat, then str
// is returned.
func trimUpTo(str, pat string) string {
	i := strings.Index(str, pat)
	if i > 0 {
		return str[i:]
	}
	return str
}

// rel returns the relative path of target from base. rel panics on error.
func rel(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		panic(err)
	}
	return rel
}

// collect returns all the file paths under root that pass the predicate p.
// Unlike filepath.Walk, symlinked directories are also traversed.
func collect(root string, p pred) paths {
	out := paths{}
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !p(path) {
			return nil
		}
		if info != nil && info.Mode()&os.ModeSymlink != 0 && isDir(path) {
			pred := and(p, func(s string) bool { return s != path })

			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			for _, p := range collect(target, pred) {
				out = append(out, filepath.Join(path, rel(target, p)))
			}
		}
		out = append(out, path)
		return nil
	})
	return out
}

// projectRoot searches upwards from the current working directory for the first
// directory containing a file called 'WORKSPACE'.
func projectRoot() (string, error) {
	root, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("Could not get the current working directory: %v", err)
	}
	for isDir(root) {
		if isFile(filepath.Join(root, "WORKSPACE")) {
			return root, nil
		}
		root, _ = filepath.Split(root)
	}
	return "", fmt.Errorf("Couldn't find project root from CWD")
}

// isFile is a predicate that returns true if there is a file at path that is
// not a directory.
func isFile(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !fi.IsDir()
}

// isFile is a predicate that returns true if there is a directory at path.
func isDir(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.IsDir()
}

// isSymlink is a predicate that returns true if there is a symlink.
func isSymlink(path string) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeSymlink) != 0
}

// isEmpty is a predicate that returns true if the directory at path is empty.
func isEmpty(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true
	}
	return false
}
