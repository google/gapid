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
package com.google.gapid.views;

import static java.util.function.Function.identity;

import com.google.common.collect.Lists;
import com.google.common.primitives.UnsignedInts;
import com.google.common.primitives.UnsignedLongs;
import com.google.gapid.models.Follower;
import com.google.gapid.proto.core.pod.Pod;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.proto.service.box.Box;
import com.google.gapid.proto.service.memory.Memory;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.util.Boxes;
import com.google.gapid.util.IntRange;
import com.google.gapid.util.Pods;
import com.google.gapid.util.ProtoDebugTextFormat;
import com.google.gapid.widgets.Theme;
import com.google.protobuf.ByteString;
import com.google.protobuf.MessageOrBuilder;

import org.eclipse.jface.viewers.StyledString;
import org.eclipse.jface.viewers.StyledString.Styler;

import java.util.Collections;
import java.util.List;
import java.util.function.Function;

/**
 * Formats varies values to {@link StylingString StylingStrings}.
 */
public class Formatter {
  private Formatter() {
  }

  public static void format(API.Command command,
      Function<Path.ConstantSet, Service.ConstantSet> constantResolver,
      Function<String, Path.Any> followResolver,
      StylingString string, Style cmdStyle, Style infoStyle) {
    string.append(command.getName(), cmdStyle);
    string.append("(", string.structureStyle());
    boolean needComma = false;

    for (int i = 0; i < command.getParametersCount(); ++i) {
      API.Parameter field = command.getParameters(i);
      if (needComma) {
        string.append(", ", string.structureStyle());
      }
      needComma = true;
      Path.Any follow = followResolver.apply(field.getName());
      Style paramStyle = (follow == null) ? infoStyle : string.linkStyle();
      string.startLink(follow);
      string.append(field.getName(), paramStyle);
      string.append(":", (follow == null) ? string.structureStyle() : string.linkStyle());
      format(field, constantResolver, string, paramStyle);
      string.endLink();
    }

    string.append(")", string.structureStyle());
    if (!command.getTerminated()) {
      string.append("-> (did not terminate)", string.structureStyle());
      return;
    }
    if (command.hasResult()) {
      string.append("->", string.structureStyle());
      Path.Any follow = followResolver.apply(Follower.RESULT_NAME);
      string.startLink(follow);
      format(command.getResult(), constantResolver, string,
          (follow == null) ? infoStyle : string.linkStyle());
      string.endLink();
    }
  }

  public static String toString(API.Command command,
      Function<Path.ConstantSet, Service.ConstantSet> constantResolver) {
    NoStyleStylingString string = new NoStyleStylingString();
    format(command, constantResolver, s -> null, string, null, null);
    return string.toString();
  }

  private static void format(API.Parameter param,
      Function<Path.ConstantSet, Service.ConstantSet> constantResolver,
      StylingString string, Style style) {
    Box.Value value = param.getValue();
    format(value, constantResolver.apply(param.getConstants()), true, string, style);
  }

  public static void format(Box.Value value, Service.ConstantSet constants, boolean isComplete,
      StylingString string, Style style) {
    if (value.getValCase() == Box.Value.ValCase.POD) {
      format(value.getPod(), constants, isComplete, string, style);
    } else {
      format(value, new Boxes.Context(), isComplete, string, style);
    }

    if (!value.getLabel().isEmpty()) {
      string.append(" " + value.getLabel(), string.labelStyle());
    }
  }

  public static String toString(
      Box.Value value, Service.ConstantSet constants, boolean isComplete) {
    NoStyleStylingString string = new NoStyleStylingString();
    format(value, constants, isComplete, string, null);
    return string.toString();
  }

