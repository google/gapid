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
import java.nio.ByteBuffer;
import java.util.Arrays;

import javax.xml.bind.DatatypeConverter;

/**
 * An ID is a codeable unique identifier.
 * These are used as 20-byte-unique identifiers, often a sha checksum.
 */
public class BinaryID {
  private static final int SIZE = 20;
  public static BinaryID INVALID = new BinaryID();

  private final byte[] mValue = new byte[SIZE];
  private final int mHashCode;

  public BinaryID() {
    mHashCode = 0;
  }

  public BinaryID(byte[] value) {
    assert value.length == SIZE;
    System.arraycopy(value, 0, mValue, 0, SIZE);
    mHashCode = ByteBuffer.wrap(mValue).getInt();
  }

  public BinaryID(Decoder d) throws IOException {
    d.read(mValue, SIZE);
    mHashCode = ByteBuffer.wrap(mValue).getInt();
  }

  public void write(Encoder e) throws IOException {
    e.write(mValue, SIZE);
  }

  @Override
  public boolean equals(Object other) {
    if (!(other instanceof BinaryID)) {
      return false;
    }
    if (other == this) {
      return true;
    }
    return Arrays.equals(mValue, ((BinaryID)other).mValue);
  }

  @Override
  public String toString() {
    return DatatypeConverter.printHexBinary(mValue).toLowerCase();
  }

  @Override
  public int hashCode() {
    return mHashCode;
  }

  public byte[] getBytes() {
    return mValue;
  }
}
