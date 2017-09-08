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
import static java.util.Comparator.comparingInt;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;

import org.eclipse.swt.widgets.Shell;

import java.util.List;
import java.util.Objects;
import java.util.logging.Logger;

/**
 * Model containing the different API contexts of a capture.
 */
public class ApiContext
    extends CaptureDependentModel<ApiContext.FilteringContext[], ApiContext.Listener> {
  private static final Logger LOG = Logger.getLogger(ApiContext.class.getName());

  private FilteringContext selectedContext = null;

  public ApiContext(Shell shell, Client client, Capture capture) {
    super(LOG, shell, client, Listener.class, capture);
  }

  @Override
  protected void reset(boolean maintainState) {
    super.reset(maintainState);
    if (!maintainState) {
      selectedContext = null;
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
  protected ListenableFuture<FilteringContext[]> doLoad(Path.Any path) {
    return Futures.transform(Futures.transformAsync(client.get(path), val -> {
      List<ListenableFuture<ApiContext.IdAndContext>> contexts = Lists.newArrayList();
      for (Path.Context ctx : val.getContexts().getListList()) {
        contexts.add(Futures.transform(client.get(Path.Any.newBuilder().setContext(ctx).build()),
            value -> new IdAndContext(ctx, value.getContext())));
      }
      return Futures.allAsList(contexts);
    }), this::unbox);
  }

  private FilteringContext[] unbox(List<IdAndContext> contexts) {
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
  protected void fireLoadStartEvent() {
    // Do nothing.
  }

  @Override
  protected void fireLoadedEvent() {
    if (count() == 1) {
      selectedContext = getData()[0];
    } else if (selectedContext != null) {
      selectedContext = stream(getData())
          .filter(c -> c.equals(selectedContext))
          .findFirst()
          .orElseGet(this::highestPriorityContext);
    } else {
      selectedContext = highestPriorityContext();
    }
    listeners.fire().onContextsLoaded();
  }

  private FilteringContext highestPriorityContext() {
    return stream(getData())
        .max(comparingInt(FilteringContext::getPriority))
        .orElse(FilteringContext.ALL);
  }

  public int count() {
    return isLoaded() ? getData().length : 0;
  }

  public FilteringContext getSelectedContext() {
    return selectedContext != null ? selectedContext : highestPriorityContext();
  }

  public void selectContext(FilteringContext context) {
    if (!Objects.equals(context, selectedContext)) {
      selectedContext = context;
      listeners.fire().onContextSelected(context);
    }
  }

  /**
   * A {@link com.google.gapid.proto.service.Service.Context} wrapper to allow filtering of the
   * command tree.
   */
  public static class FilteringContext {
    public static final FilteringContext ALL = new FilteringContext(null, null) {
      @Override
      public Path.CommandTree.Builder commandTree(Path.CommandTree.Builder path) {
        return path
            .setGroupByContext(true)
            .setIncludeNoContextGroups(true);
      }

      @Override
      public Path.Events.Builder events(Path.Events.Builder path) {
        return path;
      }

      @Override
      public Path.Report.Builder report(Path.Report.Builder path) {
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

    private final Path.ID id;
    private final Service.Context context;

    public FilteringContext(IdAndContext context) {
      this(context.id, context.context);
    }

    protected FilteringContext(Path.ID id, Service.Context context) {
      this.id = id;
      this.context = context;
    }

    public static FilteringContext withoutFilter(IdAndContext context) {
      return new FilteringContext(context.id, context.context) {
        @Override
        public Path.CommandTree.Builder commandTree(Path.CommandTree.Builder path) {
          return path
              .setGroupByFrame(true)
              .setGroupByDrawCall(true)
              .setGroupByTransformFeedback(true)
              .setGroupByUserMarkers(true)
              .setAllowIncompleteFrame(true);
        }

        @Override
        public Path.Events.Builder events(Path.Events.Builder path) {
          return path;
        }

        @Override
        public Path.Report.Builder report(Path.Report.Builder path) {
          return path;
        }
      };
    }

    public Path.CommandTree.Builder commandTree(Path.CommandTree.Builder path) {
      path.getFilterBuilder().setContext(id);
      return path
          .setGroupByFrame(true)
          .setGroupByDrawCall(true)
          .setGroupByTransformFeedback(true)
          .setGroupByUserMarkers(true)
          .setAllowIncompleteFrame(true);
    }

    public int getPriority() {
      return context != null ? context.getPriority() : 0;
    }

    public Path.State.Builder state(Path.State.Builder path) {
      if (id != null) {
        path.getContextBuilder().setData(id.getData());
      }
      return path;
    }

    public Path.StateTree.Builder stateTree(Path.StateTree.Builder path) {
      if (id != null) {
        path.getStateBuilder().getContextBuilder().setData(id.getData());
      }
      return path;
    }

    public Path.Events.Builder events(Path.Events.Builder path) {
      path.getFilterBuilder().setContext(id);
      return path;
    }

    public Path.Report.Builder report(Path.Report.Builder path) {
      path.getFilterBuilder().setContext(id);
      return path;
    }

    @Override
    public String toString() {
      return context.getName();
    }

    @Override
    public boolean equals(Object obj) {
      if (this == obj) {
        return true;
      } else if (obj == ALL || !(obj instanceof FilteringContext)) {
        return false;
      }
      return Objects.equals(id, ((FilteringContext)obj).id);
    }

    @Override
    public int hashCode() {
      return id.hashCode();
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

  protected static class IdAndContext {
    public final Path.ID id;
    public final Service.Context context;

    public IdAndContext(Path.Context path, Service.Context context) {
      this.id = path.getId();
      this.context = context;
    }
  }
}
