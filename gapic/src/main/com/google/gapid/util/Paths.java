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

import com.google.common.collect.Lists;
import com.google.common.primitives.UnsignedLongs;
import com.google.gapid.image.Images;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.Settings;
import com.google.gapid.proto.SettingsProto;
import com.google.gapid.proto.device.Device;
import com.google.gapid.proto.image.Image;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.proto.service.vertex.Vertex;
import com.google.gapid.views.Formatter;
import com.google.protobuf.Message;

import java.util.LinkedList;
import java.util.List;
import java.util.function.Predicate;

/**
 * Path utilities.
 */
public class Paths {
  private Paths() {
  }

  public static Path.Command command(Path.Capture capture, long index) {
    if (capture == null) {
      return null;
    }
    return Path.Command.newBuilder()
        .setCapture(capture)
        .addIndices(index)
        .build();
  }

  public static Path.Command firstCommand(Path.Commands commands) {
    return Path.Command.newBuilder()
        .setCapture(commands.getCapture())
        .addAllIndices(commands.getFromList())
        .build();
  }

  public static Path.Command lastCommand(Path.Commands commands) {
    return Path.Command.newBuilder()
        .setCapture(commands.getCapture())
        .addAllIndices(commands.getToList())
        .build();
  }

  public static Path.Device device(Device.ID device) {
    return Path.Device.newBuilder()
        .setID(Path.ID.newBuilder()
            .setData(device.getData()))
        .build();
  }

  public static Path.Any capture(Path.ID id, boolean excludeObservations) {
    return Path.Any.newBuilder()
        .setCapture(Path.Capture.newBuilder()
            .setID(id)
            .setExcludeMemoryRanges(excludeObservations))
        .build();
  }

  public static Path.Any command(Path.Command command) {
    return Path.Any.newBuilder()
        .setCommand(command)
        .build();
  }

  public static Path.Any commandTree(Path.Capture capture, CommandFilter filter) {
    return Path.Any.newBuilder()
        .setCommandTree(filter.toProto(
            Path.CommandTree.newBuilder()
                .setCapture(capture)
                .setGroupByFrame(true)
                .setGroupByDrawCall(true)
                .setGroupByTransformFeedback(true)
                .setGroupByUserMarkers(true)
                .setGroupBySubmission(true)
                .setAllowIncompleteFrame(true)
                .setMaxChildren(2000)
                .setMaxNeighbours(20)))
        .build();
  }

  public static Path.Any commandTree(Path.CommandTreeNode node) {
    return Path.Any.newBuilder()
        .setCommandTreeNode(node)
        .build();
  }

  public static Path.Any commandTree(Path.CommandTreeNode.Builder node) {
    return Path.Any.newBuilder()
        .setCommandTreeNode(node)
        .build();
  }

  public static Path.Any commandTreeNodeForCommand(Path.ID tree, Path.Command command, boolean preferGroup) {
    return Path.Any.newBuilder()
        .setCommandTreeNodeForCommand(Path.CommandTreeNodeForCommand.newBuilder()
            .setTree(tree)
            .setCommand(command)
            .setPreferGroup(preferGroup))
        .build();
  }

  public static Path.State stateAfter(Path.Command command) {
    if (command == null) {
      return null;
    }
    return Path.State.newBuilder()
        .setAfter(command)
        .build();
  }

  public static Path.Any stateTree(CommandIndex command) {
    if (command == null) {
      return null;
    }
    return Path.Any.newBuilder()
        .setStateTree(Path.StateTree.newBuilder()
            .setState(stateAfter(command.getCommand()))
            .setArrayGroupSize(2000))
        .build();
  }

  public static Path.Any stateTree(Path.ID tree, Path.Any statePath) {
    return Path.Any.newBuilder()
        .setStateTreeNodeForPath(Path.StateTreeNodeForPath.newBuilder()
            .setTree(tree)
            .setMember(statePath))
        .build();
  }

  public static Path.Any stateTree(Path.StateTreeNode node) {
    return Path.Any.newBuilder()
        .setStateTreeNode(node)
        .build();
  }

  public static Path.Any stateTree(Path.StateTreeNode.Builder node) {
    return Path.Any.newBuilder()
        .setStateTreeNode(node)
        .build();
  }

  public static Path.Any memoryAfter(CommandIndex index, int pool, long address, long size) {
    if (index == null) {
      return null;
    }
    return Path.Any.newBuilder()
        .setMemory(Path.Memory.newBuilder()
            .setAfter(index.getCommand())
            .setPool(pool)
            .setAddress(address)
            .setSize(size)
            .setIncludeTypes(true))
        .build();
  }

  public static Path.Any memoryAfter(CommandIndex index, int pool, Service.MemoryRange range) {
    if (index == null || range == null) {
      return null;
    }
    return Path.Any.newBuilder()
        .setMemory(Path.Memory.newBuilder()
            .setAfter(index.getCommand())
            .setPool(pool)
            .setAddress(range.getBase())
            .setSize(range.getSize())
            .setIncludeTypes(true))
        .build();
  }

  public static Path.Any observationsAfter(CommandIndex index, int pool) {
    if (index == null) {
      return null;
    }
    return Path.Any.newBuilder()
        .setMemory(Path.Memory.newBuilder()
            .setAfter(index.getCommand())
            .setPool(pool)
            .setAddress(0)
            .setSize(UnsignedLongs.MAX_VALUE)
            .setExcludeData(true)
            .setExcludeObserved(true)
            .setIncludeTypes(true))
        .build();
  }

  public static Path.Any memoryAsType(CommandIndex index, int pool, Service.TypedMemoryRange typed) {
    return Path.Any.newBuilder()
        .setMemoryAsType(Path.MemoryAsType.newBuilder()
            .setAddress(typed.getRange().getBase())
            .setSize(typed.getRange().getSize())
            .setPool(pool)
            .setAfter(index.getCommand())
            .setType(typed.getType()))
        .build();
  }

  public static Path.Any resourceAfter(CommandIndex command, Path.ID id) {
    if (command == null || id == null) {
      return null;
    }
    return Path.Any.newBuilder()
        .setResourceData(Path.ResourceData.newBuilder()
            .setAfter(command.getCommand())
            .setID(id))
        .build();
  }

  public static Path.Any resourcesAfter(CommandIndex command, Path.ResourceType type) {
    if (command == null || type == null) {
      return null;
    }
    return Path.Any.newBuilder()
        .setMultiResourceData(Path.MultiResourceData.newBuilder()
            .setAfter(command.getCommand())
            .setAll(true)
            .setType(type))
        .build();
  }

  public static Path.Any pipelinesAfter(CommandIndex command) {
    if (command == null) {
      return null;
    }
    return Path.Any.newBuilder()
        .setPipelines(Path.Pipelines.newBuilder()
            .setCommandTreeNode(command.getNode()))
        .build();
  }

  public static Path.Any framebufferAttachmentsAfter(CommandIndex command) {
    if (command == null) {
      return null;
    }
    return Path.Any.newBuilder()
        .setFramebufferAttachments(Path.FramebufferAttachments.newBuilder()
            .setAfter(command.getCommand()))
        .build();
  }

  public static Path.Any framebufferAttachmentAfter(CommandIndex command, int index, Path.RenderSettings settings, Path.UsageHints hints) {
    if (command == null) {
      return null;
    }
    return Path.Any.newBuilder()
      .setFramebufferAttachment(Path.FramebufferAttachment.newBuilder()
          .setAfter(command.getCommand())
          .setIndex(index)
          .setRenderSettings(settings)
          .setHints(hints))
        .build();
  }

  public static final Path.MeshOptions NODATA_MESH_OPTIONS = Path.MeshOptions.newBuilder()
      .setExcludeData(true)
      .build();

  public static Path.Any meshAfter(CommandIndex command, Path.MeshOptions options) {
    return Path.Any.newBuilder()
        .setMesh(Path.Mesh.newBuilder()
            .setCommandTreeNode(command.getNode())
            .setOptions(options))
        .build();
  }

