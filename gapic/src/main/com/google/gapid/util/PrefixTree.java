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
package com.google.gapid.util;

import java.util.Iterator;

/**
 * Fast prefix lookup (Trie) data structure.
 */
public class PrefixTree<V extends PrefixTree.Value> {
  private String key;
  private V value;
  private PrefixTree<V> child;
  private PrefixTree<V> sibling;
  private final char firstChar;

  private PrefixTree(String key, V value, PrefixTree<V> child, PrefixTree<V> sibling) {
    this.key = key;
    this.value = value;
    this.child = child;
    this.sibling = sibling;
    this.firstChar = (key.length() == 0) ? '\0' : key.charAt(0);
  }

  /**
   * @return a {@link PrefixTree} containing the values of the given {@link Iterable}.
   */
  public static <V extends Value> PrefixTree<V> of(Iterable<V> values) {
    return of(values.iterator());
  }

  /**
   * @return a {@link PrefixTree} containing the values of the given {@link Iterator}.
   */
  public static <V extends Value> PrefixTree<V> of(Iterator<V> values) {
    PrefixTree<V> tree = new PrefixTree<V>("", null, null, null);
    while (values.hasNext()) {
      tree.put(values.next());
    }
    return tree;
  }

  /**
   * Adds the given value to this tree.
   */
  public void put(V newValue) {
    put(newValue.getKey(), newValue);
  }

  /**
   * Looks up all values in this tree that have the given prefix.
   */
  public void find(String search, MatchCollector<V> collector) {
    if (search.length() <= key.length()) {
      if (key.startsWith(search)) {
        collect(collector);
      }
    } else if (search.startsWith(key)) {
      search = search.substring(key.length());
      PrefixTree<V> node = findChild(search);
      if (node != null) {
        node.find(search, collector);
      }
    }
  }

  /**
   * @return the value with the given exact match key or {@code null}.
   */
  public V get(String search) {
    if (!search.startsWith(key)) {
      return null;
    } else if (search.length() == key.length()) {
      return value;
    }

    search = search.substring(key.length());
    PrefixTree<V> node = findChild(search);
    return (node == null) ? null : node.get(search);
  }

  private void put(String newKey, V newValue) {
    int common = getCommonCount(newKey);
    if (common == key.length()) {
      if (common < newKey.length()) {
        String childKey = newKey.substring(common);
        PrefixTree<V> node = findChild(childKey);
        if (node != null) {
          node.put(childKey, newValue);
        } else {
          child = new PrefixTree<V>(childKey, newValue, null, child);
        }
      } else {
        value = newValue;
      }
    } else {
      child = new PrefixTree<V>(key.substring(common), value, child, null);
      key = newKey.substring(0, common);
      if (common < newKey.length()) {
        child = new PrefixTree<V>(newKey.substring(common), newValue, null, child);
        value = null;
      } else {
        value = newValue;
      }
    }
  }

  private int getCommonCount(String search) {
    int max = Math.min(key.length(), search.length());
    for (int i = 0; i < max; i++) {
      if (search.charAt(i) != key.charAt(i)) {
        return i;
      }
    }
    return max;
  }

  private PrefixTree<V> findChild(String search) {
    char searchFirstChar = search.charAt(0);
    for (PrefixTree<V> node = child; node != null; node = node.sibling) {
      if (node.firstChar == searchFirstChar) {
        return node;
      }
    }
    return null;
  }

  private boolean collect(MatchCollector<V> collector) {
    if (value != null && !collector.collect(value)) {
      return false;
    }

    for (PrefixTree<V> node = child; node != null; node = node.sibling) {
      if (!node.collect(collector)) {
        return false;
      }
    }
    return true;
  }

  /**
   * Value that can be stored in this tree.
   */
  public static interface Value {
    /**
     * @return the string key associated with this value.
     */
    public String getKey();
  }

  /**
   * Collects results of a prefix based search.
   */
  public static interface MatchCollector<V extends Value> {
    /**
     * @return whether to continue reporting matches.
     */
    public boolean collect(V value);
  }
}
