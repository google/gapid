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
import java.util.LinkedHashMap;

public final class Map extends Type {
    String mAlias;

    Type mKeyType;

    Type mValueType;

    public Map(String alias, Type keyType, Type valueType) {
        mAlias = alias;
        mKeyType = keyType;
        mValueType = valueType;
    }

    public Map(Decoder d) throws IOException {
        mKeyType = decode(d);
        mValueType = decode(d);
        mAlias = d.nonCompactString();
    }

    public String getAlias() {
        return mAlias;
    }

    public Type getKeyType() {
        return mKeyType;
    }

    @Override
    public void encodeValue(Encoder e, Object value) throws IOException {
        assert (value instanceof java.util.Map);
        java.util.Map<?, ?> map = (java.util.Map<?, ?>) value;
        e.uint32(map.size());
        for (java.util.Map.Entry<?, ?> entry : map.entrySet()) {
            mKeyType.encodeValue(e, entry.getKey());
            mValueType.encodeValue(e, entry.getValue());
        }
    }

    @Override
    public Object decodeValue(Decoder d) throws IOException {
        int size = d.uint32();
        LinkedHashMap<Object, Object> map = new LinkedHashMap<Object, Object>();
        for (int i = 0; i < size; i++) {
            map.put(mKeyType.decodeValue(d), mValueType.decodeValue(d));
        }
        return map;
    }

    public Type getValueType() {
        return mValueType;
    }

    @Override
    public void encode(Encoder e) throws IOException {
        TypeTag.MapTag.encode(e);
        mKeyType.encode(e);
        mValueType.encode(e);
        e.nonCompactString(mAlias);
    }

    @Override
    void name(StringBuilder out) {
        out.append("map<");
        mKeyType.name(out);
        out.append(',');
        mValueType.name(out);
        out.append('>');
    }

    @Override
    public void signature(StringBuilder out) {
        out.append("map[");
        mKeyType.signature(out);
        out.append(']');
        mValueType.signature(out);
    }
}
