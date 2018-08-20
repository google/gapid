/*
 * Copyright (C) 2018 Google Inc.
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

import org.lwjgl.BufferUtils;

import java.nio.Buffer;
import java.nio.ByteBuffer;

/**
 * Utilities to deal with {@link Buffer Buffers}.
 */
public class Buffers {
  private Buffers() {
  }

  // To work around JDK 9.
  public static <T extends Buffer> T flip(T buffer) {
    buffer.flip();
    return buffer;
  }

  public static ByteBuffer nativeBuffer(byte[] data) {
    return flip(BufferUtils.createByteBuffer(data.length).put(data));
  }
}
