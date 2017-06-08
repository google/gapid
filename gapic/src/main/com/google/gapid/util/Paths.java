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
import com.google.gapid.proto.core.pod.Pod;
import com.google.gapid.proto.image.Image;
import com.google.gapid.proto.service.Service.MemoryRange;
import com.google.gapid.proto.service.box.Box;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.proto.service.vertex.Vertex;
import com.google.protobuf.ByteString;
import com.google.protobuf.TextFormat;

/**
 * Path utilities.
 */
public class Paths {
  public static final Path.ID ZERO_ID = Path.ID.newBuilder()
      .setData(ByteString.copyFrom(new byte[20]))
      .build();

  private Paths() {
  }

  public static String toString(Path.Any path) {
    // TODO
    return TextFormat.shortDebugString(path);
  }

  /*
  public static Path.Command command(Path.Any atomsPath, long index) {
    if (atomsPath == null || atomsPath.getPathCase() != COMMANDS) {
      return null;
    }
    return Path.Command.newBuilder()
        .setCapture(atomsPath.getCommands().getCapture())
        .addIndex(index)
        .build();
  }
  */

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

  public static Path.Any commandTree(Path.Capture capture, FilteringContext context) {
    return Path.Any.newBuilder()
        .setCommandTree(context.commandTree(Path.CommandTree.newBuilder())
            .setCapture(capture)
            .setMaxChildren(2000))
        .build();
  }

  public static Path.Any events(Path.Capture capture, FilteringContext context) {
    return Path.Any.newBuilder()
        .setEvents(context.events(Path.Events.newBuilder())
            .setCommands(Path.Commands.newBuilder()
                .setCapture(capture))
            .setLastInFrame(true))
        .build();
  }

  public static Path.Any commandTree(Path.ID tree, Path.Command command) {
    return Path.Any.newBuilder()
        .setCommandTreeNodeForCommand(Path.CommandTreeNodeForCommand.newBuilder()
            .setTree(tree)
            .setCommand(command))
        .build();
  }