  public static Path.Any meshAfter(
      CommandIndex command, Path.MeshOptions options, Vertex.BufferFormat format) {
    return Path.Any.newBuilder()
        .setAs(Path.As.newBuilder()
            .setMesh(Path.Mesh.newBuilder()
                .setCommandTreeNode(command.getNode())
                .setOptions(options))
            .setVertexBufferFormat(format))
        .build();
  }

  public static Path.Any commandField(Path.Command command, String field) {
    if (command == null || field == null) {
      return null;
    }
    return Path.Any.newBuilder()
        .setParameter(Path.Parameter.newBuilder()
            .setCommand(command)
            .setName(field))
        .build();
  }

  public static Path.Any commandResult(Path.Command command) {
    if (command == null) {
      return null;
    }
    return Path.Any.newBuilder()
        .setResult(Path.Result.newBuilder()
            .setCommand(command))
        .build();
  }

  public static Path.Any constantSet(Path.ConstantSet set) {
    return Path.Any.newBuilder()
        .setConstantSet(set)
        .build();
  }

  public static Path.Any imageInfo(Path.ImageInfo image) {
    return Path.Any.newBuilder()
        .setImageInfo(image)
        .build();
  }

  public static Path.Any resourceInfo(Path.ResourceData resource) {
    return Path.Any.newBuilder()
        .setResourceData(resource)
        .build();
  }

  public static Path.Any imageData(Path.ImageInfo image, Image.Format format) {
    return Path.Any.newBuilder()
        .setAs(Path.As.newBuilder()
            .setImageInfo(image)
            .setImageFormat(format))
        .build();
  }

  public static Path.Any imageData(Path.ResourceData resource, Image.Format format) {
    return Path.Any.newBuilder()
        .setAs(Path.As.newBuilder()
            .setResourceData(resource)
            .setImageFormat(format))
        .build();
  }

  public static Path.RenderSettings renderSettings(int maxWidth, int maxHeight, Path.DrawMode drawMode, boolean disableReplayOptiimization) {
    return Path.RenderSettings.newBuilder()
        .setMaxWidth(maxWidth)
        .setMaxHeight(maxHeight)
        .setDrawMode(drawMode)
        .setDisableReplayOptimization(disableReplayOptiimization)
        .build();
  }

  public static Path.Thumbnail thumbnail(Path.Command command, int size, boolean disableOpt) {
    return Path.Thumbnail.newBuilder()
        .setCommand(command)
        .setDesiredFormat(Images.FMT_RGBA_U8_NORM)
        .setDesiredMaxHeight(size)
        .setDesiredMaxWidth(size)
        .setDisableOptimization(disableOpt)
        .build();
  }

  public static Path.Thumbnail thumbnail(Path.CommandTreeNode node, int size, boolean disableOpt) {
    return Path.Thumbnail.newBuilder()
        .setCommandTreeNode(node)
        .setDesiredFormat(Images.FMT_RGBA_U8_NORM)
        .setDesiredMaxHeight(size)
        .setDesiredMaxWidth(size)
        .setDisableOptimization(disableOpt)
        .build();
  }

  public static Path.Thumbnail thumbnail(Path.ResourceData resource, int size, boolean disableOpt) {
    return Path.Thumbnail.newBuilder()
        .setResource(resource)
        .setDesiredFormat(Images.FMT_RGBA_U8_NORM)
        .setDesiredMaxHeight(size)
        .setDesiredMaxWidth(size)
        .setDisableOptimization(disableOpt)
        .build();
  }

  public static Path.Thumbnail thumbnail(CommandIndex command, int attachment, int size, boolean disableOpt) {
    return Path.Thumbnail.newBuilder()
        .setFramebufferAttachment(framebufferAttachmentAfter(command, attachment,
          renderSettings(size, size, Path.DrawMode.NORMAL, disableOpt),
          Path.UsageHints.newBuilder()
            .setPreview(true)
            .build()).getFramebufferAttachment())
        .setDesiredFormat(Images.FMT_RGBA_U8_NORM)
        .setDesiredMaxHeight(size)
        .setDesiredMaxWidth(size)
        .setDisableOptimization(disableOpt)
        .build();
  }

  public static Path.Any thumbnails(
      CommandIndex command, Path.ResourceType type, int size, boolean disableOpt) {
    return thumbnail(Path.Thumbnail.newBuilder()
        .setResources(Path.MultiResourceData.newBuilder()
            .setAfter(command.getCommand())
            .setAll(true)
            .setType(type))
        .setDesiredFormat(Images.FMT_RGBA_U8_NORM)
        .setDesiredMaxHeight(size)
        .setDesiredMaxWidth(size)
        .setDisableOptimization(disableOpt)
        .build());
  }

  public static Path.Any thumbnail(Path.Thumbnail thumb) {
    return Path.Any.newBuilder()
        .setThumbnail(thumb)
        .build();
  }

  public static Path.Any blob(Image.ID id) {
    return Path.Any.newBuilder()
        .setBlob(Path.Blob.newBuilder()
            .setID(Path.ID.newBuilder()
                .setData(id.getData())))
        .build();
  }

  public static Path.Any device(Path.Device device) {
    return Path.Any.newBuilder()
        .setDevice(device)
        .build();
  }

  public static Path.Any traceInfo(Path.Device device) {
    return Path.Any.newBuilder()
        .setTraceConfig(Path.DeviceTraceConfiguration.newBuilder()
            .setDevice(device))
        .build();
  }

  public static Path.Type type(long typeIndex, Path.API api) {
    return Path.Type.newBuilder()
        .setTypeIndex(typeIndex)
        .setAPI(api)
        .build();
  }

  public static Path.Any type(Path.Type type) {
    return Path.Any.newBuilder()
        .setType(type)
        .build();
  }

  public static boolean isNull(Path.Command c) {
    return (c == null) || (c.getIndicesCount() == 0);
  }

  /**
   * Compares a and b, returning -1 if a comes before b, 1 if b comes before a and 0 if they
   * are equal.
   */
  public static int compare(Path.Command a, Path.Command b) {
    if (isNull(a)) {
      return isNull(b) ? 0 : -1;
    } else if (isNull(b)) {
      return 1;
    }
    return compareCommands(a.getIndicesList(), b.getIndicesList(), false);
  }

  public static int compareCommands(List<Long> a, List<Long> b, boolean open) {
    for (int i = 0; i < a.size(); i++) {
      if (i >= b.size()) {
        return 1;
      }
      int r = Long.compare(a.get(i), b.get(i));
      if (r != 0) {
        return r;
      }
    }
    return open || (a.size() == b.size()) ? 0 : -1;
  }

  public static boolean contains(Path.Any path, Predicate<Object> predicate) {
    return find(path, predicate) != null;
  }

  public static Path.Any find(Path.Any path, Predicate<Object> predicate) {
    for (Object p = toNode(path); p != null; p = parentOf(p)) {
      if (predicate.test(p)) {
        return toAny(p);
      }
    }
    return null;
  }

  public static Path.GlobalState findGlobalState(Path.Any path) {
    return find(path, n -> n instanceof Path.GlobalState).getGlobalState();
  }

  /**
   * @return the unboxed path node from the {@link com.google.gapid.proto.service.path.Path.Any}.
   */
  public static Object toNode(Path.Any node) {
    return dispatch(node, TO_NODE_VISITOR, null);
  }

  /**
   * @return the path node boxed into a {@link com.google.gapid.proto.service.path.Path.Any}.
   */
  public static Path.Any toAny(Object node) {
    return dispatch(node, TO_ANY_VISITOR, null);
  }

  /**
   * @return the parent path node of the given path node.
   */
  public static Object parentOf(Object path) {
    return dispatch(path, GET_PARENT_VISITOR, null);
  }

  /**
   * @return a copy of the given the path node with its parent set to the given parent.
   */
  public static Object setParent(Object path, Object parent) {
    return dispatch(path, SET_PARENT_VISITOR, parent);
  }

