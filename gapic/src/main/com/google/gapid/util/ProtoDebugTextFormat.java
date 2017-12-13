/*
 * Copyright (C) 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Protocol Buffers - Google's data interchange format
// Copyright 2008 Google Inc.  All rights reserved.
// https://developers.google.com/protocol-buffers/
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// Modified from proto's TextFormat to print ByteStrings as hex and removed a bunch of stuff.
package com.google.gapid.util;

import com.google.protobuf.ByteString;
import com.google.protobuf.Descriptors.EnumValueDescriptor;
import com.google.protobuf.Descriptors.FieldDescriptor;
import com.google.protobuf.Message;
import com.google.protobuf.MessageOrBuilder;
import com.google.protobuf.UnknownFieldSet;
import com.google.protobuf.WireFormat;

import java.math.BigInteger;
import java.nio.charset.StandardCharsets;
import java.util.List;
import java.util.Locale;
import java.util.Map;

/**
 * Provide text parsing and formatting support for proto2 instances.
 * The implementation largely follows google/protobuf/text_format.cc.
 *
 * @author wenboz@google.com Wenbo Zhu
 * @author kenton@google.com Kenton Varda
 */
public final class ProtoDebugTextFormat {
  private ProtoDebugTextFormat() {}

  private static final Printer SINGLE_LINE_PRINTER = new Printer();

  /**
   * Generates a human readable form of this message, useful for debugging and
   * other purposes, with no newline characters.
   */
  public static String shortDebugString(final MessageOrBuilder message) {
      final StringBuilder sb = new StringBuilder();
      SINGLE_LINE_PRINTER.print(message, sb);
      // Single line mode currently might have an extra space at the end.
      return sb.toString().trim();
  }

  /**
   * Generates a human readable form of the field, useful for debugging
   * and other purposes, with no newline characters.
   */
  public static String shortDebugString(final FieldDescriptor field,
                                        final Object value) {
      final StringBuilder sb = new StringBuilder();
      SINGLE_LINE_PRINTER.printField(field, value, sb);
      return sb.toString().trim();
  }

  /**
   * Generates a human readable form of the unknown fields, useful for debugging
   * and other purposes, with no newline characters.
   */
  public static String shortDebugString(final UnknownFieldSet fields) {
      final StringBuilder sb = new StringBuilder();
      SINGLE_LINE_PRINTER.printUnknownFields(fields, sb);
      // Single line mode currently might have an extra space at the end.
      return sb.toString().trim();
  }

  /** Helper class for converting protobufs to text. */
  private static final class Printer {
    public Printer() {}

    public void print(
        final MessageOrBuilder message, final StringBuilder generator) {
      for (Map.Entry<FieldDescriptor, Object> field
          : message.getAllFields().entrySet()) {
        printField(field.getKey(), field.getValue(), generator);
      }
      printUnknownFields(message.getUnknownFields(), generator);
    }

    public void printField(final FieldDescriptor field, final Object value,
        final StringBuilder generator) {
      if (field.isRepeated()) {
        // Repeated field.  Print each element.
        for (Object element : (List<?>) value) {
          printSingleField(field, element, generator);
        }
      } else {
        printSingleField(field, value, generator);
      }
    }

    private void printSingleField(final FieldDescriptor field,
                                  final Object value,
                                  final StringBuilder generator) {
      if (field.isExtension()) {
        generator.append("[");
        // We special-case MessageSet elements for compatibility with proto1.
        if (field.getContainingType().getOptions().getMessageSetWireFormat()
            && (field.getType() == FieldDescriptor.Type.MESSAGE)
            && (field.isOptional())
            // object equality
            && (field.getExtensionScope() == field.getMessageType())) {
          generator.append(field.getMessageType().getFullName());
        } else {
          generator.append(field.getFullName());
        }
        generator.append("]");
      } else {
        if (field.getType() == FieldDescriptor.Type.GROUP) {
          // Groups must be serialized with their original capitalization.
          generator.append(field.getMessageType().getName());
        } else {
          generator.append(field.getName());
        }
      }

      if (field.getJavaType() == FieldDescriptor.JavaType.MESSAGE) {
        generator.append(" { ");
      } else {
        generator.append(": ");
      }

      printFieldValue(field, value, generator);

      if (field.getJavaType() == FieldDescriptor.JavaType.MESSAGE) {
        generator.append("} ");
      } else {
        generator.append(" ");
      }
    }

    private void printFieldValue(final FieldDescriptor field,
                                 final Object value,
                                 final StringBuilder generator) {
      switch (field.getType()) {
        case INT32:
        case SINT32:
        case SFIXED32:
          generator.append(((Integer) value).toString());
          break;

        case INT64:
        case SINT64:
        case SFIXED64:
          generator.append(((Long) value).toString());
          break;

        case BOOL:
          generator.append(((Boolean) value).toString());
          break;

        case FLOAT:
          generator.append(((Float) value).toString());
          break;

        case DOUBLE:
          generator.append(((Double) value).toString());
          break;

        case UINT32:
        case FIXED32:
          generator.append(unsignedToString((Integer) value));
          break;

        case UINT64:
        case FIXED64:
          generator.append(unsignedToString((Long) value));
          break;

        case STRING:
          generator.append("\"");
          generator.append(escapeText((String) value));
          generator.append("\"");
          break;

        case BYTES:
          if (value instanceof ByteString) {
            generator.append(escapeBytes((ByteString) value));
          } else {
            generator.append(escapeBytes((byte[]) value));
          }
          break;

        case ENUM:
          generator.append(((EnumValueDescriptor) value).getName());
          break;

        case MESSAGE:
        case GROUP:
          print((Message) value, generator);
          break;
      }
    }

