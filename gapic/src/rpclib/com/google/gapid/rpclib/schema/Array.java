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

public final class Array extends Type {
    String mAlias;

    Type mValueType;

    int mSize;

    public Array(String alias, Type type, int size) {
        mAlias = alias;
        mValueType = type;
        mSize = size;
    }

    public Array(Decoder d) throws IOException {
        mSize = d.uint32();
        mValueType = decode(d);
        mAlias = d.nonCompactString();
    }

    public String getAlias() {
        return mAlias;
    }

    public Type getValueType() {
        return mValueType;
    }

    public int getSize() {
        return mSize;
    }

    @Override
    public void encodeValue(Encoder e, Object value) throws IOException {
        assert (value instanceof Object[]);
        Object[] array = (Object[]) value;
        for (int i = 0; i < mSize; i++) {
            mValueType.encodeValue(e, array[i]);
        }
    }

    @Override
    public Object decodeValue(Decoder d) throws IOException {
        Object[] array = new Object[mSize];
        for (int i = 0; i < mSize; i++) {
            array[i] = mValueType.decodeValue(d);
        }
        return array;
    }

    @Override
    public void encode(Encoder e) throws IOException {
        TypeTag.ArrayTag.encode(e);
        e.uint32(mSize);
        mValueType.encode(e);
        e.nonCompactString(mAlias);
    }

    @Override
    void name(StringBuilder out) {
        out.append("array<");
        mValueType.name(out);
        out.append('>');
    }

    @Override
    public void signature(StringBuilder out) {
        out.append('[').append(mSize).append(']');
        mValueType.signature(out);
    }
}
