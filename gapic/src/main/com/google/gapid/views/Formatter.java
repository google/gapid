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

import com.google.common.collect.ArrayListMultimap;
import com.google.common.collect.ImmutableList;
import com.google.common.collect.Iterators;
import com.google.common.collect.Lists;
import com.google.common.collect.Multimap;
import com.google.gapid.proto.service.memory.MemoryProtos.PoolNames;
import com.google.gapid.rpclib.binary.BinaryObject;
import com.google.gapid.rpclib.schema.AnyType;
import com.google.gapid.rpclib.schema.Array;
import com.google.gapid.rpclib.schema.Constant;
import com.google.gapid.rpclib.schema.ConstantSet;
import com.google.gapid.rpclib.schema.Dynamic;
import com.google.gapid.rpclib.schema.Entity;
import com.google.gapid.rpclib.schema.Field;
import com.google.gapid.rpclib.schema.Interface;
import com.google.gapid.rpclib.schema.Map;
import com.google.gapid.rpclib.schema.Method;
import com.google.gapid.rpclib.schema.Pointer;
import com.google.gapid.rpclib.schema.Primitive;
import com.google.gapid.rpclib.schema.Slice;
import com.google.gapid.rpclib.schema.Struct;
import com.google.gapid.rpclib.schema.Type;
import com.google.gapid.service.atom.DynamicAtom;
import com.google.gapid.service.memory.MemoryPointer;
import com.google.gapid.service.memory.MemoryRange;
import com.google.gapid.service.memory.MemorySliceInfo;
import com.google.gapid.service.memory.MemorySliceMetadata;
import com.google.gapid.service.snippets.CanFollow;
import com.google.gapid.service.snippets.Labels;
import com.google.gapid.service.snippets.SnippetObject;
import com.google.gapid.util.IntRange;
import com.google.gapid.widgets.Theme;

import org.eclipse.jface.viewers.StyledString;
import org.eclipse.jface.viewers.StyledString.Styler;

import java.math.BigInteger;
import java.nio.ByteBuffer;
import java.util.Collection;
import java.util.Collections;
import java.util.Iterator;
import java.util.List;

/**
 * Formats varies values to {@link StylingString StylingStrings}.
 */
public class Formatter {
  private Formatter() {
  }

  public static String toString(SnippetObject value, Type type) {
    NoStyleStylingString string = new NoStyleStylingString();
    format(value, type, string, null);
    return string.toString();
  }

  public static void format(SnippetObject value, Type type, StylingString string, Style style) {
    if (type instanceof Primitive) {
      format(value, (Primitive)type, string, style);
    } else if (type instanceof Struct) {
      format(value, (Struct)type, string, style);
    } else if (type instanceof Pointer) {
      format(value, (Pointer)type, string, style);
    } else if (type instanceof Interface) {
      format(value, (Interface)type, string, style);
    } else if (type instanceof Array) {
      format(value, (Array)type, string, style);
    } else if (type instanceof Slice) {
      format(value, (Slice)type, string, style);
    } else if (type instanceof Map) {
      format(value, (Map)type, string, style);
    } else if (type instanceof AnyType) {
      format(value, (AnyType)type, string, style);
    } else {
      format(value, string, style);
    }
  }

  private static void format(Object value, StylingString string, Style style) {
    if (value instanceof SnippetObject) {
      format((SnippetObject)value, string, style);
    } else if (value instanceof DynamicAtom) {
      format((DynamicAtom)value, string, style);
    } else if (value instanceof MemoryPointer) {
      format((MemoryPointer)value, string, style);
    } else if (value instanceof MemoryRange) {
      format((MemoryRange)value, string, style);
    } else {
      string.append(String.valueOf(value), style);
    }
  }

