/*
 * Copyright (C) 2018 Google Inc.
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

import static com.google.gapid.models.DeviceDependentModel.Source.withSource;
import static com.google.gapid.proto.service.memory.Memory.PoolNames.Application_VALUE;
import static com.google.gapid.util.Paths.memoryAfter;
import static com.google.gapid.util.Paths.type;
import static com.google.gapid.util.Ranges.memory;
import static com.google.gapid.util.Ranges.merge;
import static com.google.gapid.util.Ranges.relative;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.collect.Sets;
import com.google.common.primitives.UnsignedLongs;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.memory_box.MemoryBox;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.proto.service.types.TypeInfo;
import com.google.gapid.proto.service.types.TypeInfo.Type.TyCase;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.MemoryBoxes;
import com.google.gapid.util.Messages;
import com.google.gapid.util.MoreFutures;
import com.google.gapid.util.Paths;
import com.google.gapid.util.Ranges;
import com.google.gapid.util.TypeInfos;

import java.util.Set;
import java.util.stream.Collectors;
import org.eclipse.swt.widgets.Shell;

import java.lang.ref.SoftReference;
import java.nio.charset.Charset;
import java.util.ArrayList;
import java.util.BitSet;
import java.util.HashMap;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.logging.Logger;

/**
 * Model responsible for loading memory pool data. This model requests segments as equally sized
 * pages and maintains a cache of fetched pages.
 */
public class Memory extends DeviceDependentModel<Memory.Data, Memory.Source, Void, Memory.Listener> {
  private static final Logger LOG = Logger.getLogger(Memory.class.getName());

  private final CommandStream commands;

  public Memory(
      Shell shell, Analytics analytics, Client client, Devices devices, CommandStream commands) {
    super(LOG, shell, analytics, client, Listener.class, devices);
    this.commands = commands;

    commands.addListener(new CommandStream.Listener() {
      @Override
      public void onCommandsSelected(CommandIndex selection) {
        load(withSource(getSource(), new Source(selection, getPool())), false);
      }
    });
  }

  public int getPool() {
    DeviceDependentModel.Source<Source> source = getSource();
    return (source == null || source.source == null) ? 0 : source.source.pool;
  }

  public void setPool(int pool) {
    load(Source.withPool(getSource(), pool), false);
  }

  @Override
  protected ListenableFuture<Data> doLoad(Source source, Path.Device device) {
    return MoreFutures.transform(commands.getMemory(device, source.command), memory -> {
      List<Service.MemoryRange> reads = merge(memory.getReadsList());
      List<Service.MemoryRange> writes = merge(memory.getWritesList());
      List<Service.TypedMemoryRange> typeds = memory.getTypedRangesList();

      Observation[] obs = new Observation[reads.size() + writes.size()];
      StructObservation[] structObs = new StructObservation[typeds.size()];
      int idx = 0;
      for (Service.MemoryRange read : reads) {
        obs[idx++] = new Observation(source.command, true, read);
      }
      for (Service.MemoryRange write : writes) {
        obs[idx++] = new Observation(source.command, false, write);
      }
      idx = 0;
      for (Service.TypedMemoryRange typed : typeds) {
        structObs[idx++] = new StructObservation(typed, source, device);
      }

      return new Data(device, client, source, obs, structObs);
    });
  }

  @Override
  protected void fireLoadStartEvent() {
    listeners.fire().onMemoryLoadingStart();
  }

  @Override
  protected void fireLoadedEvent() {
    listeners.fire().onMemoryLoaded();
  }


  public static class Source {
    public final CommandIndex command;
    public final int pool;

    public Source(CommandIndex command, int pool) {
      this.command = command;
      this.pool = pool;
    }

    public static DeviceDependentModel.Source<Source> withPool(
        DeviceDependentModel.Source<Source> src, int newPool) {
      Source me = (src == null) ? null : src.source;
      return new DeviceDependentModel.Source<Source>((src == null) ? null : src.device,
          new Source((me == null) ? null : me.command, newPool));
    }

    @Override
    public boolean equals(Object obj) {
      if (obj == this) {
        return true;
      } else if (!(obj instanceof Source)) {
        return false;
      }
      Source s = (Source)obj;
      return command.equals(s.command) && pool == s.pool;
    }

    @Override
    public int hashCode() {
      return 31 * command.hashCode() + pool;
    }
  }

  public static class Data extends DeviceDependentModel.Data {
    private static final long MAX_ADDR = -1;
    private static final int PAGE_SIZE = 0x10000;

