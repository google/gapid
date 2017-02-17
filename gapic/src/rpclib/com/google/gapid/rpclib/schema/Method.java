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

import com.google.common.collect.ImmutableMap;
import com.google.gapid.rpclib.binary.Decoder;
import com.google.gapid.rpclib.binary.Encoder;

import java.io.IOException;

public final class Method {
    public static final Method Bool = new Method((byte)0, "Bool");
    public static final byte BoolValue = 0;
    public static final Method Int8 = new Method((byte)1, "Int8");
    public static final byte Int8Value = 1;
    public static final Method Uint8 = new Method((byte)2, "Uint8");
    public static final byte Uint8Value = 2;
    public static final Method Int16 = new Method((byte)3, "Int16");
    public static final byte Int16Value = 3;
    public static final Method Uint16 = new Method((byte)4, "Uint16");
    public static final byte Uint16Value = 4;
    public static final Method Int32 = new Method((byte)5, "Int32");
    public static final byte Int32Value = 5;
    public static final Method Uint32 = new Method((byte)6, "Uint32");
    public static final byte Uint32Value = 6;
    public static final Method Int64 = new Method((byte)7, "Int64");
    public static final byte Int64Value = 7;
    public static final Method Uint64 = new Method((byte)8, "Uint64");
    public static final byte Uint64Value = 8;
    public static final Method Float32 = new Method((byte)9, "Float32");
    public static final byte Float32Value = 9;
    public static final Method Float64 = new Method((byte)10, "Float64");
    public static final byte Float64Value = 10;
    public static final Method String = new Method((byte)11, "String");
    public static final byte StringValue = 11;

    private static final ImmutableMap<Byte, Method> VALUES = ImmutableMap.<Byte, Method>builder()
        .put((byte)0, Bool)
        .put((byte)1, Int8)
        .put((byte)2, Uint8)
        .put((byte)3, Int16)
        .put((byte)4, Uint16)
        .put((byte)5, Int32)
        .put((byte)6, Uint32)
        .put((byte)7, Int64)
        .put((byte)8, Uint64)
        .put((byte)9, Float32)
        .put((byte)10, Float64)
        .put((byte)11, String)
        .build();

    private final byte mValue;
    private final String mName;

    private Method(byte v, String n) {
        mValue = v;
        mName = n;
    }

    public byte getValue() {
        return mValue;
    }

    public String getName() {
        return mName;
    }

    public void encode(Encoder e) throws IOException {
        e.uint8(mValue);
    }

    public static Method decode(Decoder d) throws IOException {
        return findOrCreate(d.uint8());
    }

    public static Method find(byte value) {
        return VALUES.get(value);
    }

    public static Method findOrCreate(byte value) {
        Method result = VALUES.get(value);
        return (result == null) ? new Method(value, null) : result;
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || !(o instanceof Method)) return false;
        return mValue == ((Method)o).mValue;
    }

    @Override
    public int hashCode() {
        return mValue;
    }

    @Override
    public String toString() {
        return (mName == null) ? "Method(" + mValue + ")" : mName;
    }
}
