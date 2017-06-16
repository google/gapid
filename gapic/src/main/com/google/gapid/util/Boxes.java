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

import com.google.common.collect.Maps;
import com.google.gapid.proto.service.box.Box;

import java.util.Map;

public class Boxes {
  public static final Object NULL = new Object() {
    @Override
    public String toString() {
      return "null";
    }
  };

  private Boxes() {
  }

  public static Object unbox(Box.Value value) {
    return new Context().unbox(value);
  }

  public static Box.StructType struct(Box.Type type) {
    if (type.getTyCase() != Box.Type.TyCase.STRUCT) {
      throw new IllegalArgumentException("Invalid type, expected struct: " + type.getTyCase());
    }
    return type.getStruct();
  }

  public static Box.MapType map(Box.Type type) {
    if (type.getTyCase() != Box.Type.TyCase.MAP) {
      throw new IllegalArgumentException("Invalid type, expected map: " + type.getTyCase());
    }
    return type.getMap();
  }

  public static Box.ArrayType array(Box.Type type) {
    if (type.getTyCase() != Box.Type.TyCase.ARRAY) {
      throw new IllegalArgumentException("Invalid type, expected array: " + type.getTyCase());
    }
    return type.getArray();
  }

  public static byte[] getBytes(Box.Value box) {
    switch (box.getValCase()) {
      case POD: return Pods.getBytes(box.getPod());
      default:
        throw new RuntimeException("Don't know how to get bytes out of " + box.getValCase());
    }
  }

  public static class Context {
    private final Map<Integer, Object> values = Maps.newHashMap();
    private final Map<Integer, Box.Type> types = Maps.newHashMap();

    public Context() {
    }

    public Object getValue(Integer id) {
      return values.get(id);
    }

    public <T> T putValue(Integer id, T value) {
      values.put(id, value);
      return value;
    }

    public Box.Type getType(Integer id) {
      return types.get(id);
    }

    public Box.Type putType(Integer id, Box.Type type) {
      types.put(id, type);
      return type;
    }

    public Object unbox(Box.Value value) {
      Integer id = value.getValueId();
      switch (value.getValCase()) {
        case BACK_REFERENCE:
          return getValue(id);
        case POD:
          return putValue(id, Pods.unpod(value.getPod()));
        case POINTER:
          return putValue(id, value.getPointer());
        case SLICE:
          return putValue(id, value.getSlice());
        case REFERENCE:
          return putValue(id, unbox(value.getReference()));
        case STRUCT:
          return putValue(id, unbox(value.getStruct()));
        case MAP:
          return putValue(id, unbox(value.getMap()));
        case ARRAY:
          return putValue(id, unbox(value.getArray()));
        default:
          throw new IllegalArgumentException("Invalid box value: " + value.getValCase());
      }
    }

    private Object unbox(Box.Reference ref) {
      switch (ref.getValCase()) {
        case NULL:
          unbox(ref.getNull()); // for back references.
          return NULL;
        case VALUE:
          return unbox(ref.getValue());
        default: throw new IllegalArgumentException("Invalid reference value: " + ref.getValCase());
      }
    }

    private Object unbox(Box.Struct struct) {
      Box.StructType type = struct(unbox(struct.getType()));
      if (type.getFieldsCount() != struct.getFieldsCount()) {
        throw new IllegalArgumentException("Invalid struct value, field count doesn't match: " +
            struct.getFieldsCount() + " != " + type.getFieldsCount());
      }
      Map<String, Object> result = Maps.newHashMap();
      for (int i = 0; i < type.getFieldsCount(); i++) {
        result.put(type.getFields(i).getName(), unbox(struct.getFields(i)));
      }
      return result;
    }

    private Object unbox(Box.Map map) {
      map(unbox(map.getType())); // for back references.
      Map<Object, Object> result = Maps.newLinkedHashMap(); // maintain same order.
      for (Box.MapEntry e : map.getEntriesList()) {
        result.put(unbox(e.getKey()), unbox(e.getValue()));
      }
      return result;
    }

    private Object unbox(Box.Array array) {
      array(unbox(array.getType())); // for back references.
      Object[] result = new Object[array.getEntriesCount()];
      for (int i = 0; i < result.length; i++) {
        result[i] = unbox(array.getEntries(i));
      }
      return result;
    }

    public Box.Type unbox(Box.Type type) {
      Integer id = type.getTypeId();
      switch (type.getTyCase()) {
        case BACK_REFERENCE:
          return getType(id);
        case REFERENCE:
          return putType(id, unbox(type.getReference()));
        case STRUCT:
          for (Box.StructField field : type.getStruct().getFieldsList()) {
            unbox(field.getType());
          }
          return putType(id, type);
        case MAP:
          unbox(type.getMap().getKeyType());
          unbox(type.getMap().getValueType());
          return putType(id, type);
        case ARRAY:
          unbox(type.getArray().getElementType());
          return putType(id, type);
        default:
          return putType(id, type);
      }
    }
  }
}
