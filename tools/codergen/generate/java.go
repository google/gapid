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

package generate

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/gapid/core/text/copyright"
)

// JavaSettings holds the information and methods common to all java generators.
type JavaSettings struct {
	*Module                          // The module we are generating for.
	JavaPackage  string              // The java package name for this module.
	Copyright    string              // The copyright header to put on generatated files.
	MemberPrefix string              // The prefix to give all field members of classes.
	Imported     map[string]struct{} // The set of imports generated for the file.
}

// JavaClass is the struct handed to java binary coder generation templates.
type JavaClass struct {
	JavaSettings
	Struct *Struct
}

// JavaFactory is the struct handed to the java factory generation template.
type JavaFactory struct {
	JavaSettings
	Structs []*Struct
}

// Java is called by codergen to prepare and generate java code for a given module.
func Java(m *Module, info copyright.Info, gen chan Generate, path string) {
	settings := JavaSettings{
		Module:      m,
		JavaPackage: m.Directives["java.package"],
		Copyright:   strings.TrimSpace(copyright.Build("generated_java", info)),
	}
	settings.MemberPrefix, _ = m.Directives["java.member_prefix"]
	source, _ := m.Directives["java.source"]
	indent, _ := m.Directives["java.indent"]
	indent = strings.Trim(indent, `"`)
	pkgPath := strings.Replace(settings.JavaPackage, ".", "/", -1)
	factory := JavaFactory{JavaSettings: settings.clone(), Structs: []*Struct{}}
	for _, s := range m.Structs {
		if s.Tags.Get("java") == "disable" {
			continue
		}
		if settings.JavaPackage == "com.google.gapid.service.snippets" &&
			s.Name() == "fieldPath" {
			s.Exported = true
		}
		gen <- Generate{
			Name:   "Java.File",
			Arg:    JavaClass{JavaSettings: settings.clone(), Struct: s},
			Output: filepath.Join(path, source, pkgPath, settings.ClassName(s.Name())+".java"),
			Indent: indent,
		}
		factory.Structs = append(factory.Structs, s)
	}
	if len(factory.Structs) > 0 {
		gen <- Generate{
			Name:   "Java.Factory",
			Arg:    factory,
			Output: filepath.Join(path, source, pkgPath, "Factory.java"),
			Indent: indent,
		}
	}
}

func (settings JavaSettings) clone() JavaSettings {
	settings.Imported = map[string]struct{}{}
	return settings
}

// FieldName converts from a go struct field name to the correct java member name.
func (settings JavaSettings) FieldName(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	return settings.MemberPrefix + string(unicode.ToUpper(r)) + s[n:]
}

// Getter converts from a go struct field name to the correct java getter name.
func (settings JavaSettings) Getter(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	return "get" + string(unicode.ToUpper(r)) + s[n:]
}

// Setter converts from a go struct field name to the correct java getter name.
func (settings JavaSettings) Setter(s string) string {
	r, n := utf8.DecodeRuneInString(s)
	return "set" + string(unicode.ToUpper(r)) + s[n:]
}

// MethodName converts from a go public method name to a java method name.
func (settings JavaSettings) MethodName(s string) string {
	i := strings.IndexFunc(s, func(r rune) bool {
		return !unicode.IsUpper(r)
	})
	switch i {
	case -1:
		return strings.ToLower(s)
	case 0:
		return s
	default:
		return strings.ToLower(s[0:i]) + s[i:]
	}
}

// returns the module if found, the extracted type name and the modified java name
func (settings JavaSettings) moduleAndName(v interface{}) (*Module, string, string) {
	m, name := settings.ModuleAndName(v)
	java := strings.Title(name)
	java = strings.Title(strings.Replace(name, "_", "", -1))
	return m, name, java
}

func (settings JavaSettings) findClass(v interface{}) (string, string) {
	m, original, name := settings.moduleAndName(v)
	for _, t := range m.Structs {
		if t.Name() == original {
			if n := fmt.Sprint(t.Tags.Get("java")); n != "" {
				name = n
			} else {
				name += fmt.Sprint(m.Directive("java.class_suffix", ""))
			}
		}
	}
	if m == settings.Module {
		return "", name
	}
	return fmt.Sprint(m.Directive("java.package", "NotJava."+m.Name)), name
}

// Import returns the fully qualified name for the type, if not previously encountered.
func (settings JavaSettings) Import(v interface{}) string {
	pkg, name := settings.findClass(v)
	if pkg == "" || strings.HasPrefix(pkg, "NotJava") {
		return "" // Not an import
	}
	return settings.JavaImport(pkg + "." + name)
}

// JavaImport returns v unless it is already in the import map.
func (settings JavaSettings) JavaImport(v string) string {
	if _, ok := settings.Imported[v]; ok {
		return "" // Already imported
	}
	settings.Imported[v] = struct{}{}
	return v
}

// ClassName returns the Java name to give the class type.
func (settings JavaSettings) ClassName(v interface{}) string {
	_, name := settings.findClass(v)
	return name
}

// InterfaceName returns the Java name to give the interface type.
func (settings JavaSettings) InterfaceName(v interface{}) string {
	_, _, name := settings.moduleAndName(v)
	return name
}

// IsBox returns true if a type should be a boxing one.
func (settings JavaSettings) IsBox(v JavaClass) bool {
	return v.JavaPackage == "com.google.gapid.rpclib.any" &&
		v.Struct.Name() != "id_" &&
		v.Struct.Name() != "idSlice"
}

// Extends is a hacky way to work out which generated java classes need to extend somthing.
func (settings JavaSettings) Extends(v JavaClass) string {
	for _, i := range v.Struct.Implements {
		switch {
		case i.Package == "rpc" && i.Name == "Err":
			return "RpcException"
		case i.Package == "stringtable" && i.Name == "Node":
			return "Node"
		case i.Package == "vertex" && i.Name == "Format":
			return "Format"
		}
	}
	if v.JavaPackage == "com.google.gapid.service.snippets" {
		switch v.Struct.Name() {
		case "CanFollow", "Labels", "Observations":
			return "KindredSnippets"
		}
		if strings.HasSuffix(v.Struct.Name(), "Path") {
			return "Pathway"
		}
	}
	if settings.IsBox(v) {
		return "Box"
	}
	return ""
}
