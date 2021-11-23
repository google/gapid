/*
 * Copyright (C) 2019 Google Inc.
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

import com.google.gapid.proto.service.memory_box.MemoryBox;

/**
 * Utilities for formatting {@link MemoryBox.Value} into user-friendly Strings.
 */
public class MemoryBoxes {

  private MemoryBoxes() {
  }

  /**
   * Format the MemoryBox Value information into String format.
   */
  public static String format(MemoryBox.Value value, long rootAddress) {
    StringBuilder sb = new StringBuilder();
    switch (value.getValCase()) {
      case POD:
        // Use existing formatting method for pod values.
        Pods.append(sb, value.getPod());
        break;
      case POINTER:
        sb.append(toPointerString(value.getPointer().getAddress()));
        break;
      case SLICE:
        if (value.getSlice().hasRepresentation()) {
          Pods.append(sb, value.getSlice().getRepresentation());
          sb.append(String.format(" (%s)", toPointerString(rootAddress)));
        } else {
          sb.append("Array base: ").append(toPointerString(rootAddress));
        }
        break;
      case STRUCT:
        // No more value related info to expose.
        break;
      case ARRAY:
      if (value.getArray().hasRepresentation()) {
        Pods.append(sb, value.getArray().getRepresentation());
        sb.append(String.format(" (%s)", toPointerString(rootAddress)));
      } else {
        sb.append("Array base: ").append(toPointerString(rootAddress));
      }
        break;
      case NULL:
        sb.append("(nil)");
        break;
      default:
        break;
    }
    return sb.toString();
  }

  private static String toPointerString(long pointer) {
    if (pointer == 0) {
      return "(nil)";
    }
    String hex = "0000000" + Long.toHexString(pointer);
    if (hex.length() > 15) {
      return "*0x" + hex.substring(hex.length() - 16, hex.length());
    }
    return "*0x" + hex.substring(hex.length() - 8, hex.length());
  }
}
