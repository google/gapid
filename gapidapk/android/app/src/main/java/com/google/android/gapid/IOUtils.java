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

package com.google.android.gapid;

import android.os.Parcel;
import android.os.Parcelable;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;

public final class IOUtils {
    private IOUtils() {}

    public static byte[] readAll(InputStream is) throws IOException {
        try (Counter.Scope t = Counter.time("readAll")) {
            try (ByteArrayOutputStream os = new ByteArrayOutputStream()) {
                byte[] buf = new byte[4096];
                for (int n; (n = is.read(buf)) != -1; ) {
                    os.write(buf, 0, n);
                }
                return os.toByteArray();
            }
        }
    }

    public static void writeAll(OutputStream os, byte[] data) throws IOException {
        if (data == null) {
            return;
        }
        try (Counter.Scope t = Counter.time("writeAll")) {
            os.write(data, 0, data.length);
            os.flush();
        }
    }

    public static byte[] encode(Parcelable parcelable) {
        Parcel parcel = Parcel.obtain();
        parcelable.writeToParcel(parcel, 0);
        byte[] out = parcel.marshall();
        parcel.recycle();
        return out;
    }

    public static <T> T decode(Parcelable.Creator<T> creator, byte[] data) {
        Parcel parcel = Parcel.obtain();
        parcel.unmarshall(data, 0, data.length);
        parcel.setDataPosition(0); // This is extremely important!
        T out = creator.createFromParcel(parcel);
        parcel.recycle();
        return out;
    }
}
