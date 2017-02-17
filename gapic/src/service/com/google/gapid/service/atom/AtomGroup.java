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
import com.google.gapid.rpclib.schema.Method;
import com.google.gapid.rpclib.schema.Primitive;
import com.google.gapid.rpclib.schema.Slice;
import com.google.gapid.rpclib.schema.Struct;

import java.io.IOException;
import java.util.Arrays;

public final class AtomGroup implements BinaryObject {
  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;
    AtomGroup atomGroup = (AtomGroup)o;
    if (myName != null ? !myName.equals(atomGroup.myName) : atomGroup.myName != null) return false;
    if (myRange != null ? !myRange.equals(atomGroup.myRange) : atomGroup.myRange != null) return false;
    // the roots of different hierarchies are all the same with only the subgroups being different.
    if (!Arrays.equals(mySubGroups, atomGroup.mySubGroups)) return false;
    return true;
  }

  @Override
  public int hashCode() {
    int result = myName != null ? myName.hashCode() : 0;
    result = 31 * result + (myRange != null ? myRange.hashCode() : 0);
    // we don't want to hash each element in the array as that takes way too long.
    result = 31 * result + mySubGroups.length;
    return result;
  }

  @Override
  public String toString() {
    return "AtomGroup{" +
           "name='" + myName + '\'' + ", range=" + myRange + ", subGroups=" + Arrays.toString(mySubGroups) + '}';
  }

  //<<<Start:Java.ClassBody:1>>>
  private String myName;
  private Range myRange;
  private AtomGroup[] mySubGroups;

  // Constructs a default-initialized {@link AtomGroup}.
  public AtomGroup() {}


  public String getName() {
    return myName;
  }

  public AtomGroup setName(String v) {
    myName = v;
    return this;
  }

  public Range getRange() {
    return myRange;
  }

  public AtomGroup setRange(Range v) {
    myRange = v;
    return this;
  }

  public AtomGroup[] getSubGroups() {
    return mySubGroups;
  }

  public AtomGroup setSubGroups(AtomGroup[] v) {
    mySubGroups = v;
    return this;
  }

  @Override
  public BinaryClass klass() { return Klass.INSTANCE; }


  private static final Entity ENTITY = new Entity("atom", "Group", "", "");

  static {
    ENTITY.setFields(new Field[]{
      new Field("Name", new Primitive("string", Method.String)),
      new Field("Range", new Struct(Range.Klass.INSTANCE.entity())),
      new Field("SubGroups", new Slice("GroupList", new Struct(AtomGroup.Klass.INSTANCE.entity()))),
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
    public BinaryObject create() { return new AtomGroup(); }

    @Override
    public void encode(Encoder e, BinaryObject obj) throws IOException {
      AtomGroup o = (AtomGroup)obj;
      e.string(o.myName);
      e.value(o.myRange);
      e.uint32(o.mySubGroups.length);
      for (int i = 0; i < o.mySubGroups.length; i++) {
        e.value(o.mySubGroups[i]);
      }
    }

    @Override
    public void decode(Decoder d, BinaryObject obj) throws IOException {
      AtomGroup o = (AtomGroup)obj;
      o.myName = d.string();
      o.myRange = new Range();
      d.value(o.myRange);
      o.mySubGroups = new AtomGroup[d.uint32()];
      for (int i = 0; i <o.mySubGroups.length; i++) {
        o.mySubGroups[i] = new AtomGroup();
        d.value(o.mySubGroups[i]);
      }
    }
    //<<<End:Java.KlassBody:2>>>
  }
}