  /**
   * @return the path as a string.
   */
  public static String toString(Object path) {
    return dispatch(path, PRINT_VISITOR, new StringBuilder()).toString();
  }

  /**
   * @return the path with the {@link com.google.gapid.proto.service.path.Path.GlobalState} ancestor
   * replaced with state. If there is no
   * {@link com.google.gapid.proto.service.path.Path.GlobalState} ancestor, then null is returned.
   */
  public static Path.Any reparent(Path.Any path, Path.GlobalState state) {
    LinkedList<Object> nodes = Lists.newLinkedList();
    boolean found = false;
    for (Object p = toNode(path); p != null; p = parentOf(p)) {
      if (p instanceof Path.GlobalState) {
        found = true;
        break;
      }
      nodes.addFirst(p);
    }
    if (!found) {
      return null;
    }
    Object head = state;
    for (Object node : nodes) {
      head = setParent(node, head);
    }
    return toAny(head);
  }

  public static class CommandFilter {
    public boolean showHostCommands;
    public boolean showSubmitInfoNodes;
    public boolean showSyncCommands;
    public boolean showBeginEndCommands;

    public CommandFilter(boolean showHostCommands, boolean showSubmitInfoNodes,
        boolean showSyncCommands, boolean showBeginEndCommands) {
      this.showHostCommands = showHostCommands;
      this.showSubmitInfoNodes = showSubmitInfoNodes;
      this.showSyncCommands = showSyncCommands;
      this.showBeginEndCommands = showBeginEndCommands;
    }

    public static CommandFilter fromSettings(Settings settings) {
      SettingsProto.UI.CommandFilter filter = settings.ui().getCommandFilter();
      return new CommandFilter(filter.getShowHostCommands(), filter.getShowSubmitInfoNodes(),
          filter.getShowSyncCommands(), filter.getShowBeginEndCommands());
    }

    public CommandFilter copy() {
      return new CommandFilter(
          showHostCommands, showSubmitInfoNodes, showSyncCommands, showBeginEndCommands);
    }

    public boolean update(CommandFilter from) {
      boolean changed =
          showHostCommands != from.showHostCommands ||
          showSubmitInfoNodes != from.showSubmitInfoNodes ||
          showSyncCommands != from.showSyncCommands ||
          showBeginEndCommands != from.showBeginEndCommands;
      if (changed) {
        showHostCommands = from.showHostCommands;
        showSubmitInfoNodes = from.showSubmitInfoNodes;
        showSyncCommands = from.showSyncCommands;
        showBeginEndCommands = from.showBeginEndCommands;
      }
      return changed;
    }

    public Path.CommandTree.Builder toProto(Path.CommandTree.Builder path) {
      return path
          .setFilter(Path.CommandFilter.newBuilder()
              .setSuppressHostCommands(!showHostCommands)
              .setSuppressDeviceSideSyncCommands(!showSyncCommands)
              .setSuppressBeginEndMarkers(!showBeginEndCommands))
          .setSuppressSubmitInfoNodes(!showSubmitInfoNodes);
    }

    public void save(Settings settings) {
      SettingsProto.UI.CommandFilter.Builder filter = settings.writeUi().getCommandFilterBuilder();
      filter.setShowHostCommands(showHostCommands);
      filter.setShowSubmitInfoNodes(showSubmitInfoNodes);
      filter.setShowSyncCommands(showSyncCommands);
      filter.setShowBeginEndCommands(showBeginEndCommands);
    }
  }

  /**
   * Visitor is the interface implemented by types that operate each of the path types.
   * @param <R> the type of the result value.
   * @param <A> the type of the argument value.
   */
  private interface Visitor<R, A> {
    R visit(Image.ID path, A arg);
    R visit(Path.API path, A arg);
    R visit(Path.ArrayIndex path, A arg);
    R visit(Path.As path, A arg);
    R visit(Path.Blob path, A arg);
    R visit(Path.Capture path, A arg);
    R visit(Path.ConstantSet path, A arg);
    R visit(Path.Command path, A arg);
    R visit(Path.Commands path, A arg);
    R visit(Path.CommandTree path, A arg);
    R visit(Path.CommandTreeNode path, A arg);
    R visit(Path.CommandTreeNodeForCommand path, A arg);
    R visit(Path.Device path, A arg);
    R visit(Path.DeviceTraceConfiguration path, A arg);
    R visit(Path.FramebufferObservation path, A arg);
    R visit(Path.Field path, A arg);
    R visit(Path.GlobalState path, A arg);
    R visit(Path.ID path, A arg);
    R visit(Path.ImageInfo path, A arg);
    R visit(Path.MapIndex path, A arg);
    R visit(Path.Memory path, A arg);
    R visit(Path.Mesh path, A arg);
    R visit(Path.Parameter path, A arg);
    R visit(Path.Report path, A arg);
    R visit(Path.ResourceData path, A arg);
    R visit(Path.Resources path, A arg);
    R visit(Path.Result path, A arg);
    R visit(Path.Slice path, A arg);
    R visit(Path.State path, A arg);
    R visit(Path.StateTree path, A arg);
    R visit(Path.StateTreeNode path, A arg);
    R visit(Path.StateTreeNodeForPath path, A arg);
    R visit(Path.Thumbnail path, A arg);
  }

  /**
   * Unboxes the path node from the {@link com.google.gapid.proto.service.path.Path.Any} and
   * dispatches the node to the visitor. Throws an exception if the path is not an expected type.
   */
  private static <T, A> T dispatchAny(Path.Any path, Visitor<T, A> visitor, A arg) {
    switch (path.getPathCase()) {
      case API:
        return visitor.visit(path.getAPI(), arg);
      case ARRAY_INDEX:
        return visitor.visit(path.getArrayIndex(), arg);
      case AS:
        return visitor.visit(path.getAs(), arg);
      case BLOB:
        return visitor.visit(path.getBlob(), arg);
      case CAPTURE:
        return visitor.visit(path.getCapture(), arg);
      case CONSTANT_SET:
        return visitor.visit(path.getConstantSet(), arg);
      case COMMAND:
        return visitor.visit(path.getCommand(), arg);
      case COMMANDS:
        return visitor.visit(path.getCommands(), arg);
      case COMMAND_TREE:
        return visitor.visit(path.getCommandTree(), arg);
      case COMMAND_TREE_NODE:
        return visitor.visit(path.getCommandTreeNode(), arg);
      case COMMAND_TREE_NODE_FOR_COMMAND:
        return visitor.visit(path.getCommandTreeNodeForCommand(), arg);
      case DEVICE:
        return visitor.visit(path.getDevice(), arg);
      case TRACECONFIG:
        return visitor.visit(path.getTraceConfig(), arg);
      case FBO:
        return visitor.visit(path.getFBO(), arg);
      case FIELD:
        return visitor.visit(path.getField(), arg);
      case GLOBAL_STATE:
        return visitor.visit(path.getGlobalState(), arg);
      case IMAGE_INFO:
        return visitor.visit(path.getImageInfo(), arg);
      case MAP_INDEX:
        return visitor.visit(path.getMapIndex(), arg);
      case MEMORY:
        return visitor.visit(path.getMemory(), arg);
      case MESH:
        return visitor.visit(path.getMesh(), arg);
      case PARAMETER:
        return visitor.visit(path.getParameter(), arg);
      case REPORT:
        return visitor.visit(path.getReport(), arg);
      case RESOURCE_DATA:
        return visitor.visit(path.getResourceData(), arg);
      case RESOURCES:
        return visitor.visit(path.getResources(), arg);
      case RESULT:
        return visitor.visit(path.getResult(), arg);
      case SLICE:
        return visitor.visit(path.getSlice(), arg);
      case STATE:
        return visitor.visit(path.getState(), arg);
      case STATE_TREE:
        return visitor.visit(path.getStateTree(), arg);
      case STATE_TREE_NODE:
        return visitor.visit(path.getStateTreeNode(), arg);
      case STATE_TREE_NODE_FOR_PATH:
        return visitor.visit(path.getStateTreeNodeForPath(), arg);
      case THUMBNAIL:
        return visitor.visit(path.getThumbnail(), arg);
      default:
        throw new RuntimeException("Unexpected path case: " + path.getPathCase());
    }
  }

