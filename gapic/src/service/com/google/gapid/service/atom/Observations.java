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
import com.google.gapid.rpclib.binary.BinaryObject;
import com.google.gapid.rpclib.binary.Decoder;
import com.google.gapid.rpclib.binary.Encoder;
import com.google.gapid.rpclib.binary.Namespace;
import com.google.gapid.rpclib.schema.Entity;
import com.google.gapid.rpclib.schema.Field;
import com.google.gapid.rpclib.schema.Slice;
import com.google.gapid.rpclib.schema.Struct;

import java.io.IOException;
import java.util.Arrays;

public final class Observations implements BinaryObject {

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;
    Observations that = (Observations)o;
    if (!Arrays.equals(myReads, that.myReads)) return false;
    if (!Arrays.equals(myWrites, that.myWrites)) return false;
    return true;
  }

  @Override
  public int hashCode() {
    int result = Arrays.hashCode(myReads);
    result = 31 * result + Arrays.hashCode(myWrites);
    return result;
  }

  @Override
  public String toString() {
    return "Observations{reads=" + Arrays.toString(myReads) + ", writes=" + Arrays.toString(myWrites) + '}';
  }

  //<<<Start:Java.ClassBody:1>>>
  private Observation[] myReads;
  private Observation[] myWrites;

  // Constructs a default-initialized {@link Observations}.
  public Observations() {}


  public Observation[] getReads() {
    return myReads;
  }

  public Observations setReads(Observation[] v) {
    myReads = v;
    return this;
  }

  public Observation[] getWrites() {
    return myWrites;
  }

  public Observations setWrites(Observation[] v) {
    myWrites = v;
    return this;
  }

  @Override
  public BinaryClass klass() { return Klass.INSTANCE; }


  private static final Entity ENTITY = new Entity("atom", "Observations", "", "");

  static {
    ENTITY.setFields(new Field[]{
      new Field("Reads", new Slice("", new Struct(Observation.Klass.INSTANCE.entity()))),
      new Field("Writes", new Slice("", new Struct(Observation.Klass.INSTANCE.entity()))),
    });
    Namespace.register(Klass.INSTANCE);
  }
  public static void register() {}
  //<<<End:Java.ClassBody:1>>>
  public enum Klass implements BinaryClass {
    //<<<Start:Java.KlassBody:2>>>
    INSTANCE;

    private static final Observation[] Observation_EMPTY = {};

    @Override
    public Entity entity() { return ENTITY; }

    @Override
    public BinaryObject create() { return new Observations(); }

    @Override
    public void encode(Encoder e, BinaryObject obj) throws IOException {
      Observations o = (Observations)obj;
      e.uint32(o.myReads.length);
      for (int i = 0; i < o.myReads.length; i++) {
        e.value(o.myReads[i]);
      }
      e.uint32(o.myWrites.length);
      for (int i = 0; i < o.myWrites.length; i++) {
        e.value(o.myWrites[i]);
      }
    }

    @Override
    public void decode(Decoder d, BinaryObject obj) throws IOException {
      Observations o = (Observations)obj;
      {
        int len = d.uint32();
        o.myReads = len != 0 ? new Observation[len] : Observation_EMPTY;
      }
      for (int i = 0; i <o.myReads.length; i++) {
        o.myReads[i] = new Observation();
        d.value(o.myReads[i]);
      }
      {
        int len = d.uint32();
        o.myWrites = len != 0 ? new Observation[len] : Observation_EMPTY;
      }
      for (int i = 0; i <o.myWrites.length; i++) {
        o.myWrites[i] = new Observation();
        d.value(o.myWrites[i]);
      }
    }
    //<<<End:Java.KlassBody:2>>>
  }
}
