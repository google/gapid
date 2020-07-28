// Copyright (C) 2019 Google Inc.
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

// This is a very brittle script (see giveUp()) that tries to update the swt.bzl
// and jface.bzl files to upgrade SWT and JFace to a new Eclipse release
// version. To update to a newer version, visit
// https://download.eclipse.org/eclipse/downloads/, choose your version and
// click on it. The version to pass to this script is the last URL segment and
// should take the form "[RS]-<version>-<datetime>".
// This tool uses a bunch of regex's to parse HTML and the .bzl files, and does
// so in a rigid, but defensive manner. This means that trivial changes to the
// Eclipse HTML or the .bzl files will likely make the parsing fail, but at
// least if it succeeds, it's fairly confident that what it got is what it
// thinks it got.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/google/gapid/core/app"
	"github.com/google/gapid/core/log"
)

var (
	editSWT   = flag.Bool("swt", true, "Edit swt.bzl")
	editJFace = flag.Bool("jface", true, "Edit jface.bzl")
)

const (
	labelLinux   = "Linux (64 bit version)"
	labelWindows = "Windows (64 bit version)"
	labelMacOS   = "Mac OSX (64 bit version)"
	swtBzl       = "tools/build/third_party/swt.bzl"
	jfaceBzl     = "tools/build/third_party/jface.bzl"
)

var (
	repoBase, _    = url.Parse("https://download.eclipse.org/eclipse/downloads/drops4/")
	repoTableRegex = h3TableRegex("Repository")
	swtTableRegex  = h3TableRegex("SWT")
	tableRowRegex  = regexp.MustCompile("(?s:<tr(?: [^>]*)?>(.*?)</tr>)")
	tableCellRegex = regexp.MustCompile("(?s:<t[dh](?: [^>]*)?>(.*?)</t[dh]>)")
	linkRegex1     = regexp.MustCompile("<a(?: [^>]*)? href=\"([^\"]+)\"")
	linkRegex2     = regexp.MustCompile("<a href='([^']+)'>(.*?)</a>")
	pluginsRegex   = divRegex("dirlist")
	swtBzlRegex    = regexp.MustCompile("(?m:^_(URL|SHA)_(LINUX|WIN|OSX) = \"([^\"]*)\"$)")
	jfaceBaseRegex = regexp.MustCompile("(?m:^_BASE = \"[^\"]*\"$)")
	jfaceBzlRegex  = regexp.MustCompile(
		"(?s:struct\\(\\s*name = \"([^\"]+)\"," +
			"\\s*version = \"([^\"]+)\"," +
			"\\s*sha = \"([^\"]+)\"," +
			"\\s*sha_src = \"([^\"]+)\"," +
			")")
)

func main() {
	app.ShortHelp = "update-swt updates the SWT and JFace repository rules"
	app.ShortUsage = "<version>"
	app.Name = "update-swt"
	app.Run(run)
}

func run(ctx context.Context) error {
	if flag.NArg() != 1 {
		app.Usage(ctx, "Exactly one version expected, got %d", flag.NArg())
		return nil
	}
	version := flag.Args()[0]

	repo, err := getRepo(ctx, version)
	if err != nil {
		return log.Errf(ctx, err, "Failed to get repo for version '%s'", version)
	}

	if *editSWT {
		if err := repo.editSWTBazel(ctx); err != nil {
			return log.Errf(ctx, err, "Failed to edit the swt.bzl file")
		}
	}

	if *editJFace {
		if err := repo.editJFaceBazel(ctx); err != nil {
			return log.Errf(ctx, err, "Failed to edit the jface.bzl file")
		}
	}

	return nil
}

func giveUp(ctx context.Context, cause error, fmt string, args ...interface{}) error {
	log.E(ctx, "My input parsing is very strict and some HTML may have changed. I give up.")
	return log.Errf(ctx, cause, fmt, args...)
}

