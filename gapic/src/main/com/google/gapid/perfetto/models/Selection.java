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
import static java.util.Arrays.stream;
import static java.util.stream.Collectors.joining;

import com.google.common.collect.Iterables;
import com.google.common.collect.Maps;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.views.MultiSelectionView;
import com.google.gapid.perfetto.views.State;

import org.eclipse.swt.widgets.Composite;

import java.util.Map;

/**
 * Data about the current selection in the UI.
 */
public interface Selection {
  public String getTitle();
  public Composite buildUi(Composite parent, State state);

  public static class MultiSelection implements Selection {
    private final Selection[] selections;

    public MultiSelection(Selection[] selections) {
      this.selections = selections;
    }

    @Override
    public String getTitle() {
      return stream(selections).map(Selection::getTitle).collect(joining(", "));
    }

    @Override
    public Composite buildUi(Composite parent, State state) {
      if (selections.length == 1) {
        return selections[0].buildUi(parent, state);
      } else {
        return new MultiSelectionView(parent, selections, state);
      }
    }
  }

  public static class CombiningBuilder {
    private final Map<Comparable<?>, ListenableFuture<Combinable<?>>> selections =
        Maps.newTreeMap();

    @SuppressWarnings("unchecked")
    public <T extends Combinable<T>> void add(
        Comparable<?> type, ListenableFuture<Combinable<?>> selection) {
      selections.merge(type, selection, (f1, f2) -> transformAsync(f1, r1 ->
        transform(f2, r2 -> (((T)r1).combine((T)r2)))));
    }

    public ListenableFuture<? extends Selection> build() {
      if (selections.size() == 1) {
        return transform(Iterables.getOnlyElement(selections.values()), Combinable::build);
      }

      return transform(Futures.allAsList(selections.values()), sels -> {
        Selection[] res = new Selection[sels.size()];
        for (int i = 0; i < res.length; i++) {
          res[i] = sels.get(i).build();
        }
        return new MultiSelection(res);
      });
    }

    public static interface Combinable<T extends Combinable<T>> {
      public T combine(T other);
      public Selection build();
    }
  }
}