  private static void format(Pod.Value value, Service.ConstantSet constants, boolean isComplete,
      StylingString string, Style style) {
    if (constants == null || !Pods.mayBeConstant(value)) {
      format(value, isComplete, string, style);
    } else if (constants.getIsBitfield()) {
      long bits = Pods.getConstant(value);
      boolean first = true;
      for (Service.Constant constant : constants.getConstantsList()) {
        if ((bits & constant.getValue()) == constant.getValue()) {
          if (!first) {
            string.append(" | ", string.structureStyle());
          }
          string.append(constant.getName(), style);
          first = false;
          bits &= ~(constant.getValue());
        }
      }
      if (bits != 0) {
        // Uh-oh left over bits, probably an invalid value was passed by the app.
        if (!first) {
          string.append(" | ", string.structureStyle());
        }
        String hex = Long.toHexString(bits);
        if ((hex.length() & 1) == 1) {
          // Odd number of hex chars, add a leading 0.
          string.append("0x0" + hex, style);
        } else {
          string.append("0x" + hex, style);
        }
      }
    } else {
      long search = Pods.getConstant(value);
      for (Service.Constant constant : constants.getConstantsList()) {
        if (search == constant.getValue()) {
          string.append(constant.getName(), style);
          return;
        }
      }
      // Uh-oh value not found in constant set, probably an invalid value was passed by the app.
      format(value, isComplete, string, style);
    }
  }

  private static void format(
      Box.Value value, Boxes.Context ctx, boolean isComplete, StylingString string, Style style) {
    Integer id = value.getValueId();
    switch (value.getValCase()) {
      case BACK_REFERENCE:
        // TODO: If ever encountered, consider recursing.
        string.append("<ref:", string.structureStyle());
        string.append(id.toString(), style);
        string.append(">", string.structureStyle());
        break;
      case POD:
        format(value.getPod(), isComplete, string, style);
        break;
      case HANDLE:
        format(value.getHandle(), string, style);
        break;
      case POINTER:
        format(value.getPointer(), string, style);
        break;
      case SLICE:
        format(value.getSlice(), string, style);
        break;
      case REFERENCE:
        format(value.getReference(), ctx, isComplete, string, style);
        break;
      case STRUCT:
        format(value.getStruct(), ctx, isComplete, string, style);
        break;
      case MAP:
        format(value.getMap(), ctx, isComplete, string, style);
        break;
      case ARRAY:
        format(value.getArray(), ctx, isComplete, string, style);
        break;
      default:
        format(value, string, style);
    }
  }

  private static void format(Pod.Value v, boolean isComplete, StylingString string, Style style) {
    switch (v.getValCase()) {
      case VAL_NOT_SET:
        string.append("[null]", style); break;
      case STRING:
        string.append("\"", string.structureStyle());
        string.append(v.getString(), style);
        string.append("\"", string.structureStyle());
        break;
      case BOOL:
        string.append(Boolean.toString(v.getBool()), style);
        break;
      case FLOAT64:
        string.append(Double.toString(v.getFloat64()), style);
        break;
      case FLOAT32:
        string.append(Float.toString(v.getFloat32()), style);
        break;
      case SINT:
        string.append(Long.toString(v.getSint()), style);
        break;
      case SINT8:
        string.append(Integer.toString(v.getSint8()), style);
        break;
      case SINT16:
        string.append(Integer.toString(v.getSint16()), style);
        break;
      case SINT32:
        string.append(Integer.toString(v.getSint32()), style);
        break;
      case SINT64:
        string.append(Long.toString(v.getSint64()), style);
        break;
      case UINT:
        string.append(Long.toString(v.getUint()), style);
        break;
      case UINT8:
        string.append(Integer.toString(v.getUint8()), style);
        break;
      case UINT16:
        string.append(Integer.toString(v.getUint16()), style);
        break;
      case UINT32:
        string.append(uint32ToString(v.getUint32()), style);
        break;
      case UINT64:
        string.append(uint64ToString(v.getUint64()), style);
        break;
      case STRING_ARRAY:
        formatArray(v.getStringArray().getValList(), identity(), isComplete, string, style);
        break;
      case BOOL_ARRAY:
        formatArray(v.getBoolArray().getValList(), Object::toString, isComplete, string, style);
        break;
      case FLOAT64_ARRAY:
        formatArray(v.getFloat64Array().getValList(), Object::toString, isComplete, string, style);
        break;
      case FLOAT32_ARRAY:
        formatArray(v.getFloat32Array().getValList(), Object::toString, isComplete, string, style);
        break;
      case SINT_ARRAY:
        formatArray(v.getSintArray().getValList(), Object::toString, isComplete, string, style);
        break;
      case SINT8_ARRAY:
        formatArray(v.getSint8Array().getValList(), Object::toString, isComplete, string, style);
        break;
      case SINT16_ARRAY:
        formatArray(v.getSint16Array().getValList(), Object::toString, isComplete, string, style);
        break;
      case SINT32_ARRAY:
        formatArray(v.getSint32Array().getValList(), Object::toString, isComplete, string, style);
        break;
      case SINT64_ARRAY:
        formatArray(v.getSint64Array().getValList(), Object::toString, isComplete, string, style);
        break;
      case UINT_ARRAY:
        formatArray(v.getUintArray().getValList(), Object::toString, isComplete, string, style);
        break;
      case UINT8_ARRAY:
        formatArray(v.getUint8Array(), isComplete, string, style);
        break;
      case UINT16_ARRAY:
        formatArray(v.getUint16Array().getValList(), Object::toString, isComplete, string, style);
        break;
      case UINT32_ARRAY:
        formatArray(
            v.getUint32Array().getValList(), Formatter::uint32ToString, isComplete, string, style);
        break;
      case UINT64_ARRAY:
        formatArray(
            v.getUint64Array().getValList(), Formatter::uint64ToString, isComplete, string, style);
        break;
      default:
        format(v, string, style);
    }
  }

