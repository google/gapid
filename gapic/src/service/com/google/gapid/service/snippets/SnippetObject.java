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

import com.google.common.primitives.UnsignedLong;
import com.google.gapid.rpclib.binary.BinaryObject;
import com.google.gapid.rpclib.schema.Dynamic;
import com.google.gapid.rpclib.schema.Field;
import com.google.gapid.service.atom.Atom;

import java.util.ArrayList;
import java.util.Map;

/**
 * SnippetObject provides a way to wrap an object in the gfx data
 * model and associate it with the snippet path and snippets.
 */
public class SnippetObject {
  private final Object object;  // the underlying object
  private final Pathway path;   // the pathway for this object.
  private final KindredSnippets[] mySnippets;  // the snippets at the root.

  /**
   * Construct a snippet object from its sub-components.
   * Note the snippets for an individual object are only computed when requested.
   *
   * @param object the underlying object.
   * @param path the pathway for this object.
   * @param snippets the snippets at the root.
   */
  public SnippetObject(Object object, Pathway path, KindredSnippets[] snippets) {
    this.object = object;
    this.path = path;
    this.mySnippets = snippets;
  }

  @Override
  public String toString() {
    return object == null ? "null" : object.toString();
  }

  public Pathway getPath() {
    return path;
  }

  public Object getObject() {
    return object;
  }

  /**
   * Verify if this object has no interesting snippets.
   * @return true if this object has no interesting snippets.
   */
  public boolean isEmpty() {
    if (mySnippets.length == 0) {
      return true;
    }
    for (KindredSnippets snip : mySnippets) {
      if (path.isPrefix(snip.getPath())) {
        return false;
      }
    }
    return true;
  }

  /**
   * Compute the snippets for this object.
   * @return an array containing the snippets for this object.
   */
  public KindredSnippets[] getSnippets() {
    ArrayList<KindredSnippets> snippets = null;  // avoid allocation if not required.
    for (KindredSnippets snip : mySnippets) {
      if (path.equals(snip.getPath())) {
        snippets = KindredSnippets.append(snippets, snip);
      }
    }
    return KindredSnippets.toArray(snippets);
  }

  /**
   * Determine if this is the root object (aka the global state object).
   * @return true if this is the root object.
   */
  public boolean isRoot() {
    return path == null && mySnippets != null;
  }

  /**
   * Determine if this is a symbol object. Symbol objects are used so that
   * entity field names and map keys can share the same tree structure.
   * @return true if this is a symbol object.
   */
  public boolean isSymbol() {
    return path == null && mySnippets == null && object instanceof String;
  }

  /**
   * Determine if this is a null pointer.
   * @return true if this is a null pointer.
   */
  public boolean isNull() {
    return object == null;
  }

  /**
   * Determine if this object is an atom.
   * @return true if this is an atom.
   */
  public boolean isAtom() {
    return object instanceof Atom;
  }

  /**
   * Determine if this object is a primitive value
   * @return true if this is a primitive value.
   */
  public boolean isPrimitive() {
    return !isBinaryObject() && !isCollection() && !isSymbol() && !isNull();
  }

  /**
   * Determine if this object is a binary object.
   * @return true if this is a binary object.
   */
  public boolean isBinaryObject() {
    return object instanceof BinaryObject;
  }

  /**
   * Determine if this is a collection.
   * @return true if this a collection.
   */
  public boolean isCollection() {
    return object instanceof Map || object instanceof Object[] || object instanceof byte[];
  }

  private static Object longify(Object value) {
    // Behavior inherited from StateController.
    // Turn integers into longs, so they equal longs from paths.
    if (value instanceof Integer) {
      return ((Integer)value).longValue();
    } else if (value instanceof UnsignedLong) {
      return ((UnsignedLong)value).longValue();
    }
    return value;
  }

  /**
   * Build a symbol object. Symbol objects are used so that entity
   * field names and map keys can share the same tree structure.
   * Note the symbol themselves do not have snippets.
   * @return true if this is a symbol object.
   */
  public static SnippetObject symbol(Object symbol) {
    return new SnippetObject(longify(symbol), null, null);
  }

  /**
   * Build a key from a map entry (this object is the map).
   * @param e the map entry containing the key.
   * @return a new snippet object for the key.
   */
  public SnippetObject key(Map.Entry<?, ?> e) {
    return new SnippetObject(longify(e.getKey()), path.key(), mySnippets);
  }

  /**
   * Build an element from a map entry (this object is the map).
   * @param e the map entry containing the value.
   * @return a new snippet object for the element.
   */
  public SnippetObject elem(Map.Entry<?, ?> e) {
    return new SnippetObject(e.getValue(), path.elem(), mySnippets);
  }

  /**
   * Build an element from an array entry (this object is the array).
   * @param element the array entry.
   * @return a new snippet object for the element.
   */
  public SnippetObject elem(Object element) {
    return new SnippetObject(element, path.elem(), mySnippets);
  }

  /**
   * Build a field of a dynamic entity (this object is the entity).
   * @param obj the entity as Dynamic
   * @param fieldIndex the index of the field of the entity.
   * @return a new snippet object for the field value.
   */
  public SnippetObject field(Dynamic obj, int fieldIndex) {
    final Field info = obj.getFieldInfo(fieldIndex);
    final String name = info.getDeclared();
    // In the UI globals are treated like fields of a magic state entity.
    Pathway fieldPath = path == null ? Pathway.global(name) : path.field(name);
    return new SnippetObject(obj.getFieldValue(fieldIndex), fieldPath, mySnippets);
  }

  /**
   * lower case the first character of a string.
   * @param str the string to transform.
   * @return a new string with a lower case first character.
   */
  private static String lowerCaseFirstCharacter(String str) {
    if (str.length() == 0) {
      return str;
    }
    return Character.toLowerCase(str.charAt(0)) + str.substring(1);
  }

  /**
   * Build a new atom parameter object (this object is the atom).
   * @param atom the atom as DynamicAtom
   * @param paramIndex the index of the parameter in the atom.
   * @return a new snippet object for the parameter value.
   */

  public static SnippetObject param(Atom atom, int paramIndex) {
    final KindredSnippets[] snippets =
        KindredSnippets.fromMetadata(atom.unwrap().klass().entity().getMetadata());
    final Field info = atom.getFieldInfo(paramIndex);
    // The parameter name in the schema had the first letter capitalised.
    final String name = lowerCaseFirstCharacter(info.getDeclared());
    return new SnippetObject(
        atom.getFieldValue(paramIndex), Pathway.param(atom.getName(), name), snippets);
  }

  /**
   * Build a root object for the global state.
   * @param obj the Dynamic object containing the global state.
   * @param snippets the snippets for the state object.
   * @return a new snippet object for root object.
   */
  public static SnippetObject root(Dynamic obj, KindredSnippets[] snippets) {
    return new SnippetObject(obj, null, snippets);
  }

  // Note only the underlying object is considered in equals().
  @Override
  public boolean equals(Object o) {
    if (this == o) {
      return true;
    } else if (!(o instanceof SnippetObject)) {
      return false;
    }

    SnippetObject that = (SnippetObject)o;
    if (object != null ? !object.equals(that.object) : that.object != null) return false;

    return true;
  }

  //Note only the underlying object is considered in hashCode()
  @Override
  public int hashCode() {
    return object != null ? object.hashCode() : 0;
  }
}