    public void printUnknownFields(final UnknownFieldSet unknownFields,
                                    final StringBuilder generator) {
      for (Map.Entry<Integer, UnknownFieldSet.Field> entry :
               unknownFields.asMap().entrySet()) {
        final int number = entry.getKey();
        final UnknownFieldSet.Field field = entry.getValue();
        printUnknownField(number, WireFormat.WIRETYPE_VARINT,
            field.getVarintList(), generator);
        printUnknownField(number, WireFormat.WIRETYPE_FIXED32,
            field.getFixed32List(), generator);
        printUnknownField(number, WireFormat.WIRETYPE_FIXED64,
            field.getFixed64List(), generator);
        printUnknownField(number, WireFormat.WIRETYPE_LENGTH_DELIMITED,
            field.getLengthDelimitedList(), generator);
        for (final UnknownFieldSet value : field.getGroupList()) {
          generator.append(entry.getKey().toString());
          generator.append(" { ");
          printUnknownFields(value, generator);
          generator.append("} ");
        }
      }
    }

    private void printUnknownField(final int number,
                                   final int wireType,
                                   final List<?> values,
                                   final StringBuilder generator) {
      for (final Object value : values) {
        generator.append(String.valueOf(number));
        generator.append(": ");
        printUnknownFieldValue(wireType, value, generator);
        generator.append(" ");
      }
    }

    private void printUnknownFieldValue(final int tag,
        final Object value,
        final StringBuilder generator) {
      switch (WireFormat.getTagWireType(tag)) {
        case WireFormat.WIRETYPE_VARINT:
          generator.append(unsignedToString((Long) value));
          break;
        case WireFormat.WIRETYPE_FIXED32:
          generator.append(
              String.format((Locale) null, "0x%08x", (Integer) value));
          break;
        case WireFormat.WIRETYPE_FIXED64:
          generator.append(String.format((Locale) null, "0x%016x", (Long) value));
          break;
        case WireFormat.WIRETYPE_LENGTH_DELIMITED:
          generator.append("\"");
          generator.append(escapeBytes((ByteString) value));
          generator.append("\"");
          break;
        case WireFormat.WIRETYPE_START_GROUP:
          printUnknownFields((UnknownFieldSet) value, generator);
          break;
        default:
          throw new IllegalArgumentException("Bad tag: " + tag);
      }
    }

    /** Convert an unsigned 32-bit integer to a string. */
    private static String unsignedToString(final int value) {
      if (value >= 0) {
        return Integer.toString(value);
      } else {
        return Long.toString(value & 0x00000000FFFFFFFFL);
      }
    }

    /** Convert an unsigned 64-bit integer to a string. */
    private static String unsignedToString(final long value) {
      if (value >= 0) {
        return Long.toString(value);
      } else {
        // Pull off the most-significant bit so that BigInteger doesn't think
        // the number is negative, then set it again using setBit().
        return BigInteger.valueOf(value & 0x7FFFFFFFFFFFFFFFL)
                         .setBit(63).toString();
      }
    }

    /**
     * Escapes bytes in the format used in protocol buffer text format, which
     * is the same as the format used for C string literals.  All bytes
     * that are not printable 7-bit ASCII characters are escaped, as well as
     * backslash, single-quote, and double-quote characters.  Characters for
     * which no defined short-hand escape sequence is defined will be escaped
     * using 3-digit octal sequences.
     */
    private static String escapeText(final String input) {
      byte[] bytes = input.getBytes(StandardCharsets.UTF_8);
      final StringBuilder builder = new StringBuilder(bytes.length);
      for (byte b : bytes) {
        switch (b) {
          // Java does not recognize \a or \v, apparently.
          case 0x07: builder.append("\\a"); break;
          case '\b': builder.append("\\b"); break;
          case '\f': builder.append("\\f"); break;
          case '\n': builder.append("\\n"); break;
          case '\r': builder.append("\\r"); break;
          case '\t': builder.append("\\t"); break;
          case 0x0b: builder.append("\\v"); break;
          case '\\': builder.append("\\\\"); break;
          case '\'': builder.append("\\\'"); break;
          case '"' : builder.append("\\\""); break;
          default:
            // Only ASCII characters between 0x20 (space) and 0x7e (tilde) are
            // printable.  Other byte values must be escaped.
            if (b >= 0x20 && b <= 0x7e) {
              builder.append((char) b);
            } else {
              builder.append('\\');
              builder.append((char) ('0' + ((b >>> 6) & 3)));
              builder.append((char) ('0' + ((b >>> 3) & 7)));
              builder.append((char) ('0' + (b & 7)));
            }
            break;
        }
      }
      return builder.toString();
    }
  }

  public static char[] escapeBytes(final ByteString input) {
    return escapeBytes(input.toByteArray());
  }

  private static final char[] HEX_CHARS = {
      '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'a', 'b', 'c', 'd', 'e', 'f'
  };
  public static char[] escapeBytes(final byte[] input) {
    char[] r = new char[input.length * 2];
    for (int i = 0, j = 0; i < input.length; i++, j += 2) {
      r[j + 0] = HEX_CHARS[(input[i] & 0xF0) >> 4];
      r[j + 1] = HEX_CHARS[input[i] & 0xF];
    }
    return r;
  }
}
