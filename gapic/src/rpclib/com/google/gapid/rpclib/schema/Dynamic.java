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
import java.util.Arrays;

public class Dynamic implements BinaryObject {

    private Klass mKlass;

    /**
     * This can actually sometimes be an array of arrays!
     */
    private Object[] mFields;

    public Dynamic(Klass klass) {
        mKlass = klass;
    }

    public Entity type() {
        return mKlass.mType;
    }

    public static BinaryClass register(Entity type) {
        BinaryClass klass = new Klass(type);
        Namespace.register(klass);
        return klass;
    }

    public int getFieldCount() {
        return mFields.length;
    }

    public Field getFieldInfo(int index) {
        return mKlass.mType.getFields()[index];
    }

    /**
     * @return may return an Object, or an Object[]
     */
    public Object getFieldValue(int index) {
        return mFields[index];
    }

    public void setFieldValue(int index, Object value) {
        mFields[index] = value;
    }

    public Dynamic copy() {
        Dynamic result = new Dynamic(mKlass);
        result.mFields = Arrays.copyOf(mFields, mFields.length);
        return result;
    }


    @Override
    public Klass klass() {
        return mKlass;
    }

    @Override
    public boolean equals(Object obj) {
        if (obj == this) {
            return true;
        } else if (!(obj instanceof Dynamic)) {
            return false;
        }
        Dynamic d = (Dynamic)obj;
        return type().equals(d.type()) && Arrays.deepEquals(mFields, d.mFields);
    }

    @Override
    public int hashCode() {
        return mKlass.hashCode() + 31 * Arrays.deepHashCode(mFields);
    }

    @Override
    public String toString() {
        StringBuilder result = new StringBuilder();
        result.append(mKlass.entity().getName()).append('{');
        Field[] fields = mKlass.entity().getFields();
        for (int i = 0; i < mFields.length; i++) {
            if (i > 0) {
                result.append(", ");
            }
            String fieldsString = mFields[i] instanceof Object[] ? Arrays.toString((Object[])mFields[i]) : String.valueOf(mFields[i]);
            result.append(fields[i].getName()).append(": ").append(fieldsString);
        }
        result.append('}');
        return result.toString();
    }

    public static class Klass implements BinaryClass {

        private Entity mType;

        Klass(Entity type) {
            mType = type;
        }


        @Override
        public Entity entity() {
            return mType;
        }

        @Override

        public BinaryObject create() {
            return new Dynamic(this);
        }

        @Override
        public void encode(Encoder e, BinaryObject obj) throws IOException {
            Dynamic o = (Dynamic) obj;
            assert (o.mKlass == this);
            for (int i = 0; i < mType.getFields().length; i++) {
                Field field = mType.getFields()[i];
                Object value = o.mFields[i];
                field.getType().encodeValue(e, value);
            }
        }

        @Override
        public void decode(Decoder d, BinaryObject obj) throws IOException {
            Dynamic o = (Dynamic) obj;
            o.mFields = new Object[mType.getFields().length];
            for (int i = 0; i < mType.getFields().length; i++) {
                Field field = mType.getFields()[i];
                o.mFields[i] = field.getType().decodeValue(d);
            }
        }

        @Override
        public boolean equals(Object obj) {
            if (obj == this) {
                return true;
            } else if (!(obj instanceof Klass)) {
                return false;
            }
            return entity().equals(((Klass)obj).entity());
        }

        @Override
        public int hashCode() {
            return mType.hashCode();
        }
    }
}