  private static void format(SnippetObject obj, Primitive type, StylingString string, Style style) {
    if (tryConstantFormat(obj, type, string, style)) {
      // successfully formatted as a constant.
      return;
    }

    Object value = obj.getObject();
    // Note: casting to Number in case the value was boxed into a different Number type.
    switch (type.getMethod().getValue()) {
      case Method.BoolValue:
        string.append(String.format("%b", (Boolean)value), style);
        break;
      case Method.StringValue:
        string.appendWithEllipsis(String.valueOf(value), style);
        break;
      case Method.Float32Value:
        string.append(String.format("%f", ((Number)value).floatValue()), style);
        break;
      case Method.Float64Value:
        string.append(String.format("%f", ((Number)value).doubleValue()), style);
        break;
      default:
        string.append(toJavaIntType(type.getMethod(), (Number)value).toString(), style);
        break;
    }
  }

  private static void format(SnippetObject value, @SuppressWarnings("unused") Struct type,
      StylingString string, Style style) {
    format(value, string, style);
  }

  private static void format(SnippetObject value, @SuppressWarnings("unused") Pointer type,
      StylingString string, Style style) {
    string.append("*", string.structureStyle());
    format(value, string, style);
  }

  private static void format(SnippetObject value, @SuppressWarnings("unused") Interface type,
      StylingString string, Style style) {
    string.append("$", string.structureStyle());
    format(value, string, style);
  }

  private static void format(SnippetObject obj, Array type, StylingString string, Style style) {
    Object value = obj.getObject();
    assert (value instanceof Object[]);
    format(obj, (Object[])value, type.getValueType(), string, style);
  }

  private static void format(SnippetObject obj, Slice type, StylingString string, Style style) {
    Object value = obj.getObject();
    if (value instanceof Object[]) {
      format(obj, (Object[])value, type.getValueType(), string, style);
    } else if (value instanceof byte[]) {
      format(obj, (byte[])value, type.getValueType(), string, style);
    } else {
      assert (false);
    }
  }

  private static final int MAX_DISPLAY = 4;
  private static void format(
      SnippetObject obj, Object[] array, Type valueType, StylingString string, Style style) {
    int count = Math.min(array.length, MAX_DISPLAY);
    string.append("[", string.structureStyle());
    for (int index = 0; index < count; ++index) {
      if (index > 0) {
        string.append(",", string.structureStyle());
      }
      format(obj.elem(array[index]), valueType, string, style);
    }
    if (count < array.length) {
      string.append(", ...", string.structureStyle());
    }
    string.append("]", string.structureStyle());
  }

  private static void format(
      SnippetObject obj, byte[] array, Type valueType, StylingString string, Style style) {
    int count = Math.min(array.length, MAX_DISPLAY);
    string.append("[", string.structureStyle());
    for (int index = 0; index < count; ++index) {
      if (index > 0) {
        string.append(",", string.structureStyle());
      }
      format(obj.elem(array[index]), valueType, string, style);
    }
    if (count < array.length) {
      string.append(", ...", string.structureStyle());
    }
    string.append("]", string.structureStyle());
  }

  private static void format(SnippetObject value, Map type, StylingString string, Style style) {
    @SuppressWarnings("unchecked")
    java.util.Map<Object, Object> map = (java.util.Map<Object, Object>)value.getObject();
    Iterator<java.util.Map.Entry<Object, Object>> it = map.entrySet().iterator();

    string.append("{", string.structureStyle());
    // TODO - it looks like this is only ever used for empty maps?
    while (it.hasNext()) {
      java.util.Map.Entry<Object, Object> entry = it.next();
      format(value.key(entry), type.getKeyType(), string, style);
      string.append("=", string.structureStyle());
      SnippetObject paramValue = value.elem(entry);
      CanFollow follow = CanFollow.fromSnippets(paramValue.getSnippets());
      string.startLink(follow);
      format(
          paramValue, type.getValueType(), string, (follow == null) ? style : string.linkStyle());
      string.endLink();
      if (it.hasNext()) {
        string.append(", ", string.structureStyle());
      }
    }
    string.append("}", string.structureStyle());
  }

  private static void format(SnippetObject value, @SuppressWarnings("unused") AnyType type,
      StylingString string, Style style) {
    format(value, string, style);
  }

