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
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
)

const registry_url = "https://cvs.khronos.org/svn/repos/ogl/trunk/doc/registry/public/api/gl.xml"

// DownloadRegistry downloads the Khronos XML registry file.
func DownloadRegistry() *Registry {
	bytes := Download(registry_url)
	if len(bytes) == 0 {
		panic(fmt.Errorf("Can not download %s", registry_url))
	}
	reg := &Registry{}
	if err := xml.Unmarshal(bytes, reg); err != nil {
		panic(err.Error())
	}
	return reg
}

type KhronosAPI string
type Version string

const GLES1API = KhronosAPI("gles1")
const GLES2API = KhronosAPI("gles2") // Includes GLES 3.0 and later

func (v Version) String() string { return fmt.Sprintf("%s", string(v)) }

type Registry struct {
	Group     []*Group            `xml:"groups>group"`
	Enums     []*Enums            `xml:"enums"`
	Command   []*Command          `xml:"commands>command"`
	Feature   []*Feature          `xml:"feature"`
	Extension []*ExtensionElement `xml:"extensions>extension"`
}

type NamedElementList []NamedElement
type NamedElement struct {
	Name string `xml:"name,attr"`
}

type Group struct {
	NamedElement
	Enum NamedElementList `xml:"enum"`
}

type Enums struct {
	Namespace string `xml:"namespace,attr"`
	Group     string `xml:"group,attr"`
	Type      string `xml:"type,attr"` // "bitmask"
	Comment   string `xml:"comment,attr"`
	Enum      []Enum `xml:"enum"`
}

type Enum struct {
	NamedElement
	Value string     `xml:"value,attr"`
	Type  string     `xml:"type,attr"` // "u" or "ull"
	API   KhronosAPI `xml:"api,attr"`
	Alias string     `xml:"alias,attr"`
}

type Command struct {
	Proto ProtoOrParam   `xml:"proto"`
	Param []ProtoOrParam `xml:"param"`
	Alias NamedElement   `xml:"alias"`
}

type ProtoOrParam struct {
	InnerXML string `xml:",innerxml"`
	Chardata string `xml:",chardata"`
	Group    string `xml:"group,attr"`
	Length   string `xml:"len,attr"`
	Ptype    string `xml:"ptype"`
	Name     string `xml:"name"`
}

type Feature struct {
	NamedElement
	API     KhronosAPI          `xml:"api,attr"`
	Number  Version             `xml:"number,attr"`
	Require RequireOrRemoveList `xml:"require"`
	Remove  RequireOrRemoveList `xml:"remove"`
}

type ExtensionElement struct {
	NamedElement
	Supported string              `xml:"supported,attr"`
	Require   RequireOrRemoveList `xml:"require"`
	Remove    RequireOrRemoveList `xml:"remove"`
}

type RequireOrRemoveList []RequireOrRemove
type RequireOrRemove struct {
	API     KhronosAPI       `xml:"api,attr"` // for extensions only
	Profile string           `xml:"profile,attr"`
	Comment string           `xml:"comment,attr"`
	Enum    NamedElementList `xml:"enum"`
	Command NamedElementList `xml:"command"`
}

func (l NamedElementList) Contains(name string) bool {
	for _, v := range l {
		if v.Name == name {
			return true
		}
	}
	return false
}

func (r *RequireOrRemove) Contains(name string) bool {
	return r.Enum.Contains(name) || r.Command.Contains(name)
}

func (l RequireOrRemoveList) Contains(name string) bool {
	for _, v := range l {
		if v.Contains(name) {
			return true
		}
	}
	return false
}

func (e *ExtensionElement) IsSupported(api KhronosAPI) bool {
	for _, v := range strings.Split(e.Supported, "|") {
		if KhronosAPI(v) == api {
			return true
		}
	}
	return false
}

func (c Command) Name() string {
	return c.Proto.Name
}

func (p ProtoOrParam) Type() string {
	name := p.InnerXML
	name = name[:strings.Index(name, "<name>")]
	name = strings.Replace(name, "<ptype>", "", 1)
	name = strings.Replace(name, "</ptype>", "", 1)
	name = strings.TrimSpace(name)
	return name
}

// ParamsAndResult returns all parameters and return value as -1.
func (cmd *Command) ParamsAndResult() map[int]ProtoOrParam {
	result := cmd.Proto
	result.Name = "result"
	results := map[int]ProtoOrParam{-1: result}
	for i, param := range cmd.Param {
		results[i] = param
	}
	return results
}

// GetVersions returns sorted list of versions which support the given symbol.
func (r *Registry) GetVersions(api KhronosAPI, name string) []Version {
	version, found := Version(""), false
	for _, feature := range r.Feature {
		if feature.API == api {
			if feature.Require.Contains(name) {
				if found {
					panic(fmt.Errorf("redefinition of %s", name))
				}
				version, found = feature.Number, true
			}
			if feature.Remove != nil {
				// not used in GLES
				panic(fmt.Errorf("remove tag is not supported"))
			}
		}
	}
	if found {
		switch version {
		case "1.0":
			return []Version{"1.0"}
		case "2.0":
			return []Version{"2.0", "3.0", "3.1", "3.2"}
		case "3.0":
			return []Version{"3.0", "3.1", "3.2"}
		case "3.1":
			return []Version{"3.1", "3.2"}
		case "3.2":
			return []Version{"3.2"}
		default:
			panic(fmt.Errorf("Uknown GLES version: %v", version))
		}
	} else {
		return nil
	}
}