  /**
   * Dispatches the path node to the visitor.
   * Throws an exception if the path is not an expected type.
   */
  protected static <T, A> T dispatch(Object path, Visitor<T, A> visitor, A arg) {
    if (path instanceof Path.Any) {
      return dispatchAny((Path.Any)path, visitor, arg);
    } else if (path instanceof Image.ID) {
      return visitor.visit((Image.ID)path, arg);
    } else if (path instanceof Path.API) {
      return visitor.visit((Path.API)path, arg);
    } else if (path instanceof Path.ArrayIndex) {
      return visitor.visit((Path.ArrayIndex)path, arg);
    } else if (path instanceof Path.As) {
      return visitor.visit((Path.As)path, arg);
    } else if (path instanceof Path.Blob) {
      return visitor.visit((Path.Blob)path, arg);
    } else if (path instanceof Path.Capture) {
      return visitor.visit((Path.Capture)path, arg);
    } else if (path instanceof Path.ConstantSet) {
      return visitor.visit((Path.ConstantSet)path, arg);
    } else if (path instanceof Path.Command) {
      return visitor.visit((Path.Command)path, arg);
    } else if (path instanceof Path.Commands) {
      return visitor.visit((Path.Commands)path, arg);
    } else if (path instanceof Path.CommandTree) {
      return visitor.visit((Path.CommandTree)path, arg);
    } else if (path instanceof Path.CommandTreeNode) {
      return visitor.visit((Path.CommandTreeNode)path, arg);
    } else if (path instanceof Path.CommandTreeNodeForCommand) {
      return visitor.visit((Path.CommandTreeNodeForCommand)path, arg);
    } else if (path instanceof Path.Device) {
      return visitor.visit((Path.Device)path, arg);
    } else if (path instanceof Path.DeviceTraceConfiguration) {
      return visitor.visit((Path.DeviceTraceConfiguration)path, arg);
    } else if (path instanceof Path.FramebufferObservation) {
      return visitor.visit((Path.FramebufferObservation)path, arg);
    } else if (path instanceof Path.Field) {
      return visitor.visit((Path.Field)path, arg);
    } else if (path instanceof Path.GlobalState) {
      return visitor.visit((Path.GlobalState)path, arg);
    } else if (path instanceof Path.ID) {
      return visitor.visit((Path.ID)path, arg);
    } else if (path instanceof Path.ImageInfo) {
      return visitor.visit((Path.ImageInfo)path, arg);
    } else if (path instanceof Path.MapIndex) {
      return visitor.visit((Path.MapIndex)path, arg);
    } else if (path instanceof Path.Memory) {
      return visitor.visit((Path.Memory)path, arg);
    } else if (path instanceof Path.Mesh) {
      return visitor.visit((Path.Mesh)path, arg);
    } else if (path instanceof Path.Parameter) {
      return visitor.visit((Path.Parameter)path, arg);
    } else if (path instanceof Path.Report) {
      return visitor.visit((Path.Report)path, arg);
    } else if (path instanceof Path.ResourceData) {
      return visitor.visit((Path.ResourceData)path, arg);
    } else if (path instanceof Path.Resources) {
      return visitor.visit((Path.Resources)path, arg);
    } else if (path instanceof Path.Result) {
      return visitor.visit((Path.Result)path, arg);
    } else if (path instanceof Path.Slice) {
      return visitor.visit((Path.Slice)path, arg);
    } else if (path instanceof Path.State) {
      return visitor.visit((Path.State)path, arg);
    } else if (path instanceof Path.StateTree) {
      return visitor.visit((Path.StateTree)path, arg);
    } else if (path instanceof Path.StateTreeNode) {
      return visitor.visit((Path.StateTreeNode)path, arg);
    } else if (path instanceof Path.StateTreeNodeForPath) {
      return visitor.visit((Path.StateTreeNodeForPath)path, arg);
    } else if (path instanceof Path.Thumbnail) {
      return visitor.visit((Path.Thumbnail)path, arg);
    } else if (path instanceof Message.Builder) {
      return dispatch(((Message.Builder)path).build(), visitor, arg);
    } else {
      throw new RuntimeException("Unexpected path type: " + path.getClass().getName());
    }
  }

  /**
   * {@link Visitor} that simply returns the node type. Used by {@link #toNode(Path.Any)}.
   */
  private static final Visitor<Object, Void> TO_NODE_VISITOR = new Visitor<Object, Void>() {
    @Override public Object visit(Image.ID path, Void ignored) { return path; }
    @Override public Object visit(Path.API path, Void ignored) { return path; }
    @Override public Object visit(Path.ArrayIndex path, Void ignored) { return path; }
    @Override public Object visit(Path.As path, Void ignored) { return path; }
    @Override public Object visit(Path.Blob path, Void ignored) { return path; }
    @Override public Object visit(Path.Capture path, Void ignored) { return path; }
    @Override public Object visit(Path.ConstantSet path, Void ignored) { return path; }
    @Override public Object visit(Path.Command path, Void ignored) { return path; }
    @Override public Object visit(Path.Commands path, Void ignored) { return path; }
    @Override public Object visit(Path.CommandTree path, Void ignored) { return path; }
    @Override public Object visit(Path.CommandTreeNode path, Void ignored) { return path; }
    @Override public Object visit(Path.CommandTreeNodeForCommand path, Void ignored) { return path; }
    @Override public Object visit(Path.Device path, Void ignored) { return path; }
    @Override public Object visit(Path.DeviceTraceConfiguration path, Void ignored) { return path; }
    @Override public Object visit(Path.FramebufferObservation path, Void ignored) { return path; }
    @Override public Object visit(Path.Field path, Void ignored) { return path; }
    @Override public Object visit(Path.GlobalState path, Void ignored) { return path; }
    @Override public Object visit(Path.ID path, Void ignored) { return path; }
    @Override public Object visit(Path.ImageInfo path, Void ignored) { return path; }
    @Override public Object visit(Path.MapIndex path, Void ignored) { return path; }
    @Override public Object visit(Path.Memory path, Void ignored) { return path; }
    @Override public Object visit(Path.Mesh path, Void ignored) { return path; }
    @Override public Object visit(Path.Parameter path, Void ignored) { return path; }
    @Override public Object visit(Path.Report path, Void ignored) { return path; }
    @Override public Object visit(Path.ResourceData path, Void ignored) { return path; }
    @Override public Object visit(Path.Resources path, Void ignored) { return path; }
    @Override public Object visit(Path.Result path, Void ignored) { return path; }
    @Override public Object visit(Path.Slice path, Void ignored) { return path; }
    @Override public Object visit(Path.State path, Void ignored) { return path; }
    @Override public Object visit(Path.StateTree path, Void ignored) { return path; }
    @Override public Object visit(Path.StateTreeNode path, Void ignored) { return path; }
    @Override public Object visit(Path.StateTreeNodeForPath path, Void ignored) { return path; }
    @Override public Object visit(Path.Thumbnail path, Void ignored) { return path; }
  };

