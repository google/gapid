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
import com.google.gapid.rpclib.schema.Method;
import com.google.gapid.rpclib.schema.Primitive;
import com.google.gapid.rpclib.schema.Type;

/**
 * Plain-Old-Data utilities.
 */
public class Pods {
  private Pods() {
  }

  public static Pod.Value pod(Object o, Type type) {
    Pod.Value.Builder result = Pod.Value.newBuilder();
    if (o == null) {
      // Empty proto.
    } else if (o instanceof Number) {
      Number n = (Number)o;
      if (type instanceof Primitive) {
        switch (((Primitive)type).getMethod().getValue()) {
          case Method.BoolValue:
            result.setBool(n.intValue() != 0);
            break;
          case Method.Int8Value:
            result.setSint8(n.intValue());
            break;
          case Method.Uint8Value:
            result.setUint8(n.intValue() & 0xFF);
            break;
          case Method.Int16Value:
            result.setSint16(n.intValue());
            break;
          case Method.Uint16Value:
            result.setUint16(n.intValue() & 0xFFFF);
            break;
          case Method.Int32Value:
            result.setSint32(n.intValue());
            break;
          case Method.Uint32Value:
            result.setUint32(n.intValue());
            break;
          case Method.Int64Value:
            result.setSint64(n.longValue());
            break;
          case Method.Uint64Value:
            result.setUint64(n.longValue());
            break;
          case Method.Float32Value:
            result.setFloat32(n.floatValue());
            break;
          case Method.Float64Value:
            result.setFloat64(n.doubleValue());
            break;
          case Method.StringValue:
            result.setString(n.toString());
            break;
          default:
            throw new AssertionError();
        }
      } else {
        throw new UnsupportedOperationException("Cannot pod: " + o + " as: " + type);
      }
    } else if (o instanceof Boolean) {
      result.setBool((Boolean)o);
    } else if (o instanceof String) {
      result.setString((String)o);
    } else {
      // TODO handle arrays
      throw new UnsupportedOperationException("Cannot pod: " + o + " as: " + type);
    }
    return result.build();
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
}