  private static String uint32ToString(int val) {
    return UnsignedInts.toString(val);
  }

  private static String uint64ToString(long val) {
    return UnsignedLongs.toString(val);
  }

  private static String byteToString(byte b) {
    String s = Integer.toHexString(b & 0xFF);
    return (s.length() == 1) ? "0x0" + s : "0x" + s;
  }

  private static final int MAX_DISPLAY = 4;
  private static <T> void formatArray(List<T> list, Function<T, String> formatter,
      boolean isComplete, StylingString string, Style style) {
    string.append("[", string.structureStyle());
    if (!list.isEmpty()) {
      string.append(formatter.apply(list.get(0)), style);
      for (int i = 1; i < Math.min(MAX_DISPLAY, list.size()); i++) {
        string.append(", ", string.structureStyle());
        string.append(formatter.apply(list.get(i)), style);
      }
      if (list.size() > MAX_DISPLAY || !isComplete) {
        string.append(", ...", string.structureStyle());
      }
    }
    string.append("]", string.structureStyle());
  }

  private static void formatArray(
      ByteString bytes, boolean isComplete, StylingString string, Style style) {
    string.append("[", string.structureStyle());
    if (!bytes.isEmpty()) {
      string.append(byteToString(bytes.byteAt(0)), style);
      for (int i = 1; i < Math.min(MAX_DISPLAY, bytes.size()); i++) {
        string.append(", ", string.structureStyle());
        string.append(byteToString(bytes.byteAt(i)), style);
      }
      if (bytes.size() > MAX_DISPLAY || !isComplete) {
        string.append(", ...", string.structureStyle());
      }
    }
    string.append("]", string.structureStyle());
  }

  private static void format(Box.Handle handle, StylingString string, Style style) {
    string.append("0x", string.structureStyle());
    string.append(Long.toHexString(handle.getValue()), style);
  }

  private static void format(Box.Pointer pointer, StylingString string, Style style) {
    if (pointer.getAddress() == 0) {
      string.append("(nil)", string.structureStyle());
    } else {
      string.append("*", string.structureStyle());
      string.append(toPointerString(pointer.getAddress()), style);
    }
  }

  private static void format(Box.Slice slice, StylingString string, Style style) {
    if (slice.getCount() == 0 && slice.getBase().getAddress() == 0) {
      string.append("(nil)", string.structureStyle());
      return;
    }

    if (slice.getType() != Pod.Type.any && slice.getType() != Pod.Type.UNRECOGNIZED) {
      string.append(slice.getType().name(), style);
    } else {
      string.append("uint8", style);
    }
    string.append("[", string.structureStyle());
    string.append(String.valueOf(slice.getCount()), style);
    string.append("]", string.structureStyle());

    string.append(" (*", string.structureStyle());
    if (slice.getPool() != Memory.PoolNames.Application_VALUE) {
      string.append("p" + slice.getPool(), style);
      string.append("[", string.structureStyle());
      string.append(toPointerString(slice.getBase().getAddress()), style);
      string.append("]", string.structureStyle());
    } else {
      string.append(toPointerString(slice.getBase().getAddress()), style);
    }
    string.append(")", string.structureStyle());
  }

