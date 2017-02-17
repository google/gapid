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

import com.google.gapid.rpclib.any.Box;
import com.google.gapid.rpclib.binary.Decoder;
import com.google.gapid.rpclib.binary.Encoder;

import java.io.IOException;

public final class AnyType extends Type {
    public AnyType() {
    }

    /** @param d decoder **/
    public AnyType(Decoder d) {
    }

    @Override
    public void encodeValue(Encoder e, Object value) throws IOException {
        e.variant(Box.wrap(value));
    }

    @Override
    public Object decodeValue(Decoder d) throws IOException {
        Box boxed = (Box) d.variant();
        if (boxed == null) {
            return null;
        }
        return boxed.unwrap();
    }

    @Override
    public void encode(Encoder e) throws IOException {
        TypeTag.AnyTag.encode(e);
    }

    @Override
    void name(StringBuilder out) {
        out.append("any");
    }

    @Override
    public void signature(StringBuilder out) {
        out.append('~');
    }
}