// GetExtensions returns extensions which define the given symbol.
func (r *Registry) GetExtensions(api KhronosAPI, name string) []string {
	var extensions []string
ExtensionLoop:
	for _, extension := range r.Extension {
		if extension.IsSupported(api) {
			for _, require := range extension.Require {
				if require.API == "" || require.API == api {
					if require.Contains(name) {
						extensions = append(extensions, extension.Name)
						// sometimes the extension repeats definition - ignore
						continue ExtensionLoop
					}
				}
			}
			if extension.Remove != nil {
				// not used in GLES
				panic(fmt.Errorf("remove tag is not supported"))
			}
		}
	}
	return extensions
}

var sufix_re = regexp.MustCompile("(64|)(i_|)(I|)([1-4]|[1-4]x[1-4]|)(x|ub|f|i|ui|fi|i64|)(v|)$")

func GetCoreManpage(version Version, cmdName string) (url string, data []byte) {
	var urlFormat string
	switch version {
	case "1.0":
		urlFormat = "https://www.khronos.org/opengles/sdk/1.1/docs/man/%s.xml"
	case "2.0":
		urlFormat = "https://www.khronos.org/opengles/sdk/docs/man/xhtml/%s.xml"
	case "3.0":
		urlFormat = "https://www.khronos.org/opengles/sdk/docs/man3/html/%s.xhtml"
	case "3.1":
		urlFormat = "https://www.khronos.org/opengles/sdk/docs/man31/html/%s.xhtml"
	case "3.2":
		urlFormat = "https://www.khronos.org/opengles/sdk/docs/man32/html/%s.xhtml"
	default:
		panic(fmt.Errorf("Uknown api version: %v", version))
	}

	for _, table := range []struct{ oldPrefix, newPrefix string }{
		{"glDisable", "glEnable"},
		{"glEnd", "glBegin"},
		{"glGetBoolean", "glGet"},
		{"glGetFixed", "glGet"},
		{"glGetFloat", "glGet"},
		{"glGetInteger", "glGet"},
		{"glGetnUniform", "glGetUniform"},
		{"glMemoryBarrierByRegion", "glMemoryBarrier"},
		{"glProgramUniformMatrix", "glProgramUniform"},
		{"glReadnPixels", "glReadPixels"},
		{"glUniformMatrix", "glUniform"},
		{"glUnmapBuffer", "glMapBufferRange"},
		{"glVertexAttribIFormat", "glVertexAttribFormat"},
		{"glVertexAttribIPointer", "glVertexAttribPointer"},
		{"", ""}, // no-op
	} {
		if strings.HasPrefix(cmdName, table.oldPrefix) {
			// Replace prefix
			cmdName := table.newPrefix + strings.TrimPrefix(cmdName, table.oldPrefix)
			// Try to download URL without suffix
			if sufix_re.MatchString(cmdName) {
				cmdName := sufix_re.ReplaceAllString(cmdName, "")
				url = fmt.Sprintf(urlFormat, cmdName)
				if data := Download(url); len(data) > 0 {
					return url, data
				}
			}
			// Try to download URL with suffix
			url = fmt.Sprintf(urlFormat, cmdName)
			if data := Download(url); len(data) > 0 {
				return url, data
			}
		}
	}
	panic(fmt.Errorf("Failed to find URL for %s", cmdName))
}

func GetExtensionManpage(extension string) (url string, data []byte) {
	parts := strings.Split(extension, "_")
	vendor := parts[1]
	page := strings.Join(parts[2:], "_")
	url = fmt.Sprintf("https://www.khronos.org/registry/gles/extensions/%s/%s_%s.txt", vendor, vendor, page)
	if data := Download(url); len(data) > 0 {
		return url, data
	}
	for _, table := range []struct{ extension, page string }{
		{"GL_NV_coverage_sample", "NV/EGL_NV_coverage_sample.txt"},
		{"GL_NV_depth_nonlinear", "NV/EGL_NV_depth_nonlinear.txt"},
		{"GL_NV_EGL_NV_coverage_sample", "NV/EGL_NV_coverage_sample.txt"},
		{"GL_EXT_separate_shader_objects", "EXT/EXT_separate_shader_objects.gles.txt"},
	} {
		if table.extension == extension {
			url = fmt.Sprintf("https://www.khronos.org/registry/gles/extensions/%s", table.page)
			if data := Download(url); len(data) > 0 {
				return url, data
			}
		}
	}
	panic(fmt.Errorf("Failed to find URL for %s", extension))
}

// Download the given URL.  Returns empty slice if the page can not be found (404).
func Download(url string) []byte {
	filename := url
	filename = strings.TrimPrefix(filename, "https://")
	filename = strings.Replace(filename, "/", "-", strings.Count(filename, "/")-1)
	filename = strings.Replace(filename, "/", string(os.PathSeparator), 1)
	filename = *cacheDir + string(os.PathSeparator) + filename
	if bytes, err := ioutil.ReadFile(filename); err == nil {
		return bytes
	}
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	bytes := []byte{}
	if resp.StatusCode == 200 {
		bytes, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
	} else if resp.StatusCode != 404 {
		panic(fmt.Errorf("%s: %s", url, resp.Status))
	}
	resp.Body.Close()
	dir := filename[0:strings.LastIndex(filename, string(os.PathSeparator))]
	if err := os.MkdirAll(dir, 0750); err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(filename, bytes, 0666); err != nil {
		panic(err)
	}
	return bytes
}
