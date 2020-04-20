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
package com.google.gapid.util;

import com.google.common.collect.ImmutableMap;
import com.google.gapid.proto.stream.Stream;

import java.util.Map;

/**
 * Data {@link Stream} utilities.
 */
public class Streams {
  // U8 represents a 8-bit unsigned, integer.
  public static final Stream.DataType U8 = newInt(false, 8);

  // F10 represents a 10-bit unsigned floating-point number.
  public static final Stream.DataType F10 = newFloat(false, 5, 5);
  // F11 represents a 11-bit unsigned floating-point number.
  public static final Stream.DataType F11 = newFloat(false, 5, 6);
  // F16 represents a 16-bit signed, floating-point number.
  public static final Stream.DataType F16 = newFloat(true, 5, 10);
  // F32 represents a 32-bit signed, floating-point number.
  public static final Stream.DataType F32 = newFloat(true, 8, 23);
  // F64 represents a 64-bit signed, floating-point number.
  public static final Stream.DataType F64 = newFloat(true, 11, 52);

  // LINEAR is a Sampling state using a linear curve.
  public static final Stream.Sampling LINEAR = Stream.Sampling.newBuilder()
      .setCurve(Stream.Curve.Linear)
      .build();

  // LINEAR_NORMALIZED is a Sampling state using a linear curve with normalized sampling.
  public static final Stream.Sampling LINEAR_NORMALIZED = Stream.Sampling.newBuilder()
          .setCurve(Stream.Curve.Linear)
          .setNormalized(true)
          .build();

  public static final Stream.Format FMT_XYZ_F32 = Stream.Format.newBuilder()
      .addComponents(Stream.Component.newBuilder()
          .setChannel(Stream.Channel.X)
          .setDataType(F32)
          .setSampling(LINEAR))
      .addComponents(Stream.Component.newBuilder()
          .setChannel(Stream.Channel.Y)
          .setDataType(F32)
          .setSampling(LINEAR))
      .addComponents(Stream.Component.newBuilder()
          .setChannel(Stream.Channel.Z)
          .setDataType(F32)
          .setSampling(LINEAR))
      .build();

  public static final Stream.Format FMT_RGBA_U8_NORM = Stream.Format.newBuilder()
      .addComponents(Stream.Component.newBuilder()
          .setChannel(Stream.Channel.Red)
          .setDataType(U8)
          .setSampling(LINEAR_NORMALIZED))
      .addComponents(Stream.Component.newBuilder()
          .setChannel(Stream.Channel.Green)
          .setDataType(U8)
          .setSampling(LINEAR_NORMALIZED))
      .addComponents(Stream.Component.newBuilder()
          .setChannel(Stream.Channel.Blue)
          .setDataType(U8)
          .setSampling(LINEAR_NORMALIZED))
      .addComponents(Stream.Component.newBuilder()
          .setChannel(Stream.Channel.Alpha)
          .setDataType(U8)
          .setSampling(LINEAR_NORMALIZED))
      .build();

  public static final Stream.Format FMT_RGBA_FLOAT = Stream.Format.newBuilder()
      .addComponents(Stream.Component.newBuilder()
          .setChannel(Stream.Channel.Red)
          .setDataType(F32)
          .setSampling(LINEAR))
      .addComponents(Stream.Component.newBuilder()
          .setChannel(Stream.Channel.Green)
          .setDataType(F32)
          .setSampling(LINEAR))
      .addComponents(Stream.Component.newBuilder()
          .setChannel(Stream.Channel.Blue)
          .setDataType(F32)
          .setSampling(LINEAR))
      .addComponents(Stream.Component.newBuilder()
          .setChannel(Stream.Channel.Alpha)
          .setDataType(F32)
          .setSampling(LINEAR))
      .build();

  public static final Stream.Format FMT_LUMINANCE_FLOAT = Stream.Format.newBuilder()
      .addComponents(Stream.Component.newBuilder()
          .setChannel(Stream.Channel.Luminance)
          .setDataType(F32)
          .setSampling(LINEAR))
      .build();

  public static final Stream.Format FMT_DEPTH_U8_NORM = Stream.Format.newBuilder()
      .addComponents(Stream.Component.newBuilder()
          .setChannel(Stream.Channel.Depth)
          .setDataType(U8)
          .setSampling(LINEAR_NORMALIZED))
      .build();

  public static final Stream.Format FMT_DEPTH_FLOAT = Stream.Format.newBuilder()
      .addComponents(Stream.Component.newBuilder()
          .setChannel(Stream.Channel.Depth)
          .setDataType(F32)
          .setSampling(LINEAR))
      .build();

  public static final Stream.Format FMT_COUNT_U8 = Stream.Format.newBuilder()
      .addComponents(Stream.Component.newBuilder()
          .setChannel(Stream.Channel.Count)
          .setDataType(U8)
          .setSampling(LINEAR))
      .build();

  public static Stream.DataType newInt(boolean signed, int bits) {
    return Stream.DataType.newBuilder()
        .setSigned(signed)
        .setInteger(Stream.Integer.newBuilder()
            .setBits(bits)
        ).build();
  }

  public static Stream.DataType newFloat(boolean signed, int exponentBits, int mantissaBits) {
    return Stream.DataType.newBuilder()
        .setSigned(signed)
        .setFloat(Stream.Float.newBuilder()
            .setExponentBits(exponentBits)
            .setMantissaBits(mantissaBits)
        ).build();
  }

  private static final Map<Stream.DataType, String> FIXED_DATATYPE_NAMES =
      ImmutableMap.<Stream.DataType, String>builder()
          .put(F10, "float10")
          .put(F11, "float11")
          .put(F16, "float16")
          .put(F32, "float32")
          .put(F64, "float64")
          .build();

  private Streams() {
  }

  public static String toString(Stream.Format format) {
    StringBuilder sb = new StringBuilder();
    int count = 0;
    String lastType = null;
    for (Stream.Component c : format.getComponentsList()) {
      String type = toString(c.getDataType());
      if (lastType == null || type.equals(lastType)) {
        lastType = type;
        count++;
      } else {
        append(sb, lastType, count).append(", ");
        lastType = type;
        count = 1;
      }
    }
    if (lastType != null) {
      append(sb, lastType, count);
    }
    return sb.toString();
  }

  private static StringBuilder append(StringBuilder sb, String type, int count) {
    sb.append(type);
    if (count > 1) {
      sb.append(" vec").append(count);
    }
    return sb;
  }

  public static String toString(Stream.DataType type) {
    String name = FIXED_DATATYPE_NAMES.get(type);
    if (name != null) {
      return name;
    }

    switch (type.getKindCase()) {
      case FLOAT:
        Stream.Float f = type.getFloat();
        return "float" + (type.getSigned() ? "S" : "U") + f.getMantissaBits() + "E" + f.getExponentBits();
      case FIXED:
        Stream.Fixed x = type.getFixed();
        return "fixed" + (type.getSigned() ? "S" : "U") + x.getIntegerBits() + "." + x.getFractionalBits();
      case INTEGER:
        Stream.Integer i = type.getInteger();
        return (type.getSigned() ? "s" : "u") + "int" + i.getBits();
      default:
        return "unknown";
    }
  }
}
