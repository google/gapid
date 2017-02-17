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
package com.google.gapid.models;

import static com.google.gapid.util.Ranges.last;

import com.google.gapid.proto.service.Service.CommandRange;
import com.google.gapid.proto.service.Service.Context;
import com.google.gapid.proto.service.Service.Value;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.proto.service.path.Path.Contexts;
import com.google.gapid.server.Client;
import com.google.gapid.service.atom.AtomList;
import com.google.gapid.util.Events;
import com.google.gapid.util.Events.ListenerCollection;
import com.google.gapid.util.Paths;
import com.google.gapid.util.Ranges;

import org.eclipse.swt.widgets.Shell;

import java.util.Arrays;
import java.util.List;
import java.util.Objects;
import java.util.logging.Logger;

public class ApiContext extends CaptureDependentModel<ApiContext.FilteringContext[]> {
  private static final Logger LOG = Logger.getLogger(ApiContext.class.getName());

  private final ListenerCollection<Listener> listeners = Events.listeners(Listener.class);
  private FilteringContext selectedContext = FilteringContext.ALL;

  public ApiContext(Shell shell, Client client, Capture capture) {
    super(LOG, shell, client, capture);
  }

  @Override
  protected void reset() {
    super.reset();
    selectedContext = FilteringContext.ALL;
  }

  @Override
  protected Path.Any getPath(Path.Capture capturePath) {
    return Path.Any.newBuilder()
        .setContexts(Contexts.newBuilder()
            .setCapture(capturePath))
        .build();
  }

  @Override
  protected FilteringContext[] unbox(Value value) {
    List<Context> contexts = value.getContexts().getListList();
    if (contexts.isEmpty()) {
      return new FilteringContext[0];
    } else if (contexts.size() == 1) {
      return new FilteringContext[] { FilteringContext.withoutFilter(contexts.get(0)) };
    } else {
      FilteringContext[] result = new FilteringContext[contexts.size() + 1];
      result[0] = FilteringContext.ALL;
      for (int i = 0; i < contexts.size(); i++) {
        result[i + 1] = new FilteringContext(contexts.get(i));
      }
      return result;
    }
  }

  @Override
  protected void fireLoadEvent() {
    if (selectedContext == FilteringContext.ALL && count() == 1) {
      selectedContext = getData()[0];
    }
    listeners.fire().onContextsLoaded();
  }

  public int count() {
    return isLoaded() ? getData().length : 0;
  }

  public FilteringContext getSelectedContext() {
    return selectedContext;
  }

  public void selectContext(FilteringContext context) {
    if (!Objects.equals(context, selectedContext)) {
      selectedContext = context;
      listeners.fire().onContextSelected(context);
    }
  }

  public void selectContextContaining(CommandRange atoms) {
    if (isLoaded() && !selectedContext.contains(last(atoms))) {
      for (FilteringContext context : getData()) {
        if (context != FilteringContext.ALL && context.contains(last(atoms))) {
          selectContext(context);
          return;
        }
      }
      // Fallback to selecting the ALL context.
      selectContext(FilteringContext.ALL);
    }
  }

  public void addListener(Listener listener) {
    listeners.addListener(listener);
  }

  public void removeListener(Listener listener) {
    listeners.removeListener(listener);
  }

  public static class FilteringContext {
    public static final FilteringContext ALL = new FilteringContext(null) {
      @Override
      public Path.ID getId() {
        return Paths.ZERO_ID;
      }

      @Override
      public List<CommandRange> getRanges(AtomList atoms) {
        return Arrays.asList(CommandRange.newBuilder().setCount(atoms.getAtoms().length).build());
      }

      @Override
      public boolean contains(CommandRange range) {
        return true;
      }

      @Override
      public boolean contains(long index) {
        return true;
      }

      @Override
      public String toString() {
        return "All contexts";
      }
    };

    private final Context context;

    public FilteringContext(Context context) {
      this.context = context;
    }

    public static FilteringContext withoutFilter(Context context) {
      return new FilteringContext(context) {
        @Override
        public List<CommandRange> getRanges(AtomList atoms) {
          return ALL.getRanges(atoms);
        }

        @Override
        public boolean contains(CommandRange range) {
          return true;
        }

        @Override
        public boolean contains(long index) {
          return true;
        }
      };
    }

    public Path.ID getId() {
      return context.getId();
    }

    public List<CommandRange> getRanges(@SuppressWarnings("unused") AtomList atoms) {
      return context.getRangesList();
    }

    public boolean contains(CommandRange range) {
      return Ranges.overlaps(context.getRangesList(), range);
    }

    public boolean contains(long index) {
      return Ranges.contains(context.getRangesList(), index) >= 0;
    }

    @Override
    public String toString() {
      return context.getName();
    }

    @Override
    public boolean equals(Object obj) {
      if (this == obj) {
        return true;
      } else if (!(obj instanceof FilteringContext)) {
        return false;
      }
      return Objects.equals(getId(), ((FilteringContext)obj).getId());
    }

    @Override
    public int hashCode() {
      return getId().hashCode();
    }
  }

  @SuppressWarnings("unused")
  public static interface Listener extends Events.Listener {
    public default void onContextsLoaded() { /* empty */ }
    public default void onContextSelected(FilteringContext context) { /* empty */ }
  }
}