type repo struct {
	base       *url.URL
	swtLinux   *url.URL
	swtWindows *url.URL
	swtMacOS   *url.URL
}

func getRepo(ctx context.Context, version string) (*repo, error) {
	repoUrl, err := repoBase.Parse(url.PathEscape(version) + "/")
	if err != nil {
		return nil, log.Errf(ctx, err, "Version '%s' caused an invalid URL", version)
	}
	html, err := fetchString(ctx, repoUrl.String())
	if err != nil {
		return nil, log.Errf(ctx, err, "Failed to download repo page")
	}

	repoTable, err := extractTable(ctx, html, repoTableRegex)
	if err != nil {
		return nil, giveUp(ctx, err, "Failed to extract repository table")
	}
	if len(repoTable) != 2 || len(repoTable[1]) != 1 {
		return nil, giveUp(ctx, err, "Repo table has unexpected dimensions (%v)", repoTable)
	}
	base, err := extractLink(ctx, repoUrl, repoTable[1][0])
	if err != nil {
		return nil, giveUp(ctx, err, "Failed to extract repo URL")
	}

	r := &repo{base: base}

	swtTable, err := extractTable(ctx, html, swtTableRegex)
	if err != nil {
		return nil, giveUp(ctx, err, "Failed to extract SWT table")
	}
	if len(swtTable) < 2 {
		return nil, giveUp(ctx, err, "SWT table has not enough rows (%v)", swtTable)
	}
	for _, row := range swtTable[1:] { // skip over the table headers
		if len(row) != 3 {
			return nil, giveUp(ctx, err, "SWT table has unexpected dimensions (%v)", swtTable)
		}
		link, err := extractLink(ctx, repoUrl, row[1])
		if err != nil {
			return nil, giveUp(ctx, err, "Failed to extract SWT URL")
		}
		file := link.Query().Get("dropFile")
		if file == "" {
			return nil, giveUp(ctx, nil, "URL %v didn't contain a dropFile query parameter", link)
		}
		link, err = link.Parse(file)
		if err != nil {
			return nil, giveUp(ctx, err, "Drop file '%s' caused an invalid URL", file)
		}

		switch row[0] {
		case labelLinux:
			r.swtLinux = link
		case labelWindows:
			r.swtWindows = link
		case labelMacOS:
			r.swtMacOS = link
		default:
			log.I(ctx, "Ignoring unknown OS '%s'", row[0])

		}
	}

	if r.swtLinux == nil || r.swtWindows == nil || r.swtMacOS == nil {
		return nil, giveUp(ctx, nil, "Didn't find SWT for every OS: %v %v %v", r.swtLinux, r.swtWindows, r.swtMacOS)
	}

	return r, nil
}

func (r *repo) editSWTBazel(ctx context.Context) (err error) {
	data := map[string]*struct {
		url  string
		sha  string
		done int
	}{
		"LINUX": {r.swtLinux.String(), "", 0},
		"WIN":   {r.swtWindows.String(), "", 0},
		"OSX":   {r.swtMacOS.String(), "", 0},
	}
	for os, d := range data {
		d.sha, err = sha(ctx, d.url)
		if err != nil {
			return log.Errf(ctx, err, "Getting SHA for SWT for OS %s", os)
		}
	}

	src, err := ioutil.ReadFile(swtBzl)
	if err != nil {
		return giveUp(ctx, err, "Failed to read swt.bzl at %s", swtBzl)
	}
	matches := swtBzlRegex.FindAllSubmatchIndex(src, -1)
	if len(matches) != 6 {
		return giveUp(ctx, err, "Failed to match the swt.bzl regex (%b)", matches)
	}

	var b bytes.Buffer
	last := 0
	for _, loc := range matches {
		b.Write(src[last:loc[0]])
		last = loc[1]

		ty := string(src[loc[2]:loc[3]])
		os := string(src[loc[4]:loc[5]])
		if d, ok := data[os]; ok {
			b.WriteString("_" + ty + "_" + os + " = \"")
			switch ty {
			case "URL":
				b.WriteString(d.url)
				d.done |= 1
			case "SHA":
				b.WriteString(d.sha)
				d.done |= 2
			}
			b.WriteString("\"")
		}
	}
	b.Write(src[last:])

	for os, d := range data {
		if d.done != 3 {
			return giveUp(ctx, nil, "Didn't find SHA/URL for OS %s (%d)", os, d.done)
		}
	}
	if err := ioutil.WriteFile(swtBzl, b.Bytes(), 0644); err != nil {
		return giveUp(ctx, err, "Failed to write output to swt.bzl")
	}

	log.I(ctx, "I've updated %s successfully... I think.", swtBzl)
	return nil
}