  private static void format(SnippetObject obj, StylingString string, Style style) {
    if (obj.getObject() instanceof Dynamic) {
      format(obj, (Dynamic)obj.getObject(), string, style);
      return;
    }
    format(obj.getObject(), string, style);
  }

  private static void format(SnippetObject obj, Dynamic dynamic, StylingString string, Style style) {
    MemoryPointer mp = tryMemoryPointer(dynamic);
    if (mp != null) {
      format(mp, string, style);
      return;
    }

    if (dynamic.getFieldCount() == 1 && dynamic.getFieldValue(0) instanceof MemorySliceInfo) {
      format((MemorySliceInfo)dynamic.getFieldValue(0), getSliceMetadata(dynamic), string, style);
      return;
    }

    string.append("{", string.structureStyle());
    for (int index = 0; index < dynamic.getFieldCount(); ++index) {
      if (index > 0) {
        string.append(", ", string.structureStyle());
      }
      Field field = dynamic.getFieldInfo(index);
      SnippetObject paramValue = obj.field(dynamic, index);
      CanFollow follow = CanFollow.fromSnippets(paramValue.getSnippets());
      Style paramStyle = (follow == null) ? style : string.linkStyle();
      string.startLink(follow);
      string.append(field.getName(), paramStyle);
      string.append(":", (follow == null) ? string.structureStyle() : string.linkStyle());
      format(paramValue, field.getType(), string, style);
      string.endLink();
    }
    string.append("}", string.structureStyle());
  }

  private static void format(MemorySliceInfo info, MemorySliceMetadata metaData,
      StylingString string, Style style) {
    if (metaData != null) {
      string.append(metaData.getElementTypeName(), style);
    }
    string.append("[", string.structureStyle());
    string.append(String.valueOf(info.getCount()), style);
    string.append("]", string.structureStyle());

    if (info.getPool() != PoolNames.Application_VALUE || info.getBase() != 0) {
      string.append(" (", string.structureStyle());
      MemoryPointer pointer = new MemoryPointer();
      pointer.setAddress(info.getBase());
      pointer.setPool(info.getPool());
      format(pointer, string, style);
      string.append(")", string.structureStyle());
    }
  }

  private static void format(MemoryPointer pointer, StylingString string, Style style) {
    if (PoolNames.Application_VALUE != pointer.getPool()) {
      if (pointer.getAddress() != 0) {
        string.append(toPointerString(pointer.getAddress()) + " ", style);
      }
      string.append("Pool: ", style);
      string.append("0x" + Long.toHexString(pointer.getPool()), style);
    } else {
      string.append(toPointerString(pointer.getAddress()), style);
    }
  }

  private static void format(MemoryRange range, StylingString string, Style style) {
    string.append(Long.toString(range.getSize()), style);
    string.append(" bytes at ", string.structureStyle());
    string.append(toPointerString(range.getBase()), style);
  }

  public static String toString(DynamicAtom atom) {
    NoStyleStylingString string = new NoStyleStylingString();
    format(atom, string, null);
    return string.toString();
  }

  public static void format(DynamicAtom atom, StylingString string, Style style) {
    string.append(atom.getName(), string.labelStyle());
    string.append("(", string.structureStyle());
    int resultIndex = atom.getResultIndex();
    int extrasIndex = atom.getExtrasIndex();
    boolean needComma = false;

    for (int i = 0; i < atom.getFieldCount(); ++i) {
      if (i == resultIndex || i == extrasIndex) continue;
      Field field = atom.getFieldInfo(i);
      if (needComma) {
        string.append(", ", string.structureStyle());
      }
      needComma = true;
      SnippetObject paramValue = SnippetObject.param(atom, i);
      CanFollow follow = CanFollow.fromSnippets(paramValue.getSnippets());
      Style paramStyle = (follow == null) ? style : string.linkStyle();
      string.startLink(follow);
      string.append(field.getDeclared(), paramStyle);
      string.append(":", (follow == null) ? string.structureStyle() : string.linkStyle());
      format(paramValue, field.getType(), string, paramStyle);
      string.endLink();
    }

    string.append(")", string.structureStyle());
    if (resultIndex >= 0) {
      string.append("->", string.structureStyle());
      SnippetObject paramValue = SnippetObject.param(atom, resultIndex);
      Field field = atom.getFieldInfo(resultIndex);
      CanFollow follow = CanFollow.fromSnippets(paramValue.getSnippets());
      string.startLink(follow);
      format(paramValue, field.getType(), string, (follow == null) ? style : string.linkStyle());
      string.endLink();
    }
  }

