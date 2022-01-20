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

import static com.google.common.base.Functions.identity;
import static com.google.gapid.rpc.UiErrorCallback.error;
import static com.google.gapid.rpc.UiErrorCallback.success;
import static com.google.gapid.util.Logging.throttleLogRpcError;
import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.Paths.lastCommand;
import static java.util.logging.Level.WARNING;
import static java.util.stream.Collectors.toMap;

import com.google.common.base.Objects;
import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.CommandStream.CommandIndex;
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
import com.google.gapid.util.Ranges;

import org.eclipse.swt.widgets.Shell;

import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

public class Profile
    extends CaptureDependentModel<Profile.Data, Profile.Source, Loadable.Message, Profile.Listener> {
  private static final Logger LOG = Logger.getLogger(Profile.class.getName());
  private static final int FRAME_LOOP_COUNT = 10;

  private static final int SUBMIT_LEVEL = 0;
  private static final int SUBMIT_INFO_LEVEL = 1;
  private static final int COMMAND_BUFFER_LEVEL = 2;
  private static final int COMMAND_LEVEL = 3;

  private final Capture capture;
  private final CommandStream commands;
  private int selectedGroupId;
  private final Settings settings;

  public Profile(
      Shell shell, Analytics analytics, Client client, Capture capture, Devices devices, CommandStream commands, Settings settings) {
    super(LOG, shell, analytics, client, Listener.class, capture, devices);
    this.capture = capture;
    this.commands = commands;
    this.selectedGroupId = -1;
    this.settings = settings;
  }

  @Override
  protected Source getSource(Capture.Data data) {
    return new Source(data.path, new ProfileExperiments());
  }

  @Override
  protected boolean shouldLoad(Capture.Data data) {
    return data.isGraphics();
  }

  @Override
  protected ListenableFuture<Data> doLoad(Source source, Path.Device device) {
    int loopCount = this.settings.preferences().getUseFrameLooping() ? FRAME_LOOP_COUNT : 0;
    return transform(client.profile(capture.getData().path, device, source.experiments, loopCount), r -> new Data(device, r));
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

  public void selectGroup(Service.ProfilingData.Group group) {
    if (group.getId() != selectedGroupId) {
      selectedGroupId = group.getId();
      listeners.fire().onGroupSelected(group);
    }
  }

  public void linkCommandToGpuGroup(List<Long> commandIndex) {
    if (getData() == null || commandIndex == null) {
      return;
    }
    for (Service.ProfilingData.Group group : getData().getGroups()) {
      if (group.getLink().getToList().toString().equals(commandIndex.toString())) {
        selectGroup(group);
      }
    }
  }

  public void linkGpuGroupToCommand(Service.ProfilingData.Group group) {
    // Use a real CommandStream.Node's CommandIndex to trigger the command selection, rather
    // than using a CommandIndex stitched together on the spot. In this way the selection
    // behavior aligns to what happens when selection is from the UI side, where the resource
    // tabs' loading result is based on a "representation" command in the grouping node.
    ListenableFuture<CommandStream.Node> node = MoreFutures.transformAsync(
        commands.getGroupingNodePath(lastCommand(group.getLink())),
        commands::findNode);
    Rpc.listen(node, new UiCallback<CommandStream.Node, CommandStream.Node>(shell, LOG) {
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
          commands.selectCommands(CommandIndex.forCommand(lastCommand(group.getLink())), false);
        } else {
          commands.selectCommands(node.getIndex(), false);
        }
      }
    });
  }

  public ProfileExperiments getExperiments() {
    DeviceDependentModel.Source<Source> src = getSource();
    return (src == null || src.source == null) ? null : src.source.experiments;
  }

  public void updateExperiments(ProfileExperiments experiments) {
    load(Source.withExperiments(getSource(), experiments), false);
  }

  public static class Source {
    public final Path.Capture capture;
    public final ProfileExperiments experiments;

    public Source(Path.Capture capture, ProfileExperiments experiments) {
      this.capture = capture;
      this.experiments = experiments;
    }

    public static DeviceDependentModel.Source<Source> withExperiments(
        DeviceDependentModel.Source<Source> src, ProfileExperiments newExperiments) {
      Source me = (src == null) ? null : src.source;
      return new DeviceDependentModel.Source<Source>((src == null) ? null : src.device,
          new Source((me == null) ? null : me.capture, newExperiments));
    }

    @Override
    public boolean equals(Object obj) {
      if (obj == this) {
        return true;
      } else if (!(obj instanceof Source)) {
        return false;
      }
      Source s = (Source)obj;
      return Objects.equal(capture, s.capture) && Objects.equal(experiments, s.experiments);
    }

    @Override
    public int hashCode() {
      return (capture == null) ? 0 : capture.hashCode();
    }
  }

  public static class Data extends DeviceDependentModel.Data {
    public final Service.ProfilingData profile;
    private final PerfNode rootNode;

    public Data(Path.Device device, Service.ProfilingData profile) {
      super(device);
      this.profile = profile;
      this.rootNode = createPerfNodes(profile);
    }

    private static PerfNode createPerfNodes(Service.ProfilingData profile) {
      Map<Integer, Service.ProfilingData.GpuCounters.Entry> entries =
          profile.getGpuCounters().getEntriesList().stream()
              .collect(toMap(Service.ProfilingData.GpuCounters.Entry::getGroupId, identity()));
      Map<Integer, PerfNode> nodes = Maps.newHashMap(); // groupId -> node.
      nodes.put(0, new PerfNode(null, null));
      // Groups in the proto are sorted and parents come before children.
      for (Service.ProfilingData.Group group : profile.getGroupsList()) {
        PerfNode node = new PerfNode(entries.get(group.getId()), group);
        nodes.put(group.getId(), node);
        nodes.get(group.getParentId()).addChild(node);
      }
      return nodes.get(0);
    }

    public List<Service.ProfilingData.Group> getGroups() {
      return profile.getGroupsList();
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
      return rootNode.children;
    }

    // TODO: this function makes some assumptions about command/sub command IDs. For more details
    // see gapis/trace/android/profile/groups.go.
    public PerfNode getPerfNode(Service.CommandTreeNode treeNode) {
      Path.Commands commands = treeNode.getCommands();
      if (commands.getFromCount() == SUBMIT_INFO_LEVEL + 1) {
        // We don't have perf data for submit infos, bail out early.
        return null;
      }
      PerfNode submit = rootNode.findNode(commands, SUBMIT_LEVEL);
      if (submit == null || commands.getFromCount() == SUBMIT_LEVEL + 1) {
        return submit;
      }
      PerfNode cmdBuf = submit.findNode(commands, COMMAND_BUFFER_LEVEL);
      if (cmdBuf == null || commands.getFromCount() == COMMAND_BUFFER_LEVEL + 1) {
        return cmdBuf;
      }
      PerfNode rp = cmdBuf.findNodeContaining(commands);
      if (rp == null || Ranges.compare(rp.getGroup().getLink(), commands) == 0) {
        return rp;
      }
      return rp.findNodeContaining(commands);
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
    private final Service.ProfilingData.GpuCounters.Entry entry;
    private final Service.ProfilingData.Group group;
    protected final List<PerfNode> children;

    public PerfNode(
        Service.ProfilingData.GpuCounters.Entry entry, Service.ProfilingData.Group group) {
      this.entry = entry;
      this.group = group;
      this.children = Lists.newArrayList();
    }

    public Service.ProfilingData.GpuCounters.Entry getEntry() {
      return entry;
    }

    public Service.ProfilingData.Group getGroup() {
      return group;
    }

    public Map<Integer, Service.ProfilingData.GpuCounters.Perf> getPerfs() {
      return entry.getMetricToValueMap();
    }

    public boolean hasChildren() {
      return children.size() > 0;
    }

    public List<PerfNode> getChildren() {
      return children;
    }

    protected void addChild(PerfNode node) {
      children.add(node);
    }

    // Finds this node's child that matches the given command path up to the given level.
    protected PerfNode findNode(Path.Commands commands, int level) {
      int idx = Collections.binarySearch(children, null, (node, $) -> {
        Path.Commands current = node.group.getLink();
        int result = 0;
        for (int i = 0; result == 0 && i <= level; i++) {
          result = Long.compare(current.getFrom(i), commands.getFrom(i));
          if (result == 0) {
            result = Long.compare(current.getTo(i), commands.getTo(i));
          }
        }
        return result;
      });
      return idx < 0 ? null : children.get(idx);
    }

    // Finds this node's child which fully contains the given command path.
    protected PerfNode findNodeContaining(Path.Commands commands) {
      int idx = Collections.binarySearch(children, null, (node, $) -> {
        Path.Commands current = node.group.getLink();
        int result = 0;
        for (int i = 0; result == 0 && i <= COMMAND_BUFFER_LEVEL; i++) {
          result = Long.compare(current.getFrom(i), commands.getFrom(i));
          if (result == 0) {
            result = Long.compare(current.getTo(i), commands.getTo(i));
          }
        }
        if (result == 0) {
          result = Long.compare(current.getFrom(COMMAND_LEVEL), commands.getTo(COMMAND_LEVEL));
          if (result <= 0) {
            result = Long.compare(current.getTo(COMMAND_LEVEL), commands.getTo(COMMAND_LEVEL));
            if (result >= 0) {
              result = 0;
            }
          }
        }
        return result;
      });
      return idx < 0 ? null : children.get(idx);
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
     *
     * @param group the selected group.
     */
    public default void onGroupSelected(Service.ProfilingData.Group group) { /* empty */ }
  }
}
