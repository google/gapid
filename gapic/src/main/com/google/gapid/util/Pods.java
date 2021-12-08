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

import com.google.common.primitives.UnsignedLong;
import com.google.gapid.proto.core.pod.Pod;

/**
 * Plain-Old-Data utilities.
 */
public class Pods {
  private Pods() {
  }

  public static Pod.Value pod(String s) {
    return Pod.Value.newBuilder().setString(s).build();
  }

  public static Object unpod(Pod.Value o) {
    switch (o.getValCase()) {
      case FLOAT32: return o.getFloat32();
      case FLOAT64: return o.getFloat64();
      case UINT: return UnsignedLong.fromLongBits(o.getUint());
      case SINT: return o.getSint();
      case UINT8: return o.getUint8();
      case SINT8: return o.getSint8();
      case UINT16: return o.getUint16();
      case SINT16: return o.getSint16();
      case UINT32: return o.getUint32() & 0xFFFFFFFFL;
      case SINT32: return o.getSint32();
      case UINT64: return UnsignedLong.fromLongBits(o.getUint64());
      case SINT64: return o.getSint64();
      case BOOL: return o.getBool();
      case STRING: return o.getString();
      case UINT8_ARRAY:
        return o.getUint8Array();
      case FLOAT32_ARRAY:
        Pod.Float32Array a = o.getFloat32Array();
        float[] result = new float[a.getValCount()];
        for (int i = 0; i < result.length; i++) {
          result[i] = a.getVal(i);
        }
        return result;
      default:
        // TODO handle other arrays
        throw new UnsupportedOperationException("Cannot unpod: " + o);
    }
  }

  public static StringBuilder append(StringBuilder sb, Pod.Value v) {
    switch (v.getValCase()) {
      case VAL_NOT_SET: return sb.append("[null]");
      case CHAR: {
        int charValue = v.getChar();
        if (charValue == 0) return sb.append("\\0");
        return sb.append((char)charValue);
      }
      case STRING: return sb.append(v.getString());
      case BOOL: return sb.append(v.getBool());
      case FLOAT64: return sb.append(v.getFloat64());
      case FLOAT32: return sb.append(v.getFloat32());
      case SINT: return sb.append(v.getSint());
      case SINT8: return sb.append(v.getSint8());
      case SINT16: return sb.append(v.getSint16());
      case SINT32: return sb.append(v.getSint32());
      case SINT64: return sb.append(v.getSint64());
      case UINT: return sb.append(v.getUint());
      case UINT8: return sb.append(v.getUint8());
      case UINT16: return sb.append(v.getUint16());
      case UINT32: return sb.append(v.getUint32());
      case UINT64: return sb.append(v.getUint64());
      case STRING_ARRAY: return sb.append(v.getStringArray().getValList());
      case BOOL_ARRAY: return sb.append(v.getBoolArray().getValList());
      case FLOAT64_ARRAY: return sb.append(v.getFloat64Array().getValList());
      case FLOAT32_ARRAY: return sb.append(v.getFloat32Array().getValList());
      case SINT_ARRAY: return sb.append(v.getSintArray().getValList());
      case SINT8_ARRAY: return sb.append(v.getSint8Array().getValList());
      case SINT16_ARRAY: return sb.append(v.getSint16Array().getValList());
      case SINT32_ARRAY: return sb.append(v.getSint32Array().getValList());
      case SINT64_ARRAY: return sb.append(v.getSint64Array().getValList());
      case UINT_ARRAY: return sb.append(v.getUintArray().getValList());
      case UINT8_ARRAY: return sb.append(ProtoDebugTextFormat.escapeBytes(v.getUint8Array()));
      case UINT16_ARRAY: return sb.append(v.getUint16Array().getValList());
      case UINT32_ARRAY: return sb.append(v.getUint32Array().getValList());
      case UINT64_ARRAY: return sb.append(v.getUint64Array().getValList());
      default:
        assert false;
        return sb.append(ProtoDebugTextFormat.shortDebugString(v));
    }
  }

  public static boolean mayBeConstant(Pod.Value v) {
    switch (v.getValCase()) {
      case BOOL:
      case SINT:
      case SINT8:
      case SINT16:
      case SINT32:
      case SINT64:
      case UINT:
      case UINT8:
      case UINT16:
      case UINT32:
      case UINT64:
        return true;
      default: return false;
    }
  }

