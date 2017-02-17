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
import com.google.gapid.rpclib.schema.Variant;

import java.io.IOException;

public final class AtomList implements BinaryObject {
  public Atom get(long index) {
    return myAtoms[(int)index];
  }

  //<<<Start:Java.ClassBody:1>>>
  private Atom[] myAtoms;

  // Constructs a default-initialized {@link AtomList}.
  public AtomList() {}


  public Atom[] getAtoms() {
    return myAtoms;
  }

  public AtomList setAtoms(Atom[] v) {
    myAtoms = v;
    return this;
  }

  @Override
  public BinaryClass klass() { return Klass.INSTANCE; }


  private static final Entity ENTITY = new Entity("atom", "List", "", "");

  static {
    ENTITY.setFields(new Field[]{
      new Field("Atoms", new Slice("", new Variant("Atom"))),
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
    public BinaryObject create() { return new AtomList(); }

    @Override
    public void encode(Encoder e, BinaryObject obj) throws IOException {
      AtomList o = (AtomList)obj;
      e.uint32(o.myAtoms.length);
      for (int i = 0; i < o.myAtoms.length; i++) {
        e.variant(o.myAtoms[i] == null ? null : o.myAtoms[i].unwrap());
      }
    }

    @Override
    public void decode(Decoder d, BinaryObject obj) throws IOException {
      AtomList o = (AtomList)obj;
      o.myAtoms = new Atom[d.uint32()];
      for (int i = 0; i <o.myAtoms.length; i++) {
        o.myAtoms[i] = Atom.wrap(d.variant());
      }
    }
    //<<<End:Java.KlassBody:2>>>
  }
}
