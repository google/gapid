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
package com.google.gapid.views;

import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.util.Paths.command;
import static com.google.gapid.util.Ranges.last;
import static com.google.gapid.util.Ranges.memory;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.ifNotDisposed;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.primitives.UnsignedLong;
import com.google.common.primitives.UnsignedLongs;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.common.util.concurrent.MoreExecutors;
import com.google.gapid.Server.GapisInitException;
import com.google.gapid.models.AtomStream;
import com.google.gapid.models.AtomStream.TypedObservation;
import com.google.gapid.models.Capture;
import com.google.gapid.models.Follower;
import com.google.gapid.models.Models;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.CommandRange;
import com.google.gapid.proto.service.Service.MemoryInfo;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;
import com.google.gapid.util.IntRange;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.util.Paths;
import com.google.gapid.widgets.InfiniteScrolledComposite;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.resource.JFaceResources;
import org.eclipse.jface.viewers.ArrayContentProvider;
import org.eclipse.jface.viewers.ComboViewer;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.StructuredSelection;
import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.graphics.Font;
import org.eclipse.swt.graphics.GC;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Combo;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Label;

import java.lang.ref.SoftReference;
import java.math.BigInteger;
import java.nio.ByteOrder;
import java.util.Arrays;
import java.util.BitSet;
import java.util.Collections;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.NoSuchElementException;
import java.util.function.Consumer;

