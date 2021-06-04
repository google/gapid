/*
 * Copyright (C) 2019 Google Inc.
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
package com.google.gapid.perfetto.models;

import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.MoreFutures.transformAsync;

import com.google.common.collect.Maps;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.views.MultiSelectionView;
import com.google.gapid.perfetto.views.State;

import org.eclipse.swt.widgets.Composite;

import java.util.Iterator;
import java.util.Map;
import java.util.NavigableMap;
import java.util.TreeMap;
import java.util.function.Consumer;

/**
 * Data about the current selection in the UI.
 */
public interface Selection<T extends Selection<T>> {
  public String getTitle();
  public boolean contains(Long key);
  public T combine(T other);
  public Composite buildUi(Composite parent, State state);

  public default void getRange(@SuppressWarnings("unused") Consumer<TimeSpan> span) {
    /* do nothing */
  }

  public default boolean isEmpty() {
    return false;
  }

  public static final Selection<EmptySelection> EMPTY_SELECTION = new EmptySelection();

  @SuppressWarnings("unchecked")
  public static <T extends Selection<T>> T emptySelection() {
      return (T)EMPTY_SELECTION;
  }

  public static class EmptySelection implements Selection<EmptySelection> {
    @Override
    public String getTitle() {
      return "";
    }

    @Override
    public boolean contains(Long key) {
      return false;
    }

    @Override
    public boolean isEmpty() {
      return true;
    }

    @Override
    public Composite buildUi(Composite parent, State state) {
      return null;
    }

    @Override
    public EmptySelection combine(EmptySelection other) {
      return this;
    }
  }

  /**
   * MultiSelection stores selections across different {@link Kind}s.
   * */
  public static class MultiSelection {
    private final NavigableMap<Kind, Selection<?>> selections;

    public MultiSelection(Kind type, Selection<?> selection) {
      this.selections = Maps.newTreeMap();
      this.selections.put(type, selection);
    }

    public MultiSelection(NavigableMap<Kind, Selection<?>> selections) {
      this.selections = selections;
    }

    public Composite buildUi(Composite parent, State state) {
      if (selections.size() == 1) {
        return firstSelection().buildUi(parent, state);
      } else {
        return new MultiSelectionView(parent, selections, state);
      }
    }

    @SuppressWarnings("unchecked")
    public <T extends Selection<T>> T getSelection(Kind type) {
      return (T)(selections.containsKey(type) ? selections.get(type) : Selection.emptySelection());
    }

    public void addSelection(MultiSelection other) {
      for (Selection.Kind k : other.selections.keySet()) {
        this.addSelection(k, other.getSelection(k));
      }
    }

    public void addSelection(Kind kind, Selection<?> selection) {
      Selection<?> old = getSelection(kind);
      if (old == null || old == Selection.EMPTY_SELECTION) {
        selections.put(kind, selection);
      } else {
        selections.put(kind, combine(old, selection));
      }
    }

    public void markTime(State state) {
      getRange().ifNotEmpty(state::setHighlight);
    }

    public void zoom(State state) {
      getRange().ifNotEmpty(state::setVisibleTime);
    }

    private TimeSpan getRange() {
      TimeSpan[] range = new TimeSpan[] { TimeSpan.ZERO };
      for (Selection<?> sel : selections.values()) {
        sel.getRange(r -> range[0] = range[0].expand(r));
      }
      return range[0];
    }

    private Selection<?> firstSelection() {
      return selections.firstEntry().getValue();
    }

    @SuppressWarnings("unchecked")
    protected static <T extends Selection<T>> Selection<?> combine(Selection<?> a, Selection<?> b) {
      return ((T)a).combine((T)b);
    }
  }

  /**
   * Selection builder for combining selections across different {@link Kind}s.
   * */
  public static class CombiningBuilder {
    private final Map<Kind, ListenableFuture<Selection<?>>> selections = Maps.newTreeMap();

    @SuppressWarnings("unchecked")
    public <T extends Selection<T>> void add(Kind type, ListenableFuture<T> selection) {
      selections.merge(type, (ListenableFuture<Selection<?>>)selection, (f1, f2) ->
          transformAsync(f1, r1 -> transform(f2, r2 -> (MultiSelection.combine(r1, r2)))));
    }

    public ListenableFuture<MultiSelection> build() {
      return transform(Futures.allAsList(selections.values()), sels -> {
        Iterator<Kind> keys = selections.keySet().iterator();
        Iterator<Selection<?>> vals = sels.iterator();
        TreeMap<Kind, Selection<?>> res = Maps.newTreeMap();
        while (keys.hasNext()) {
          res.put(keys.next(), vals.next());
        }
        return new MultiSelection(res);
      });
    }
  }

  public static class Kind implements Comparable<Kind>{
    public static final Kind Thread = new Kind(1000);
    public static final Kind ThreadState = new Kind(1010);
    public static final Kind Cpu = new Kind(1020);
    public static final Kind Async = new Kind(1030);
    public static final Kind Gpu = new Kind(1040);
    public static final Kind VulkanEvent = new Kind(1050);
    public static final Kind Counter = new Kind(1060);
    public static final Kind FrameEvents = new Kind(1070);
    public static final Kind Memory = new Kind(1080);
    public static final Kind Battery = new Kind(1090);
    public static final Kind ProcessMemory = new Kind(1100);

    private final int priority;

    public Kind(int priority) {
      this.priority = priority;
    }

    @Override
    public int compareTo(Kind other) {
      return this.priority - other.priority;
    }
  }
}