    private final Client client;
    private final Source src;
    private final Observation[] observations;
    private final StructObservation[] structObservations;
    private final Map<Long, SoftReference<Segment>> cache = Maps.newHashMap();

    public Data(Path.Device device, Client client, Source src, Observation[] observations,
        StructObservation[] structObservations) {
      super(device);
      this.client = client;
      this.src = src;
      this.observations = observations;
      this.structObservations = structObservations;
    }

    public int getPool() {
      return src.pool;
    }

    public long getEndAddress() {
      return MAX_ADDR; // TODO
    }

    public Observation[] getObservations() {
      return observations;
    }

    public StructObservation[] getStructObservations() {
      return structObservations;
    }

    public ListenableFuture<Segment> load(long offset, int length) {
      length = (int)UnsignedLongs.min(MAX_ADDR - offset, length - 1) + 1;

      long firstPage = getPageForOffset(offset);
      long lastPage = getPageForOffset(offset + length - 1);
      if (firstPage == lastPage) {
        return getPage(firstPage, getOffsetInPage(offset), length);
      }
      List<ListenableFuture<Segment>> futures = Lists.newArrayList();
      futures.add(getPage(firstPage, getOffsetInPage(offset), PAGE_SIZE - getOffsetInPage(offset)));
      for (long page = firstPage + 1, left = length - PAGE_SIZE + getOffsetInPage(offset);
          page <= lastPage; page++, left -= PAGE_SIZE) {
        futures.add(getPage(page, 0, (int)Math.min(left, PAGE_SIZE)));
      }

      final int totalLength = length;
      return MoreFutures.transform(
          Futures.allAsList(futures), segments -> Segment.combine(segments, totalLength));
    }

    private ListenableFuture<Segment> getPage(long page, int offset, int length) {
      return MoreFutures.transform(getFromCacheOrServer(page),
          memory -> memory.subSegment(offset, length));
    }

    private ListenableFuture<Segment> getFromCacheOrServer(long page) {
      Segment cached = null;
      synchronized (cache) {
        SoftReference<Segment> reference = cache.get(page);
        if (reference != null) {
          cached = reference.get();
          if (cached == null) {
            cache.remove(page);
          }
        }
      }
      if (cached != null) {
        return Futures.immediateFuture(cached);
      }

      return MoreFutures.transform(getFromServer(page), mem -> {
        addToCache(page, mem);
        return mem;
      });
    }

    private void addToCache(long page, Segment data) {
      synchronized (cache) {
        cache.put(page, new SoftReference<Segment>(data));
      }
    }

    private ListenableFuture<Segment> getFromServer(long page) {
      return MoreFutures.transform(
          client.get(memoryAfter(src.command, src.pool, getOffsetForPage(page), PAGE_SIZE), device),
          Segment::new);
    }

    private static long getPageForOffset(long offset) {
      return Long.divideUnsigned(offset, PAGE_SIZE);
    }

    private static long getOffsetForPage(long page) {
      return page * PAGE_SIZE;
    }

    private static int getOffsetInPage(long offset) {
      return (int)Long.remainderUnsigned(offset, PAGE_SIZE);
    }
  }

  /**
   * A segment of memory data.
   */
  public static class Segment {
    private final byte[] data;
    private final BitSet known;
    private final int offset;
    private final int length;

    private final List<Service.MemoryRange> reads;
    private final List<Service.MemoryRange> writes;

    private Segment(byte[] data, BitSet known, int offset, int length,
        List<Service.MemoryRange> reads, List<Service.MemoryRange> writes) {
      this.data = data;
      this.offset = offset;
      this.length = length;
      this.known = known;
      this.reads = reads;
      this.writes = writes;
    }

    public Segment(Service.Value value) {
      Service.Memory mem = value.getMemory();
      data = mem.getData().toByteArray();
      offset = 0;
      known = computeKnown(mem);
      length = data.length;
      reads = merge(mem.getReadsList());
      writes = merge(mem.getWritesList());
    }

