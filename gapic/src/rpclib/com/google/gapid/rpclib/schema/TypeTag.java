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

public final class TypeTag {
    public static final TypeTag PrimitiveTag = new TypeTag((byte)0, "PrimitiveTag");
    public static final byte PrimitiveTagValue = 0;
    public static final TypeTag StructTag = new TypeTag((byte)1, "StructTag");
    public static final byte StructTagValue = 1;
    public static final TypeTag PointerTag = new TypeTag((byte)2, "PointerTag");
    public static final byte PointerTagValue = 2;
    public static final TypeTag InterfaceTag = new TypeTag((byte)3, "InterfaceTag");
    public static final byte InterfaceTagValue = 3;
    public static final TypeTag VariantTag = new TypeTag((byte)4, "VariantTag");
    public static final byte VariantTagValue = 4;
    public static final TypeTag AnyTag = new TypeTag((byte)5, "AnyTag");
    public static final byte AnyTagValue = 5;
    public static final TypeTag SliceTag = new TypeTag((byte)6, "SliceTag");
    public static final byte SliceTagValue = 6;
    public static final TypeTag ArrayTag = new TypeTag((byte)7, "ArrayTag");
    public static final byte ArrayTagValue = 7;
    public static final TypeTag MapTag = new TypeTag((byte)8, "MapTag");
    public static final byte MapTagValue = 8;

    private static final ImmutableMap<Byte, TypeTag> VALUES = ImmutableMap.<Byte, TypeTag>builder()
        .put((byte)0, PrimitiveTag)
        .put((byte)1, StructTag)
        .put((byte)2, PointerTag)
        .put((byte)3, InterfaceTag)
        .put((byte)4, VariantTag)
        .put((byte)5, AnyTag)
        .put((byte)6, SliceTag)
        .put((byte)7, ArrayTag)
        .put((byte)8, MapTag)
        .build();

    private final byte mValue;
    private final String mName;

    private TypeTag(byte v, String n) {
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

    public static TypeTag decode(Decoder d) throws IOException {
        return findOrCreate(d.uint8());
    }

    public static TypeTag find(byte value) {
        return VALUES.get(value);
    }

    public static TypeTag findOrCreate(byte value) {
        TypeTag result = VALUES.get(value);
        return (result == null) ? new TypeTag(value, null) : result;
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || !(o instanceof TypeTag)) return false;
        return mValue == ((TypeTag)o).mValue;
    }

    @Override
    public int hashCode() {
        return mValue;
    }

    @Override
    public String toString() {
        return (mName == null) ? "TypeTag(" + mValue + ")" : mName;
    }
}
