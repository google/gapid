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
package com.google.gapid.models;
import static com.google.gapid.rpc.UiErrorCallback.error;
import static com.google.gapid.rpc.UiErrorCallback.success;
import static com.google.gapid.util.Logging.throttleLogRpcError;
import static com.google.gapid.util.MoreFutures.transform;
import static java.util.Collections.binarySearch;
import static java.util.logging.Level.WARNING;
import static java.util.stream.Collectors.groupingBy;
import static java.util.stream.Collectors.mapping;
import static java.util.stream.Collectors.toList;
import static java.util.stream.Collectors.toSet;

import com.google.common.base.Objects;
import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.CommandStream.Node;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.rpc.UiErrorCallback.ResultOrError;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.MoreFutures;
import com.google.gapid.util.Paths;
import com.google.gapid.util.Ranges;

import org.eclipse.swt.widgets.Shell;

import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

public class Profile
    extends CaptureDependentModel<Profile.Data, Profile.Source, Loadable.Message, Profile.Listener> {
  private static final Logger LOG = Logger.getLogger(Profile.class.getName());

  private final Capture capture;
  private final CommandStream commands;

  public Profile(
      Shell shell, Analytics analytics, Client client, Capture capture, Devices devices, CommandStream commands) {
    super(LOG, shell, analytics, client, Listener.class, capture, devices);
    this.capture = capture;
    this.commands = commands;
  }

  @Override
  protected Source getSource(Capture.Data data) {
    return new Source(data.path);
  }

  @Override
  protected boolean shouldLoad(Capture.Data data) {
    return data.isGraphics();
  }

  @Override
  protected ListenableFuture<Data> doLoad(Source source, Path.Device device) {
    return transform(client.profile(capture.getData().path, device), r -> new Data(device, r));
  }

  @Override
  protected ResultOrError<Data, Loadable.Message> processResult(Rpc.Result<Data> result) {
    try {
      return success(result.get());
    } catch (RpcException e) {
      LOG.log(WARNING, "Failed to load the GPU profile", e);
      return error(Loadable.Message.error(e));
    } catch (ExecutionException e) {
      if (!shell.isDisposed()) {
        throttleLogRpcError(LOG, "Failed to load the GPU profile", e);
      }
      return error(Loadable.Message.error("Failed to load the GPU profile"));
    }
  }

  @Override
  protected void updateError(Loadable.Message error) {
    listeners.fire().onProfileLoaded(error);
  }

  @Override
  protected void fireLoadStartEvent() {
    listeners.fire().onProfileLoadingStart();
  }

  @Override
  protected void fireLoadedEvent() {
    listeners.fire().onProfileLoaded(null);
  }

  public void selectGroup(Service.ProfilingData.GpuSlices.Group group) {
    listeners.fire().onGroupSelected(group);
  }

  public void linkCommandToGpuGroup(List<Long> commandIndex) {
    if (getData() == null || commandIndex == null) {
      return;
    }
    for (Service.ProfilingData.GpuSlices.Group group : getData().getSlices().getGroupsList()) {
      if (group.getLink().getIndicesList().toString().equals(commandIndex.toString())) {
        selectGroup(group);
      }
    }
  }

  public void linkGpuGroupToCommand(Service.ProfilingData.GpuSlices.Group group) {
    // Use a real CommandStream.Node's CommandIndex to trigger the command selection, rather
    // than using a CommandIndex stitched together on the spot. In this way the selection
    // behavior aligns to what happens when selection is from the UI side, where the resource
    // tabs' loading result is based on a "representation" command in the grouping node.
    ListenableFuture<CommandStream.Node> node = MoreFutures.transformAsync(
        commands.getGroupingNodePath(group.getLink()),
        commands::findNode);
    Rpc.listen(node, new UiCallback<Node, Node>(shell, LOG) {
      @Override
      protected CommandStream.Node onRpcThread(Rpc.Result<CommandStream.Node> result)
          throws RpcException, ExecutionException {
        return result.get();
      }

      @Override
      protected void onUiThread(CommandStream.Node node) {
        if (node == null) {
          // A fallback.
          LOG.log(WARNING, "Profile: failed to find the CommandStream.Node for command index: %s", group.getLink());
          commands.selectCommands(CommandIndex.forCommand(group.getLink()), false);
        } else {
          commands.selectCommands(node.getIndex(), false);
        }
      }
    });
  }

  public static class Source {
    public final Path.Capture capture;

    public Source(Path.Capture capture) {
      this.capture = capture;
    }

    @Override
    public boolean equals(Object obj) {
      if (obj == this) {
        return true;
      } else if (!(obj instanceof Source)) {
        return false;
      }
      return Objects.equal(capture, ((Source)obj).capture);
    }

    @Override
    public int hashCode() {
      return (capture == null) ? 0 : capture.hashCode();
    }
  }

  public static class Data extends DeviceDependentModel.Data {
    public final Service.ProfilingData profile;
    private final Map<Integer, List<TimeSpan>> spansByGroup;
    private final List<Service.ProfilingData.GpuSlices.Group> groups;
    private final List<PerfNode> perfNodes;

    public Data(Path.Device device, Service.ProfilingData profile) {
      super(device);
      this.profile = profile;
      this.spansByGroup = aggregateSliceTimeByGroup(profile);
      this.groups = getSortedGroups(profile, spansByGroup.keySet());
      this.perfNodes = createPerfNodes(profile);
    }

    private static Map<Integer, List<TimeSpan>>
        aggregateSliceTimeByGroup(Service.ProfilingData profile) {
      Set<Integer> groups = profile.getSlices().getGroupsList().stream()
          .map(Service.ProfilingData.GpuSlices.Group::getId)
          .collect(toSet());
      return profile.getSlices().getSlicesList().stream()
          .filter(s -> (s.getDepth() == 0) && groups.contains(s.getGroupId()))
          .collect(groupingBy(Service.ProfilingData.GpuSlices.Slice::getGroupId,
              mapping(s -> new TimeSpan(s.getTs(), s.getTs() + s.getDur()), toList())));
    }

    private static List<Service.ProfilingData.GpuSlices.Group> getSortedGroups(
        Service.ProfilingData profile, Set<Integer> ids) {
      return profile.getSlices().getGroupsList().stream()
          .filter(g -> ids.contains(g.getId()))
          .sorted((g1, g2) -> Paths.compare(g1.getLink(), g2.getLink()))
          .collect(toList());
    }

    private static List<PerfNode> createPerfNodes(Service.ProfilingData profile) {
      Map<Integer, PerfNode> nodes = Maps.newHashMap(); // groupId -> node.
      // Create the nodes.
      for (Service.ProfilingData.GpuCounters.Entry entry : profile.getGpuCounters().getEntriesList()) {
        nodes.put(entry.getGroup().getId(), new PerfNode(entry));
      }
      // Link the nodes.
      for (PerfNode node : nodes.values()) {
        if (node.group.hasParent()) {
          nodes.get(node.group.getParent().getId()).children.add(node);
        }
      }
      // Return the root nodes.
      List<PerfNode> rootNodes = Lists.newArrayList();
      for (PerfNode node : nodes.values()) {
        if (!node.group.hasParent()) {
          rootNodes.add(node);
        }
      }
      return rootNodes;
    }

    public boolean hasSlices() {
      return profile.getSlices().getSlicesCount() > 0 &&
          profile.getSlices().getTracksCount() > 0;
    }

    public Service.ProfilingData.GpuSlices getSlices() {
      return profile.getSlices();
    }

    public List<Service.ProfilingData.Counter> getCounters() {
      return profile.getCountersList();
    }

    public Service.ProfilingData.GpuCounters getGpuPerformance() {
      return profile.getGpuCounters();
    }

    public List<PerfNode> getPerfNodes() {
      return perfNodes;
    }

    public TimeSpan getSlicesTimeSpan() {
      if (!hasSlices()) {
        return TimeSpan.ZERO;
      }

      long start = Long.MAX_VALUE, end = 0;
      for (Service.ProfilingData.GpuSlices.Slice slice : profile.getSlices().getSlicesList()) {
        start = Math.min(slice.getTs(), start);
        end = Math.max(slice.getTs() + slice.getDur(), end);
      }
      return new TimeSpan(start, end);
    }

    public Duration getDuration(Path.Commands range) {
      List<TimeSpan> spans = getSpans(range);
      if (spans.size() == 0) {
        return Duration.NONE;
      }
      long start = Long.MAX_VALUE, end = Long.MIN_VALUE;
      long gpuTime = 0, wallTime = 0;
      TimeSpan last = TimeSpan.ZERO;
      for (TimeSpan span : spans) {
        start = Math.min(start, span.start);
        end = Math.max(end, span.end);
        long duration = span.getDuration();
        gpuTime += duration;
        if (span.start < last.end) {
          if (span.end <= last.end) {
            continue; // completely contained within the other, can ignore it.
          }
          duration -= last.end - span.start;
        }
        wallTime += duration;
        last = span;
      }

      TimeSpan ts = start < end ? new TimeSpan(start, end) : TimeSpan.ZERO;
      return new Duration(gpuTime, wallTime, ts);
    }

    private List<TimeSpan> getSpans(Path.Commands range) {
      List<TimeSpan> spans = Lists.newArrayList();
      int idx = binarySearch(groups, null, (g, $) -> Ranges.compare(range, g.getLink()));
      if (idx < 0) {
        return spans;
      }
      for (int i = idx; i >= 0 && Ranges.contains(range, groups.get(i).getLink()); i--) {
        spans.addAll(spansByGroup.get(groups.get(i).getId()));
      }
      for (int i = idx + 1; i < groups.size() && Ranges.contains(range, groups.get(i).getLink()); i++) {
        spans.addAll(spansByGroup.get(groups.get(i).getId()));
      }
      Collections.sort(spans, (s1, s2) -> Long.compare(s1.start, s2.start));
      return spans;
    }
  }

  public static class Duration {
    public static final Duration NONE = new Duration(0, 0, TimeSpan.ZERO) {
      @Override
      public String formatGpuTime() {
        return "";
      }

      @Override
      public String formatWallTime() {
        return "";
      }
    };

    public final long gpuTime;
    public final long wallTime;
    public final TimeSpan timeSpan;

    public Duration(long gpuTime, long wallTime, TimeSpan timeSpan) {
      this.gpuTime = gpuTime;
      this.wallTime = wallTime;
      this.timeSpan = timeSpan;
    }

    public String formatGpuTime() {
      return String.format("%.3fms", gpuTime / 1e6);
    }

    public String formatWallTime() {
      return String.format("%.3fms", wallTime / 1e6);
    }
  }

  public static class PerfNode {
    private final Service.ProfilingData.GpuSlices.Group group;
    private final Map<Integer, Service.ProfilingData.GpuCounters.Perf> perfs;
    private final List<PerfNode> children;

    public PerfNode(Service.ProfilingData.GpuCounters.Entry entry) {
      this.group = entry.getGroup();
      this.perfs = entry.getMetricToValueMap();
      this.children = Lists.newArrayList();
    }

    public Service.ProfilingData.GpuSlices.Group getGroup() {
      return group;
    }

    public Map<Integer, Service.ProfilingData.GpuCounters.Perf> getPerfs() {
      return perfs;
    }

    public boolean hasChildren() {
      return children.size() > 0;
    }

    public List<PerfNode> getChildren() {
      return children;
    }
  }

  public static interface Listener extends Events.Listener {
    /**
     * Event indicating that profiling data is being loaded.
     */
    public default void onProfileLoadingStart() { /* empty */ }

    /**
     * Event indicating that the profiling data has finished loading.
     *
     * @param error the loading error or {@code null} if loading was successful.
     */
    public default void onProfileLoaded(Loadable.Message error) { /* empty */ }

    /**
     * Event indicating that the currently selected group has changed.
     */
    public default void onGroupSelected(Service.ProfilingData.GpuSlices.Group group) { /* empty */ }
  }
}