  /**
   * {@link Visitor} that returns the passed node type boxed in a
   * {@link com.google.gapid.proto.service.path.Path.Any}. Used by {@link #toAny(Object)}.
   */
  private static final Visitor<Path.Any, Void> TO_ANY_VISITOR = new Visitor<Path.Any, Void>() {
    @Override
    public Path.Any visit(Image.ID path, Void ignored) {
      throw new RuntimeException("Image.ID cannot be stored in a Path.Any");
    }

    @Override
    public Path.Any visit(Path.API path, Void ignored) {
      return Path.Any.newBuilder().setAPI(path).build();
    }

    @Override
    public Path.Any visit(Path.ArrayIndex path, Void ignored) {
      return Path.Any.newBuilder().setArrayIndex(path).build();
    }

    @Override
    public Path.Any visit(Path.As path, Void ignored) {
      return Path.Any.newBuilder().setAs(path).build();
    }

    @Override
    public Path.Any visit(Path.Blob path, Void ignored) {
      return Path.Any.newBuilder().setBlob(path).build();
    }

    @Override
    public Path.Any visit(Path.Capture path, Void ignored) {
      return Path.Any.newBuilder().setCapture(path).build();
    }

    @Override
    public Path.Any visit(Path.ConstantSet path, Void ignored) {
      return Path.Any.newBuilder().setConstantSet(path).build();
    }

    @Override
    public Path.Any visit(Path.Command path, Void ignored) {
      return Path.Any.newBuilder().setCommand(path).build();
    }

    @Override
    public Path.Any visit(Path.Commands path, Void ignored) {
      return Path.Any.newBuilder().setCommands(path).build();
    }

    @Override
    public Path.Any visit(Path.CommandTree path, Void ignored) {
      return Path.Any.newBuilder().setCommandTree(path).build();
    }

    @Override
    public Path.Any visit(Path.CommandTreeNode path, Void ignored) {
      return Path.Any.newBuilder().setCommandTreeNode(path).build();
    }

    @Override
    public Path.Any visit(Path.CommandTreeNodeForCommand path, Void ignored) {
      return Path.Any.newBuilder().setCommandTreeNodeForCommand(path).build();
    }

    @Override
    public Path.Any visit(Path.Device path, Void ignored) {
      return Path.Any.newBuilder().setDevice(path).build();
    }

    @Override
    public Path.Any visit(Path.DeviceTraceConfiguration path, Void ignored) {
      return Path.Any.newBuilder().setTraceConfig(path).build();
    }

    @Override
    public Path.Any visit(Path.FramebufferObservation path, Void ignored) {
      return Path.Any.newBuilder().setFBO(path).build();
    }

    @Override
    public Path.Any visit(Path.Field path, Void ignored) {
      return Path.Any.newBuilder().setField(path).build();
    }

    @Override
    public Path.Any visit(Path.GlobalState path, Void ignored) {
      return Path.Any.newBuilder().setGlobalState(path).build();
    }

    @Override
    public Path.Any visit(Path.ID path, Void ignored) {
      throw new RuntimeException("Path.ID cannot be stored in a Path.Any");
    }

    @Override
    public Path.Any visit(Path.ImageInfo path, Void ignored) {
      return Path.Any.newBuilder().setImageInfo(path).build();
    }

    @Override
    public Path.Any visit(Path.MapIndex path, Void ignored) {
      return Path.Any.newBuilder().setMapIndex(path).build();
    }

    @Override
    public Path.Any visit(Path.Memory path, Void ignored) {
      return Path.Any.newBuilder().setMemory(path).build();
    }

    @Override
    public Path.Any visit(Path.Mesh path, Void ignored) {
      return Path.Any.newBuilder().setMesh(path).build();
    }

    @Override
    public Path.Any visit(Path.Parameter path, Void ignored) {
      return Path.Any.newBuilder().setParameter(path).build();
    }

    @Override
    public Path.Any visit(Path.Report path, Void ignored) {
      return Path.Any.newBuilder().setReport(path).build();
    }

    @Override
    public Path.Any visit(Path.ResourceData path, Void ignored) {
      return Path.Any.newBuilder().setResourceData(path).build();
    }

    @Override
    public Path.Any visit(Path.Resources path, Void ignored) {
      return Path.Any.newBuilder().setResources(path).build();
    }

    @Override
    public Path.Any visit(Path.Result path, Void ignored) {
      return Path.Any.newBuilder().setResult(path).build();
    }

    @Override
    public Path.Any visit(Path.Slice path, Void ignored) {
      return Path.Any.newBuilder().setSlice(path).build();
    }

    @Override
    public Path.Any visit(Path.State path, Void ignored) {
      return Path.Any.newBuilder().setState(path).build();
    }

    @Override
    public Path.Any visit(Path.StateTree path, Void ignored) {
      return Path.Any.newBuilder().setStateTree(path).build();
    }

    @Override
    public Path.Any visit(Path.StateTreeNode path, Void ignored) {
      return Path.Any.newBuilder().setStateTreeNode(path).build();
    }

    @Override
    public Path.Any visit(Path.StateTreeNodeForPath path, Void ignored) {
      return Path.Any.newBuilder().setStateTreeNodeForPath(path).build();
    }

    @Override
    public Path.Any visit(Path.Thumbnail path, Void ignored) {
      return Path.Any.newBuilder().setThumbnail(path).build();
    }
  };

  /**
   * {@link Visitor} that returns the parent node of the given path node.
   * Used by {@link #parentOf(Object)}.
   */
  private static final Visitor<Object, Void> GET_PARENT_VISITOR = new Visitor<Object, Void>() {
    @Override
    public Object visit(Image.ID path, Void ignored) {
      return null;
    }

    @Override
    public Object visit(Path.API path, Void ignored) {
      return null;
    }

    @Override
    public Object visit(Path.ArrayIndex path, Void ignored) {
      switch (path.getArrayCase()) {
        case FIELD:
          return path.getField();
        case ARRAY_INDEX:
          return path.getArrayIndex();
        case SLICE:
          return path.getSlice();
        case MAP_INDEX:
          return path.getMapIndex();
        default:
          return null;
      }
    }

    @Override
    public Object visit(Path.As path, Void ignored) {
      switch (path.getFromCase()) {
        case FIELD:
          return path.getField();
        case SLICE:
          return path.getSlice();
        case ARRAY_INDEX:
          return path.getArrayIndex();
        case MAP_INDEX:
          return path.getMapIndex();
        case IMAGE_INFO:
          return path.getImageInfo();
        case RESOURCE_DATA:
          return path.getResourceData();
        case MESH:
          return path.getMesh();
        default:
          return null;
      }
    }

    @Override
    public Object visit(Path.Blob path, Void ignored) {
      return null;
    }

    @Override
    public Object visit(Path.Capture path, Void ignored) {
      return null;
    }

    @Override
    public Object visit(Path.ConstantSet path, Void ignored) {
      return path.getAPI();
    }

    @Override
    public Object visit(Path.Command path, Void ignored) {
      return path.getCapture();
    }

    @Override
    public Object visit(Path.Commands path, Void ignored) {
      return path.getCapture();
    }

    @Override
    public Object visit(Path.CommandTree path, Void ignored) {
      return path.getCapture();
    }

    @Override
    public Object visit(Path.CommandTreeNode path, Void ignored) {
      return null;
    }

    @Override
    public Object visit(Path.CommandTreeNodeForCommand path, Void ignored) {
      return path.getCommand();
    }

    @Override
    public Object visit(Path.Device path, Void ignored) {
      return null;
    }

    @Override
    public Object visit(Path.DeviceTraceConfiguration path, Void ignored) {
      return path.getDevice();
    }

    @Override
    public Object visit(Path.FramebufferObservation path, Void ignored) {
      return path.getCommand();
    }

    @Override
    public Object visit(Path.Field path, Void ignored) {
      switch (path.getStructCase()) {
        case STATE:
          return path.getState();
        case GLOBAL_STATE:
          return path.getGlobalState();
        case FIELD:
          return path.getField();
        case ARRAY_INDEX:
          return path.getArrayIndex();
        case SLICE:
          return path.getSlice();
        case MAP_INDEX:
          return path.getMapIndex();
        default:
          return null;
      }
    }

    @Override
    public Object visit(Path.GlobalState path, Void ignored) {
      return path.getAfter();
    }

    @Override
    public Object visit(Path.ID path, Void ignored) {
      return null;
    }

    @Override
    public Object visit(Path.ImageInfo path, Void ignored) {
      return null;
    }

    @Override
    public Object visit(Path.MapIndex path, Void ignored) {
      switch (path.getMapCase()) {
        case STATE:
          return path.getState();
        case FIELD:
          return path.getField();
        case ARRAY_INDEX:
          return path.getArrayIndex();
        case SLICE:
          return path.getSlice();
        case MAP_INDEX:
          return path.getMapIndex();
        default:
          return null;
      }
    }

    @Override
    public Object visit(Path.Memory path, Void ignored) {
      return path.getAfter();
    }

    @Override
    public Object visit(Path.Mesh path, Void ignored) {
      switch (path.getObjectCase()) {
        case COMMAND:
          return path.getCommand();
        case COMMAND_TREE_NODE:
          return path.getCommandTreeNode();
        default:
          return null;
      }
    }

    @Override
    public Object visit(Path.Parameter path, Void ignored) {
      return path.getCommand();
    }

    @Override
    public Object visit(Path.Report path, Void ignored) {
      return path.getCapture();
    }

    @Override
    public Object visit(Path.ResourceData path, Void ignored) {
      return path.getAfter();
    }

    @Override
    public Object visit(Path.Resources path, Void ignored) {
      return path.getCapture();
    }

    @Override
    public Object visit(Path.Result path, Void ignored) {
      return path.getCommand();
    }

    @Override
    public Object visit(Path.Slice path, Void ignored) {
      switch (path.getArrayCase()) {
        case FIELD:
          return path.getField();
        case ARRAY_INDEX:
          return path.getArrayIndex();
        case SLICE:
          return path.getSlice();
        case MAP_INDEX:
          return path.getMapIndex();
        default:
          return null;
      }
    }

    @Override
    public Object visit(Path.State path, Void ignored) {
      return path.getAfter();
    }

    @Override
    public Object visit(Path.StateTree path, Void ignored) {
      return path.getState();
    }

    @Override
    public Object visit(Path.StateTreeNode path, Void ignored) {
      return null;
    }

    @Override
    public Object visit(Path.StateTreeNodeForPath path, Void ignored) {
      return null;
    }

    @Override
    public Object visit(Path.Thumbnail path, Void ignored) {
      switch (path.getObjectCase()) {
        case RESOURCE:
          return path.getResource();
        case COMMAND:
          return path.getCommand();
        case COMMAND_TREE_NODE:
          return path.getCommandTreeNode();
        default:
          return null;
      }
    }
  };

