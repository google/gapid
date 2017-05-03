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

//  @SuppressWarnings("unused")
//  public static interface Visitor {
//    public default void backReference(Integer id) { /* empty */ }
//    public default void pod(Integer id, Pod.Value pod) { /* empty */ }
//    public default void nil(Integer id) { /* empty */ }
//    public default void reference(Integer id) { /* empty */ }
//
//    public default void structStart(Integer id) { /* empty */ }
//    public default void field(String name) { /* empty */ }
//    public default void structEnd() { /* empty */ }
//
//    public default void mapStart(Integer id) { /* empty */ }
//    public default void key() { /* empty */ }
//    public default void value() { /* empty */ }
//    public default void mapEnd() { /* empty */ }
//
//    public Box.Type type(Box.Type type);
//
//    public default void visit(Box.Value value) {
//      Integer id = value.getValueId();
//      switch (value.getValCase()) {
//        case BACK_REFERENCE:
//          backReference(id);
//          break;
//        case POD:
//          pod(id, value.getPod());
//          break;
//        case REFERENCE:
//          Box.Reference ref = value.getReference();
//          switch (ref.getValCase()) {
//            case NULL:
//              nil(id);
//              break;
//            case VALUE:
//              reference(id);
//              visit(ref.getValue());
//              break;
//            default:
//              throw new IllegalArgumentException("Invalid reference value: " + ref.getValCase());
//          }
//          break;
//        case STRUCT:
//          structStart(id);
//          visit(value.getStruct());
//          structEnd();
//          break;
//        case MAP:
//          mapStart(id);
//          visit(value.getMap());
//          mapEnd();
//          break;
//        default:
//          throw new IllegalArgumentException("Invalid box value: " + value.getValCase());
//      }
//    }
//
//    public default void visit(Box.Struct struct) {
//      Box.StructType type = struct(type(struct.getType()));
//      if (type.getFieldsCount() != struct.getFieldsCount()) {
//        throw new IllegalArgumentException("Invalid struct value, field count doesn't match: " +
//            struct.getFieldsCount() + " != " + type.getFieldsCount());
//      }
//      for (int i = 0; i < type.getFieldsCount(); i++) {
//        field(type.getFields(i).getName());
//        type(type.getFields(i).getType()); // for back references
//        visit(struct.getFields(i));
//      }
//    }
//
//    public default void visit(Box.Map map) {
//      Box.MapType type = map(type(map.getType()));
//      type(type.getKeyType()); // for back references
//      type(type.getValueType()); // for back references
//
//      for (Box.MapEntry e : map.getEntriesList()) {
//        key();
//        visit(e.getKey());
//        value();
//        visit(e.getValue());
//      }
//    }
//  }
//
//  public static class TypeMaintinginVisitor implements Visitor {
//    private final Map<Integer, Box.Type> types = Maps.newHashMap();
//
//    public TypeMaintinginVisitor() {
//    }
//
//    @Override
//    public Box.Type type(Box.Type type) {
//      Integer id = type.getTypeId();
//      switch (type.getTyCase()) {
//        case BACK_REFERENCE:
//          return types.get(id);
//        default:
//          types.put(id, type);
//          return type;
//      }
//    }
//  }

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
        default:
          return putType(id, type);
      }
    }
  }
}
