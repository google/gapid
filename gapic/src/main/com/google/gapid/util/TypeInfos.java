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
import com.google.gapid.proto.service.types.TypeInfo;

/**
 * Utilities for formatting {@link TypeInfo.Type} into user-friendly Strings.
 */
public class TypeInfos {

  private TypeInfos() {
  }

  /**
   * Format the memory TypeInfo information into String format.
   */
  public static String format(TypeInfo.Type type, MemoryBox.Value value) {
    StringBuilder sb = new StringBuilder(type.getName());
    switch (type.getTyCase()) {
      case POINTER:
        if (type.getPointer().getIsConst()) {
          sb.insert(0, "const ");
        }
        break;
      case SLICE:
        sb.append('[').append(value.getSlice().getValuesCount()).append(']');
        break;
      case ARRAY:
        sb.append('[').append(value.getArray().getEntriesCount()).append(']');
        break;
      case SIZED:
        // format the type name using C type standard.
        sb.delete(0, sb.length());
        switch (type.getSized()) {
          case sized_int:
            sb.append("int");
            break;
          case sized_uint:
            sb.append("uint");
            break;
          case sized_size:
            sb.append("size_t");
            break;
          case sized_char:
            sb.append("char");
            break;
          default:
            break;
        }
        break;
      default:
        break;
    }
    return sb.toString();
  }
}