  private static String toPointerString(long pointer) {
    String hex = "0000000" + Long.toHexString(pointer);
    if (hex.length() > 15) {
      return "0x" + hex.substring(hex.length() - 16, hex.length());
    }
    return "0x" + hex.substring(hex.length() - 8, hex.length());
  }

  private static void format(
      Box.Reference ref, Boxes.Context ctx, boolean isComplete, StylingString string, Style style) {
    switch (ref.getValCase()) {
      case NULL:
        ctx.unbox(ref.getNull());
        string.append("(nil)", string.structureStyle());
        break;
      case VALUE:
        format(ref.getValue(), ctx, isComplete, string, style);
        break;
      default:
        format(ref, string, style);
    }
  }

  private static void format(
      Box.Struct struct, Boxes.Context ctx, boolean isComplete, StylingString string, Style style) {
    Box.StructType type = Boxes.struct(ctx.unbox(struct.getType()));

    string.append("{", string.structureStyle());
    int count = Math.min(type.getFieldsCount(), struct.getFieldsCount());
    for (int i = 0; i < count; i++) {
      if (i > 0) {
        string.append(", ", string.structureStyle());
      }

      ctx.unbox(type.getFields(i).getType()); // for back references
      string.append(type.getFields(i).getName(), style);
      string.append(":", string.structureStyle());
      format(struct.getFields(i), ctx, isComplete, string, style);
    }
    string.append("}", string.structureStyle());

    for (int i = count; i < type.getFieldsCount(); i++) {
      ctx.unbox(type.getFields(i).getType()); // for back references
    }
  }

  private static void format(
      Box.Map map, Boxes.Context ctx, boolean isComplete, StylingString string, Style style) {
    // TODO - from old serialization style: it looked like this was only ever used for empty maps?

    Box.MapType type = Boxes.map(ctx.unbox(map.getType()));
    ctx.unbox(type.getKeyType()); // for back references
    ctx.unbox(type.getValueType()); // for back references

    string.append("{", string.structureStyle());
    for (int i = 0; i < Math.min(MAX_DISPLAY, map.getEntriesCount()); i++) {
      if (i > 0) {
        string.append(", ", string.structureStyle());
      }

      Box.MapEntry e = map.getEntries(i);
      format(e.getKey(), ctx, isComplete, string, style);
      string.append("=", string.structureStyle());
    }
    if (!isComplete || map.getEntriesCount() > MAX_DISPLAY) {
      string.append(", ...", string.structureStyle());
    }
    string.append("}", string.structureStyle());
  }

  private static void format(
      Box.Array arr, Boxes.Context ctx, boolean isComplete, StylingString string, Style style) {
    ctx.unbox(arr.getType());
    string.append("[", string.structureStyle());
    if (arr.getEntriesCount() > 0) {
      format(arr.getEntries(0), ctx, isComplete, string, style);
      for (int i = 1; i < Math.min(MAX_DISPLAY, arr.getEntriesCount()); i++) {
        string.append(", ", string.structureStyle());
        format(arr.getEntries(i), ctx, isComplete, string, style);
      }
      if (!isComplete || arr.getEntriesCount() > MAX_DISPLAY) {
        string.append(", ...", string.structureStyle());
      }
    }
    string.append("]", string.structureStyle());
  }

  private static void format(MessageOrBuilder proto, StylingString string, Style style) {
    // This is the fallback, which if used means a bug.
    string.append("!!BUG!!" + ProtoDebugTextFormat.shortDebugString(proto), style);
  }

  public static String commandIndex(Path.Command cmd) {
    return index(cmd.getIndicesList());
  }

  public static String firstIndex(Path.Commands cmd) {
    return index(cmd.getFromList());
  }

  public static String lastIndex(Path.Commands cmd) {
    return index(cmd.getToList());
  }

