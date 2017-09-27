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

import android.content.Context;

import java.io.File;
import java.io.FileInputStream;
import java.io.FileNotFoundException;
import java.io.FileOutputStream;
import java.io.IOException;

/**
 * FileCache is a {@link Cache} that also uses the applications cache directory as a second level
 * cache.
 */
public abstract class FileCache {
    private FileCache() {
    }

    public static <K, V> Cache<K, V> create(
            Context context,
            final String name,
            final Builder<K, V> builder) {

        final File cacheDir = context.getCacheDir();

        return Cache.create(new Cache.Builder<K, V>() {
            @Override
            public V build(K key) {
                try (Counter.Scope t = Counter.time("FileCache<" + name + ">")) {
                    return scopedBuild(key);
                }
            }

            private V scopedBuild(K key) {
                String name;
                try (Counter.Scope t = Counter.time("filename")) {
                    try (Counter.Scope t2 = Counter.time("generate")) {
                        name = builder.filename(key);
                    }
                    try (Counter.Scope t2 = Counter.time("sanitize")) {
                        char[] src = name.toCharArray();
                        char[] dst = new char[src.length];
                        for (int i = 0; i < src.length; i++) {
                            char c = src[i];
                            if ((c != '.') && (c != '-') &&
                                    !(c >= 'a' && c <= 'z') &&
                                    !(c >= 'A' && c <= 'Z') &&
                                    !(c >= '0' && c <= '9')) {
                                c = '_';
                            }
                            dst[i] = c;
                        }
                        name = new String(dst);
                    }
                }

                File path = new File(cacheDir, name);
                if (path.exists()) {
                    try (Counter.Scope t = Counter.time("load")) {
                        try (FileInputStream file = new FileInputStream(path)) {
                            byte[] bytes = IOUtils.readAll(file);
                            if (bytes.length == 0) {
                                return null;
                            }
                            try (Counter.Scope t2 = Counter.time("decode")) {
                                return builder.decode(bytes);
                            }
                        } catch (FileNotFoundException e) {
                            // Fallthrough
                        } catch (IOException e) {
                            // Fallthrough
                        }
                    }
                }

                V value;
                try (Counter.Scope t = Counter.time("build")) {
                    value = builder.build(key);
                }

                try (Counter.Scope t = Counter.time("store")) {
                    try (FileOutputStream file = new FileOutputStream(path)) {
                        if (value != null) {
                            byte[] data;
                            try (Counter.Scope t2 = Counter.time("encode")) {
                                data = builder.encode(value);
                            }
                            IOUtils.writeAll(file, data);
                        }
                    } catch (FileNotFoundException e) {
                        // Fallthrough
                    } catch (IOException e) {
                        // Fallthrough
                    }
                }

                return value;
            }
        });
    }

    /**
     * Builder is the interface implemented by users of the {@link FileCache} to encode, decode
     * and name the cache files.
     * @param <K> the cache key type.
     * @param <V> the cache value type.
     */
    public interface Builder<K, V> extends Cache.Builder<K, V> {
        /**
         * @return the unique filename for the given data key.
         */
        String filename(K key);

        /**
         * @return the encoded value data as bytes that can be serialized.
         */
        byte[] encode(V value);

        /**
         * @return the value deserialized from the encoded byte data.
         */
        V decode(byte[] data);
    }
}
