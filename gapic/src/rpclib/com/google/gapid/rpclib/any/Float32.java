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
package com.google.gapid.rpclib.any;

import com.google.gapid.rpclib.binary.BinaryClass;
import com.google.gapid.rpclib.binary.BinaryObject;
import com.google.gapid.rpclib.binary.Decoder;
import com.google.gapid.rpclib.binary.Encoder;
import com.google.gapid.rpclib.binary.Namespace;
import com.google.gapid.rpclib.schema.Entity;
import com.google.gapid.rpclib.schema.Field;
import com.google.gapid.rpclib.schema.Method;
import com.google.gapid.rpclib.schema.Primitive;

import java.io.IOException;

final class Float32 extends Box implements BinaryObject {
    @Override
    public Object unwrap() {
        return getValue();
    }

    //<<<Start:Java.ClassBody:1>>>
    private float mValue;

    // Constructs a default-initialized {@link Float32}.
    public Float32() {}


    public float getValue() {
        return mValue;
    }

    public Float32 setValue(float v) {
        mValue = v;
        return this;
    }

    @Override
    public BinaryClass klass() { return Klass.INSTANCE; }


    private static final Entity ENTITY = new Entity("any", "float32_", "", "");

    static {
        ENTITY.setFields(new Field[]{
            new Field("Value", new Primitive("float32", Method.Float32)),
        });
        Namespace.register(Klass.INSTANCE);
    }
    public static void register() {}
    //<<<End:Java.ClassBody:1>>>
    public enum Klass implements BinaryClass {
        //<<<Start:Java.KlassBody:2>>>
        INSTANCE;

        @Override
        public Entity entity() { return ENTITY; }

        @Override
        public BinaryObject create() { return new Float32(); }

        @Override
        public void encode(Encoder e, BinaryObject obj) throws IOException {
            Float32 o = (Float32)obj;
            e.float32(o.mValue);
        }

        @Override
        public void decode(Decoder d, BinaryObject obj) throws IOException {
            Float32 o = (Float32)obj;
            o.mValue = d.float32();
        }
        //<<<End:Java.KlassBody:2>>>
    }
}