  public static String toString(Service.Constant constant) {
    return constant.getName() + " (0x" + Long.toHexString(constant.getValue()) + ")";
  }

  public static String index(List<Long> cmd) {
    switch (cmd.size()) {
      case 0: return "";
      case 1: return Long.toString(cmd.get(0));
      default:
        StringBuilder sb = new StringBuilder();
        sb.append(cmd.get(0));
        for (int i = 1; i < cmd.size(); i++) {
          sb.append('.').append(cmd.get(i));
        }
        return sb.toString();
    }
  }

   /**
   * Tagging interface implemented by the various styles.
   */
  public static interface Style {
    // Empty tagging interface.
  }

  /**
   * String consisting of styled segments.
   */
  public static interface StylingString {
    /**
     * @return the default "no style" style.
     */
    public Style defaultStyle();

    /**
     * @return the style to use for structure elements like '{' or ','.
     */
    public Style structureStyle();

    /**
     * @return the style to use for identifiers.
     */
    public Style identifierStyle();

    /**
     * @return the style to use for labels.
     */
    public Style labelStyle();

    /**
     * @return the style to use for disabled labels.
     */
    public Style disabledLabelStyle();

    /**
     * @return the style to use for partially disabled labels.
     */
    public Style semiDisabledLabelStyle();

    /**
     * @return the style to use for links.
     */
    public Style linkStyle();

    /**
     * @return the style to use for errors (e.g. in the report view).
     */
    public Style errorStyle();

    /**
     * @return the style to use for warnings (e.g. in the report view).
     */
    public Style warningStyle();

    /**
     * Appends the given segment with the given style to this string.
     */
    public StylingString append(String text, Style style);

    /**
     * Appends the given segment with the given style to this string, optionally eliding it.
     */
    public StylingString appendWithEllipsis(String text, Style style);

    /**
     * Indicates the start of a link with the given target.
     */
    public void startLink(Object target);

    /**
     * Indicates the end of the most recently started link.
     */
    public void endLink();
  }

  /**
   * {@link StylingString} that doesn't actually style.
   */
  private static class NoStyleStylingString implements StylingString {
    private final StringBuilder string = new StringBuilder();

    public NoStyleStylingString() {
    }

    @Override
    public StylingString append(String text, Style style) {
      string.append(text);
      return this;
    }

    @Override
    public StylingString appendWithEllipsis(String text, Style style) {
      string.append(text);
      return this;
    }

    @Override
    public void startLink(Object target) {
      // Ignore.
    }

    @Override
    public void endLink() {
      // Ignore.
    }

    @Override
    public String toString() {
      return string.toString();
    }

    @Override
    public Style defaultStyle() {
      return null;
    }

    @Override
    public Style structureStyle() {
      return null;
    }

    @Override
    public Style identifierStyle() {
      return null;
    }

    @Override
    public Style labelStyle() {
      return null;
    }

    @Override
    public Style disabledLabelStyle() {
      return null;
    }

    @Override
    public Style semiDisabledLabelStyle() {
      return null;
    }

    @Override
    public Style linkStyle() {
      return null;
    }

    @Override
    public Style errorStyle() {
      return null;
    }

    @Override
    public Style warningStyle() {
      return null;
    }
  }

  /**
   * {@link StylingString} that uses the {@link Theme} stylers to style the string.
   */
  private abstract static class ThemedStylingString implements StylingString {
    private final StylerStyle deflt;
    private final StylerStyle structure;
    private final StylerStyle identifier;
    private final StylerStyle label;
    private final StylerStyle disabledLabel;
    private final StylerStyle semiDisabledLabel;
    private final StylerStyle link;
    private final StylerStyle error;
    private final StylerStyle warning;

    public ThemedStylingString(Theme theme) {
      deflt = new StylerStyle(null);
      structure = new StylerStyle(theme.structureStyler());
      identifier = new StylerStyle(theme.identifierStyler());
      label = new StylerStyle(theme.labelStyler());
      disabledLabel = new StylerStyle(theme.disabledlabelStyler());
      semiDisabledLabel = new StylerStyle(theme.semiDisabledlabelStyler());
      link = new StylerStyle(theme.linkStyler());
      error = new StylerStyle(theme.errorStyler());
      warning = new StylerStyle(theme.warningStyler());
    }