  /**
   * {@link Visitor} that returns the a copy of the provided path node, but with the parent changed
   * to the specified parent node.
   * Used by {@link #setParent(Object, Object)}.
   */
  private static final Visitor<Object, Object> SET_PARENT_VISITOR = new Visitor<Object, Object>() {
    @Override
    public Object visit(Image.ID path, Object parent) {
      throw new RuntimeException("Image.ID has no parent to set");
    }

    @Override
    public Object visit(Path.API path, Object parent) {
      throw new RuntimeException("Path.API has no parent to set");
    }

    @Override
    public Object visit(Path.ArrayIndex path, Object parent) {
      if (parent instanceof Path.Field) {
        return path.toBuilder().setField((Path.Field) parent).build();
      } else if (parent instanceof Path.ArrayIndex) {
        return path.toBuilder().setArrayIndex((Path.ArrayIndex) parent).build();
      } else if (parent instanceof Path.Slice) {
        return path.toBuilder().setSlice((Path.Slice) parent).build();
      } else if (parent instanceof Path.MapIndex) {
        return path.toBuilder().setMapIndex((Path.MapIndex) parent).build();
      } else {
        throw new RuntimeException("Path.ArrayIndex cannot set parent to " + parent.getClass().getName());
      }
    }

    @Override
    public Object visit(Path.As path, Object parent) {
      if (parent instanceof Path.Field) {
        return path.toBuilder().setField((Path.Field) parent).build();
      } else if (parent instanceof Path.ArrayIndex) {
        return path.toBuilder().setArrayIndex((Path.ArrayIndex) parent).build();
      } else if (parent instanceof Path.Slice) {
        return path.toBuilder().setSlice((Path.Slice) parent).build();
      } else if (parent instanceof Path.MapIndex) {
        return path.toBuilder().setMapIndex((Path.MapIndex) parent).build();
      } else if (parent instanceof Path.ImageInfo) {
        return path.toBuilder().setImageInfo((Path.ImageInfo) parent).build();
      } else if (parent instanceof Path.ResourceData) {
        return path.toBuilder().setResourceData((Path.ResourceData) parent).build();
      } else if (parent instanceof Path.Mesh) {
        return path.toBuilder().setMesh((Path.Mesh) parent).build();
      } else {
        throw new RuntimeException("Path.As cannot set parent to " + parent.getClass().getName());
      }
    }

    @Override
    public Object visit(Path.DeviceTraceConfiguration path, Object parent) {
      if (!(parent instanceof Path.Device)) {
        throw new RuntimeException("Path.DeviceTraceConfiguration cannot set parent to " + parent.getClass().getName());
      }
      return path.toBuilder().setDevice((Path.Device) parent).build();
    }

    @Override
    public Object visit(Path.Blob path, Object parent) {
      throw new RuntimeException("Path.Blob has no parent to set");
    }

    @Override
    public Object visit(Path.Capture path, Object parent) {
      throw new RuntimeException("Path.Capture has no parent to set");
    }

    @Override
    public Object visit(Path.ConstantSet path, Object parent) {
      if (parent instanceof Path.API) {
        return path.toBuilder().setAPI((Path.API) parent).build();
      } else {
        throw new RuntimeException("Path.ConstantSet cannot set parent to " + parent.getClass().getName());
      }
    }

    @Override
    public Object visit(Path.Command path, Object parent) {
      if (parent instanceof Path.Capture) {
        return path.toBuilder().setCapture((Path.Capture) parent).build();
      } else {
        throw new RuntimeException("Path.Command cannot set parent to " + parent.getClass().getName());
      }
    }

    @Override
    public Object visit(Path.Commands path, Object parent) {
      if (parent instanceof Path.Capture) {
        return path.toBuilder().setCapture((Path.Capture) parent).build();
      } else {
        throw new RuntimeException("Path.Commands cannot set parent to " + parent.getClass().getName());
      }
    }

    @Override
    public Object visit(Path.CommandTree path, Object parent) {
      if (parent instanceof Path.Capture) {
        return path.toBuilder().setCapture((Path.Capture) parent).build();
      } else {
        throw new RuntimeException("Path.CommandTree cannot set parent to " + parent.getClass().getName());
      }
    }

    @Override
    public Object visit(Path.CommandTreeNode path, Object parent) {
      throw new RuntimeException("Path.CommandTreeNode has no parent to set");
    }

    @Override
    public Object visit(Path.CommandTreeNodeForCommand path, Object parent) {
      if (parent instanceof Path.Command) {
        return path.toBuilder().setCommand((Path.Command) parent).build();
      } else {
        throw new RuntimeException("Path.CommandTreeNodeForCommand cannot set parent to " + parent.getClass().getName());
      }
    }

    @Override
    public Object visit(Path.Device path, Object parent) {
      throw new RuntimeException("Path.Device has no parent to set");
    }

    @Override
    public Object visit(Path.FramebufferObservation path, Object parent) {
      if (parent instanceof Path.Command) {
        return path.toBuilder().setCommand((Path.Command) parent).build();
      } else {
        throw new RuntimeException("Path.FramebufferObservation cannot set parent to " + parent.getClass().getName());
      }
    }

    @Override
    public Object visit(Path.Field path, Object parent) {
      if (parent instanceof Path.State) {
        return path.toBuilder().setState((Path.State) parent).build();
      } else if (parent instanceof Path.GlobalState) {
        return path.toBuilder().setGlobalState((Path.GlobalState) parent).build();
      } else if (parent instanceof Path.Field) {
        return path.toBuilder().setField((Path.Field) parent).build();
      } else if (parent instanceof Path.ArrayIndex) {
        return path.toBuilder().setArrayIndex((Path.ArrayIndex) parent).build();
      } else if (parent instanceof Path.Slice) {
        return path.toBuilder().setSlice((Path.Slice) parent).build();
      } else if (parent instanceof Path.MapIndex) {
        return path.toBuilder().setMapIndex((Path.MapIndex) parent).build();
      } else {
        throw new RuntimeException("Path.Field cannot set parent to " + parent.getClass().getName());
      }
    }

    @Override
    public Object visit(Path.GlobalState path, Object parent) {
      if (parent instanceof Path.Command) {
        return path.toBuilder().setAfter((Path.Command) parent).build();
      } else {
        throw new RuntimeException("Path.GlobalState cannot set parent to " + parent.getClass().getName());
      }
    }

    @Override
    public Object visit(Path.ID path, Object parent) {
      throw new RuntimeException("Path.ID has no parent to set");
    }

    @Override
    public Object visit(Path.ImageInfo path, Object parent) {
      throw new RuntimeException("Path.ImageInfo has no parent to set");
    }

    @Override
    public Object visit(Path.MapIndex path, Object parent) {
      if (parent instanceof Path.State) {
        return path.toBuilder().setState((Path.State) parent).build();
      } else if (parent instanceof Path.Field) {
        return path.toBuilder().setField((Path.Field) parent).build();
      } else if (parent instanceof Path.ArrayIndex) {
        return path.toBuilder().setArrayIndex((Path.ArrayIndex) parent).build();
      } else if (parent instanceof Path.Slice) {
        return path.toBuilder().setSlice((Path.Slice) parent).build();
      } else if (parent instanceof Path.MapIndex) {
        return path.toBuilder().setMapIndex((Path.MapIndex) parent).build();
      } else {
        throw new RuntimeException("Path.MapIndex cannot set parent to " + parent.getClass().getName());
      }
    }

    @Override
    public Object visit(Path.Memory path, Object parent) {
      if (parent instanceof Path.Command) {
        return path.toBuilder().setAfter((Path.Command) parent).build();
      } else {
        throw new RuntimeException("Path.Memory cannot set parent to " + parent.getClass().getName());
      }
    }

    @Override
    public Object visit(Path.Mesh path, Object parent) {
      if (parent instanceof Path.Command) {
        return path.toBuilder().setCommand((Path.Command) parent).build();
      } else if (parent instanceof Path.CommandTreeNode) {
        return path.toBuilder().setCommandTreeNode((Path.CommandTreeNode) parent).build();
      } else {
        throw new RuntimeException("Path.Mesh cannot set parent to " + parent.getClass().getName());
      }
    }

    @Override
    public Object visit(Path.Parameter path, Object parent) {
      if (parent instanceof Path.Command) {
        return path.toBuilder().setCommand((Path.Command) parent).build();
      } else {
        throw new RuntimeException("Path.Parameter cannot set parent to " + parent.getClass().getName());
      }
    }

    @Override
    public Object visit(Path.Report path, Object parent) {
      if (parent instanceof Path.Capture) {
        return path.toBuilder().setCapture((Path.Capture) parent).build();
      } else {
        throw new RuntimeException("Path.Report cannot set parent to " + parent.getClass().getName());
      }
    }

    @Override
    public Object visit(Path.ResourceData path, Object parent) {
      if (parent instanceof Path.Command) {
        return path.toBuilder().setAfter((Path.Command) parent).build();
      } else {
        throw new RuntimeException("Path.ResourceData cannot set parent to " + parent.getClass().getName());
      }
    }

    @Override
    public Object visit(Path.Resources path, Object parent) {
      if (parent instanceof Path.Capture) {
        return path.toBuilder().setCapture((Path.Capture) parent).build();
      } else {
        throw new RuntimeException("Path.Resources cannot set parent to " + parent.getClass().getName());
      }
    }

    @Override
    public Object visit(Path.Result path, Object parent) {
      if (parent instanceof Path.Command) {
        return path.toBuilder().setCommand((Path.Command) parent).build();
      } else {
        throw new RuntimeException("Path.Result cannot set parent to " + parent.getClass().getName());
      }
    }

    @Override
    public Object visit(Path.Slice path, Object parent) {
      if (parent instanceof Path.Field) {
        return path.toBuilder().setField((Path.Field) parent).build();
      } else if (parent instanceof Path.ArrayIndex) {
        return path.toBuilder().setArrayIndex((Path.ArrayIndex) parent).build();
      } else if (parent instanceof Path.Slice) {
        return path.toBuilder().setSlice((Path.Slice) parent).build();
      } else if (parent instanceof Path.MapIndex) {
        return path.toBuilder().setMapIndex((Path.MapIndex) parent).build();
      } else {
        throw new RuntimeException("Path.Slice cannot set parent to " + parent.getClass().getName());
      }
    }

    @Override
    public Object visit(Path.State path, Object parent) {
      if (parent instanceof Path.Command) {
        return path.toBuilder().setAfter((Path.Command) parent).build();
      } else {
        throw new RuntimeException("Path.State cannot set parent to " + parent.getClass().getName());
      }
    }

    @Override
    public Object visit(Path.StateTree path, Object parent) {
      if (parent instanceof Path.State) {
        return path.toBuilder().setState((Path.State) parent).build();
      } else {
        throw new RuntimeException("Path.StateTree cannot set parent to " + parent.getClass().getName());
      }
    }

    @Override
    public Object visit(Path.StateTreeNode path, Object parent) {
      throw new RuntimeException("Path.StateTreeNode has no parent to set");
    }

    @Override
    public Object visit(Path.StateTreeNodeForPath path, Object parent) {
      throw new RuntimeException("Path.StateTreeNodeForPath has no parent to set");
    }

    @Override
    public Object visit(Path.Thumbnail path, Object parent) {
      if (parent instanceof Path.ResourceData) {
        return path.toBuilder().setResource((Path.ResourceData) parent).build();
      } else if (parent instanceof Path.Command) {
        return path.toBuilder().setCommand((Path.Command) parent).build();
      } else if (parent instanceof Path.CommandTreeNode) {
        return path.toBuilder().setCommandTreeNode((Path.CommandTreeNode) parent).build();
      } else {
        throw new RuntimeException("Path.Thumbnail cannot set parent to " + parent.getClass().getName());
      }
    }
  };

