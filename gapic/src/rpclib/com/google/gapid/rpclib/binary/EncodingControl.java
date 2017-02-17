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
package com.google.gapid.rpclib.binary;

import java.io.IOException;

public final class EncodingControl {
    public static final int Version = 0;
    public static final int Compact = 0;
    public static final int Full = 1;

    public int mode;

    public void encode(Encoder e) throws IOException {
        e.uint32(Version);
        e.uint32(mode);
    }

    public void decode(Decoder d) throws IOException {
        int version = d.uint32();
        if (version != Version) {
            throw new IOException("(Invalid control block version");
        }
        mode = d.uint32();
    }
}
