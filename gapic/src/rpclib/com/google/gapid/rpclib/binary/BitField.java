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

/**
 * A set of constants that can be combined into a single field.
 */
public class BitField<T extends BitField<T>> {
  protected int bits;

  public BitField(Iterable<Bit<T>> bits) {
    for (Bit<T> bit : bits) {
      set(bit);
    }
  }

  protected BitField(int value) {
    bits = value;
  }

  public boolean isSet(Bit<T> bit) {
    return (bits & bit.value) == bit.value;
  }

  public void set(Bit<T> bit) {
    bits |= bit.value;
  }

  public void clear(Bit<T> bit) {
    bits &= ~bit.value;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || o.getClass() != getClass()) return false;
    return bits == ((BitField<T>)o).bits;
  }

  @Override
  public int hashCode() {
    return bits;
  }

  public static class Bit<T extends BitField<T>> {
    public final int value;
    public final String name;

    public Bit(int value, String name) {
      this.value = value;
      this.name = name;
    }
  }
}