func (r *repo) editJFaceBazel(ctx context.Context) error {
	base, _ := r.base.Parse("plugins/")
	html, err := fetchString(ctx, base.String())
	if err != nil {
		return log.Errf(ctx, err, "Failed fetching the plugins repo")
	}
	div, err := extractDiv(ctx, html, pluginsRegex)
	if err != nil {
		return giveUp(ctx, err, "Failed to extract plugins file list")
	}
	links, err := extractLinks(ctx, base, div)
	if err != nil {
		return giveUp(ctx, err, "Failed to extract plugins links")
	}
	links = filterJars(links)

	src, err := ioutil.ReadFile(jfaceBzl)
	if err != nil {
		return giveUp(ctx, err, "Failed to read jface.bzl at %s", jfaceBzl)
	}

	matches := jfaceBaseRegex.FindAllIndex(src, -1)
	if len(matches) != 1 {
		return giveUp(ctx, err, "Failed to match the base JFace regex (%v)", matches)
	}
	src = append(src[:matches[0][0]],
		append([]byte("_BASE = \""+base.String()+"\""), src[matches[0][1]:]...)...)

	matches = jfaceBzlRegex.FindAllSubmatchIndex(src, -1)
	if len(matches) == 0 {
		return giveUp(ctx, nil, "Failed to match the JFace regex (%v)", matches)
	}
	if len(matches) != 8 {
		log.W(ctx, "I found %d JFace jars, but I expected 8. Closing my eyes and continuing...", len(matches))
	}

	var b bytes.Buffer
	last := 0
	for _, loc := range matches {
		b.Write(src[last:loc[0]])
		last = loc[1]

		name := string(src[loc[2]:loc[3]])
		var jar, srcJar *link
		for i := range links {
			if strings.HasPrefix(links[i].label, name+"_") {
				jar = &links[i]
			} else if strings.HasPrefix(links[i].label, name+".source_") {
				srcJar = &links[i]
			}
		}
		if jar == nil || srcJar == nil {
			return giveUp(ctx, nil, "Didn't find jar/src jar for %s", name)
		}
		version := jar.label[len(name)+1 : len(jar.label)-4]
		jarSha, err := sha(ctx, jar.url.String())
		if err != nil {
			return giveUp(ctx, err, "Failed to get SHA for JAR %s", jar.label)
		}
		srcSha, err := sha(ctx, srcJar.url.String())
		if err != nil {
			return giveUp(ctx, err, "Failed to get SHA for src JAR %s", srcJar.label)
		}
		fmt.Println(jarSha + " " + srcSha)
		b.Write(src[loc[0]:loc[4]])
		b.WriteString(version)
		b.Write(src[loc[5]:loc[6]])
		b.WriteString(jarSha)
		b.Write(src[loc[7]:loc[8]])
		b.WriteString(srcSha)
		b.Write(src[loc[9]:loc[1]])
	}
	b.Write(src[last:])

	if err := ioutil.WriteFile(jfaceBzl, b.Bytes(), 0644); err != nil {
		return giveUp(ctx, err, "Failed to write output to jface.bzl")
	}

	log.I(ctx, "I've updated %s successfully... I think.", jfaceBzl)
	return nil
}

