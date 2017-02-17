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

import com.google.gapid.rpclib.binary.BinaryClass;
import com.google.gapid.rpclib.binary.BinaryObject;
import com.google.gapid.rpclib.binary.Decoder;
import com.google.gapid.rpclib.binary.Encoder;
import com.google.gapid.rpclib.binary.Namespace;

import java.io.IOException;

public final class Struct extends Type {
    Entity mEntity;

    public Struct(Entity entity) {
        mEntity = entity;
    }

    public Struct(Decoder d) throws IOException {
        mEntity = d.entity();
        d.nonCompactString();
    }

    public Entity getEntity() {
        return mEntity;
    }

    @Override
    public void encodeValue(Encoder e, Object value) throws IOException {
        assert (value instanceof BinaryObject);
        e.value((BinaryObject) value);
    }

    @Override
    public Object decodeValue(Decoder d) throws IOException {
        BinaryClass klass = Namespace.lookup(mEntity);
        if (klass == null) {
            throw new IOException("Unknown type: " + mEntity);
        }
        BinaryObject obj = klass.create();
        klass.decode(d, obj);
        return obj;
    }

    @Override
    public void encode(Encoder e) throws IOException {
        TypeTag.StructTag.encode(e);
        e.entity(mEntity);
        e.nonCompactString("");
    }

    @Override
    void name(StringBuilder out) {
        out.append(mEntity.getName());
    }

    @Override
    public void signature(StringBuilder out) {
        out.append('$');
    }


    public boolean is(BinaryClass klass) {
        return mEntity.signature().equals(klass.entity().signature());
    }
}
