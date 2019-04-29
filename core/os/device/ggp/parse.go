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

package ggp

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// ParseList contains the output from ggp xxx list command
type ParsedList struct {
	Header []string
	Rows   [][]string
}

// ColumnByName returns the content of the table for the specified column in
// a list of string.
func (t ParsedList) ColumnByName(name string) ([]string, error) {
	if len(t.Header) == 0 {
		return []string{}, nil
	}
	ci := -1
	for i, h := range t.Header {
		if h == name {
			ci = i
		}
	}
	if ci == -1 {
		return nil, fmt.Errorf("Could not find column with name: %v", name)
	}
	ret := make([]string, len(t.Rows))
	for i, r := range t.Rows {
		ret[i] = r[ci]
	}
	return ret, nil
}

// ParseListOutput parses the output from ggp xxx list command.
// For example:
// blablabla
// header1    header2    header3
// =======    =======    =======
// value1     value2     value3
func ParseListOutput(stdout *bytes.Buffer) (*ParsedList, error) {
	reader := bufio.NewReader(stdout)
	schemaLine := ""
	contentLines := []string{}
	fieldIndices := []int{}
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		line = strings.TrimSuffix(line, "\n")
		if strings.HasPrefix(line, "no results") {
			return &ParsedList{}, nil
		}
		if strings.HasPrefix(line, "==") {
			line = strings.TrimSpace(line)
			for i, _ := range line {
				if i == 0 {
					fieldIndices = append(fieldIndices, i)
					continue
				}
				if line[i] == '=' && line[i-1] == ' ' {
					fieldIndices = append(fieldIndices, i)
				}
				if line[i] != '=' && line[i] != ' ' {
					return nil, fmt.Errorf("Separator line contains character other than '=' and blankspace")
				}
			}
			continue
		}
		if len(fieldIndices) == 0 {
			schemaLine = line
		} else {
			contentLines = append(contentLines, line)
		}
	}

	if len(schemaLine) == 0 {
		return nil, fmt.Errorf("No table header (Column names) found")
	}

	extractFieldsFromLine := func(line string) ([]string, error) {
		ret := make([]string, 0, len(fieldIndices))
		for i, _ := range fieldIndices {
			start := fieldIndices[i]
			var end int
			if i == len(fieldIndices)-1 {
				end = len(line)
			} else {
				end = fieldIndices[i+1]
			}
			if start >= len(line) || end > len(line) {
				return nil, fmt.Errorf("Unexpected length of line: %v, substr [%v:%d] failed", len(line), start, end)
			}
			ret = append(ret, strings.TrimSpace(line[start:end]))
		}
		return ret, nil
	}

	schema, err := extractFieldsFromLine(schemaLine)
	if err != nil {
		return nil, err
	}
	rows := make([][]string, len(contentLines))
	for i, l := range contentLines {
		fields, err := extractFieldsFromLine(l)
		if err != nil {
			return nil, err
		}
		rows[i] = fields
	}
	return &ParsedList{
		Header: schema,
		Rows:   rows,
	}, nil
}