public class MemoryView extends Composite
    implements Capture.Listener, AtomStream.Listener, Follower.Listener {
  private final Client client;
  private final Models models;
  private final Selections selections;
  private final MemoryPanel memoryPanel;
  protected final LoadablePanel<InfiniteScrolledComposite> loading;
  protected final InfiniteScrolledComposite memoryScroll;
  private final State uiState = new State();
  private MemoryDataModel memoryData;

  public MemoryView(Composite parent, Client client, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.client = client;
    this.models = models;

    memoryPanel = new MemoryPanel(this, new Loadable() {
      @Override
      public void startLoading() {
        ifNotDisposed(loading, loading::startLoading);
      }

      @Override
      public void stopLoading() {
        scheduleIfNotDisposed(loading, () -> {
          if (loading.isLoading()) {
            loading.stopLoading();
            memoryScroll.redraw();
          }
        });
      }
    }, widgets.theme);
    setLayout(new GridLayout(1, true));

    selections = new Selections(this, this::setDataType, this::setObservation);
    loading = LoadablePanel.create(this, widgets,
        panel -> new InfiniteScrolledComposite(panel, SWT.H_SCROLL | SWT.V_SCROLL, memoryPanel));
    memoryScroll = loading.getContents();

    selections.setLayoutData(new GridData(SWT.FILL, SWT.TOP, true, false));
    loading.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    models.capture.addListener(this);
    models.atoms.addListener(this);
    models.follower.addListener(this);
  }

  @Override
  public void onCaptureLoadingStart() {
    loading.showMessage(Info, Messages.LOADING_CAPTURE);
  }

  @Override
  public void onCaptureLoaded(GapisInitException error) {
    if (error != null) {
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    }
  }

  @Override
  public void onAtomsLoaded() {
    loading.showMessage(Info, Messages.SELECT_MEMORY);
  }

  @Override
  public void onAtomsSelected(CommandRange range) {
    TypedObservation[] obs = models.atoms.getObservations(last(range));
    selections.setObservations(obs);

    if (obs.length > 0 && !uiState.isComplete()) {
      // If the memory view is not showing anything yet, show the first observation.
      setObservation(obs[0]);
    } else {
      uiState.update(command(models.atoms.getPath(), range));
      update(getCurrentAddress());
    }
  }

  @Override
  public void onMemoryFollowed(Path.Memory path) {
    uiState.update(path);
    update(path.getAddress());
  }

  private void setDataType(DataType dataType) {
    if (uiState.setDataType(dataType)) {
      update(getCurrentAddress());
    }
  }

  private void setObservation(TypedObservation obs) {
    Path.Memory memoryPath = obs.getPath(models.atoms);
    uiState.update(memoryPath);
    update(memoryPath.getAddress());
  }

  private void update(long address) {
    if (!uiState.isComplete()) {
      loading.showMessage(Info, Messages.SELECT_MEMORY);
      return;
    }

    loading.stopLoading();
    selections.setPool(uiState.pool);
    memoryData = uiState.createMemoryDataModel(client);
    selections.setDataType(uiState.dataType);
    memoryPanel.setModel(uiState.getMemoryModel(memoryData));
    memoryScroll.updateMinSize();
    getDisplay().asyncExec(() -> goToAddress(address));
    selections.updateSelectedObservation(address);
  }

  private void goToAddress(long address) {
    memoryScroll.scrollTo(BigInteger.ZERO, UnsignedLong.fromLongBits(address).bigIntegerValue()
        .divide(BigInteger.valueOf(FixedMemoryModel.BYTES_PER_ROW))
        .multiply(BigInteger.valueOf(memoryPanel.lineHeight)));
  }

  private long getCurrentAddress() {
    if (memoryData == null) {
      return 0;
    }

    return memoryScroll.getScrollLocation().y
        .divide(BigInteger.valueOf(memoryPanel.lineHeight))
        .multiply(BigInteger.valueOf(FixedMemoryModel.BYTES_PER_ROW))
        .add(UnsignedLong.fromLongBits(memoryData.getAddress()).bigIntegerValue()).longValue();
  }

  private static class Selections extends Composite {
    private final Label poolLabel;
    private final Combo typeCombo;
    private final Label obsLabel;
    private final ComboViewer obsCombo;

    public Selections(Composite parent, Consumer<DataType> dataTypeListener,
        Consumer<TypedObservation> observationListener) {
      super(parent, SWT.NONE);
      setLayout(new GridLayout(6, false));

      createLabel(this, "Pool:").setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, false, false));
      poolLabel = createLabel(this, "0");
      poolLabel.setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, false, false));

      createLabel(this, "Type:").setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, false, false));
      typeCombo = DataType.createCombo(this);

      obsLabel = createLabel(this, "Range:");
      obsCombo = createObservationSelector();

      obsLabel.setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, false, false));
      typeCombo.setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, false, false));

      typeCombo.addListener(SWT.Selection,
          e -> dataTypeListener.accept(DataType.values()[typeCombo.getSelectionIndex()]));
      obsCombo.getCombo().addListener(SWT.Selection, e -> {
        TypedObservation obs =
            (TypedObservation)obsCombo.getElementAt(obsCombo.getCombo().getSelectionIndex());
        if (obs != TypedObservation.NULL_OBSERVATION) {
          observationListener.accept(obs);
        }
      });

      obsLabel.setVisible(false);
      obsCombo.getCombo().setVisible(false);
    }

    private ComboViewer createObservationSelector() {
      ComboViewer combo = new ComboViewer(this, SWT.READ_ONLY);
      combo.setContentProvider(ArrayContentProvider.getInstance());
      combo.setLabelProvider(new LabelProvider());
      combo.setUseHashlookup(true);
      combo.getCombo().setVisibleItemCount(10);
      combo.getCombo().setLayoutData(new GridData(SWT.FILL, SWT.CENTER, false, false));
      return combo;
    }

    public void setDataType(DataType dataType) {
      typeCombo.select(dataType.ordinal());
    }

    public void setObservations(TypedObservation[] observations) {
      if (observations.length == 0) {
        obsCombo.setInput(Arrays.asList(observations));
      } else {
        obsCombo.setInput(Lists.asList(TypedObservation.NULL_OBSERVATION, observations));
      }

      obsLabel.setVisible(observations.length != 0);
      obsCombo.getCombo().setVisible(observations.length != 0);
      obsCombo.getControl().requestLayout();
    }

    @SuppressWarnings("unchecked")
    public void updateSelectedObservation(long address) {
      for (TypedObservation obs : (Iterable<TypedObservation>)obsCombo.getInput()) {
        if (obs.contains(address)) {
          obsCombo.setSelection(new StructuredSelection(obs), true);
          return;
        }
      }
      if (obsCombo.getCombo().getItemCount() > 0) {
        obsCombo.setSelection(new StructuredSelection(TypedObservation.NULL_OBSERVATION));
      }
    }

    public void setPool(int pool) {
      poolLabel.setText(String.valueOf(pool));
      poolLabel.requestLayout();
    }
  }

  private static class State {
    public DataType dataType = DataType.Bytes;
    public Path.Command atomPath;
    public int pool = -1;
    public long offset = -1;
    public long lastAddress;

    public State() {
    }

    public void update(Path.Memory memoryPath) {
      if (memoryPath != null) {
        pool = memoryPath.getPool();
        offset = Long.remainderUnsigned(memoryPath.getAddress(), FixedMemoryModel.BYTES_PER_ROW);
        lastAddress = UnsignedLong.MAX_VALUE.longValue(); // TODO
        atomPath = memoryPath.getAfter();
      }
    }

    public void update(Path.Command newAtomPath) {
      if (newAtomPath != null) {
        atomPath = newAtomPath;
      }
    }

    public boolean setDataType(DataType newType) {
      if (dataType != newType) {
        dataType = newType;
        return true;
      }
      return false;
    }

    public boolean isComplete() {
      return atomPath != null && offset >= 0 && pool >= 0;
    }

    public MemoryDataModel createMemoryDataModel(Client client) {
      final Path.Command curAtomPath = atomPath;
      final int curPool = pool;
      PagedMemoryDataModel.MemoryFetcher fetcher = (address, count) ->
          Futures.transform(client.get(Paths.memoryAfter(curAtomPath, curPool, address, count)),
              value -> value.getMemoryInfo());


      return new PagedMemoryDataModel(fetcher, offset, lastAddress);
    }

    public MemoryModel getMemoryModel(MemoryDataModel data) {
      return dataType.getMemoryModel(data);
    }
  }

  private static enum DataType {
    Bytes() {
      @Override
      public MemoryModel getMemoryModel(MemoryDataModel memory) {
        return new BytesMemoryModel(memory);
      }
    }, Shorts() {
      @Override
      public MemoryModel getMemoryModel(MemoryDataModel memory) {
        return new ShortsMemoryModel(memory);
      }
    }, Ints() {
      @Override
      public MemoryModel getMemoryModel(MemoryDataModel memory) {
        return new IntsMemoryModel(memory);
      }
    }, Longs() {
      @Override
      public MemoryModel getMemoryModel(MemoryDataModel memory) {
        return new LongsMemoryModel(memory);
      }
    }, Floats() {
      @Override
      public MemoryModel getMemoryModel(MemoryDataModel memory) {
        return new FloatsMemoryModel(memory);
      }
    }, Doubles() {
      @Override
      public MemoryModel getMemoryModel(MemoryDataModel memory) {
        return new DoublesMemoryModel(memory);
      }
    };

    public abstract MemoryModel getMemoryModel(MemoryDataModel memory);

    public static Combo createCombo(Composite parent) {
      Combo combo = new Combo(parent, SWT.READ_ONLY);
      String[] names = new String[values().length];
      for (int i = 0; i < names.length; i++) {
        names[i] = values()[i].name();
      }
      combo.setItems(names);
      combo.select(0);
      return combo;
    }
  }

  private static class MemoryPanel implements InfiniteScrolledComposite.Scrollable {
    public final int lineHeight;
    private final int[] charOffset = new int[256];

    private final Loadable loadable;
    private final Theme theme;
    private final Font font;
    private MemoryModel model;

    public MemoryPanel(Composite parent, Loadable loadable, Theme theme) {
      this.loadable = loadable;
      this.theme = theme;
      font = JFaceResources.getFont(JFaceResources.TEXT_FONT);
      GC gc = new GC(parent);
      gc.setFont(font);
      lineHeight = gc.getFontMetrics().getHeight();
      StringBuilder sb = new StringBuilder();
      for (int i = 1; i < charOffset.length; i++) {
        charOffset[i] = gc.stringExtent(sb.append('0').toString()).x;
      }
      gc.dispose();
    }

    public void setModel(MemoryModel model) {
      this.model = model;
    }

    @Override
    public BigInteger getWidth() {
      return (model == null) ?
          BigInteger.ZERO : BigInteger.valueOf(charOffset[model.getLineLength()]);
    }

    @Override
    public BigInteger getHeight() {
      return (model == null) ? BigInteger.ZERO :
        BigInteger.valueOf(model.getLineCount()).multiply(BigInteger.valueOf(lineHeight));
    }

    @Override
    public void paint(BigInteger xOffset, BigInteger yOffset, GC gc) {
      if (model == null) {
        return;
      }

      gc.setFont(font);
      Rectangle clip = gc.getClipping();
      BigInteger startY = yOffset.add(BigInteger.valueOf(clip.y));
      long startRow = startY.divide(BigInteger.valueOf(lineHeight))
          .max(BigInteger.ZERO).min(BigInteger.valueOf(model.getLineCount() - 1)).longValueExact();
      long endRow = startY.add(BigInteger.valueOf(clip.height + lineHeight - 1))
          .divide(BigInteger.valueOf(lineHeight))
          .max(BigInteger.ZERO).min(BigInteger.valueOf(model.getLineCount())).longValueExact();

      Color background = gc.getBackground();
      gc.setBackground(theme.memoryReadHighlight());
      for (Selection read : model.getReads(startRow, endRow, loadable)) {
        highlight(gc, yOffset, read);
      }

      gc.setBackground(theme.memoryWriteHighlight());
      for (Selection write : model.getWrites(startRow, endRow, loadable)) {
        highlight(gc, yOffset, write);
      }
      gc.setBackground(background);

      int y = getY(startRow, yOffset);
      Iterator<Segment> it = model.getLines(startRow, endRow, loadable);
      for (; it.hasNext(); y += lineHeight) {
        Segment segment = it.next();
        gc.drawString(new String(segment.array, segment.offset, segment.count), 0, y, true);
      }
    }

    private int getY(long line, BigInteger yOffset) {
      return BigInteger.valueOf(line)
          .multiply(BigInteger.valueOf(lineHeight)).subtract(yOffset).intValueExact();
    }

    private void highlight(GC gc, BigInteger yOffset, Selection selection) {
      if (selection.startRow == selection.endRow) {
        int so = charOffset[selection.startCol], eo = charOffset[selection.endCol];
        gc.fillRectangle(so, getY(selection.startRow, yOffset), eo - so, lineHeight);
      } else {
        int so = charOffset[selection.startCol], eo = charOffset[selection.endCol];
        int fo = charOffset[selection.range.from], to = charOffset[selection.range.to];
        gc.fillRectangle(so, getY(selection.startRow, yOffset), to - so, lineHeight);
        gc.fillRectangle(fo, getY(selection.endRow, yOffset), eo - fo, lineHeight);
        gc.fillRectangle(fo, getY(selection.startRow + 1, yOffset), to - fo,
            (int)(selection.endRow - selection.startRow - 1) * lineHeight);
      }
    }
  }

  private interface MemoryDataModel {
    long getAddress();
    long getEndAddress();
    ListenableFuture<MemorySegment> get(long offset, int length);
    MemoryDataModel align(int byteAlign);
  }

  private static class PagedMemoryDataModel implements MemoryDataModel {
    private static final int PAGE_SIZE = 0x10000;

    private final MemoryFetcher fetcher;
    private final long address;
    private final long lastAddress;

    private final Map<Long, SoftReference<ListenableFuture<MemoryInfo>>> cache = Maps.newHashMap();

    public PagedMemoryDataModel(MemoryFetcher fetcher, long address, long lastAddress) {
      this.fetcher = fetcher;
      this.address = address;
      this.lastAddress = lastAddress;
    }

    @Override
    public long getAddress() {
      return address;
    }

    @Override
    public long getEndAddress() {
      return lastAddress;
    }

    @Override
    public ListenableFuture<MemorySegment> get(long offset, int length) {
      offset = UnsignedLongs.min(lastAddress - address, offset);
      length = (int)UnsignedLongs.min(lastAddress - address - offset, length - 1) + 1;

      long firstPage = getPageForOffset(offset);
      long lastPage = getPageForOffset(offset + length - 1);
      if (firstPage == lastPage) {
        return getPage(firstPage, getOffsetInPage(offset), length);
      }
      List<ListenableFuture<MemorySegment>> futures = Lists.newArrayList();
      futures.add(getPage(firstPage, getOffsetInPage(offset), PAGE_SIZE - getOffsetInPage(offset)));
      for (long page = firstPage + 1, left = length - PAGE_SIZE + getOffsetInPage(offset);
          page <= lastPage; page++, left -= PAGE_SIZE) {
        futures.add(getPage(page, 0, (int)Math.min(left, PAGE_SIZE)));
      }

      final int totalLength = length;
      return Futures.transform(
          Futures.allAsList(futures), segments -> MemorySegment.combine(segments, totalLength));
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

    private ListenableFuture<MemorySegment> getPage(long page, int offset, int length) {
      ListenableFuture<Service.MemoryInfo> mem = getFromCache(page);
      if (mem == null) {
        long base = address + getOffsetForPage(page);
        mem = fetcher.get(base, (int)UnsignedLongs.min(lastAddress - base, PAGE_SIZE - 1) + 1);
        addToCache(page, mem);
      }
      return Futures.transform(mem, memory -> new MemorySegment(memory).subSegment(offset, length));
    }

    private ListenableFuture<Service.MemoryInfo> getFromCache(long page) {
      ListenableFuture<Service.MemoryInfo> result = null;
      synchronized (cache) {
        SoftReference<ListenableFuture<MemoryInfo>> reference = cache.get(page);
        if (reference != null) {
          result = reference.get();
          if (result == null) {
            cache.remove(page);
          }
        }
      }
      return result;
    }

    private void addToCache(long page, ListenableFuture<MemoryInfo> data) {
      synchronized (cache) {
        cache.put(page, new SoftReference<ListenableFuture<MemoryInfo>>(data));
      }
    }

    @Override
    public MemoryDataModel align(int byteAlign) {
      return this;
    }

    public interface MemoryFetcher {
      ListenableFuture<MemoryInfo> get(long address, long count);
    }
  }

  private static interface MemoryModel {
    long getLineCount();
    int getLineLength();
    Iterator<Segment> getLines(long start, long end, Loadable loadable);
    IntRange getSelectableRegion(int column);
    Selection[] getReads(long startRow, long endRow, Loadable loadable);
    Selection[] getWrites(long startRow, long endRow, Loadable loadable);
  }

  private static abstract class FixedMemoryModel implements MemoryModel {
    protected static final char UNKNOWN_CHAR = '?';
    protected static final int BYTES_PER_ROW = 16;
    private final static Selection[] NO_SELECTIONS = new Selection[0];

    protected final MemoryDataModel data;
    protected final long rows;

    public FixedMemoryModel(MemoryDataModel data) {
      this.data = data;
      this.rows =
          UnsignedLong.fromLongBits(data.getEndAddress() - data.getAddress()).bigIntegerValue()
          .add(BigInteger.valueOf(BYTES_PER_ROW))
          .divide(BigInteger.valueOf(BYTES_PER_ROW))
          .longValueExact();
    }

    @Override
    public long getLineCount() {
      return rows;
    }

    private MemorySegment getMemorySegment(long startRow, long endRow, Loadable loadable) {
      if (startRow < 0 || endRow < startRow || endRow > getLineCount()) {
        throw new IndexOutOfBoundsException(
            "[" + startRow + ", " + endRow + ") outside of [0, " + getLineCount() + ")");
      }
      ListenableFuture<MemorySegment> future =
          data.get(startRow * BYTES_PER_ROW, (int)(endRow - startRow) * BYTES_PER_ROW);
      if (future.isDone()) {
        loadable.stopLoading();
        return Futures.getUnchecked(future);
      } else {
        loadable.startLoading();
        future.addListener(loadable::stopLoading, MoreExecutors.directExecutor());
        return null;
      }
    }

    @Override
    public Iterator<Segment> getLines(long startRow, long endRow, Loadable loadable) {
      MemorySegment segment = getMemorySegment(startRow, endRow, loadable);
      if (segment != null) {
        return getLines(startRow, endRow, segment);
      } else {
        return Collections.emptyIterator();
      }
    }

    protected Iterator<Segment> getLines(long startRow, long endRow, MemorySegment memory) {
      return new Iterator<Segment>() {
        private long pos = startRow;
        private int offset = 0;
        private final Segment segment = new Segment(null, 0, 0);

        @Override
        public boolean hasNext() {
          return pos < endRow;
        }

        @Override
        public Segment next() {
          if (!hasNext()) {
            throw new NoSuchElementException();
          }
          getLine(segment, memory.subSegment(offset, BYTES_PER_ROW), pos);
          pos++;
          offset += BYTES_PER_ROW;
          return segment;
        }

        @Override
        public void remove() {
          throw new UnsupportedOperationException();
        }
      };
    }

    protected abstract void getLine(Segment segment, MemorySegment memory, long line);

    protected abstract IntRange[] getDataRanges();

    @Override
    public Selection[] getReads(long startRow, long endRow, Loadable loadable) {
      MemorySegment memory = getMemorySegment(startRow, endRow, loadable);
      return memory == null || memory.reads == null || memory.reads.isEmpty() ? NO_SELECTIONS
          : getSelections(memory.reads, (startRow * BYTES_PER_ROW) - memory.offset);
    }

    @Override
    public Selection[] getWrites(long startRow, long endRow, Loadable loadable) {
      MemorySegment memory = getMemorySegment(startRow, endRow, loadable);
      return memory == null || memory.writes == null || memory.writes.isEmpty() ? NO_SELECTIONS
          : getSelections(memory.writes, (startRow * BYTES_PER_ROW) - memory.offset);
    }

    private Selection[] getSelections(List<Service.MemoryRange> operation, long offset) {
      IntRange[] ranges = getDataRanges();
      Selection[] shapes = new Selection[operation.size() * ranges.length];
      for (int ri = 0; ri < ranges.length; ri++) {
        for (int oi = 0; oi < operation.size(); oi++) {
          Service.MemoryRange memoryRange = operation.get(oi);
          long startOffset = offset + memoryRange.getBase();
          int startCol = getColForOffset(ranges[ri], startOffset, true);
          long endOffset = offset + memoryRange.getBase() + memoryRange.getSize();
          int endCol = getColForOffset(ranges[ri], endOffset, false);
          shapes[ri * operation.size() + oi] = new Selection(ranges[ri], startCol,
              Long.divideUnsigned(startOffset, BYTES_PER_ROW), endCol,
              Long.divideUnsigned(endOffset, BYTES_PER_ROW));
        }
      }
      return shapes;
    }

    private static int getColForOffset(IntRange range, long offset, boolean start) {
      double positionOffset = (range.to - range.from) *
          ((double)Long.remainderUnsigned(offset, BYTES_PER_ROW)) / BYTES_PER_ROW;
      return range.from + (int)(start ? Math.ceil(positionOffset) : positionOffset);
    }
  }

  private static abstract class CharBufferMemoryModel extends FixedMemoryModel {
    protected static final int CHARS_PER_ADDRESS = 16; // 8 byte addresses
    protected static final int ADDRESS_SEPARATOR = 1;
    protected static final int ADDRESS_CHARS = CHARS_PER_ADDRESS + ADDRESS_SEPARATOR;
    protected static final IntRange ADDRESS_RANGE = new IntRange(0, CHARS_PER_ADDRESS);
    protected static final char[] HEX_DIGITS = "0123456789abcdef".toCharArray();

    protected final int charsPerRow;
    protected final IntRange memoryRange;

    public CharBufferMemoryModel(MemoryDataModel data, int charsPerRow, IntRange memoryRange) {
      super(data);
      this.charsPerRow = charsPerRow;
      this.memoryRange = memoryRange;
    }

    @Override
    public int getLineLength() {
      return charsPerRow;
    }

    protected static void appendUnknown(StringBuilder str, int n) {
      for (int i = 0; i < n; i++) {
        str.append(UNKNOWN_CHAR);
      }
    }

    @Override
    protected void getLine(Segment segment, MemorySegment memory, long line) {
      segment.array = new char[charsPerRow];
      segment.offset = 0;
      segment.count = charsPerRow;
      formatLine(segment.array, memory, line);
    }

    private void formatLine(char[] array, MemorySegment memory, long line) {
      Arrays.fill(array, ' ');
      long address = data.getAddress() + line * BYTES_PER_ROW;
      for (int i = CHARS_PER_ADDRESS - 1; i >= 0; i--, address >>>= 4) {
        array[i] = HEX_DIGITS[(int)address & 0xF];
      }
      array[CHARS_PER_ADDRESS] = ':';
      formatMemory(array, memory);
    }

    protected abstract void formatMemory(char[] buffer, MemorySegment memory);

    @Override
    public IntRange getSelectableRegion(int column) {
      if (ADDRESS_RANGE.isWithin(column)) {
        return ADDRESS_RANGE;
      } else if (memoryRange.isWithin(column)) {
        return memoryRange;
      }
      return null;
    }

    @Override
    public IntRange[] getDataRanges() {
      return new IntRange[] { memoryRange };
    }
  }

  private static class BytesMemoryModel extends CharBufferMemoryModel {
    private static final int CHARS_PER_BYTE = 2; // 2 hex chars per byte

    private static final int BYTE_SEPARATOR = 1;
    private static final int ASCII_SEPARATOR = 2;

    private static final int BYTES_CHARS = (CHARS_PER_BYTE + BYTE_SEPARATOR) * BYTES_PER_ROW;
    private static final int ASCII_CHARS = BYTES_PER_ROW + ASCII_SEPARATOR;
    private static final int CHARS_PER_ROW = ADDRESS_CHARS + BYTES_CHARS + ASCII_CHARS;

    private static final IntRange BYTES_RANGE =
        new IntRange(ADDRESS_CHARS + BYTE_SEPARATOR, ADDRESS_CHARS + BYTES_CHARS);
    private static final IntRange ASCII_RANGE =
        new IntRange(ADDRESS_CHARS + BYTES_CHARS + ASCII_SEPARATOR, CHARS_PER_ROW);

    public BytesMemoryModel(MemoryDataModel data) {
      super(data, CHARS_PER_ROW, BYTES_RANGE);
    }

    @Override
    public IntRange getSelectableRegion(int column) {
      if (ASCII_RANGE.isWithin(column)) {
        return ASCII_RANGE;
      } else {
        return super.getSelectableRegion(column);
      }
    }

    @Override
    protected void formatMemory(char[] buffer, MemorySegment memory) {
      for (int i = 0, j = ADDRESS_CHARS; i < memory.length;
          i++, j += CHARS_PER_BYTE + BYTE_SEPARATOR) {
        int b = memory.getByte(i);
        if (memory.getByteKnown(i)) {
          buffer[j + 1] = HEX_DIGITS[(b >> 4) & 0xF];
          buffer[j + 2] = HEX_DIGITS[(b >> 0) & 0xF];
        } else {
          buffer[j + 1] = UNKNOWN_CHAR;
          buffer[j + 2] = UNKNOWN_CHAR;
        }
      }

      for (int i = 0, j = ADDRESS_CHARS + BYTES_CHARS + ASCII_SEPARATOR; i < memory.length;
          i++, j++) {
        int b = memory.getByte(i);
        buffer[j] = memory.getByteKnown(i) && (b >= 32 && b < 127) ? (char)b : '.';
      }
    }

    @Override
    public IntRange[] getDataRanges() {
      return new IntRange[] { memoryRange, ASCII_RANGE };
    }
  }

  private static class IntegersMemoryModel extends CharBufferMemoryModel {
    // Little endian assumption
    private static final ByteOrder ENDIAN = ByteOrder.LITTLE_ENDIAN;
    private static final int ITEM_SEPARATOR = 1;
    private final int size;

    /**
     * @return the number of characters used to represent a single item
     */
    private static int charsPerItem(int size) {
      return size * 2;
    }

    /**
     * @return the number of characters per row needed for items including separators.
     */
    private static int itemChars(int size) {
      int itemsPerRow = BYTES_PER_ROW / size;
      int charsPerItem = charsPerItem(size); // hex chars per item
      return (charsPerItem + ITEM_SEPARATOR) * itemsPerRow;
    }

    /**
     * @return the number of characters per row including the address and items.
     */
    private static int charsPerRow(int size) {
      return ADDRESS_CHARS + itemChars(size);
    }

    /**
     * @return the character column range where the items are present
     */
    private static IntRange itemsRange(int size) {
      return new IntRange(ADDRESS_CHARS + ITEM_SEPARATOR, ADDRESS_CHARS + itemChars(size));
    }

    protected IntegersMemoryModel(MemoryDataModel data, int size) {
      super(data.align(size), charsPerRow(size), itemsRange(size));
      this.size = size;
    }

    @Override
    protected void formatMemory(char[] buffer, MemorySegment memory) {
      final int charsPerItem = charsPerItem(size);
      for (int i = 0, j = ADDRESS_CHARS; i + size <= memory.length;
          i += size, j += charsPerItem + ITEM_SEPARATOR) {
        if (memory.getByteKnown(i, size)) {
          for (int k = 0; k < size; ++k) {
            int val = memory.getByte(i + k);
            int chOff = ENDIAN.equals(ByteOrder.LITTLE_ENDIAN) ?
                charsPerItem(size - 1 - k) : charsPerItem(k);
            buffer[j + chOff + 1] = HEX_DIGITS[(val >> 4) & 0xF];
            buffer[j + chOff + 2] = HEX_DIGITS[val & 0xF];
          }
        } else {
          for (int k = 0; k < charsPerItem; ++k) {
            buffer[j + k + 1] = UNKNOWN_CHAR;
          }
        }
      }
    }
  }

  private static class ShortsMemoryModel extends IntegersMemoryModel {
    public ShortsMemoryModel(MemoryDataModel data) {
      super(data, 2);
    }
  }

  private static class IntsMemoryModel extends IntegersMemoryModel {
    public IntsMemoryModel(MemoryDataModel data) {
      super(data, 4);
    }
  }

  private static class LongsMemoryModel extends IntegersMemoryModel {
    public LongsMemoryModel(MemoryDataModel data) {
      super(data, 8);
    }
  }

  private static class FloatsMemoryModel extends CharBufferMemoryModel {
    private static final int FLOATS_PER_ROW = BYTES_PER_ROW / 4;
    private static final int CHARS_PER_FLOAT = 15;

    private static final int FLOAT_SEPARATOR = 1;

    private static final int FLOATS_CHARS = (CHARS_PER_FLOAT + FLOAT_SEPARATOR) * FLOATS_PER_ROW;
    private static final int CHARS_PER_ROW = ADDRESS_CHARS + FLOATS_CHARS;

    private static final IntRange FLOATS_RANGE =
        new IntRange(ADDRESS_CHARS + FLOAT_SEPARATOR, ADDRESS_CHARS + FLOATS_CHARS);

    public FloatsMemoryModel(MemoryDataModel data) {
      super(data.align(4), CHARS_PER_ROW, FLOATS_RANGE);
    }

    @Override
    protected void formatMemory(char[] buffer, MemorySegment memory) {
      StringBuilder sb = new StringBuilder(50);
      for (int i = 0, j = ADDRESS_CHARS; i + 3 < memory.length;
          i += 4, j += CHARS_PER_FLOAT + FLOAT_SEPARATOR) {
        sb.setLength(0);
        if (memory.getIntKnown(i)) {
          sb.append(Float.intBitsToFloat(memory.getInt(i)));
        } else {
          appendUnknown(sb, CHARS_PER_FLOAT);
        }
        int count = Math.min(CHARS_PER_FLOAT, sb.length());
        sb.getChars(0, count, buffer, j + CHARS_PER_FLOAT - count + 1);
      }
    }
  }

  private static class DoublesMemoryModel extends CharBufferMemoryModel {
    private static final int DOUBLES_PER_ROW = BYTES_PER_ROW / 8;
    private static final int CHARS_PER_DOUBLE = 24;

    private static final int DOUBLE_SEPARATOR = 1;

    private static final int DOUBLES_CHARS = (CHARS_PER_DOUBLE + DOUBLE_SEPARATOR) * DOUBLES_PER_ROW;
    private static final int CHARS_PER_ROW = ADDRESS_CHARS + DOUBLES_CHARS;

    private static final IntRange DOUBLES_RANGE =
        new IntRange(ADDRESS_CHARS + DOUBLE_SEPARATOR, ADDRESS_CHARS + DOUBLES_CHARS);

    public DoublesMemoryModel(MemoryDataModel data) {
      super(data.align(8), CHARS_PER_ROW, DOUBLES_RANGE);
    }

    @Override
    protected void formatMemory(char[] buffer, MemorySegment memory) {
      StringBuilder sb = new StringBuilder(50);
      for (int i = 0, j = ADDRESS_CHARS; i + 7 < memory.length;
          i += 8, j += CHARS_PER_DOUBLE + DOUBLE_SEPARATOR) {
        sb.setLength(0);
        if (memory.getLongKnown(i)) {
          sb.append(Double.longBitsToDouble(memory.getLong(i)));
        } else {
          appendUnknown(sb, CHARS_PER_DOUBLE);
        }
        int count = Math.min(CHARS_PER_DOUBLE, sb.length());
        sb.getChars(0, count, buffer, j + CHARS_PER_DOUBLE - count + 1);
      }
    }
  }

  private static class Selection {
    public final IntRange range;

    public final int startCol;
    public final long startRow;

    public final int endCol;
    public final long endRow;

    public Selection(IntRange range, int startCol, long startRow, int endCol, long endRow) {
      this.range = range;
      this.startCol = startCol;
      this.startRow = startRow;
      this.endCol = endCol;
      this.endRow = endRow;
    }

    /*TODO
    public boolean isEmpty() {
      return startCol == endCol && startRow == endRow;
    }

    public boolean isSelectionVisible(long startRow, long endRow) {
      return startRow <= endRow && startRow <= endRow;
    }
    */

    @Override
    public String toString() {
      return range + " (" + startCol + "," + startRow + ") -> (" + endCol + "," + endRow + ")";
    }
  }

  private static class MemorySegment {
    private final byte[] data;
    private final BitSet known;
    protected final int offset;
    protected final int length;

    protected final List<Service.MemoryRange> reads;
    protected final List<Service.MemoryRange> writes;

    private MemorySegment(byte[] data, BitSet known, int offset, int length,
        List<Service.MemoryRange> reads, List<Service.MemoryRange> writes) {
      this.data = data;
      this.offset = offset;
      this.length = length;
      this.known = known;
      this.reads = reads;
      this.writes = writes;
    }

    public MemorySegment(MemoryInfo info) {
      data = info.getData().toByteArray();
      offset = 0;
      known = computeKnown(info);
      length = data.length;
      reads = info.getReadsList();
      writes = info.getWritesList();
    }

    public static MemorySegment combine(List<MemorySegment> segments, int length) {
      byte[] data = new byte[length];
      BitSet known = new BitSet(length);
      int done = 0;

      List<Service.MemoryRange> reads = Lists.newArrayList();
      List<Service.MemoryRange> writes = Lists.newArrayList();

      for (Iterator<MemorySegment> it = segments.iterator(); it.hasNext() && done < length; ) {
        MemorySegment segment = it.next();
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
      return new MemorySegment(data, known, 0, done, reads, writes);
    }

    public MemorySegment subSegment(int start, int count) {
      return new MemorySegment(
          data, known, offset + start, Math.min(count, length - start), reads, writes);
    }

    /*TODO
    public String asString(int start, int count) {
      return new String(
          data, offset + start, Math.min(count, length - start), Charset.forName("US-ASCII"));
    }
    */

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

    public boolean getIntKnown(int off) {
      return getByteKnown(off, 4);
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

    private static BitSet computeKnown(MemoryInfo data) {
      BitSet known = new BitSet(data.getData().size());
      for (Service.MemoryRange rng : data.getObservedList()) {
        known.set((int)rng.getBase(), (int)rng.getBase() + (int)rng.getSize());
      }
      return known;
    }
  }

  private static class Segment {
    public char[] array;
    public int offset;
    public int count;

    public Segment(char[] array, int offset, int count) {
      this.array = array;
      this.offset = offset;
      this.count = count;
    }
  }
}
