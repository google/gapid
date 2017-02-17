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

public class DynamicAtom extends Atom {
  private final Dynamic myValue;
  private final AtomMetadata myMetadata;
  private final Observations myObservations;

  private static final Object[] EMPTY_OBJECT_ARRAY = {};

  public DynamicAtom(Dynamic value) {
    myValue = value;
    myMetadata = AtomMetadata.find(value.type());
    if (myMetadata.myExtrasIndex >= 0) {
      myObservations = getObservations(getFieldValue(myMetadata.myExtrasIndex));
      myValue.setFieldValue(myMetadata.myExtrasIndex, EMPTY_OBJECT_ARRAY); // Dump the extras to save RAM.
    } else {
      myObservations = null;
    }
  }

  private static Observations getObservations(Object extras) {
    assert (extras instanceof Object[]);
    for (Object extra : (Object[])extras) {
      if (extra instanceof Observations) {
        return (Observations)extra;
      }
    }
    return null;
  }


  @Override
  public BinaryObject unwrap() {
    return myValue;
  }

  @Override
  public String getName() {
    return myMetadata.getDisplayName();
  }

  @Override
  public int getFieldCount() {
    return myValue.getFieldCount();
  }

  @Override
  public Field getFieldInfo(int index) {
    return myValue.getFieldInfo(index);
  }

  @Override
  public Object getFieldValue(int index) {
    return myValue.getFieldValue(index);
  }

  @Override
  public int getExtrasIndex() {
    return myMetadata.myExtrasIndex;
  }

  @Override
  public Observations getObservations() {
    return myObservations;
  }

  @Override
  public int getResultIndex() {
    return myMetadata.myResultIndex;
  }

  @Override
  public boolean isEndOfFrame() {
    return myMetadata.getEndOfFrame();
  }

  @Override
  public boolean isDrawCall() {
    return myMetadata.getDrawCall();
  }

  @Override
  public boolean equals(Object o) {
    if (this == o) return true;
    if (o == null || getClass() != o.getClass()) return false;
    DynamicAtom that = (DynamicAtom)o;
    if (!myValue.equals(that.myValue)) return false;
    if (!myMetadata.equals(that.myMetadata)) return false;
    return true;
  }

  @Override
  public int hashCode() {
    int result = myValue.hashCode();
    result = 31 * result + myMetadata.hashCode();
    return result;
  }

  @Override
  public String toString() {
    StringBuilder sb = new StringBuilder();
    sb.append(getName());
    sb.append('(');
    boolean needComma = false;
    for (int i = 0, c = getFieldCount(); i < c; i++) {
      if (!isParameter(i)) { continue; }
      if (needComma) { sb.append(", "); }
      needComma = true;
      sb.append(getFieldInfo(i).getName());
      sb.append(": ");
      sb.append(getFieldValue(i));
    }
    sb.append(')');
    return sb.toString();
  }
}
