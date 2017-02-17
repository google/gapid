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

import com.google.gapid.rpclib.binary.BinaryObject;
import com.google.gapid.rpclib.binary.Decoder;
import com.google.gapid.rpclib.binary.Encoder;

import java.io.IOException;

public final class Variant extends Type {
    String mName;

    public Variant(String name) {
        mName = name;
    }

    public Variant(Decoder d) throws IOException {
        mName = d.nonCompactString();
    }

    @Override
    public void encodeValue(Encoder e, Object value) throws IOException {
        assert (value instanceof BinaryObject);
        e.object((BinaryObject) value);
    }

    @Override
    public Object decodeValue(Decoder d) throws IOException {
        return d.object();
    }

    @Override
    public void encode(Encoder e) throws IOException {
        TypeTag.VariantTag.encode(e);
        e.nonCompactString(mName);
    }

    @Override
    void name(StringBuilder out) {
        out.append(mName);
    }

    @Override
    public void signature(StringBuilder out) {
        out.append('&');
    }
}
