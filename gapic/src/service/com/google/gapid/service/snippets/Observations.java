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

import com.google.gapid.proto.service.snippets.SnippetsProtos.ObservationType;
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
import com.google.gapid.rpclib.schema.Slice;

import java.io.IOException;

public final class Observations extends KindredSnippets implements BinaryObject {
  //<<<Start:Java.ClassBody:1>>>
  private Pathway myPath;
  private ObservationType[] myObservations;

  // Constructs a default-initialized {@link Observations}.
  public Observations() {}


  public Pathway getPath() {
    return myPath;
  }

  public Observations setPath(Pathway v) {
    myPath = v;
    return this;
  }

  public ObservationType[] getObservations() {
    return myObservations;
  }

  public Observations setObservations(ObservationType[] v) {
    myObservations = v;
    return this;
  }

  @Override
  public BinaryClass klass() { return Klass.INSTANCE; }


  private static final Entity ENTITY = new Entity("snippets", "Observations", "", "");

  static {
    ENTITY.setFields(new Field[]{
      new Field("path", new Interface("Pathway")),
      new Field("observations", new Slice("", new Primitive("ObservationType", Method.Int32))),
    });
    Namespace.register(Klass.INSTANCE);
  }
  public static void register() {}
  //<<<End:Java.ClassBody:1>>>

  /**
   * find the observations in the snippets.
   * @param snippets any kind of snippets.
   * @return the observations maybe null.
   */
  public static Observations fromSnippets(KindredSnippets[] snippets) {
    for (KindredSnippets obj : snippets) {
      if (obj instanceof Observations) {
        return (Observations)obj;
      }
    }
    return null;
  }

  public enum Klass implements BinaryClass {
    //<<<Start:Java.KlassBody:2>>>
    INSTANCE;

    @Override
    public Entity entity() { return ENTITY; }

    @Override
    public BinaryObject create() { return new Observations(); }

    @Override
    public void encode(Encoder e, BinaryObject obj) throws IOException {
      Observations o = (Observations)obj;
      e.object(o.myPath == null ? null : o.myPath.unwrap());
      e.uint32(o.myObservations.length);
      for (int i = 0; i < o.myObservations.length; i++) {
        e.int32(o.myObservations[i].getNumber());
      }
    }

    @Override
    public void decode(Decoder d, BinaryObject obj) throws IOException {
      Observations o = (Observations)obj;
      o.myPath = Pathway.wrap(d.object());
      o.myObservations = new ObservationType[d.uint32()];
      for (int i = 0; i <o.myObservations.length; i++) {
        o.myObservations[i] = ObservationType.valueOf(d.int32());
      }
    }
    //<<<End:Java.KlassBody:2>>>
  }
}
