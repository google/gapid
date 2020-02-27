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

import static com.google.gapid.util.Paths.context;
import static java.util.Arrays.stream;
import static java.util.Comparator.comparingInt;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.MoreFutures;

import org.eclipse.swt.widgets.Shell;

import java.util.List;
import java.util.Objects;
import java.util.logging.Logger;

/**
 * Model containing the different API contexts of a capture.
 */
public class ApiContext
    extends CaptureDependentModel.ForPath<ApiContext.Contexts, ApiContext.Listener> {
  private static final Logger LOG = Logger.getLogger(ApiContext.class.getName());

  private FilteringContext selectedContext = null;

  public ApiContext(
      Shell shell, Analytics analytics, Client client, Capture capture, Devices devices) {
    super(LOG, shell, analytics, client, Listener.class, capture, devices);
  }

  @Override
  protected void reset(boolean maintainState) {
    super.reset(maintainState);
    if (!maintainState) {
      selectedContext = null;
    }
  }

  @Override
  protected Path.Any getSource(Capture.Data capture) {
    return Path.Any.newBuilder()
        .setContexts(Path.Contexts.newBuilder()
            .setCapture(capture.path))
        .build();
  }

  @Override
  protected boolean shouldLoad(Capture.Data capture) {
    return capture != null && capture.isGraphics();
  }

  @Override
  protected ListenableFuture<Contexts> doLoad(Path.Any path, Path.Device device) {
    return MoreFutures.transform(MoreFutures.transformAsync(client.get(path, device), val -> {
      List<ListenableFuture<ApiContext.IdAndContext>> contexts = Lists.newArrayList();
      for (Path.Context ctx : val.getContexts().getListList()) {
        contexts.add(MoreFutures.transform(client.get(context(ctx), device),
            value -> new IdAndContext(ctx, value.getContext())));
      }
      return Futures.allAsList(contexts);
    }), ctxList -> new Contexts(device, unbox(ctxList)));
  }

  private static FilteringContext[] unbox(List<IdAndContext> contexts) {
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
      selectedContext = getData().contexts[0];
    } else if (selectedContext != null) {
      selectedContext = stream(getData().contexts)
          .filter(c -> c.equals(selectedContext))
          .findFirst()
          .orElseGet(this::highestPriorityContext);
    } else {
      selectedContext = highestPriorityContext();
    }
    listeners.fire().onContextsLoaded();
  }

  private FilteringContext highestPriorityContext() {
    return stream(getData().contexts)
        .max(comparingInt(FilteringContext::getPriority))
        .orElse(FilteringContext.ALL);
  }

  public int count() {
    return isLoaded() ? getData().contexts.length : 0;
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

  public static class Contexts extends DeviceDependentModel.Data {
    public final FilteringContext[] contexts;

    public Contexts(Path.Device device, FilteringContext[] contexts) {
      super(device);
      this.contexts = contexts;
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

      @Override
      public boolean matches(Path.Context path) {
        return true;
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
              .setGroupBySubmission(true)
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
          .setGroupBySubmission(true)
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

    public boolean matches(Path.Context path) {
      return Objects.equals(id, path.getID());
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
      this.id = path.getID();
      this.context = context;
    }
  }
}
