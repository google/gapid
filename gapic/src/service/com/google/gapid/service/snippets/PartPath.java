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
package com.google.gapid.service.snippets;

import com.google.gapid.proto.service.snippets.SnippetsProtos.PartKind;
import com.google.gapid.rpclib.binary.BinaryClass;
import com.google.gapid.rpclib.binary.BinaryObject;
import com.google.gapid.rpclib.binary.Decoder;
import com.google.gapid.rpclib.binary.Encoder;
import com.google.gapid.rpclib.binary.Namespace;
import com.google.gapid.rpclib.schema.Entity;
import com.google.gapid.rpclib.schema.Field;
import com.google.gapid.rpclib.schema.Interface;
import com.google.gapid.rpclib.schema.Method;
import com.google.gapid.rpclib.schema.Primitive;

import java.io.IOException;

final class PartPath extends Pathway implements BinaryObject {

  @Override
  public String getSegmentString() {
    return "PartPath(" + myKind + ")";
  }

  //<<<Start:Java.ClassBody:1>>>
  private Pathway myBase;
  private PartKind myKind;

  // Constructs a default-initialized {@link PartPath}.
  public PartPath() {}


  public Pathway getBase() {
    return myBase;
  }

  public PartPath setBase(Pathway v) {
    myBase = v;
    return this;
  }

  public PartKind getKind() {
    return myKind;
  }

  public PartPath setKind(PartKind v) {
    myKind = v;
    return this;
  }

  @Override
  public BinaryClass klass() { return Klass.INSTANCE; }


  private static final Entity ENTITY = new Entity("snippets", "partPath", "", "");

  static {
    ENTITY.setFields(new Field[]{
      new Field("base", new Interface("Pathway")),
      new Field("kind", new Primitive("PartKind", Method.Int32)),
    });
    Namespace.register(Klass.INSTANCE);
  }
  public static void register() {}
  //<<<End:Java.ClassBody:1>>>

  public PartPath(Pathway base, PartKind kind) {
    myBase = base;
    myKind = kind;
  }

  @Override
  public Pathway base() {
    return getBase();
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    PartPath partPath = (PartPath)o;

    if (myBase != null ? !myBase.equals(partPath.myBase) : partPath.myBase != null) return false;
    if (myKind != null ? !myKind.equals(partPath.myKind) : partPath.myKind != null) return false;

    return true;
  }

  @Override
  public int hashCode() {
    int result = myBase != null ? myBase.hashCode() : 0;
    result = 31 * result + (myKind != null ? myKind.hashCode() : 0);
    return result;
  }

  public enum Klass implements BinaryClass {
    //<<<Start:Java.KlassBody:2>>>
    INSTANCE;

    @Override
    public Entity entity() { return ENTITY; }

    @Override
    public BinaryObject create() { return new PartPath(); }

    @Override
    public void encode(Encoder e, BinaryObject obj) throws IOException {
      PartPath o = (PartPath)obj;
      e.object(o.myBase == null ? null : o.myBase.unwrap());
      e.int32(o.myKind.getNumber());
    }

    @Override
    public void decode(Decoder d, BinaryObject obj) throws IOException {
      PartPath o = (PartPath)obj;
      o.myBase = Pathway.wrap(d.object());
      o.myKind = PartKind.valueOf(d.int32());
    }
    //<<<End:Java.KlassBody:2>>>
  }
}
