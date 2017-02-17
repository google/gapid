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
package com.google.gapid.service.memory;

import com.google.gapid.rpclib.binary.BinaryClass;
import com.google.gapid.rpclib.binary.BinaryObject;
import com.google.gapid.rpclib.binary.Decoder;
import com.google.gapid.rpclib.binary.Encoder;
import com.google.gapid.rpclib.binary.Namespace;
import com.google.gapid.rpclib.schema.Entity;
import com.google.gapid.rpclib.schema.Field;
import com.google.gapid.rpclib.schema.Method;
import com.google.gapid.rpclib.schema.Primitive;

import java.io.IOException;

public final class MemoryPointer implements BinaryObject {

  @Override
  public String toString() {
    return "0x" + Long.toHexString(myAddress) + "@" + myPool;
  }

  //<<<Start:Java.ClassBody:1>>>
  private long myAddress;
  private int myPool;

  // Constructs a default-initialized {@link MemoryPointer}.
  public MemoryPointer() {}


  public long getAddress() {
    return myAddress;
  }

  public MemoryPointer setAddress(long v) {
    myAddress = v;
    return this;
  }

  public int getPool() {
    return myPool;
  }

  public MemoryPointer setPool(int v) {
    myPool = v;
    return this;
  }

  @Override
  public BinaryClass klass() { return Klass.INSTANCE; }


  private static final Entity ENTITY = new Entity("memory", "Pointer", "", "");

  static {
    ENTITY.setFields(new Field[]{
      new Field("Address", new Primitive("uint64", Method.Uint64)),
      new Field("Pool", new Primitive("PoolID", Method.Uint32)),
    });
    Namespace.register(Klass.INSTANCE);
  }
  public static void register() {}
  //<<<End:Java.ClassBody:1>>>
  public enum Klass implements BinaryClass {
    //<<<Start:Java.KlassBody:2>>>
    INSTANCE;

    @Override
    public Entity entity() { return ENTITY; }

    @Override
    public BinaryObject create() { return new MemoryPointer(); }

    @Override
    public void encode(Encoder e, BinaryObject obj) throws IOException {
      MemoryPointer o = (MemoryPointer)obj;
      e.uint64(o.myAddress);
      e.uint32(o.myPool);
    }

    @Override
    public void decode(Decoder d, BinaryObject obj) throws IOException {
      MemoryPointer o = (MemoryPointer)obj;
      o.myAddress = d.uint64();
      o.myPool = d.uint32();
    }
    //<<<End:Java.KlassBody:2>>>
  }
}
