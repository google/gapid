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

import com.google.common.primitives.UnsignedLongs;
import com.google.gapid.image.Images;
import com.google.gapid.models.ApiContext.FilteringContext;
import com.google.gapid.models.AtomStream.AtomIndex;
import com.google.gapid.proto.image.Image;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.proto.service.vertex.Vertex;
import com.google.gapid.views.Formatter;

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
    return Path.Command.newBuilder().setCapture(capture).addIndices(index).build();
  }

  public static Path.Command firstCommand(Path.Commands commands) {
    return Path.Command.newBuilder().setCapture(commands.getCapture())
        .addAllIndices(commands.getFromList()).build();
  }

  public static Path.Command lastCommand(Path.Commands commands) {
    return Path.Command.newBuilder().setCapture(commands.getCapture())
        .addAllIndices(commands.getToList()).build();
  }

  public static Path.Any commandTree(Path.Capture capture, FilteringContext context) {
    return Path.Any.newBuilder().setCommandTree(
        context.commandTree(Path.CommandTree.newBuilder())
            .setCapture(capture)
            .setMaxChildren(2000)
            .setMaxNeighbours(20))
        .build();
  }

  public static Path.Any events(Path.Capture capture, FilteringContext context) {
    return Path.Any.newBuilder()
        .setEvents(
            context.events(Path.Events.newBuilder()).setCapture(capture).setLastInFrame(true))
        .build();
  }

  public static Path.Any commandTree(Path.ID tree, Path.Command command) {
    return Path.Any.newBuilder().setCommandTreeNodeForCommand(
        Path.CommandTreeNodeForCommand.newBuilder().setTree(tree).setCommand(command)).build();
  }

  public static Path.State stateAfter(Path.Command command) {
    if (command == null) {
      return null;
    }
    return Path.State.newBuilder().setAfter(command).build();
  }

  public static Path.Any stateTree(AtomIndex atom) {
    if (atom == null) {
      return null;
    }
    return Path.Any.newBuilder()
        .setStateTree(
            Path.StateTree.newBuilder()
                .setState(stateAfter(atom.getCommand()))
                .setArrayGroupSize(2000))
        .build();
  }

  public static Path.Any stateTree(Path.ID tree, Path.Any statePath) {
    return Path.Any.newBuilder().setStateTreeNodeForPath(
        Path.StateTreeNodeForPath.newBuilder().setTree(tree).setMember(statePath)).build();
  }

  public static Path.Any memoryAfter(Path.Command after, int pool, long address, long size) {
    if (after == null) {
      return null;
    }
    return Path.Any.newBuilder().setMemory(
        Path.Memory.newBuilder().setAfter(after).setPool(pool).setAddress(address).setSize(size))
        .build();
  }

  public static Path.Any memoryAfter(AtomIndex index, int pool, Service.MemoryRange range) {
    if (index == null || range == null) {
      return null;
    }
    return Path.Any.newBuilder().setMemory(Path.Memory.newBuilder().setAfter(index.getCommand())
        .setPool(pool).setAddress(range.getBase()).setSize(range.getSize())).build();
  }

  public static Path.Any observationsAfter(AtomIndex index, int pool) {
    if (index == null) {
      return null;
    }
    return Path.Any.newBuilder()
        .setMemory(Path.Memory.newBuilder().setAfter(index.getCommand()).setPool(pool).setAddress(0)
            .setSize(UnsignedLongs.MAX_VALUE).setExcludeData(true).setExcludeObserved(true))
        .build();
  }

  public static Path.Any resourceAfter(AtomIndex atom, Path.ID id) {
    if (atom == null || id == null) {
      return null;
    }
    return Path.Any.newBuilder()
        .setResourceData(Path.ResourceData.newBuilder().setAfter(atom.getCommand()).setId(id))
        .build();
  }

  public static Path.Any meshAfter(AtomIndex atom, Path.MeshOptions options,
      Vertex.BufferFormat format) {
    return Path.Any.newBuilder()
        .setAs(Path.As.newBuilder().setVertexBufferFormat(format)
            .setMesh(Path.Mesh.newBuilder().setCommandTreeNode(atom.getNode()).setOptions(options)))
        .build();
  }

  public static Path.Any atomField(Path.Command atom, String field) {
    if (atom == null || field == null) {
      return null;
    }
    return Path.Any.newBuilder()
        .setParameter(Path.Parameter.newBuilder().setCommand(atom).setName(field)).build();
  }

  public static Path.Any atomResult(Path.Command atom) {
    if (atom == null) {
      return null;
    }
    return Path.Any.newBuilder().setResult(Path.Result.newBuilder().setCommand(atom)).build();
  }

  public static Path.Any imageInfo(Path.ImageInfo image) {
    return Path.Any.newBuilder().setImageInfo(image).build();
  }

  public static Path.Any resourceInfo(Path.ResourceData resource) {
    return Path.Any.newBuilder().setResourceData(resource).build();
  }

  public static Path.Any imageData(Path.ImageInfo image, Image.Format format) {
    return Path.Any.newBuilder()
        .setAs(Path.As.newBuilder().setImageInfo(image).setImageFormat(format)).build();
  }

  public static Path.Any imageData(Path.ResourceData resource, Image.Format format) {
    return Path.Any.newBuilder()
        .setAs(Path.As.newBuilder().setResourceData(resource).setImageFormat(format)).build();
  }

  public static Path.Thumbnail thumbnail(Path.ResourceData resource, int size) {
    return Path.Thumbnail.newBuilder().setResource(resource)
        .setDesiredFormat(Images.FMT_RGBA_U8_NORM).setDesiredMaxHeight(size)
        .setDesiredMaxWidth(size).build();
  }

  public static Path.Thumbnail thumbnail(Path.Command command, int size) {
    return Path.Thumbnail.newBuilder().setCommand(command).setDesiredFormat(Images.FMT_RGBA_U8_NORM)
        .setDesiredMaxHeight(size).setDesiredMaxWidth(size).build();
  }

  public static Path.Thumbnail thumbnail(Path.CommandTreeNode node, int size) {
    return Path.Thumbnail.newBuilder().setCommandTreeNode(node)
        .setDesiredFormat(Images.FMT_RGBA_U8_NORM).setDesiredMaxHeight(size)
        .setDesiredMaxWidth(size).build();
  }

  public static Path.Any thumbnail(Path.Thumbnail thumb) {
    return Path.Any.newBuilder().setThumbnail(thumb).build();
  }

  public static Path.Any blob(Image.ID id) {
    return Path.Any.newBuilder()
        .setBlob(Path.Blob.newBuilder().setId(Path.ID.newBuilder().setData(id.getData()))).build();
  }

  public static Path.Any device(Path.Device device) {
    return Path.Any.newBuilder().setDevice(device).build();
  }

  public static Path.Any any(Path.Command command) {
    return Path.Any.newBuilder().setCommand(command).build();
  }

  public static Path.Any any(Path.CommandTreeNode node) {
    return Path.Any.newBuilder().setCommandTreeNode(node).build();
  }

  public static Path.Any any(Path.CommandTreeNode.Builder node) {
    return Path.Any.newBuilder().setCommandTreeNode(node).build();
  }

  public static Path.Any any(Path.ConstantSet constants) {
    return Path.Any.newBuilder().setConstantSet(constants).build();
  }

  public static Path.Any any(Path.StateTreeNode node) {
    return Path.Any.newBuilder().setStateTreeNode(node).build();
  }

  public static Path.Any any(Path.StateTreeNode.Builder node) {
    return Path.Any.newBuilder().setStateTreeNode(node).build();
  }

  /**
   * Compares a and b, returning -1 if a comes before b, 1 if b comes before a and 0 if they
   * are equal.
   */
  public static int compare(Path.Command a, Path.Command b) {
    if (a == null) {
      return (b == null) ? 0 : -1;
    } else if (b == null) {
      return 1;
    }

    for (int i = 0; i < a.getIndicesCount(); i++) {
      if (i >= b.getIndicesCount()) {
        return 1;
      }
      int r = Long.compare(a.getIndices(i), b.getIndices(i));
      if (r != 0) {
        return r;
      }
    }
    return (a.getIndicesCount() == b.getIndicesCount()) ? 0 : -1;
  }

  public static Path.State findState(Path.Any path) {
    switch (path.getPathCase()) {
      case STATE:
        return path.getState();
      case FIELD:
        return findState(path.getField());
      case ARRAY_INDEX:
        return findState(path.getArrayIndex());
      case SLICE:
        return findState(path.getSlice());
      case MAP_INDEX:
        return findState(path.getMapIndex());
      default:
        return null;
    }
  }

  public static Path.State findState(Path.Field path) {
    switch (path.getStructCase()) {
      case STATE:
        return path.getState();
      case FIELD:
        return findState(path.getField());
      case ARRAY_INDEX:
        return findState(path.getArrayIndex());
      case SLICE:
        return findState(path.getSlice());
      case MAP_INDEX:
        return findState(path.getMapIndex());
      default:
        return null;
    }
  }

  public static Path.State findState(Path.ArrayIndex path) {
    switch (path.getArrayCase()) {
      case FIELD:
        return findState(path.getField());
      case ARRAY_INDEX:
        return findState(path.getArrayIndex());
      case SLICE:
        return findState(path.getSlice());
      case MAP_INDEX:
        return findState(path.getMapIndex());
      default:
        return null;
    }
  }

  public static Path.State findState(Path.Slice path) {
    switch (path.getArrayCase()) {
      case FIELD:
        return findState(path.getField());
      case ARRAY_INDEX:
        return findState(path.getArrayIndex());
      case SLICE:
        return findState(path.getSlice());
      case MAP_INDEX:
        return findState(path.getMapIndex());
      default:
        return null;
    }
  }

  public static Path.State findState(Path.MapIndex path) {
    switch (path.getMapCase()) {
      case STATE:
        return path.getState();
      case FIELD:
        return findState(path.getField());
      case ARRAY_INDEX:
        return findState(path.getArrayIndex());
      case SLICE:
        return findState(path.getSlice());
      case MAP_INDEX:
        return findState(path.getMapIndex());
      default:
        return null;
    }
  }

  public static Path.Any reparent(Path.Any path, Path.State newState) {
    Path.Any.Builder builder = path.toBuilder();
    switch (path.getPathCase()) {
      case STATE:
        return builder.setState(newState).build();
      case FIELD:
        return reparent(builder.getFieldBuilder(), newState) ? builder.build() : null;
      case ARRAY_INDEX:
        return reparent(builder.getArrayIndexBuilder(), newState) ? builder.build() : null;
      case SLICE:
        return reparent(builder.getSliceBuilder(), newState) ? builder.build() : null;
      case MAP_INDEX:
        return reparent(builder.getMapIndexBuilder(), newState) ? builder.build() : null;
      default:
        return null;
    }
  }

  public static boolean reparent(Path.Field.Builder path, Path.State newState) {
    switch (path.getStructCase()) {
      case STATE:
        path.setState(newState);
        return true;
      case FIELD:
        return reparent(path.getFieldBuilder(), newState);
      case ARRAY_INDEX:
        return reparent(path.getArrayIndexBuilder(), newState);
      case SLICE:
        return reparent(path.getSliceBuilder(), newState);
      case MAP_INDEX:
        return reparent(path.getMapIndexBuilder(), newState);
      default:
        return false;
    }
  }

  public static boolean reparent(Path.ArrayIndex.Builder path, Path.State newState) {
    switch (path.getArrayCase()) {
      case FIELD:
        return reparent(path.getFieldBuilder(), newState);
      case ARRAY_INDEX:
        return reparent(path.getArrayIndexBuilder(), newState);
      case SLICE:
        return reparent(path.getSliceBuilder(), newState);
      case MAP_INDEX:
        return reparent(path.getMapIndexBuilder(), newState);
      default:
        return false;
    }
  }

  public static boolean reparent(Path.Slice.Builder path, Path.State newState) {
    switch (path.getArrayCase()) {
      case FIELD:
        return reparent(path.getFieldBuilder(), newState);
      case ARRAY_INDEX:
        return reparent(path.getArrayIndexBuilder(), newState);
      case SLICE:
        return reparent(path.getSliceBuilder(), newState);
      case MAP_INDEX:
        return reparent(path.getMapIndexBuilder(), newState);
      default:
        return false;
    }
  }

  public static boolean reparent(Path.MapIndex.Builder path, Path.State newState) {
    switch (path.getMapCase()) {
      case STATE:
        path.setState(newState);
        return true;
      case FIELD:
        return reparent(path.getFieldBuilder(), newState);
      case ARRAY_INDEX:
        return reparent(path.getArrayIndexBuilder(), newState);
      case SLICE:
        return reparent(path.getSliceBuilder(), newState);
      case MAP_INDEX:
        return reparent(path.getMapIndexBuilder(), newState);
      default:
        return false;
    }
  }

  public static String toString(Path.ID id) {
    return ProtoDebugTextFormat.shortDebugString(id);
  }

  public static String toString(Image.ID id) {
    return ProtoDebugTextFormat.shortDebugString(id);
  }

  public static String toString(Path.Any path) {
    switch (path.getPathCase()) {
      case API:
        return toString(path.getApi());
      case ARRAY_INDEX:
        return toString(path.getArrayIndex());
      case AS:
        return toString(path.getAs());
      case BLOB:
        return toString(path.getBlob());
      case CAPTURE:
        return toString(path.getCapture());
      case COMMAND:
        return toString(path.getCommand());
      case COMMANDS:
        return toString(path.getCommands());
      case COMMAND_TREE:
        return toString(path.getCommandTree());
      case COMMAND_TREE_NODE:
        return toString(path.getCommandTreeNode());
      case COMMAND_TREE_NODE_FOR_COMMAND:
        return toString(path.getCommandTreeNodeForCommand());
      case CONSTANT_SET:
        return toString(path.getConstantSet());
      case CONTEXT:
        return toString(path.getContext());
      case CONTEXTS:
        return toString(path.getContexts());
      case DEVICE:
        return toString(path.getDevice());
      case EVENTS:
        return toString(path.getEvents());
      case FIELD:
        return toString(path.getField());
      case IMAGE_INFO:
        return toString(path.getImageInfo());
      case MAP_INDEX:
        return toString(path.getMapIndex());
      case MEMORY:
        return toString(path.getMemory());
      case MESH:
        return toString(path.getMesh());
      case PARAMETER:
        return toString(path.getParameter());
      case REPORT:
        return toString(path.getReport());
      case RESOURCES:
        return toString(path.getResources());
      case RESOURCE_DATA:
        return toString(path.getResourceData());
      case RESULT:
        return toString(path.getResult());
      case SLICE:
        return toString(path.getSlice());
      case STATE:
        return toString(path.getState());
      case STATE_TREE:
        return toString(path.getStateTree());
      case STATE_TREE_NODE:
        return toString(path.getStateTreeNode());
      case STATE_TREE_NODE_FOR_PATH:
        return toString(path.getStateTreeNodeForPath());
      case THUMBNAIL:
        return toString(path.getThumbnail());
      default:
        return ProtoDebugTextFormat.shortDebugString(path);
    }
  }

  public static String toString(Path.API api) {
    return "API{" + toString(api.getId()) + "}";
  }

  public static String toString(Path.ArrayIndex arrayIndex) {
    String parent;
    switch (arrayIndex.getArrayCase()) {
      case ARRAY_INDEX:
        parent = toString(arrayIndex.getArrayIndex());
        break;
      case FIELD:
        parent = toString(arrayIndex.getField());
        break;
      case MAP_INDEX:
        parent = toString(arrayIndex.getMapIndex());
        break;
      case PARAMETER:
        parent = toString(arrayIndex.getParameter());
        break;
      case REPORT:
        parent = toString(arrayIndex.getReport());
        break;
      case SLICE:
        parent = toString(arrayIndex.getSlice());
        break;
      default:
        parent = "??";
        break;
    }
    return parent + "[" + UnsignedLongs.toString(arrayIndex.getIndex()) + "]";
  }

  public static String toString(Path.As as) {
    String parent;
    switch (as.getFromCase()) {
      case ARRAY_INDEX:
        parent = toString(as.getArrayIndex());
        break;
      case FIELD:
        parent = toString(as.getField());
        break;
      case IMAGE_INFO:
        parent = toString(as.getImageInfo());
        break;
      case MAP_INDEX:
        parent = toString(as.getMapIndex());
        break;
      case MESH:
        parent = toString(as.getMesh());
        break;
      case RESOURCE_DATA:
        parent = toString(as.getResourceData());
        break;
      case SLICE:
        parent = toString(as.getSlice());
        break;
      default:
        parent = "??";
        break;
    }
    switch (as.getToCase()) {
      case IMAGE_FORMAT:
        return parent + ".as(" + as.getImageFormat().getName() + ")"; // TODO
      case VERTEX_BUFFER_FORMAT:
        return parent + ".as(VBF)"; // TODO
      default:
        return parent + ".as(??)";
    }
  }

  public static String toString(Path.Blob blob) {
    return "blob{" + toString(blob.getId()) + "}";
  }

  public static String toString(Path.Capture capture) {
    return "capture{" + toString(capture.getId()) + "}";
  }

  public static String toString(Path.Command command) {
    return toString(command.getCapture()) + ".command[" + Formatter.atomIndex(command) + "]";
  }

  public static String toString(Path.Commands commands) {
    return toString(commands.getCapture()) + ".command[" + Formatter.firstIndex(commands) + ":"
        + Formatter.lastIndex(commands) + "]";
  }

  public static String toString(Path.CommandTree tree) {
    StringBuilder sb = new StringBuilder().append(toString(tree.getCapture())).append(".tree");
    append(sb, tree.getFilter()).append('[');
    if (tree.getGroupByApi()) sb.append('A');
    if (tree.getGroupByThread()) sb.append('T');
    if (tree.getGroupByContext()) sb.append('C');
    if (tree.getIncludeNoContextGroups()) sb.append('n');
    if (tree.getGroupByFrame()) sb.append('F');
    if (tree.getAllowIncompleteFrame()) sb.append('i');
    if (tree.getGroupByDrawCall()) sb.append('D');
    if (tree.getGroupByUserMarkers()) sb.append('M');
    if (tree.getMaxChildren() != 0) {
      sb.append(",max=").append(tree.getMaxChildren());
    }
    return sb.append(']').toString();
  }

  public static StringBuilder append(StringBuilder sb, Path.CommandFilter filter) {
    String sep = "(", end = "";
    if (filter.hasContext()) {
      sb.append(sep).append("context=").append(toString(filter.getContext()));
      sep = ",";
      end = ")";
    }
    if (filter.getThreadsCount() > 0) {
      sb.append(sep).append("threads=").append(filter.getThreadsList());
      sep = ",";
      end = ")";
    }
    return sb.append(end);
  }

  public static String toString(Path.CommandTreeNode n) {
    return "tree{" + toString(n.getTree()) + "}.node(" + Formatter.index(n.getIndicesList()) + ")";
  }

  public static String toString(Path.CommandTreeNodeForCommand nfc) {
    return "tree{" + toString(nfc.getTree()) + "}.command(" + toString(nfc.getCommand()) + ")";
  }

  public static String toString(Path.ConstantSet cs) {
    return toString(cs.getApi()) + ".constants[" + cs.getIndex() + "]";
  }

  public static String toString(Path.Context context) {
    return toString(context.getCapture()) + ".context[" + toString(context.getId()) + "]";
  }

  public static String toString(Path.Contexts contexts) {
    return toString(contexts.getCapture()) + ".contexts";
  }

  public static String toString(Path.Device device) {
    return "device{" + toString(device.getId()) + "}";
  }

  public static String toString(Path.Events events) {
    StringBuilder sb = new StringBuilder().append(toString(events.getCapture())).append(".events");
    append(sb, events.getFilter()).append('[');
    if (events.getFirstInFrame()) sb.append("Fs");
    if (events.getLastInFrame()) sb.append("Fe");
    if (events.getClears()) sb.append("C");
    if (events.getDrawCalls()) sb.append("D");
    if (events.getUserMarkers()) sb.append("M");
    if (events.getPushUserMarkers()) sb.append("Ms");
    if (events.getPopUserMarkers()) sb.append("Me");
    if (events.getFramebufferObservations()) sb.append("O");
    return sb.append(']').toString();
  }

  public static String toString(Path.Field field) {
    String parent;
    switch (field.getStructCase()) {
      case FIELD:
        parent = toString(field.getField());
        break;
      case SLICE:
        parent = toString(field.getSlice());
        break;
      case ARRAY_INDEX:
        parent = toString(field.getArrayIndex());
        break;
      case MAP_INDEX:
        parent = toString(field.getMapIndex());
        break;
      case STATE:
        parent = toString(field.getState());
        break;
      case PARAMETER:
        parent = toString(field.getParameter());
        break;
      default:
        parent = "??";
        break;
    }
    return parent + "." + field.getName();
  }

  public static String toString(Path.ImageInfo image) {
    return "image{" + toString(image.getId()) + "}";
  }

  public static String toString(Path.MapIndex mapIndex) {
    String parent;
    switch (mapIndex.getMapCase()) {
      case FIELD:
        parent = toString(mapIndex.getField());
        break;
      case SLICE:
        parent = toString(mapIndex.getSlice());
        break;
      case ARRAY_INDEX:
        parent = toString(mapIndex.getArrayIndex());
        break;
      case MAP_INDEX:
        parent = toString(mapIndex.getMapIndex());
        break;
      case STATE:
        parent = toString(mapIndex.getState());
        break;
      case PARAMETER:
        parent = toString(mapIndex.getParameter());
        break;
      default:
        parent = "??";
        break;
    }
    switch (mapIndex.getKeyCase()) {
      case BOX:
        return parent + "[" + Formatter.toString(mapIndex.getBox(), null, true) + "]";
      default:
        return parent + "[??]";
    }
  }

  public static String toString(Path.Memory mem) {
    StringBuilder sb = new StringBuilder().append(toString(mem.getAfter())).append(".memory(")
        .append("pool:").append(mem.getPool()).append(',')
        .append(Long.toHexString(mem.getAddress())).append('[').append(mem.getSize()).append(']');
    if (mem.getExcludeData()) sb.append(",nodata");
    if (mem.getExcludeObserved()) sb.append(",noobs");
    return sb.append(')').toString();
  }

  public static String toString(Path.Mesh mesh) {
    StringBuilder sb = new StringBuilder();
    switch (mesh.getObjectCase()) {
      case COMMAND:
        sb.append(toString(mesh.getCommand()));
        break;
      case COMMAND_TREE_NODE:
        sb.append(toString(mesh.getCommandTreeNode()));
        break;
      default:
        sb.append("??");
        break;
    }
    sb.append(".mesh(");
    if (mesh.getOptions().getFaceted()) sb.append("faceted");
    return sb.append(')').toString();
  }

  public static String toString(Path.Parameter parameter) {
    return toString(parameter.getCommand()) + "." + parameter.getName();
  }

  public static String toString(Path.Report report) {
    StringBuilder sb = new StringBuilder().append(toString(report.getCapture())).append(".report");
    if (report.hasDevice()) {
      sb.append('[').append(toString(report.getDevice()));
    }
    return append(sb, report.getFilter()).toString();
  }

  public static String toString(Path.ResourceData res) {
    return toString(res.getAfter()) + ".resource{" + toString(res.getId()) + "}";
  }

  public static String toString(Path.Resources res) {
    return toString(res.getCapture()) + ".resources";
  }

  public static String toString(Path.Result result) {
    return toString(result.getCommand()) + ".<result>";
  }

  public static String toString(Path.Slice slice) {
    String parent;
    switch (slice.getArrayCase()) {
      case FIELD:
        parent = toString(slice.getField());
        break;
      case SLICE:
        parent = toString(slice.getSlice());
        break;
      case ARRAY_INDEX:
        parent = toString(slice.getArrayIndex());
        break;
      case MAP_INDEX:
        parent = toString(slice.getMapIndex());
        break;
      case PARAMETER:
        parent = toString(slice.getParameter());
        break;
      default:
        parent = "??";
        break;
    }
    return parent + "[" + slice.getStart() + ":" + slice.getEnd() + "]";
  }

  public static String toString(Path.State state) {
    return toString(state.getAfter()) + ".state";
  }

  public static String toString(Path.StateTree tree) {
    StringBuilder sb = new StringBuilder().append(toString(tree.getState())).append(".tree");
    if (tree.getArrayGroupSize() > 0) {
      sb.append("(groupSize=").append(tree.getArrayGroupSize()).append(')');
    }
    return sb.toString();
  }

  public static String toString(Path.StateTreeNode node) {
    return "stateTree{" + toString(node.getTree()) + "}.node("
        + Formatter.index(node.getIndicesList()) + ")";
  }

  public static String toString(Path.StateTreeNodeForPath nfp) {
    return "stateTree{" + toString(nfp.getTree()) + "}.path(" + toString(nfp.getMember()) + ")";
  }

  public static String toString(Path.Thumbnail thumbnail) {
    StringBuilder sb = new StringBuilder();
    switch (thumbnail.getObjectCase()) {
      case RESOURCE:
        sb.append(toString(thumbnail.getResource()));
        break;
      case COMMAND:
        sb.append(toString(thumbnail.getCommand()));
        break;
      case COMMAND_TREE_NODE:
        sb.append(toString(thumbnail.getCommandTreeNode()));
        break;
      default:
        sb.append("??");
        break;
    }
    sb.append(".thumbnail");
    String sep = "(", end = "";
    if (thumbnail.getDesiredMaxWidth() > 0) {
      sb.append(sep).append("w=").append(thumbnail.getDesiredMaxWidth());
      sep = ",";
      end = ")";
    }
    if (thumbnail.getDesiredMaxHeight() > 0) {
      sb.append(sep).append("h=").append(thumbnail.getDesiredMaxHeight());
      sep = ",";
      end = ")";
    }
    if (thumbnail.hasDesiredFormat()) {
      sb.append(sep).append("f=").append(thumbnail.getDesiredFormat().getName()); // TODO
      sep = ",";
      end = ")";
    }
    return sb.append(end).toString();
  }
}