    public static Segment combine(List<Segment> segments, int length) {
      byte[] data = new byte[length];
      BitSet known = new BitSet(length);
      int done = 0;

      List<Service.MemoryRange> reads = Lists.newArrayList();
      List<Service.MemoryRange> writes = Lists.newArrayList();

      for (Iterator<Segment> it = segments.iterator(); it.hasNext() && done < length; ) {
        Segment segment = it.next();
        int count = Math.min(length - done, segment.length);
        System.arraycopy(segment.data, segment.offset, data, done, count);
        for (int i = 0; i < count; ++i) {
          known.set(done + i, segment.known.get(segment.offset + i));
        }

        for (Service.MemoryRange range : segment.reads) {
          reads.add((done == 0 && segment.offset == 0) ?
              range : memory(done - segment.offset + range.getBase(), range.getSize()));
        }
        for (Service.MemoryRange range : segment.writes) {
          writes.add((done == 0 && segment.offset == 0) ?
              range : memory(done - segment.offset + range.getBase(), range.getSize()));
        }

        done += count;
      }
      return new Segment(data, known, 0, done, merge(reads), merge(writes));
    }

    public Segment subSegment(int start, int count) {
      return new Segment(
          data, known, offset + start, Math.min(count, length - start), reads, writes);
    }

    public String asString(int start, int count) {
      return new String(
          data, offset + start, Math.min(count, length - start), Charset.forName("US-ASCII"));
    }

    public int length() {
      return length;
    }

    public boolean getByteKnown(int off, int size) {
      if (off < 0 || size < 0 || offset + off + size > data.length) {
        return false;
      }
      if (known == null) {
        return true;
      }
      for (int o = off; o < off + size; o++) {
        if (!known.get(offset + o)) {
          return false;
        }
      }
      return true;
    }

    public boolean getByteKnown(int off) {
      return getByteKnown(off, 1);
    }

    public int getByte(int off) {
      return data[offset + off] & 0xFF;
    }

    public boolean getShortKnown(int off) {
      return getByteKnown(off, 2);
    }

    public boolean getIntKnown(int off) {
      return getByteKnown(off, 4);
    }

    public int getShort(int off) {
      off += offset;
      // TODO: figure out BigEndian vs LittleEndian.
      return (data[off + 0] & 0xFF) |
          ((data[off + 1] & 0xFF) << 8);
    }

    public int getInt(int off) {
      off += offset;
      // TODO: figure out BigEndian vs LittleEndian.
      return (data[off + 0] & 0xFF) |
          ((data[off + 1] & 0xFF) << 8) |
          ((data[off + 2] & 0xFF) << 16) |
          (data[off + 3] << 24);
    }

    public boolean getLongKnown(int off) {
      return getByteKnown(off, 8);
    }

    public long getLong(int off) {
      // TODO: figure out BigEndian vs LittleEndian.
      return (getInt(off) & 0xFFFFFFFFL) | ((long)getInt(off + 4) << 32);
    }

    public boolean hasReads() {
      return !reads.isEmpty();
    }

    public boolean hasWrites() {
      return !writes.isEmpty();
    }

    public Iterator<Service.MemoryRange> getReads() {
      return adjustedRanges(reads);
    }

    public Iterator<Service.MemoryRange> getWrites() {
      return adjustedRanges(writes);
    }

    private Iterator<Service.MemoryRange> adjustedRanges(List<Service.MemoryRange> ranges) {
      return ranges.stream()
          .filter(r -> Ranges.overlap(r, offset, length))
          .map(r -> relative(offset, length, r))
          .iterator();
    }

    private static BitSet computeKnown(Service.Memory data) {
      BitSet known = new BitSet(data.getData().size());
      for (Service.MemoryRange rng : data.getObservedList()) {
        known.set((int)rng.getBase(), (int)rng.getBase() + (int)rng.getSize());
      }
      return known;
    }
  }

  /**
   * Read or write memory observation at a specific command.
   */
  public static class Observation {
    public static final Observation NULL_OBSERVATION = new Observation(null, false, null) {
      @Override
      public String toString() {
        return Messages.SELECT_OBSERVATION;
      }

      @Override
      public boolean contains(long address) {
        return false;
      }
    };

    private final CommandIndex index;
    private final boolean read;
    private final Service.MemoryRange range;

    public Observation(CommandIndex index, boolean read, Service.MemoryRange range) {
      this.index = index;
      this.read = read;
      this.range = range;
    }

    public Path.Memory getPath() {
      return Paths.memoryAfter(index, Application_VALUE, range).getMemory();
    }

    public boolean contains(long address) {
      return Ranges.contains(range, address);
    }

    @Override
    public String toString() {
      long base = range.getBase(), count = range.getSize();
      return (read ? "Read " : "Write ") + count + " byte" + (count == 1 ? "" : "s") +
          String.format(" at 0x%016x", base);
    }
  }

