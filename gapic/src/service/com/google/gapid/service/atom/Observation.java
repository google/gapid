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
package com.google.gapid.service.atom;

import com.google.gapid.rpclib.binary.BinaryClass;
import com.google.gapid.rpclib.binary.BinaryID;
import com.google.gapid.rpclib.binary.BinaryObject;
import com.google.gapid.rpclib.binary.Decoder;
import com.google.gapid.rpclib.binary.Encoder;
import com.google.gapid.rpclib.binary.Namespace;
import com.google.gapid.rpclib.schema.Array;
import com.google.gapid.rpclib.schema.Entity;
import com.google.gapid.rpclib.schema.Field;
import com.google.gapid.rpclib.schema.Method;
import com.google.gapid.rpclib.schema.Primitive;
import com.google.gapid.rpclib.schema.Struct;
import com.google.gapid.service.memory.MemoryRange;

import java.io.IOException;

public final class Observation implements BinaryObject {

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;
    Observation that = (Observation)o;
    if (myRange != null ? !myRange.equals(that.myRange) : that.myRange != null) return false;
    if (myID != null ? !myID.equals(that.myID) : that.myID != null) return false;
    return true;
  }

  @Override
  public int hashCode() {
    int result = myRange != null ? myRange.hashCode() : 0;
    result = 31 * result + (myID != null ? myID.hashCode() : 0);
    return result;
  }

  @Override
  public String toString() {
    return "Observation{ID=" + myID + ", range=" + myRange + '}';
  }

  //<<<Start:Java.ClassBody:1>>>
  private MemoryRange myRange;
  private BinaryID myID;

  // Constructs a default-initialized {@link Observation}.
  public Observation() {}


  public MemoryRange getRange() {
    return myRange;
  }

  public Observation setRange(MemoryRange v) {
    myRange = v;
    return this;
  }

  public BinaryID getID() {
    return myID;
  }

  public Observation setID(BinaryID v) {
    myID = v;
    return this;
  }

  @Override
  public BinaryClass klass() { return Klass.INSTANCE; }


  private static final Entity ENTITY = new Entity("atom", "Observation", "", "");

  static {
    ENTITY.setFields(new Field[]{
      new Field("Range", new Struct(MemoryRange.Klass.INSTANCE.entity())),
      new Field("ID", new Array("id.ID", new Primitive("byte", Method.Uint8), 20)),
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
    public BinaryObject create() { return new Observation(); }

    @Override
    public void encode(Encoder e, BinaryObject obj) throws IOException {
      Observation o = (Observation)obj;
      e.value(o.myRange);
      o.myID.write(e);

    }

    @Override
    public void decode(Decoder d, BinaryObject obj) throws IOException {
      Observation o = (Observation)obj;
      o.myRange = new MemoryRange();
      d.value(o.myRange);
      o.myID = new BinaryID(d);
    }
    //<<<End:Java.KlassBody:2>>>
  }
}