  public static Path.Any stateTree(AtomIndex atom) {
    if (atom == null) {
      return null;
    }
    return Path.Any.newBuilder()
        .setStateTree(Path.StateTree.newBuilder()
            .setAfter(atom.getCommand())
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

  /*
  public static Path.Any memoryAfter(
      Path.Any atomsPath, AtomIndex atom, int pool, long address, long size) {
    if (atomsPath == null || atom == null || atomsPath.getPathCase() != COMMANDS) {
      return null;
    }
    return Path.Any.newBuilder()
        .setMemory(Path.Memory.newBuilder()
            .setAfter(atom.toProto(Path.Command.newBuilder())
                .setCapture(atomsPath.getCommands().getCapture()))
            .setPool(pool)
            .setAddress(address)
            .setSize(size))
        .build();
  }
  */

  public static Path.Any memoryAfter(Path.Command after, int pool, long address, long size) {
    if (after == null) {
      return null;
    }
    return Path.Any.newBuilder()
        .setMemory(Path.Memory.newBuilder()
            .setAfter(after)
            .setPool(pool)
            .setAddress(address)
            .setSize(size))
        .build();
  }

  public static Path.Any memoryAfter(AtomIndex index, int pool, MemoryRange range) {
    if (index == null || range == null) {
      return null;
    }
    return Path.Any.newBuilder()
        .setMemory(Path.Memory.newBuilder()
            .setAfter(index.getCommand())
            .setPool(pool)
            .setAddress(range.getBase())
            .setSize(range.getSize()))
        .build();
  }

  public static Path.Any observationsAfter(AtomIndex index, int pool) {
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
            .setExcludeObserved(true))
        .build();
  }

  public static Path.Any resourceAfter(AtomIndex atom, Path.ID id) {
    if (atom == null || id == null) {
      return null;
    }
    return Path.Any.newBuilder()
        .setResourceData(Path.ResourceData.newBuilder()
            .setAfter(atom.getCommand())
            .setId(id))
        .build();
  }

  public static Path.Any meshAfter(
      AtomIndex atom, Path.MeshOptions options, Vertex.BufferFormat format) {
    return Path.Any.newBuilder()
        .setAs(Path.As.newBuilder()
            .setVertexBufferFormat(format)
            .setMesh(Path.Mesh.newBuilder()
                .setCommandTreeNode(atom.getNode())
                .setOptions(options)))
        .build();
  }

  public static Path.Any atomField(Path.Command atom, String field) {
    if (atom == null || field == null) {
      return null;
    }
    return Path.Any.newBuilder()
        .setParameter(Path.Parameter.newBuilder()
            .setCommand(atom)
            .setName(field))
        .build();
  }

  public static Path.Any atomResult(Path.Command atom) {
    if (atom == null) {
      return null;
    }
    return Path.Any.newBuilder()
        .setResult(Path.Result.newBuilder()
            .setCommand(atom))
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

  public static Path.Thumbnail thumbnail(Path.ResourceData resource, int size) {
    return Path.Thumbnail.newBuilder()
        .setResource(resource)
        .setDesiredFormat(Images.FMT_RGBA_U8_NORM)
        .setDesiredMaxHeight(size)
        .setDesiredMaxWidth(size)
        .build();
  }

  public static Path.Thumbnail thumbnail(Path.Command command, int size) {
    return Path.Thumbnail.newBuilder()
        .setCommand(command)
        .setDesiredFormat(Images.FMT_RGBA_U8_NORM)
        .setDesiredMaxHeight(size)
        .setDesiredMaxWidth(size)
        .build();
  }

  public static Path.Thumbnail thumbnail(Path.CommandTreeNode node, int size) {
    return Path.Thumbnail.newBuilder()
        .setCommandTreeNode(node)
        .setDesiredFormat(Images.FMT_RGBA_U8_NORM)
        .setDesiredMaxHeight(size)
        .setDesiredMaxWidth(size)
        .build();
  }

  public static Path.Any thumbnail(Path.Thumbnail thumb) {
    return Path.Any.newBuilder()
        .setThumbnail(thumb)
        .build();
  }

  public static Path.Any blob(Image.ID id) {
    return Path.Any.newBuilder()
        .setBlob(Path.Blob.newBuilder()
            .setId(Path.ID.newBuilder()
                .setData(id.getData())))
        .build();
  }

  public static Path.Any device(Path.Device device) {
    return Path.Any.newBuilder()
        .setDevice(device)
        .build();
  }

  public static Path.State findState(Path.Any path) {
    switch (path.getPathCase()) {
      case STATE: return path.getState();
      case FIELD: return findState(path.getField());
      case ARRAY_INDEX: return findState(path.getArrayIndex());
      case SLICE: return findState(path.getSlice());
      case MAP_INDEX: return findState(path.getMapIndex());
      default: return null;
    }
  }

  public static Path.State findState(Path.Field path) {
    switch (path.getStructCase()) {
      case STATE: return path.getState();
      case FIELD: return findState(path.getField());
      case ARRAY_INDEX: return findState(path.getArrayIndex());
      case SLICE: return findState(path.getSlice());
      case MAP_INDEX: return findState(path.getMapIndex());
      default: return null;
    }
  }

  public static Path.State findState(Path.ArrayIndex path) {
    switch (path.getArrayCase()) {
      case FIELD: return findState(path.getField());
      case ARRAY_INDEX: return findState(path.getArrayIndex());
      case SLICE: return findState(path.getSlice());
      case MAP_INDEX: return findState(path.getMapIndex());
      default: return null;
    }
  }

  public static Path.State findState(Path.Slice path) {
    switch (path.getArrayCase()) {
      case FIELD: return findState(path.getField());
      case ARRAY_INDEX: return findState(path.getArrayIndex());
      case SLICE: return findState(path.getSlice());
      case MAP_INDEX: return findState(path.getMapIndex());
      default: return null;
    }
  }

  public static Path.State findState(Path.MapIndex path) {
    switch (path.getMapCase()) {
      case STATE: return path.getState();
      case FIELD: return findState(path.getField());
      case ARRAY_INDEX: return findState(path.getArrayIndex());
      case SLICE: return findState(path.getSlice());
      case MAP_INDEX: return findState(path.getMapIndex());
      default: return null;
    }
  }

  // TODO Incomplete.
  @SuppressWarnings("unused")
  public static interface PathBuilder {
    public static final PathBuilder INVALID_BUILDER = new PathBuilder() {
      @Override
      public Path.Any build() {
        return null;
      }
    };

    public default PathBuilder map(Pod.Value key) { return INVALID_BUILDER; }
    public default PathBuilder array(long index) { return INVALID_BUILDER; }
    public default PathBuilder field(String name) { return INVALID_BUILDER; }
    public Path.Any build();

    public static class State implements PathBuilder {
      private final Path.State state;

      public State(Path.State state) {
        this.state = state;
      }

      @Override
      public PathBuilder map(Pod.Value key) {
        return new MapIndex(Path.MapIndex.newBuilder().setState(state), key);
      }

      @Override
      public PathBuilder field(String name) {
        return new Field(Path.Field.newBuilder().setState(state), name);
      }

      @Override
      public Path.Any build() {
        return Path.Any.newBuilder().setState(state).build();
      }
    }

    public static class MapIndex implements PathBuilder {
      private final Path.MapIndex.Builder mapIndex;

      public MapIndex(Path.MapIndex.Builder mapIndex, Pod.Value key) {
        this.mapIndex = mapIndex.setBox(Box.Value.newBuilder().setPod(key));
      }

      @Override
      public PathBuilder map(Pod.Value key) {
        return new MapIndex(Path.MapIndex.newBuilder().setMapIndex(mapIndex), key);
      }

      @Override
      public PathBuilder array(long index) {
        return new ArrayIndex(Path.ArrayIndex.newBuilder().setMapIndex(mapIndex), index);
      }

      @Override
      public PathBuilder field(String name) {
        return new Field(Path.Field.newBuilder().setMapIndex(mapIndex), name);
      }

      @Override
      public Path.Any build() {
        return Path.Any.newBuilder().setMapIndex(mapIndex).build();
      }
    }

    public static class ArrayIndex implements PathBuilder {
      private final Path.ArrayIndex.Builder arrayIndex;

      public ArrayIndex(Path.ArrayIndex.Builder arrayIndex, long index) {
        this.arrayIndex = arrayIndex.setIndex(index);
      }

      @Override
      public PathBuilder map(Pod.Value key) {
        return new MapIndex(Path.MapIndex.newBuilder().setArrayIndex(arrayIndex), key);
      }

      @Override
      public PathBuilder array(long index) {
        return new ArrayIndex(Path.ArrayIndex.newBuilder().setArrayIndex(arrayIndex), index);
      }

      @Override
      public PathBuilder field(String name) {
        return new Field(Path.Field.newBuilder().setArrayIndex(arrayIndex), name);
      }

      @Override
      public Path.Any build() {
        return Path.Any.newBuilder().setArrayIndex(arrayIndex).build();
      }
    }

    public static class Field implements PathBuilder {
      private final Path.Field.Builder field;

      public Field(Path.Field.Builder field, String name) {
        this.field = field.setName(name);
      }

      @Override
      public PathBuilder map(Pod.Value key) {
        return new MapIndex(Path.MapIndex.newBuilder().setField(field), key);
      }

      @Override
      public PathBuilder array(long index) {
        return new ArrayIndex(Path.ArrayIndex.newBuilder().setField(field), index);
      }

      @Override
      public PathBuilder field(String name) {
        return new Field(Path.Field.newBuilder().setField(field), name);
      }

      @Override
      public Path.Any build() {
        return Path.Any.newBuilder().setField(field).build();
      }
    }
  }

  // TODO Incomplete.
  @SuppressWarnings("unused")
  public static interface Visitor {
    public default void array(Path.ArrayIndex array) { /* empty */ }
    public default void field(Path.Field field) { /* empty */ }
    public default void map(Path.MapIndex map) { /* empty */ }
    public default void state(Path.State state) { /* empty */ }
  }

  public static void visit(Path.Any path, Visitor visitor) {
    switch (path.getPathCase()) {
      case ARRAY_INDEX: visit(path.getArrayIndex(), visitor); break;
      case FIELD: visit(path.getField(), visitor); break;
      case MAP_INDEX: visit(path.getMapIndex(), visitor); break;
      case STATE: visit(path.getState(), visitor); break;
      default: throw new UnsupportedOperationException();
    }
  }

  public static void visit(Path.ArrayIndex array, Visitor visitor) {
    visitor.array(array);
    switch (array.getArrayCase()) {
      case ARRAY_INDEX: visit(array.getArrayIndex(), visitor); break;
      case MAP_INDEX: visit(array.getMapIndex(), visitor); break;
      case FIELD: visit(array.getField(), visitor); break;
      default: throw new UnsupportedOperationException();
    }
  }

  public static void visit(Path.Field field, Visitor visitor) {
    visitor.field(field);
    switch (field.getStructCase()) {
      case ARRAY_INDEX: visit(field.getArrayIndex(), visitor); break;
      case FIELD: visit(field.getField(), visitor); break;
      case MAP_INDEX: visit(field.getMapIndex(), visitor); break;
      case STATE: visit(field.getState(), visitor); break;
      default: throw new UnsupportedOperationException();
    }
  }

  public static void visit(Path.MapIndex map, Visitor visitor) {
    visitor.map(map);
    switch (map.getMapCase()) {
      case ARRAY_INDEX: visit(map.getArrayIndex(), visitor); break;
      case FIELD: visit(map.getField(), visitor); break;
      case MAP_INDEX: visit(map.getMapIndex(), visitor); break;
      case STATE: visit(map.getState(), visitor); break;
      default: throw new UnsupportedOperationException();
    }
  }

  public static void visit(Path.State state, Visitor visitor) {
    visitor.state(state);
  }
}