  /**
   * Structured memory observation, a lightweight data structure for struct memory, containing
   * all the needed information to query server to ask for decoded result.
   */
  public static class StructObservation {
    public final Service.TypedMemoryRange range;
    public final Source source;
    public final Path.Device device;

    public StructObservation(Service.TypedMemoryRange range, Source source, Path.Device device) {
      this.range = range;
      this.source = source;
      this.device = device;
    }

    public Service.TypedMemoryRange getRange() {
      return range;
    }
  }

  /**
   * Structured memory node, containing decoded struct memory information.
   */
  public static class StructNode {
    private static final int MAX_CHILDREN_SIZE = 100;

    private final Path.API api;
    private final TypeInfo.Type type;
    private final MemoryBox.Value value;
    private final long rootAddress;     // The root address of the observation this node belongs to.
    private final MemoryTypes typesModel;
    private List<StructNode> children;
    private String structName = "";     // Name information for node of type TypeInfo.StructField.
    private boolean isLargeArray = false;   // True if this node denotes a large array or slice.

    public StructNode(Path.API api, TypeInfo.Type type, MemoryBox.Value value, long rootAddress,
        MemoryTypes typesModel) {
      this.api = api;
      this.type = type;
      this.value = value;
      this.rootAddress = rootAddress;
      this.typesModel = typesModel;
      this.children = loadChildren();
    }

    public StructNode(Path.API api, TypeInfo.Type type, MemoryBox.Value value, long rootAddress,
        MemoryTypes typesModel, String name) {
      this.api = api;
      this.type = type;
      this.value = value;
      this.rootAddress = rootAddress;
      this.typesModel = typesModel;
      this.structName = name;
      this.children = loadChildren();
    }

    public TypeInfo.Type getType() {
      return type;
    }

    public TypeInfo.Type.TyCase getTypeCase() {
      return type.getTyCase();
    }

    public String getTypeName() {
      return type.getName();
    }

    public String getTypeFormatted() {
      return TypeInfos.format(type, value);
    }

    public MemoryBox.Value getValue() {
      return value;
    }

    public String getValueFormatted() {
      if (type.getTyCase() == TyCase.ENUM && type.getEnum().hasConstants()) {
        return ConstantSets.find(typesModel.constants.getConstants(type.getEnum().getConstants()),
            value.getPod()).getName();
      } else {
        return MemoryBoxes.format(value, rootAddress);
      }
    }

    public long getRootAddress() {
      return rootAddress;
    }

    public boolean hasChildren() {
      return children.size() > 0;
    }

    public List<StructNode> getChildren() {
      return children;
    }

    public void setStructName(String name) {
      structName = name;
    }

    public String getStructName() {
      return structName;
    }

    public boolean isLargeArray() {
      return isLargeArray;
    }

    private boolean mayHaveChildren() {
      TypeInfo.Type.TyCase tyCase = type.getTyCase();
      return tyCase == TypeInfo.Type.TyCase.SLICE || tyCase == TypeInfo.Type.TyCase.STRUCT ||
          tyCase == TypeInfo.Type.TyCase.ARRAY || tyCase == TypeInfo.Type.TyCase.PSEUDONYM;
    }

    private List<StructNode> loadChildren() {
      children = new ArrayList<Memory.StructNode>();
      if (!mayHaveChildren()) {
        return children;
      }

      TypeInfo.Type childType;
      switch (type.getTyCase()) {
        case SLICE:
          // Don't create and append children nodes if it's a large slice.
          if (value.getSlice().getValuesCount() < MAX_CHILDREN_SIZE) {
            TypeInfo.SliceType slice = type.getSlice();
            childType = typesModel.getType(type(slice.getUnderlying(), api));
            for (MemoryBox.Value childValue : value.getSlice().getValuesList()) {
              children.add(new StructNode(api, childType, childValue, rootAddress, typesModel));
            }
          } else {
            isLargeArray = true;
          }
          break;
        case STRUCT:
          TypeInfo.StructType struct = type.getStruct();
          List<TypeInfo.StructField> childrenTypes = struct.getFieldsList();
          List<MemoryBox.Value> childrenValues = value.getStruct().getFieldsList();
          for (int i = 0; i < childrenValues.size(); i++) {
            StructNode childNode = new StructNode(api,
                typesModel.getType(type(childrenTypes.get(i).getType(), api)), childrenValues.get(i),
                rootAddress, typesModel, childrenTypes.get(i).getName());
            children.add(childNode);
          }
          break;
        case ARRAY:
          // Don't create and append children nodes if it's a large array.
          if (value.getArray().getEntriesCount() < MAX_CHILDREN_SIZE) {
            TypeInfo.ArrayType array = type.getArray();
            childType = typesModel.getType(type(array.getElementType(), api));
            for (MemoryBox.Value childValue : value.getArray().getEntriesList()) {
              children.add(new StructNode(api, childType, childValue, rootAddress, typesModel));
            }
          } else {
            isLargeArray = true;
          }
          break;
        case PSEUDONYM:
          TypeInfo.PseudonymType pseudonym = type.getPseudonym();
          childType = typesModel.getType(type(pseudonym.getUnderlying(), api));
          StructNode childNode = new StructNode(api, childType, value, rootAddress, typesModel);
          children.add(childNode);
          childNode.setStructName(structName);
          break;
        default:
          break;
      }
      return children;
    }

