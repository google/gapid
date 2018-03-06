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
package com.google.gapid.models;

import com.google.gapid.proto.stringtable.Stringtable;
import com.google.gapid.util.Paths;
import com.google.gapid.views.Formatter;

import java.util.Collections;
import java.util.Map;
import java.util.concurrent.atomic.AtomicReference;

/**
 * {@link Stringtable} utilities.
 */
public class Strings {
  private static final AtomicReference<Stringtable.StringTable> current =
      new AtomicReference<Stringtable.StringTable>();

  private Strings() {
  }

  public static void setCurrent(Stringtable.StringTable table) {
    current.set(table);
  }

  public static Stringtable.Msg create(String identifier) {
    return Stringtable.Msg.newBuilder().setIdentifier(identifier).build();
  }

  public static String getMessage(String identifier) {
    return getMessage(identifier, Collections.emptyMap());
  }

  public static String getMessage(Stringtable.Msg reason) {
    return getMessage(reason.getIdentifier(), reason.getArgumentsMap());
  }

  public static String getMessage(String identifier, Map<String, Stringtable.Value> arguments) {
    Stringtable.StringTable table = current.get();
    if (table != null) {
      Stringtable.Node node = getNode(table, identifier);
      if (node != null) {
        return getString(node, new StringBuilder(), arguments).toString();
      }
    }
    return identifier + (arguments == null ? "" : " " + arguments);
  }

  private static Stringtable.Node getNode(Stringtable.StringTable table, String identifier) {
    return table.getEntriesMap().get(identifier);
  }

  private static StringBuilder getString(
      Stringtable.Node node, StringBuilder sb, Map<String, Stringtable.Value> arguments) {
    switch (node.getNodeCase()) {
      case NODE_NOT_SET: return sb;
      case BLOCK:
        for (Stringtable.Node n : node.getBlock().getChildrenList()) {
          getString(n, sb, arguments);
        }
        return sb;
      case BOLD: return getString(node.getBold().getBody(), sb, arguments);
      case CODE: return getString(node.getCode().getBody(), sb, arguments);
      case FORMATTER: throw new UnsupportedOperationException("TODO"); // TODO (todo in proto)
      case HEADING: return getString(node.getHeading().getBody(), sb, arguments);
      case ITALIC: return getString(node.getItalic().getBody(), sb, arguments);
      case LINE_BREAK: return sb.append('\n');
      case LINK: return getString(node.getLink().getBody(), sb, arguments);
      case LIST:
        for (Stringtable.Node n : node.getList().getItemsList()) {
          getString(n, sb.append("• "), arguments).append('\n');
        }
        return sb;
      case PARAMETER:
        Stringtable.Value argument = arguments.get(node.getParameter().getKey());
        if (argument == null) {
          return sb.append('<').append(node.getParameter().getKey()).append('>');
        }
        // TODO formatter
        return append(sb, argument);
      case TEXT: return sb.append(node.getText().getText());
      case UNDERLINED: return getString(node.getUnderlined().getBody(), sb, arguments);
      case WHITESPACE: return sb.append(' ');
      default:
        throw new UnsupportedOperationException("Unsupported message type: " + node.getNodeCase());
    }
  }

  private static StringBuilder append(StringBuilder sb, Stringtable.Value value) {
    switch (value.getValueCase()) {
      case VALUE_NOT_SET: return sb.append("[null]");
      case BOX: return sb.append(Formatter.toString(value.getBox(), null, true));
      case PATH: return sb.append(Paths.toString(value.getPath()));
      default:
        throw new UnsupportedOperationException("Unsupported value type: " + value.getValueCase());
    }
  }
}
