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

public abstract class Type {
    private String mName = null;

    public final String getName() {
        if (mName == null) {
            StringBuilder out = new StringBuilder();
            name(out);
            mName = out.toString();
        }
        return mName;
    }

    public abstract void encodeValue(Encoder e, Object value) throws IOException;

    public abstract Object decodeValue(Decoder d) throws IOException;

    public abstract void encode(Encoder e) throws IOException;

    public static Type decode(Decoder d) throws IOException {
        byte v = d.uint8();
        switch (v & 0xf) {
            case TypeTag.PrimitiveTagValue:
                return new Primitive(d, Method.find((byte)((v >> 4) & 0xf)));
            case TypeTag.StructTagValue:
                return new Struct(d);
            case TypeTag.PointerTagValue:
                return new Pointer(d);
            case TypeTag.InterfaceTagValue:
                return new Interface(d);
            case TypeTag.VariantTagValue:
                return new Variant(d);
            case TypeTag.AnyTagValue:
                return new AnyType(d);
            case TypeTag.SliceTagValue:
                return new Slice(d);
            case TypeTag.ArrayTagValue:
                return new Array(d);
            case TypeTag.MapTagValue:
                return new Map(d);
            default:
                throw new IOException("Decode unknown type " + v);
        }
    }

    abstract void name(StringBuilder out);

    public abstract void signature(StringBuilder out);

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (!(o instanceof Type)) return false;
        return (getName().equals(((Type)o).getName()));
    }

    @Override
    public int hashCode() {
        return getName().hashCode();
    }

    @Override
    public String toString() {
        return getName();
    }
}