    /**
     * Utility method. Simplify trees, especially for vulkan structs.
     * 1. Remove redundant layers for all the trees.
     * 2. Combine trees together by appending some smaller trees to the main trees, if they are
     *    related through a pointer field.
     */
    public static List<StructNode> simplifyTrees(List<StructNode> trees) {
      // Remove redundant layers.
      List<StructNode> allNodes = Lists.newArrayList();
      Map<Long, List<StructNode>> nodesMap = new HashMap<Long, List<StructNode>>();
      for (StructNode tree : trees) {
        StructNode simpleTree = removeExtraLayers(tree);
        nodesMap.putIfAbsent(tree.getRootAddress(), Lists.newArrayList());
        nodesMap.get(tree.getRootAddress()).add(simpleTree);
        allNodes.add(simpleTree);
      }
      // Append pointed nodes to corresponding pointer field if possible.
      Set<StructNode> appended = Sets.newHashSet(); // Whether the node is appended to other nodes, if so, it shouldn't get displayed a second time as a basic independent tree.
      Set<StructNode> treated = Sets.newHashSet();  // Whether the node got treated so that all of its children pointer field got appended.
      for (StructNode node : allNodes) {
        appendPointedNodes(node, nodesMap, appended, treated);
      }
      return allNodes.stream().filter(n -> !appended.contains(n)).collect(Collectors.toList());
    }

    /**
     * Remove redundant layers and return the new root.
     */
    private static StructNode removeExtraLayers(StructNode root) {
      TypeInfo.Type.TyCase tyCase = root.getTypeCase();
      // Remove the outer layer for SLICE type. E.g. {[()]} -> [()].
      if ((tyCase == TyCase.SLICE) && root.hasChildren() && root.children.size() == 1
          && root.children.get(0).hasChildren()) {
        root = removeExtraLayers(root.children.get(0));
      }
      // Treat PSEUDONYM types as leaves, and remove any children they may have had.
      if (tyCase == TyCase.PSEUDONYM) {
        root.children.clear();
      }
      for (int i = 0; i < root.children.size(); i++) {
        root.children.set(i, removeExtraLayers(root.children.get(i)));
      }
      return root;
    }

    /**
     * Check whether the tree contains any node with type TypeInfo.StructType.
     */
    private static boolean containsStructType(StructNode root) {
      if (root == null) {
        return false;
      }
      if (root.getTypeCase() == TyCase.STRUCT) {
        return true;
      }
      for (StructNode child : root.getChildren()) {
        if (containsStructType(child)) {
          return true;
        }
      }
      return false;
    }

    /**
     * Find all the nodes with type TypeInfo.PointerType in this tree, append the pointed tree to
     * these nodes if possible.
     */
    private static void appendPointedNodes(StructNode root, Map<Long, List<StructNode>> nodesMap,
        Set<StructNode> appended, Set<StructNode> treated) {
      if (root == null || treated.contains(root)) {
        return;
      }
      if (root.getTypeCase() == TyCase.POINTER) {
        long pointedAddress = root.getValue().getPointer().getAddress();
        if (nodesMap.containsKey(pointedAddress)) {
          root.getChildren().addAll(nodesMap.get(pointedAddress));
          appended.addAll(nodesMap.get(pointedAddress));
        }
      }
      treated.add(root);
      for (StructNode child : root.getChildren()) {
        appendPointedNodes(child, nodesMap, appended, treated);
      }
    }
  }

  public static interface Listener extends Events.Listener {
    /**
     * Event indicating that memory data is being loaded.
     */
    public default void onMemoryLoadingStart() { /* empty */ }

    /**
     * Event indicating that the memory data has finished loading.
     */
    public default void onMemoryLoaded() { /* empty */ }
  }
}
