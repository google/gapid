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
package com.google.gapid.rpclib.schema;

import com.google.gapid.rpclib.binary.Decoder;
import com.google.gapid.rpclib.binary.Encoder;

import java.io.IOException;

public final class Primitive extends Type {

    String mName;

    Method mMethod;

    public Primitive(String name, Method method) {
        mName = name;
        mMethod = method;
    }

    public Primitive(Decoder d, Method method) throws IOException {
        mMethod = method;
        mName = d.nonCompactString();
    }

    @Override
    public String toString() {
        return mName + "(" + mMethod + ")";
    }

    public Method getMethod() {
        return mMethod;
    }

    @Override
    public void encodeValue(Encoder e, Object value) throws IOException {
        switch (mMethod.getValue()) {
            case Method.BoolValue:
                e.bool((Boolean) value);
                break;
            case Method.Int8Value:
                e.int8(((Number)value).byteValue());
                break;
            case Method.Uint8Value:
                e.uint8(((Number)value).shortValue());
                break;
            case Method.Int16Value:
                e.int16(((Number)value).shortValue());
                break;
            case Method.Uint16Value:
                e.uint16(((Number)value).intValue());
                break;
            case Method.Int32Value:
                e.int32(((Number)value).intValue());
                break;
            case Method.Uint32Value:
                e.uint32(((Number)value).longValue());
                break;
            case Method.Int64Value:
                e.int64(((Number)value).longValue());
                break;
            case Method.Uint64Value:
                e.uint64(((Number)value).longValue());
                break;
            case Method.Float32Value:
                e.float32(((Number)value).floatValue());
                break;
            case Method.Float64Value:
                e.float64(((Number)value).doubleValue());
                break;
            case Method.StringValue:
                e.string((value == null) ? null : value.toString());
                break;
            default:
                throw new IOException("Invalid primitive method in encode");
        }
    }

    @Override
    public Object decodeValue(Decoder d) throws IOException {
        switch (mMethod.getValue()) {
            case Method.BoolValue:
                return d.bool();
            case Method.Int8Value:
                return d.int8();
            case Method.Uint8Value:
                return d.uint8();
            case Method.Int16Value:
                return d.int16();
            case Method.Uint16Value:
                return d.uint16();
            case Method.Int32Value:
                return d.int32();
            case Method.Uint32Value:
                return d.uint32();
            case Method.Int64Value:
                return d.int64();
            case Method.Uint64Value:
                return d.uint64();
            case Method.Float32Value:
                return d.float32();
            case Method.Float64Value:
                return d.float64();
            case Method.StringValue:
                return d.string();
            default:
                throw new IOException("Invalid primitive method in decode");
        }
    }

    @Override
    public void encode(Encoder e) throws IOException {
        //noinspection PointlessBitwiseExpression
        e.uint8((short)(TypeTag.PrimitiveTagValue | ( mMethod.getValue() << 4)));
        e.nonCompactString(mName);
    }

    @Override
    void name(StringBuilder out) {
        out.append(mName);
    }

    @Override
    public void signature(StringBuilder out) {
        out.append(mMethod);
    }

    /**
     * @return true if ty is a primitive of base type method.
     */
    public static boolean isMethod(Type ty, Method method) {
        if (!(ty instanceof Primitive)) {
            return false;
        }
        return ((Primitive)(ty)).mMethod.equals(method);
    }
}
