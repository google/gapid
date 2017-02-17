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

import com.google.gapid.rpclib.binary.BinaryObject;

import java.util.ArrayList;

public abstract class KindredSnippets implements BinaryObject {
  /**
   * Single empty list to avoid unnecessary empty list allocations.
   */
  private static final KindredSnippets[] Empty = new KindredSnippets[0];

  public static KindredSnippets wrap(BinaryObject obj) {
    return (KindredSnippets)obj;
  }

  public BinaryObject unwrap() {
    return this;
  }

  /**
   * the pathway from the top-level to these snippets.
   * @return the pathway from top-level to these snippets.
   */
  public abstract Pathway getPath();

  /**
   * find the snippets in the metadata.
   * @param metadata arbitrary metadata.
   * @return metadata of type KindredSnippets.
   */
  public static KindredSnippets[] fromMetadata(BinaryObject[] metadata) {
    ArrayList<KindredSnippets> snippets = null;
    for (BinaryObject obj : metadata) {
      if (obj instanceof KindredSnippets) {
        snippets = append(snippets, (KindredSnippets)obj);
      }
    }
    return toArray(snippets);
  }

  /**
   * Convert an ArrayList of snippets to an array of snippets. Use the shared
   * Empty array if the ArrayList is empty or null.
   * @param snippets the ArrayList to convert to an array or null for empty.
   * @return an array list of the snippets.
   */
  public static KindredSnippets[] toArray(ArrayList<KindredSnippets> snippets) {
    if (snippets == null || snippets.isEmpty()) {
      return Empty;
    }
    return snippets.toArray(new KindredSnippets[snippets.size()]);
  }

  /**
   * Add to an array list, but allow null to signify an empty list.
   * @param list the array list to be added to of null.
   * @param snip the snippet to add to the list.
   * @return the array list with the item added.
   */
  public static ArrayList<KindredSnippets> append(ArrayList<KindredSnippets> list, KindredSnippets snip) {
    if (list == null) {
      list = new ArrayList<KindredSnippets>();
    }
    list.add(snip);
    return list;
  }
}

