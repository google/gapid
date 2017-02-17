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

import com.google.common.collect.Maps;
import com.google.gapid.rpclib.schema.Entity;

import java.io.IOException;
import java.io.OutputStream;
import java.io.UnsupportedEncodingException;
import java.util.Map;

/**
 * An encoder of various primitive types.
 * The encoding format is documented at the following link:
 * https://github.com/google/gapid/+/master/binary/doc.go
 */
public class Encoder {
  private final OutputStream mOutputStream;
  private final Map<Entity, Integer> mEntities;
  private final Map<BinaryObject, Integer> mObjects;
  private final byte[] mBuffer;
  private boolean mControlNeeded = false;
  private final EncodingControl mControl = new EncodingControl();

  public Encoder(OutputStream out) {
    mEntities = Maps.newHashMap();
    mObjects = Maps.newHashMap();
    mOutputStream = out;
    mBuffer = new byte[9];
  }

  public void write(byte[] b, int len) throws IOException {
    mOutputStream.write(b, 0, len);
  }

  public void bool(boolean v) throws IOException {
    mBuffer[0] = (byte)(v ? 1 : 0);
    mOutputStream.write(mBuffer, 0, 1);
  }

  public void int8(byte v) throws IOException {
    mBuffer[0] = v;
    mOutputStream.write(mBuffer, 0, 1);
  }

  public void uint8(short v) throws IOException {
    mBuffer[0] = (byte)(v & 0xff);
    mOutputStream.write(mBuffer, 0, 1);
  }

  private void intv(long v) throws IOException {
    long uv = v << 1;
    if (v < 0) uv = ~uv;
    uintv(uv);
  }

  private void uintv(long v) throws IOException {
    long space = ~0x7fL;
    int tag = 0;
    for (int o = 8; true; o--) {
      if ((v & space) == 0) {
        mBuffer[o] = (byte)(v | tag);
        mOutputStream.write(mBuffer, o, 9 - o);
        return;
      }
      mBuffer[o] = (byte)(v&0xff);
      v >>>= 8;
      space >>= 1;
      tag =(tag >> 1) | 0x80;
    }
  }

  public void int16(short v) throws IOException {
    intv(v);
  }

  public void uint16(int v) throws IOException {
    uintv(v);
  }

  public void int32(int v) throws IOException {
    intv(v);
  }

  public void uint32(long v) throws IOException {
    uintv(v);
  }

  public void int64(long v) throws IOException {
    intv(v);
  }

  public void uint64(long v) throws IOException {
    uintv(v);
  }

  public void float32(float v) throws IOException {
    int bits = Float.floatToIntBits(v);
    int shuffled = ((bits & 0x000000ff) <<  24) |
                   ((bits & 0x0000ff00) <<   8) |
                   ((bits & 0x00ff0000) >> 8) |
                   ((bits & 0xff000000) >>> 24);
    uintv(shuffled);
  }

  public void float64(double v) throws IOException {
    long bits = Double.doubleToLongBits(v);
    long shuffled = ((bits & 0x00000000000000ffL) <<  56) |
                    ((bits & 0x000000000000ff00L) <<  40) |
                    ((bits & 0x0000000000ff0000L) <<  24) |
                    ((bits & 0x00000000ff000000L) <<   8) |
                    ((bits & 0x000000ff00000000L) >>   8) |
                    ((bits & 0x0000ff0000000000L) >>  24) |
                    ((bits & 0x00ff000000000000L) >> 40) |
                    ((bits & 0xff00000000000000L) >>> 56);
    uintv(shuffled);
  }

  public void string(String v) throws IOException {
    try {
      if (v == null) {
        uint32(0);
        return;
      }

      byte[] bytes = v.getBytes("UTF-8");
      uint32(bytes.length);
      for (byte b : bytes) {
        int8(b);
      }
    }
    catch (UnsupportedEncodingException e) {
      throw new RuntimeException(e); // Should never happen
    }
  }

  public void nonCompactString(String v) throws IOException {
    if (mControl.mode != EncodingControl.Compact) {
      string(v);
    }
  }

  public void writeSid(int sid, boolean encoded) throws IOException {
    if (mControlNeeded) {
      mControlNeeded = false; // set early to prevent recursive triggering
      uint32(1);             // encoded sid 0 is a special marker
      mControl.encode(this);      // and write the control block itself
    }
    uint32((sid << 1) | (encoded ? 1 : 0));
  }

  public void entity(Entity entity) throws IOException {
    if (entity == null) {
      writeSid(0, false);
    } else if (mEntities.containsKey(entity)) {
      writeSid(mEntities.get(entity), false);
    } else {
      int sid = mEntities.size() + 1;
      mEntities.put(entity, sid);
      writeSid(sid, true);
      entity.encode(this);
    }
  }

  public void value(BinaryObject obj) throws IOException {
    obj.klass().encode(this, obj);
  }

  public void variant(BinaryObject obj) throws IOException {
    if (obj == null) {
      entity(null);
    } else {
      BinaryClass c = obj.klass();
      entity(c.entity());
      c.encode(this, obj);
    }
  }

  public void object(BinaryObject obj) throws IOException {
    if (obj == null) {
      writeSid(0, false);
    } else if (mObjects.containsKey(obj)) {
      writeSid(mObjects.get(obj), false);
    } else {
      int sid = mObjects.size() + 1;
      mObjects.put(obj, sid);
      writeSid(sid, true);
      variant(obj);
    }
  }

  public OutputStream stream() {
    return mOutputStream;
  }

  public int getMode() {
    return mControl.mode;
  }

  public int setMode(int mode) {
    int oldMode = mControl.mode;
    if (oldMode != mode) {
      mControlNeeded = true;
      mControl.mode = mode;
    }
    return oldMode;
  }
}