  public static long getConstant(Pod.Value v) {
    switch (v.getValCase()) {
      case BOOL: return v.getBool() ? 1 : 0;
      case SINT: return v.getSint();
      case SINT8: return v.getSint8();
      case SINT16: return v.getSint16();
      case SINT32: return v.getSint32();
      case SINT64: return v.getSint64();
      case UINT: return v.getUint();
      case UINT8: return v.getUint8();
      case UINT16: return v.getUint16();
      case UINT32: return v.getUint32() & 0xFFFFFFFFL;
      case UINT64: return v.getUint64();
      default: return 0;
    }
  }

  public static Pod.Value.Builder setConstant(Pod.Value.Builder pod, long value) {
    switch (pod.getValCase()) {
      case BOOL: return pod.setBool(value != 0);
      case SINT: return pod.setSint(value);
      case SINT8: return pod.setSint8((int)value);
      case SINT16: return pod.setSint16((int)value);
      case SINT32: return pod.setSint32((int)value);
      case SINT64: return pod.setSint64(value);
      case UINT: return pod.setUint(value);
      case UINT8: return pod.setUint8((int)value);
      case UINT16: return pod.setUint16((int)value);
      case UINT32: return pod.setUint32((int)value);
      case UINT64: return pod.setUint64(value);
      default: return pod;
    }
  }

  public static boolean isInt(Pod.Value v) {
    switch (v.getValCase()) {
      case SINT:
      case SINT8:
      case SINT16:
      case SINT32:
      case UINT8:
      case UINT16:
        return true;
      default: return false;
    }
  }

  public static int getInt(Pod.Value v) {
    switch (v.getValCase()) {
      case SINT: return (int)v.getSint();
      case SINT8: return v.getSint8();
      case SINT16: return v.getSint16();
      case SINT32: return v.getSint32();
      case UINT8: return v.getUint8();
      case UINT16: return v.getUint16();
      default: return 0;
    }
  }

  public static Pod.Value.Builder setInt(Pod.Value.Builder pod, int value) {
    switch (pod.getValCase()) {
      case SINT: return pod.setSint(value);
      case SINT8: return pod.setSint8(value);
      case SINT16: return pod.setSint16(value);
      case SINT32: return pod.setSint32(value);
      case UINT8: return pod.setUint8(value);
      case UINT16: return pod.setUint16(value);
      default: return pod;
    }
  }

  public static boolean isLong(Pod.Value v) {
    switch (v.getValCase()) {
      case SINT64:
      case UINT:
      case UINT32:
      case UINT64:
        return true;
      default: return false;
    }
  }

  public static long getLong(Pod.Value v) {
    switch (v.getValCase()) {
      case SINT64: return v.getSint64();
      case UINT: return v.getUint();
      case UINT32: return v.getUint32();
      case UINT64: return v.getUint64(); // TODO unsigned
      default: return 0;
    }
  }

  public static Pod.Value.Builder setLong(Pod.Value.Builder pod, long value) {
    switch (pod.getValCase()) {
      case SINT64: return pod.setSint64(value);
      case UINT: return pod.setUint(value);
      case UINT32: return pod.setUint32((int)value);
      case UINT64: return pod.setUint64(value);
      default: return pod;
    }
  }

  public static boolean isFloat(Pod.Value v) {
    switch (v.getValCase()) {
      case FLOAT32:
      case FLOAT64:
        return true;
      default: return false;
    }
  }

  public static double getFloat(Pod.Value v) {
    switch (v.getValCase()) {
      case FLOAT32: return v.getFloat32();
      case FLOAT64: return v.getFloat64();
      default: return 0;
    }
  }

  public static Pod.Value.Builder setFloat(Pod.Value.Builder pod, double value) {
    switch (pod.getValCase()) {
      case FLOAT32: return pod.setFloat32((float)value);
      case FLOAT64: return pod.setFloat64(value);
      default: return pod;
    }
  }

  public static byte[] getBytes(Pod.Value pod) {
    switch (pod.getValCase()) {
      case UINT8_ARRAY: return pod.getUint8Array().toByteArray();
      default:
        throw new RuntimeException("Don't know how to get bytes out of " + pod.getValCase());
    }
  }
}