 private static String toPointerString(long pointer) {
    String hex = "0000000" + Long.toHexString(pointer);
    if (hex.length() > 15) {
      return "0x" + hex.substring(hex.length() - 16, hex.length());
    }
    return "0x" + hex.substring(hex.length() - 8, hex.length());
  }

  /**
   * Try to format a primitive value using it's constant name.
   * @return true if obj was formatted as a constant, false means format underlying value.
   */
  private static boolean tryConstantFormat(
      SnippetObject obj, Primitive type, StylingString string, Style style) {
    Collection<Constant> value = findConstant(obj, type);
    if (!value.isEmpty()) {
      boolean first = true;
      for (Constant constant : value) {
        if (!first) {
          string.append(" | ", style);
        }
        first = false;
        string.append(constant.getName(), string.identifierStyle());
      }
      return true;
    }
    return false;
  }

  /**
   * @return empty list if not a constant, single value for constants, more values, for bitfileds.
   */
  public static Collection<Constant> findConstant(SnippetObject obj, Primitive type) {
    final ConstantSet constants = ConstantSet.lookup(type);
    if (constants == null || constants.getEntries().length == 0) {
      return Collections.emptyList();
    }

    // first, try and find exact match
    List<Constant> byValue = constants.getByValue(obj.getObject());
    if (byValue != null && byValue.size() != 0) {
      if (byValue.size() == 1) {
        // perfect, we have just 1 match
        return byValue;
      }
      // try and find the best match
      Labels labels = Labels.fromSnippets(obj.getSnippets());
      Constant result = disambiguate(byValue, labels);
      return result == null ? Collections.emptyList() : ImmutableList.of(result);
    }

    // we can not find any exact match,
    // but for a number, maybe we can find a combination of constants that match (bit flags)
    Object value = obj.getObject();
    if (!(value instanceof Number) || value instanceof Double || value instanceof Float) {
      return Collections.emptyList();
    }

    long valueNumber = ((Number)value).longValue();
    long leftToFind = valueNumber;
    Multimap<Number, Constant> resultMap = ArrayListMultimap.create();

    for (Constant constant : constants.getEntries()) {
      long constantValue = ((Number)constant.getValue()).longValue();
      if (Long.bitCount(constantValue) == 1 && (valueNumber & constantValue) != 0) {
        resultMap.put(constantValue, constant);
        leftToFind &= ~constantValue; // remove bit
      }
    }

    // we did not find enough flags to cover this constant
    if (leftToFind != 0) {
      return Collections.emptyList();
    }

    // we found exactly 1 of each constant to cover the whole value
    if (resultMap.keySet().size() == resultMap.size()) {
      return resultMap.values();
    }

    // we have more than 1 matching constant per flag to we need to disambiguate
    Labels labels = Labels.fromSnippets(obj.getSnippets());
    for (Number key : resultMap.keySet()) {
      Collection<Constant> flagConstants = resultMap.get(key);
      if (flagConstants.size() == 1) {
        // perfect, we only have 1 value for this
        continue;
      }

      Constant con = disambiguate(flagConstants, labels);
      if (con != null) {
        // we have several values, but we found 1 to use
        resultMap.replaceValues(key, ImmutableList.of(con));
      } else {
        // we have several values and we don't know what one to use
        return Collections.emptyList();
      }
    }
    // assert all constants are disambiguated now
    assert resultMap.keySet().size() == resultMap.size();
    return resultMap.values();
  }

