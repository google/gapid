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

import com.google.gapid.rpclib.binary.BinaryObject;
import com.google.gapid.rpclib.schema.Dynamic;
import com.google.gapid.rpclib.schema.Field;

public abstract class Atom {
  public static Atom wrap(BinaryObject object) {
    if (object instanceof Dynamic) {
      return new DynamicAtom((Dynamic)object);
    }
    return (Atom)object;
  }

  public int getObservationCount() {
    Observations obs = getObservations();
    return (obs == null) ? 0 : obs.getReads().length + obs.getWrites().length;
  }


  public abstract BinaryObject unwrap();

  public abstract String getName();

  public abstract int getFieldCount();

  public abstract Field getFieldInfo(int index);

  public abstract Object getFieldValue(int index);

  public abstract int getExtrasIndex();

  public abstract Observations getObservations();

  public abstract int getResultIndex();

  public abstract boolean isEndOfFrame();

  public abstract boolean isDrawCall();

  public final boolean isParameter(int fieldIndex) {
    return fieldIndex != getExtrasIndex() && fieldIndex != getResultIndex();
  }
}
