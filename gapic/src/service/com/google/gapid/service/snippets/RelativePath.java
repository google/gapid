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

final class RelativePath extends Pathway implements BinaryObject {

  @Override
  public String getSegmentString() {
    return "RelativePath(" + myTypeName + ")";
  }

  //<<<Start:Java.ClassBody:1>>>
  private String myTypeName;

  // Constructs a default-initialized {@link RelativePath}.
  public RelativePath() {}


  public String getTypeName() {
    return myTypeName;
  }

  public RelativePath setTypeName(String v) {
    myTypeName = v;
    return this;
  }

  @Override
  public BinaryClass klass() { return Klass.INSTANCE; }


  private static final Entity ENTITY = new Entity("snippets", "relativePath", "", "");

  static {
    ENTITY.setFields(new Field[]{
      new Field("typeName", new Primitive("string", Method.String)),
    });
    Namespace.register(Klass.INSTANCE);
  }
  public static void register() {}
  //<<<End:Java.ClassBody:1>>>

  @Override
  public Pathway base() {
    return null;
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;

    RelativePath that = (RelativePath)o;

    if (myTypeName != null ? !myTypeName.equals(that.myTypeName) : that.myTypeName != null) return false;

    return true;
  }

  @Override
  public int hashCode() {
    return myTypeName != null ? myTypeName.hashCode() : 0;
  }

  public enum Klass implements BinaryClass {
    //<<<Start:Java.KlassBody:2>>>
    INSTANCE;

    @Override
    public Entity entity() { return ENTITY; }

    @Override
    public BinaryObject create() { return new RelativePath(); }

    @Override
    public void encode(Encoder e, BinaryObject obj) throws IOException {
      RelativePath o = (RelativePath)obj;
      e.string(o.myTypeName);
    }

    @Override
    public void decode(Decoder d, BinaryObject obj) throws IOException {
      RelativePath o = (RelativePath)obj;
      o.myTypeName = d.string();
    }
    //<<<End:Java.KlassBody:2>>>
  }
}
