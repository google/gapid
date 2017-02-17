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
import com.google.gapid.rpclib.binary.EncodingControl;
import com.google.gapid.rpclib.binary.Namespace;

import java.io.IOException;

/**
 * This class is used to transmit schema data to a client.
 * It is not a normal binary object because it requires custom encoding for the Entity objects.
 */
public final class Message implements BinaryObject {
    public Entity[] entities;
    public ConstantSet[] constants;

    @Override
    public BinaryClass klass() {
        return Klass.INSTANCE;
    }

    static {
        Namespace.register(Klass.INSTANCE);
    }

    public static void register() {
    }

    public enum Klass implements BinaryClass {
        INSTANCE;

        private static final Entity ENTITY = new Entity("schema","Message","","");

        @Override
        public Entity entity() {
            return ENTITY;
        }

        @Override
        public BinaryObject create() {
            return new Message();
        }

        @Override
        public void encode(Encoder e, BinaryObject obj) throws IOException {
            int oldMode = e.setMode(EncodingControl.Full);
            try {
                Message o = (Message)obj;
                e.uint32(o.entities.length);
                for (Entity entity : o.entities) {
                    e.entity(entity);
                }
                e.uint32(o.constants.length);
                for (ConstantSet set : o.constants) {
                    set.encode(e);
                }
            } finally {
                e.setMode(oldMode);
            }
        }

        @Override
        public void decode(Decoder d, BinaryObject obj) throws IOException {
            Message o = (Message) obj;
            o.entities = new Entity[d.uint32()];
            for (int i = 0; i < o.entities.length; i++) {
                o.entities[i] = d.entity();
            }
            o.constants = new ConstantSet[d.uint32()];
            for (int i = 0; i < o.constants.length; i++) {
                o.constants[i] = new ConstantSet(d);
            }
        }
    }
}