func h3TableRegex(id string) *regexp.Regexp {
	return regexp.MustCompile("(?s:<h3 id=\"" + id + "\">.*?<table [^>]*>(.*?)</table>)")
}

func divRegex(id string) *regexp.Regexp {
	return regexp.MustCompile("(?s:<div id='" + id + "'(?: [^>]*)?>(.*?)</div>)")
}

func fetch(ctx context.Context, url string) ([]byte, error) {
	log.I(ctx, "Fetching %s...", url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, giveUp(ctx, err, "Failed to download '%s'", url)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, giveUp(ctx, err, "Got a failure response (%d) from '%s'", resp.StatusCode, url)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, giveUp(ctx, err, "Failed to read response from '%s'", url)
	}
	return data, nil
}

func fetchString(ctx context.Context, url string) (string, error) {
	data, err := fetch(ctx, url)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func sha(ctx context.Context, url string) (string, error) {
	data, err := fetch(ctx, url)
	if err != nil {
		return "", log.Errf(ctx, err, "Failed to fetch content to compute SHA")
	}
	return fmt.Sprintf("%x", sha256.Sum256(data)), nil
}

func extractTable(ctx context.Context, html string, re *regexp.Regexp) ([][]string, error) {
	tables := re.FindAllStringSubmatch(html, -1)
	if len(tables) != 1 {
		log.I(ctx, "%s", html)
		return nil, log.Errf(ctx, nil, "Table regex did not match (%v)", tables)
	}

	rows := tableRowRegex.FindAllStringSubmatch(tables[0][1], -1)
	if len(rows) == 0 {
		log.I(ctx, "%s", tables[0][1])
		return nil, log.Errf(ctx, nil, "No rows found in table")
	}

	r := [][]string{}
	for _, row := range rows {
		cells := tableCellRegex.FindAllStringSubmatch(row[1], -1)
		if len(cells) == 0 {
			log.I(ctx, "%s", row[1])
			return nil, log.Errf(ctx, nil, "No cells found in row")
		}
		c := make([]string, len(cells))
		for i, cell := range cells {
			c[i] = cell[1]
		}
		r = append(r, c)
	}

	return r, nil
}

func extractDiv(ctx context.Context, html string, re *regexp.Regexp) (string, error) {
	div := re.FindAllStringSubmatch(html, -1)
	if len(div) != 1 {
		log.I(ctx, "%s", html)
		return "", log.Errf(ctx, nil, "Div regex did not match (%v)", div)
	}
	return div[0][1], nil
}

func extractLink(ctx context.Context, base *url.URL, html string) (*url.URL, error) {
	matches := linkRegex1.FindAllStringSubmatch(html, -1)
	if len(matches) != 1 {
		log.I(ctx, "%s", html)
		return nil, log.Errf(ctx, nil, "Link regex did not match (%v)", matches)
	}

	u, err := base.Parse(matches[0][1])
	if err != nil {
		return nil, log.Errf(ctx, err, "Extracted invalid url")
	}
	return u, nil
}

type link struct {
	url   *url.URL
	label string
}

func extractLinks(ctx context.Context, base *url.URL, html string) ([]link, error) {
	matches := linkRegex2.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		log.I(ctx, "%s", html)
		return nil, log.Errf(ctx, nil, "Links regex did not match (%v)", matches)
	}

	r := []link{}
	for _, match := range matches {
		u, err := base.Parse(match[1])
		if err != nil {
			return nil, log.Errf(ctx, err, "Extracted invalid url")
		}
		r = append(r, link{u, strings.TrimSpace(match[2])})
	}
	return r, nil
}

func filterJars(links []link) []link {
	r := []link{}
	for _, l := range links {
		if strings.HasSuffix(l.label, ".jar") {
			r = append(r, l)
		}
	}
	return r
}