    @Override
    public Style defaultStyle() {
      return deflt;
    }

    @Override
    public Style structureStyle() {
      return structure;
    }

    @Override
    public Style identifierStyle() {
      return identifier;
    }

    @Override
    public Style labelStyle() {
      return label;
    }

    @Override
    public Style disabledLabelStyle() {
      return disabledLabel;
    }

    @Override
    public Style semiDisabledLabelStyle() {
      return semiDisabledLabel;
    }

    @Override
    public Style linkStyle() {
      return link;
    }

    @Override
    public Style errorStyle() {
      return error;
    }

    @Override
    public Style warningStyle() {
      return warning;
    }

    protected static class StylerStyle implements Style {
      public final Styler styler;

      public StylerStyle(Styler styler) {
        this.styler = styler;
      }
    }
  }

  /**
   * {@link StylingString} implementations.
   */
  public static interface LinkableStyledString extends StylingString {
    public static final int MAX_STR_LEN = 45;

    public Object getLinkTarget(int offset);
    /* Do not modify the returned string. Use this only to render this string. */
    public StyledString getString();

    public static LinkableStyledString create(Theme theme) {
      return new LinkableStyledStringImpl(theme);
    }

    /** @return a {@link LinkableStyledString} that ignores the linking part. */
    public static LinkableStyledString ignoring(Theme theme) {
      return new IgnoringLinkableStyledString(theme, false);
    }

    /** @return a {@link LinkableStyledString} that ignores the linking and ellipsis part. */
    public static LinkableStyledString ignoringAndExpanded(Theme theme) {
      return new IgnoringLinkableStyledString(theme, true);
    }

    public static class IgnoringLinkableStyledString extends ThemedStylingString
    implements LinkableStyledString {
      protected final StyledString string = new StyledString();
      private final boolean ignoreEllipsis;

      IgnoringLinkableStyledString(Theme theme, boolean ignoreEllipsis) {
        super(theme);
        this.ignoreEllipsis = ignoreEllipsis;
      }

      @Override
      public StylingString append(String text, Style style) {
        if (!ignoreEllipsis) {
          text = text.replaceAll("[\n\r]+", "[\\\\n]");
        }
        string.append(text, ((StylerStyle)style).styler);
        return this;
      }

      @Override
      public StylingString appendWithEllipsis(String text, Style style) {
        if (ignoreEllipsis) {
          string.append(text, ((StylerStyle)style).styler);
          return this;
        }
        text = text.replaceAll("[\n\r]+", "[\\\\n]");
        if (text.length() < MAX_STR_LEN + 3) {
          return append(text, style);
        } else {
          return append(text.substring(0, MAX_STR_LEN) + "...", style);
        }
      }

      @Override
      public void startLink(Object target) {
        // Ignore.
      }

      @Override
      public void endLink() {
        // Ignore.
      }

      @Override
      public Object getLinkTarget(int offset) {
        return null;
      }

      @Override
      public StyledString getString() {
        return string;
      }
    }

    public static class LinkableStyledStringImpl extends IgnoringLinkableStyledString {
      private final List<Entry> entries = Lists.newArrayList();
      private int currentStart;
      private Object currentTarget;

      LinkableStyledStringImpl(Theme theme) {
        super(theme, false);
      }

      @Override
      public void startLink(Object target) {
        endLink();
        currentStart = string.length();
        currentTarget = target;
      }

      @Override
      public void endLink() {
        if (currentTarget != null) {
          int end = string.length();
          if (end > currentStart) {
            entries.add(new Entry(new IntRange(currentStart, end - 1), currentTarget));
          }
          currentTarget = null;
        }
      }

      @Override
      public Object getLinkTarget(int offset) {
        int index = Collections.binarySearch(entries, null, (x, ignored) ->
        (offset < x.range.from) ? 1 : (offset > x.range.to) ? -1 : 0);
        return (index < 0) ? null : entries.get(index).value;
      }

      private static class Entry {
        public final IntRange range;
        public final Object value;

        public Entry(IntRange range, Object value) {
          this.range = range;
          this.value = value;
        }
      }
    }
  }
}