  /**
   * {@link Visitor} that prints the path to the provided {@link StringBuilder}, then returns that
   * {@link StringBuilder}.
   * Used by {@link #toString(Object)}.
   */
  private static final Visitor<StringBuilder, StringBuilder> PRINT_VISITOR = new Visitor<StringBuilder, StringBuilder>() {
    @Override
    public StringBuilder visit(Image.ID path, StringBuilder sb) {
      return sb.append(ProtoDebugTextFormat.shortDebugString(path));
    }

    @Override
    public StringBuilder visit(Path.API path, StringBuilder sb) {
      sb.append("API{");
      visit(path.getID(), sb);
      sb.append("}");
      return sb;
    }

    @Override
    public StringBuilder visit(Path.ArrayIndex path, StringBuilder sb) {
      return dispatch(parentOf(path), this, sb)
          .append("[")
          .append(UnsignedLongs.toString(path.getIndex()))
          .append("]");
    }

    @Override
    public StringBuilder visit(Path.As path, StringBuilder sb) {
      dispatch(parentOf(path), this, sb);
      switch (path.getToCase()) {
        case IMAGE_FORMAT:
          sb.append(".as(");
          sb.append(path.getImageFormat().getName()); // TODO
          sb.append(")");
          break;
        case VERTEX_BUFFER_FORMAT:
          sb.append(".as(VBF)"); // TODO
          break;
        default:
          sb.append(".as(??)");
      }
      return sb;
    }

    @Override
    public StringBuilder visit(Path.Blob path, StringBuilder sb) {
      sb.append("blob{");
      visit(path.getID(), sb);
      sb.append("}");
      return sb;
    }

    @Override
    public StringBuilder visit(Path.Capture path, StringBuilder sb) {
      sb.append("capture{");
      visit(path.getID(), sb);
      sb.append("}");
      return sb;
    }

    @Override
    public StringBuilder visit(Path.ConstantSet path, StringBuilder sb) {
      return visit(path.getAPI(), sb)
          .append(".constants[")
          .append(path.getIndex())
          .append("]");
    }

    @Override
    public StringBuilder visit(Path.Command path, StringBuilder sb) {
      return visit(path.getCapture(), sb)
          .append(".command[")
          .append(Formatter.commandIndex(path))
          .append("]");
    }

    @Override
    public StringBuilder visit(Path.Commands path, StringBuilder sb) {
      return visit(path.getCapture(), sb)
          .append(".command[")
          .append(Formatter.firstIndex(path))
          .append(":")
          .append(Formatter.lastIndex(path))
          .append("]");
    }

    @Override
    public StringBuilder visit(Path.CommandTree path, StringBuilder sb) {
      visit(path.getCapture(), sb)
          .append(".tree");
      append(sb, path.getFilter()).append('[');
      if (path.getGroupByApi()) sb.append('A');
      if (path.getGroupByThread()) sb.append('T');
      if (path.getGroupByFrame()) sb.append('F');
      if (path.getAllowIncompleteFrame()) sb.append('i');
      if (path.getGroupByDrawCall()) sb.append('D');
      if (path.getGroupByUserMarkers()) sb.append('M');
      if (path.getGroupBySubmission()) sb.append('S');
      if (path.getMaxChildren() != 0) {
        sb.append(",max=").append(path.getMaxChildren());
      }
      sb.append(']');
      return sb;
    }

    @Override
    public StringBuilder visit(Path.CommandTreeNode path, StringBuilder sb) {
      sb.append("tree{");
      visit(path.getTree(), sb);
      sb.append("}.node(");
      sb.append(Formatter.index(path.getIndicesList()));
      sb.append(")");
      return sb;
    }

    @Override
    public StringBuilder visit(Path.CommandTreeNodeForCommand path, StringBuilder sb) {
      sb.append("tree{");
      visit(path.getTree(), sb);
      sb.append("}.command(");
      visit(path.getCommand(), sb);
      sb.append(")");
      return sb;
    }

    @Override
    public StringBuilder visit(Path.Device path, StringBuilder sb) {
      sb.append("device{");
      visit(path.getID(), sb);
      sb.append("}");
      return sb;
    }

    @Override
    public StringBuilder visit(Path.DeviceTraceConfiguration path, StringBuilder sb) {
      sb.append("device_configuration{");
      visit(path.getDevice(), sb);
      sb.append("}");
      return sb;
    }

    @Override
    public StringBuilder visit(Path.FramebufferObservation path, StringBuilder sb) {
      return visit(path.getCommand(), sb).append(".fbo");
    }

    @Override
    public StringBuilder visit(Path.Field path, StringBuilder sb) {
      return dispatch(parentOf(path), this, sb)
          .append(".")
          .append(path.getName());
    }

    @Override
    public StringBuilder visit(Path.GlobalState path, StringBuilder sb) {
      return visit(path.getAfter(), sb).append(".global-state");
    }

    @Override
    public StringBuilder visit(Path.ID path, StringBuilder sb) {
      return sb.append(ProtoDebugTextFormat.shortDebugString(path));
    }

    @Override
    public StringBuilder visit(Path.ImageInfo path, StringBuilder sb) {
      sb.append("image{");
      visit(path.getID(), sb);
      sb.append("}");
      return sb;
    }

    @Override
    public StringBuilder visit(Path.MapIndex path, StringBuilder sb) {
      dispatch(parentOf(path), this, sb);
      switch (path.getKeyCase()) {
        case BOX:
          return sb.append("[").append(Formatter.toString(path.getBox(), null, true)).append("]");
        default:
          return sb.append("[??]");
      }
    }

    @Override
    public StringBuilder visit(Path.Memory path, StringBuilder sb) {
      visit(path.getAfter(), sb)
          .append(".memory(")
          .append("pool:").append(path.getPool()).append(',')
          .append(Long.toHexString(path.getAddress())).append('[').append(path.getSize()).append(']');
      if (path.getExcludeData()) sb.append(",nodata");
      if (path.getExcludeObserved()) sb.append(",noobs");
      return sb.append(')');
    }

    @Override
    public StringBuilder visit(Path.Mesh path, StringBuilder sb) {
      visit(path.getCommand(), sb).append(".mesh(");
      if (path.getOptions().getFaceted()) sb.append("faceted");
      return sb.append(')');
    }

    @Override
    public StringBuilder visit(Path.Parameter path, StringBuilder sb) {
      return visit(path.getCommand(), sb).append(".").append(path.getName());
    }

    @Override
    public StringBuilder visit(Path.Report path, StringBuilder sb) {
      visit(path.getCapture(), sb).append(".report");
      if (path.hasDevice()) {
        sb.append('[');
        visit(path.getDevice(), sb);
        sb.append(']');
      }
      return sb;
    }

    @Override
    public StringBuilder visit(Path.ResourceData path, StringBuilder sb) {
      visit(path.getAfter(), sb).append(".resource{");
      visit(path.getID(), sb).append("}");
      return sb;
    }

    @Override
    public StringBuilder visit(Path.Resources path, StringBuilder sb) {
      return visit(path.getCapture(), sb).append(".resources");
    }

    @Override
    public StringBuilder visit(Path.Result path, StringBuilder sb) {
      return visit(path.getCommand(), sb).append(".<result>");
    }

    @Override
    public StringBuilder visit(Path.Slice path, StringBuilder sb) {
      return dispatch(parentOf(path), this, sb)
          .append("[")
          .append(path.getStart())
          .append(":")
          .append(path.getEnd())
          .append("]");
    }

    @Override
    public StringBuilder visit(Path.State path, StringBuilder sb) {
      return visit(path.getAfter(), sb).append(".state");
    }

    @Override
    public StringBuilder visit(Path.StateTree path, StringBuilder sb) {
      visit(path.getState(), sb).append(".tree");
      if (path.getArrayGroupSize() > 0) {
        sb.append("(groupSize=").append(path.getArrayGroupSize()).append(')');
      }
      return sb;
    }

    @Override
    public StringBuilder visit(Path.StateTreeNode path, StringBuilder sb) {
      sb.append("stateTree{");
      visit(path.getTree(), sb);
      return sb.append("}.node(")
          .append(Formatter.index(path.getIndicesList()))
          .append(")");
    }

    @Override
    public StringBuilder visit(Path.StateTreeNodeForPath path, StringBuilder sb) {
      sb.append("stateTree{");
      visit(path.getTree(), sb);
      return sb.append("}.path(")
          .append(Paths.toString(path.getMember()))
          .append(")");
    }

    @Override
    public StringBuilder visit(Path.Thumbnail path, StringBuilder sb) {
      dispatch(parentOf(path), this, sb)
          .append(".thumbnail");
      String sep = "(", end = "";
      if (path.getDesiredMaxWidth() > 0) {
        sb.append(sep).append("w=").append(path.getDesiredMaxWidth());
        sep = ",";
        end = ")";
      }
      if (path.getDesiredMaxHeight() > 0) {
        sb.append(sep).append("h=").append(path.getDesiredMaxHeight());
        sep = ",";
        end = ")";
      }
      if (path.hasDesiredFormat()) {
        sb.append(sep).append("f=").append(path.getDesiredFormat().getName()); // TODO
        sep = ",";
        end = ")";
      }
      return sb.append(end);
    }

    private StringBuilder append(StringBuilder sb, Path.CommandFilter filter) {
      String sep = "(", end = "";
      if (filter.getThreadsCount() > 0) {
        sb.append(sep).append("threads=").append(filter.getThreadsList());
        sep = ",";
        end = ")";
      }
      return sb.append(end);
    }
  };
}
