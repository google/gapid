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

package lingo

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/google/gapid/core/fault"
	"github.com/google/gapid/core/log"
)

type scanError struct {
	scanner   *Scanner
	offset    int
	cause     interface{}
	message   string
	watermark error
}

const scanFailure = fault.Const("No match")

func WasScannerError(v interface{}) error {
	if v == scanFailure {
		return scanFailure
	}
	if err, ok := v.(scanError); ok {
		return err
	}
	return nil
}

// Error wraps an error and message into a new error that knows the scanner stream position.
func (s *Scanner) Error(err error, msg string) error {
	result := scanError{
		scanner: s,
		offset:  s.offset,
		message: msg,
		cause:   err,
	}
	se, found := err.(scanError)
	if found {
		if strings.HasPrefix(se.message, result.message) {
			// repeated message, just add > for brevity
			result.message += ">" + se.message[len(result.message):]
		} else if result.message == "" {
			// first message in the chain
			result.message = se.message
		} else {
			// chaining a message
			result.message += "->" + se.message
		}
		result.cause = se.cause
		if result.offset < se.offset {
			result.offset = se.offset
		}
	}
	switch {
	case s.skipping:
		result.watermark = s.watermark
	case s.watermark.offset <= result.offset:
		s.watermark = result
	case s.watermark == se:
		// wrapping
		s.watermark = result
	default:
		// include watermark
		result.watermark = s.watermark
	}
	return result
}

func positionOf(data []byte, offset int) (line, column int) {
	last := 0
	line = 1
	for {
		count := bytes.IndexRune(data[last:], '\n')
		if count < 0 || last+count >= offset {
			return line, offset + 1 - last
		}
		last += count + 1
		line++
	}
}

// Error is to make scanError conform to the error interface.
// It returns a message that includes the scan stream position and associated messages/errors.
func (err scanError) Error() string {
	line, col := positionOf(err.scanner.data, err.offset)
	result := fmt.Sprintf("%s:%d:%d:%s", err.scanner.name, line, col, err.message)
	if err.cause != nil {
		result = fmt.Sprintf("%s:%s", result, err.cause)
	}
	if err.watermark != nil {
		result += fmt.Sprintf("\n    @%s", err.watermark)
	}
	return result
}

// Trace writes a message to the context at info level
func (s *Scanner) Trace(msg string) {
	line, col := positionOf(s.data, s.offset)
	log.I(s.ctx, "%s:%d:%d:%s", s.name, line, col, msg)
}
