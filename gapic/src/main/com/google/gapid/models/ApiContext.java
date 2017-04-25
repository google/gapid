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

import static java.util.Arrays.stream;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.Events.ListenerCollection;

import org.eclipse.swt.widgets.Shell;

import java.util.List;
import java.util.Objects;
import java.util.logging.Logger;

/**
 * Model containing the different API contexts of a capture.
 */
public class ApiContext
    extends CaptureDependentModel<ApiContext.FilteringContext[], List<Service.Value>> {
  private static final Logger LOG = Logger.getLogger(ApiContext.class.getName());

  private final ListenerCollection<Listener> listeners = Events.listeners(Listener.class);
  private FilteringContext selectedContext = FilteringContext.ALL;

  public ApiContext(Shell shell, Client client, Capture capture) {
    super(LOG, shell, client, capture);
  }

  @Override
  protected void reset(boolean maintainState) {
    super.reset(maintainState);
    if (!maintainState) {
      selectedContext = FilteringContext.ALL;
    }
  }

  @Override
  protected Path.Any getPath(Path.Capture capturePath) {
    return Path.Any.newBuilder()
        .setContexts(Path.Contexts.newBuilder()
            .setCapture(capturePath))
        .build();
  }

  @Override
  protected ListenableFuture<List<Service.Value>> doLoad(Path.Any path) {
    return Futures.transformAsync(client.get(path), val -> {
      List<ListenableFuture<Service.Value>> contexts = Lists.newArrayList();
      for (Path.Context ctx : val.getContexts().getListList()) {
        contexts.add(client.get(Path.Any.newBuilder().setContext(ctx).build()));
      }
      return Futures.allAsList(contexts);
    });
  }

  @Override
  protected FilteringContext[] unbox(List<Service.Value> contexts) {
    if (contexts.isEmpty()) {
      return new FilteringContext[0];
    } else if (contexts.size() == 1) {
      return new FilteringContext[] {
          FilteringContext.withoutFilter(contexts.get(0).getContext())
      };
    } else {
      FilteringContext[] result = new FilteringContext[contexts.size() + 1];
      result[0] = FilteringContext.ALL;
      for (int i = 0; i < contexts.size(); i++) {
        result[i + 1] = new FilteringContext(contexts.get(i).getContext());
      }
      return result;
    }
  }

  @Override
  protected void fireLoadEvent() {
    if (count() == 1) {
      selectedContext = getData()[0];
    } else if (selectedContext != FilteringContext.ALL) {
      selectedContext = stream(getData())
          .filter(c -> c.equals(selectedContext))
          .findFirst()
          .orElse(FilteringContext.ALL);
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

  /*
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
  */

  public void addListener(Listener listener) {
    listeners.addListener(listener);
  }

  public void removeListener(Listener listener) {
    listeners.removeListener(listener);
  }

  /**
   * A {@link Context} wrapper to allow filtering of commands.
   */
  public static class FilteringContext {
    public static final FilteringContext ALL = new FilteringContext(null) {
      @Override
      public Path.CommandTree.Builder commandTree(Path.CommandTree.Builder path) {
        return path;
      }

      @Override
      public Path.Events.Builder events(Path.Events.Builder path) {
        return path;
      }

      @Override
      public String toString() {
        return "All contexts";
      }

      @Override
      public boolean equals(Object obj) {
        return obj == ALL;
      }

      @Override
      public int hashCode() {
        return 0;
      }
    };

    private final Service.Context context;

    public FilteringContext(Service.Context context) {
      this.context = context;
    }

    public static FilteringContext withoutFilter(Service.Context context) {
      return new FilteringContext(context) {
        @Override
        public Path.CommandTree.Builder commandTree(Path.CommandTree.Builder path) {
          return path;
        }

        @Override
        public Path.Events.Builder events(Path.Events.Builder path) {
          return path;
        }
      };
    }

    public Path.CommandTree.Builder commandTree(Path.CommandTree.Builder path) {
      return path.setContext(context.getPath().getId());
    }

    public Path.Events.Builder events(Path.Events.Builder path) {
      return path.setContext(context.getPath().getId());
    }

    @Override
    public String toString() {
      return context.getName();
    }

    @Override
    public boolean equals(Object obj) {
      if (this == obj) {
        return true;
      } else if (!(obj instanceof FilteringContext) || obj == ALL) {
        return false;
      }
      return Objects.equals(context.getPath(), ((FilteringContext)obj).context.getPath());
    }

    @Override
    public int hashCode() {
      return context.getPath().hashCode();
    }
  }

  @SuppressWarnings("unused")
  public static interface Listener extends Events.Listener {
    /**
     * Event indicating that the contexts have finished loading from the server.
     */
    public default void onContextsLoaded() { /* empty */ }

    /**
     * Event indicating that the currently selected context has changed.
     */
    public default void onContextSelected(FilteringContext context) { /* empty */ }
  }
}
