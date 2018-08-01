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

import static com.google.gapid.util.Paths.memoryAfter;
import static com.google.gapid.util.Ranges.memory;
import static com.google.gapid.util.Ranges.merge;
import static com.google.gapid.util.Ranges.relative;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.primitives.UnsignedLongs;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.CommandStream.Observation;
import com.google.gapid.proto.service.Service;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.Ranges;

import org.eclipse.swt.widgets.Shell;

import java.lang.ref.SoftReference;
import java.nio.charset.Charset;
import java.util.BitSet;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.logging.Logger;

/**
 * Model responsible for loading memory pool data. This model requests segments as equally sized
 * pages and maintains a cache of fetched pages.
 */
public class Memory extends ModelBase<Memory.Data, Memory.Source, Void, Memory.Listener> {
  private static final Logger LOG = Logger.getLogger(Memory.class.getName());

  private final CommandStream commands;

  public Memory(Shell shell, Analytics analytics, Client client, CommandStream commands) {
    super(LOG, shell, analytics, client, Listener.class);
    this.commands = commands;

    commands.addListener(new CommandStream.Listener() {
      @Override
      public void onCommandsSelected(CommandIndex selection) {
        load(new Source(selection, getPool()), false);
      }
    });
  }

  public int getPool() {
    Source source = getSource();
    return (source == null) ? 0 : source.pool;
  }

  public void setPool(int pool) {
    Source source = getSource();
    if (source != null) {
      load(source.withPool(pool), false);
    }
  }

  @Override
  protected ListenableFuture<Data> doLoad(Source source) {
    return Futures.transform(commands.getObservations(source.command),
        obs -> new Data(client, source, obs));
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

    public Source withPool(int newPool) {
      return new Source(command, newPool);
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

  public static class Data {
    private static final long MAX_ADDR = -1;
    private static final int PAGE_SIZE = 0x10000;

    private final Client client;
    private final Source src;
    private final Observation[] observations;
    private final Map<Long, SoftReference<Segment>> cache = Maps.newHashMap();

    public Data(Client client, Source src, Observation[] observations) {
      this.client = client;
      this.src = src;
      this.observations = observations;
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
      return Futures.transform(
          Futures.allAsList(futures), segments -> Segment.combine(segments, totalLength));
    }

    private ListenableFuture<Segment> getPage(long page, int offset, int length) {
      return Futures.transform(getFromCacheOrServer(page),
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

      return Futures.transform(getFromServer(page), mem -> {
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
      return Futures.transform(
          client.get(memoryAfter(src.command, src.pool, getOffsetForPage(page), PAGE_SIZE)),
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
