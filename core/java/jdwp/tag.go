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

package jdwp

import "fmt"

// Tag is a type kind enumerator.
type Tag uint8

const (
	TagArray       = Tag(91)  // '[' - an array object (objectID size).
	TagByte        = Tag(66)  // 'B' - a byte value (1 byte).
	TagChar        = Tag(67)  // 'C' - a character value (2 bytes).
	TagObject      = Tag(76)  // 'L' - an object (objectID size).
	TagFloat       = Tag(70)  // 'F' - a float value (4 bytes).
	TagDouble      = Tag(68)  // 'D' - a double value (8 bytes).
	TagInt         = Tag(73)  // 'I' - an int value (4 bytes).
	TagLong        = Tag(74)  // 'J' - a long value (8 bytes).
	TagShort       = Tag(83)  // 'S' - a short value (2 bytes).
	TagVoid        = Tag(86)  // 'V' - a void value (no bytes).
	TagBoolean     = Tag(90)  // 'Z' - a boolean value (1 byte).
	TagString      = Tag(115) // 's' - a String object (objectID size).
	TagThread      = Tag(116) // 't' - a Thread object (objectID size).
	TagThreadGroup = Tag(103) // 'g' - a ThreadGroup object (objectID size).
	TagClassLoader = Tag(108) // 'l' - a ClassLoader object (objectID size).
	TagClassObject = Tag(99)  // 'c' - a class object object (objectID size).
)

func (t Tag) String() string {
	switch t {
	case TagArray:
		return "Array"
	case TagByte:
		return "Byte"
	case TagChar:
		return "Char"
	case TagObject:
		return "Object"
	case TagFloat:
		return "Float"
	case TagDouble:
		return "Double"
	case TagInt:
		return "Int"
	case TagLong:
		return "Long"
	case TagShort:
		return "Short"
	case TagVoid:
		return "Void"
	case TagBoolean:
		return "Boolean"
	case TagString:
		return "String"
	case TagThread:
		return "Thread"
	case TagThreadGroup:
		return "ThreadGroup"
	case TagClassLoader:
		return "ClassLoader"
	case TagClassObject:
		return "ClassObject"
	default:
		return fmt.Sprintf("Tag<%v>", int(t))
	}
}
