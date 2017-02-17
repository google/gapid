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

public final class Field {

    String mDeclared;

    Type mType;

    public Field(String name, Type type) {
        mDeclared = name;
        mType = type;
    }

    public Field() {
    }

    public String getDeclared() {
        return mDeclared;
    }

    public Type getType() {
        return mType;
    }

    public String getName() {
        if (mDeclared.length() == 0) {
            return mType.getName();
        }
        return mDeclared;
    }

    @Override
    public boolean equals(Object o) {
        if (this == o) return true;
        if (o == null || getClass() != o.getClass()) return false;

        Field field = (Field)o;

        if (!mDeclared.equals(field.mDeclared)) return false;
        // This is the best we can do at the moment. Type.equals() is
        // buggy because it doesn't know the package or relative types.
        if (!mType.getClass().equals(field.mType.getClass())) return false;

        return true;
    }

    @Override
    public int hashCode() {
        int result = mDeclared.hashCode();
        result = 31 * result + mType.getClass().hashCode();
        return result;
    }
}