  private static Constant disambiguate(Collection<Constant> constants, Labels labels) {
    Collection<Constant> preferred;
    if (labels != null) {
      // There are label snippets, use them to disambiguate.
      preferred = labels.preferred(constants);
      if (preferred.size() == 1) {
        return Iterators.get(preferred.iterator(), 0);
      } else if (preferred.size() == 0) {
        // No matches, continue with the unfiltered constants.
        preferred = constants;
      }
    } else {
      preferred = constants;
    }
    // labels wasn't enough, try the heuristic.
    // Using an ambiguity threshold of 8. This side steps the most egregious misinterpretations.
    if (preferred.size() < 8) {
      return pickShortestName(preferred);
    }
    // Nothing worked we will show a numeric value.
    return null;
  }

  private static Constant pickShortestName(Collection<Constant> constants) {
    int len = Integer.MAX_VALUE;
    Constant shortest = null;
    for (Constant constant : constants) {
      int l = constant.getName().length();
      if (l < len) {
        len = l;
        shortest = constant;
      }
    }
    return shortest;
  }

  private static Number toJavaIntType(Method type, Number value) {
    switch (type.getValue()) {
      case Method.Int8Value:
        return value.byteValue();
      case Method.Uint8Value:
        return (short)(value.intValue() & 0xff);
      case Method.Int16Value:
        return value.shortValue();
      case Method.Uint16Value:
        return value.intValue() & 0xffff;
      case Method.Int32Value:
        return value.intValue();
      case Method.Uint32Value:
        return value.longValue() & 0xffffffffL;
      case Method.Int64Value:
        return value.longValue();
      case Method.Uint64Value:
        ByteBuffer buffer = ByteBuffer.allocate(Long.BYTES);
        buffer.putLong(value.longValue());
        return new BigInteger(1, buffer.array());
      default:
        throw new IllegalArgumentException("not int type: " + type);
    }
  }

  /**
   * Tries to convert a dynamic to a memory pointer if the schema representation is compatible.
   * There are several aliases for Memory.Pointer which are unique types, but we want to format
   * them as pointers.
   *
   * @param dynamic object to attempt to convert to a memory pointer.
   * @return a memory pointer if the conversion is possible, otherwise null.
   */
  private static MemoryPointer tryMemoryPointer(Dynamic dynamic) {
    Entity entity = dynamic.klass().entity();
    Field[] fields = entity.getFields();
    MemoryPointer mp = new MemoryPointer();
    Field[] mpFields = mp.klass().entity().getFields();
    if (mpFields.length != fields.length) {
      return null;
    }
    for (int i = 0; i < fields.length; ++i) {
      if (!fields[i].equals(mpFields[i])) {
        return null;
      }
    }
    long address = ((Long)dynamic.getFieldValue(0)).longValue();
    int poolId = ((Number)dynamic.getFieldValue(1)).intValue();
    mp.setAddress(address);
    mp.setPool(poolId);
    return mp;
  }

  private static MemorySliceMetadata getSliceMetadata(Dynamic dynamic) {
    BinaryObject[] metaData = dynamic.type().getMetadata();
    for (BinaryObject md : metaData) {
      if (md instanceof MemorySliceMetadata) {
        return (MemorySliceMetadata)md;
      }
    }
    return null;
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
    private final StylerStyle link;
    private final StylerStyle error;
    private final StylerStyle warning;

    public ThemedStylingString(Theme theme) {
      deflt = new StylerStyle(null);
      structure = new StylerStyle(theme.structureStyler());
      identifier = new StylerStyle(theme.identifierStyler());
      label = new StylerStyle(theme.labelStyler());
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
        string.append(text, ((StylerStyle)style).styler);
        return this;
      }

      @Override
      public StylingString appendWithEllipsis(String text, Style style) {
        if (ignoreEllipsis) {
          return append(text, style);
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
