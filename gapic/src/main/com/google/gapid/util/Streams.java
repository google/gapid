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

import com.google.gapid.proto.stream.Stream;

public class Streams {
  // U8 represents a 8-bit unsigned, integer.
  public static final Stream.DataType U8 = newInt(false, 8);

  // F32 represents a 32-bit signed, floating-point number.
  public static final Stream.DataType F32 = newFloat(true, 7, 24);

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

  private Streams() {
  }
}
