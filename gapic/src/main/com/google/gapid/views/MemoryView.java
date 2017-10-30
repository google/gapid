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

import static com.google.gapid.util.GeoUtils.top;
import static com.google.gapid.util.Loadable.Message.error;
import static com.google.gapid.util.Loadable.Message.info;
import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.util.Logging.throttleLogRpcError;
import static com.google.gapid.util.Ranges.memory;
import static com.google.gapid.widgets.Widgets.createDropDown;
import static com.google.gapid.widgets.Widgets.createDropDownViewer;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.ifNotDisposed;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;
import static java.util.Collections.emptyList;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.primitives.UnsignedLong;
import com.google.common.primitives.UnsignedLongs;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.common.util.concurrent.MoreExecutors;
import com.google.gapid.models.AtomStream;
import com.google.gapid.models.AtomStream.AtomIndex;
import com.google.gapid.models.AtomStream.Observation;
import com.google.gapid.models.Capture;
import com.google.gapid.models.Follower;
import com.google.gapid.models.Models;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.SingleInFlight;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.server.Client;
import com.google.gapid.server.Client.DataUnavailableException;
import com.google.gapid.util.BigPoint;
import com.google.gapid.util.Float16;
import com.google.gapid.util.IntRange;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.LongPoint;
import com.google.gapid.util.Messages;
import com.google.gapid.util.MouseAdapter;
import com.google.gapid.util.Paths;
import com.google.gapid.widgets.CopyPaste;
import com.google.gapid.widgets.CopyPaste.CopyData;
import com.google.gapid.widgets.CopyPaste.CopySource;
import com.google.gapid.widgets.InfiniteScrolledComposite;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.viewers.ArrayContentProvider;
import org.eclipse.jface.viewers.ComboViewer;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.StructuredSelection;
import org.eclipse.swt.SWT;
import org.eclipse.swt.events.MouseEvent;
import org.eclipse.swt.events.SelectionEvent;
import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.graphics.Font;
import org.eclipse.swt.graphics.GC;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Combo;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Label;

import java.lang.ref.SoftReference;
import java.math.BigInteger;
import java.nio.ByteOrder;
import java.nio.charset.Charset;
import java.util.Arrays;
import java.util.BitSet;
import java.util.Collections;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.NoSuchElementException;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;
import java.util.function.Consumer;
import java.util.logging.Logger;

/**
 * View that displays the observed memory contents in an infinite scrolling panel.
 */
public class MemoryView extends Composite
    implements Tab, Capture.Listener, AtomStream.Listener, Follower.Listener {
  protected static final Logger LOG = Logger.getLogger(MemoryView.class.getName());

  private final Client client;
  private final Models models;
  private final Selections selections;
  private final MemoryPanel memoryPanel;
  protected final LoadablePanel<InfiniteScrolledComposite> loading;
  protected final InfiniteScrolledComposite memoryScroll;
  private final State uiState = new State();
  private final SingleInFlight rpcController = new SingleInFlight();
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

      @Override
      public void showMessage(Message message) {
        ifNotDisposed(loading, () -> loading.showMessage(message));
      }
    }, widgets);
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
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.atoms.removeListener(this);
      models.follower.removeListener(this);
    });
    memoryPanel.registerMouseEvents(memoryScroll);
  }

  @Override
  public Control getControl() {
    return this;
  }

  @Override
  public void reinitialize() {
    if (!models.capture.isLoaded() || !models.atoms.isLoaded()) {
      onCaptureLoadingStart(false);
    } else if (models.atoms.getSelectedAtoms() == null) {
      onAtomsLoaded();
    } else {
      onAtomsSelected(models.atoms.getSelectedAtoms());
    }
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    loading.showMessage(Info, Messages.LOADING_CAPTURE);
  }

  @Override
  public void onCaptureLoaded(Loadable.Message error) {
    if (error != null) {
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    }
  }

  @Override
  public void onAtomsLoaded() {
    loading.showMessage(Info, Messages.SELECT_MEMORY);
  }

  @Override
  public void onAtomsSelected(AtomIndex range) {
    rpcController.start().listen(models.atoms.getObservations(range),
        new UiCallback<Observation[], Observation[]>(this, LOG) {
      @Override
      protected Observation[] onRpcThread(Rpc.Result<Observation[]> result)
          throws RpcException, ExecutionException {
        return result.get();
      }

      @Override
      protected void onUiThread(Observation[] obs) {
        setObservations(range, obs);
      }
    });
  }

  protected void setObservations(AtomIndex range, Observation[] obs) {
    selections.setObservations(obs);
    if (obs.length > 0 && !uiState.isComplete()) {
      // If the memory view is not showing anything yet, show the first observation.
      setObservation(obs[0]);
    } else {
      uiState.update(range.getCommand());
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

  private void setObservation(Observation obs) {
    Path.Memory memoryPath = obs.getPath();
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
    scheduleIfNotDisposed(memoryScroll, () -> goToAddress(address));
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

  /**
   * Bookkeeping of the current user selections.
   */
  private static class Selections extends Composite {
    private final Label poolLabel;
    private final Combo typeCombo;
    private final Label obsLabel;
    private final ComboViewer obsCombo;

    public Selections(Composite parent, Consumer<DataType> dataTypeListener,
        Consumer<Observation> observationListener) {
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
        Observation obs =
            (Observation)obsCombo.getElementAt(obsCombo.getCombo().getSelectionIndex());
        if (obs != Observation.NULL_OBSERVATION) {
          observationListener.accept(obs);
        }
      });

      obsLabel.setVisible(false);
      obsCombo.getCombo().setVisible(false);
    }

    private ComboViewer createObservationSelector() {
      ComboViewer combo = createDropDownViewer(this);
      combo.setContentProvider(ArrayContentProvider.getInstance());
      combo.setLabelProvider(new LabelProvider());
      combo.getCombo().setLayoutData(new GridData(SWT.FILL, SWT.CENTER, false, false));
      combo.setInput(emptyList());
      return combo;
    }

    public void setDataType(DataType dataType) {
      typeCombo.select(dataType.ordinal());
    }

    public void setObservations(Observation[] observations) {
      if (observations.length == 0) {
        obsCombo.setInput(emptyList());
      } else {
        obsCombo.setInput(Lists.asList(Observation.NULL_OBSERVATION, observations));
      }

      obsLabel.setVisible(observations.length != 0);
      obsCombo.getCombo().setVisible(observations.length != 0);
      obsCombo.getControl().requestLayout();
    }

    @SuppressWarnings("unchecked")
    public void updateSelectedObservation(long address) {
      for (Observation obs : (Iterable<Observation>)obsCombo.getInput()) {
        if (obs.contains(address)) {
          obsCombo.setSelection(new StructuredSelection(obs), true);
          return;
        }
      }
      if (obsCombo.getCombo().getItemCount() > 0) {
        obsCombo.setSelection(new StructuredSelection(Observation.NULL_OBSERVATION));
      }
    }

    public void setPool(int pool) {
      poolLabel.setText(String.valueOf(pool));
      poolLabel.requestLayout();
    }
  }

  /**
   * Bookkeeping of the current UI state.
   */
  private static class State {
    public DataType dataType = DataType.Byte;
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
              value -> value.getMemory());


      return new PagedMemoryDataModel(fetcher, offset, lastAddress);
    }

    public MemoryModel getMemoryModel(MemoryDataModel data) {
      return dataType.getMemoryModel(data);
    }
  }

  /**
   * The memory data can be visualized as different atomic data types to ease buffer inspection.
   */
  private static enum DataType {
    Byte() {
      @Override
      public MemoryModel getMemoryModel(MemoryDataModel memory) {
        return new ByteMemoryModel(memory);
      }
    }, Int16() {
      @Override
      public MemoryModel getMemoryModel(MemoryDataModel memory) {
        return new Int16MemoryModel(memory);
      }
    }, Int32() {
      @Override
      public MemoryModel getMemoryModel(MemoryDataModel memory) {
        return new Int32MemoryModel(memory);
      }
    }, Int64() {
      @Override
      public MemoryModel getMemoryModel(MemoryDataModel memory) {
        return new Int64MemoryModel(memory);
      }
    }, Float16() {
      @Override
      public MemoryModel getMemoryModel(MemoryDataModel memory) {
        return new Float16MemoryModel(memory);
      }
    }, Float32() {
      @Override
      public MemoryModel getMemoryModel(MemoryDataModel memory) {
        return new Float32MemoryModel(memory);
      }
    }, Float64() {
      @Override
      public MemoryModel getMemoryModel(MemoryDataModel memory) {
        return new Float64MemoryModel(memory);
      }
    };

    public abstract MemoryModel getMemoryModel(MemoryDataModel memory);

    public static Combo createCombo(Composite parent) {
      Combo combo = createDropDown(parent);
      String[] names = new String[values().length];
      for (int i = 0; i < names.length; i++) {
        names[i] = values()[i].name();
      }
      combo.setItems(names);
      combo.select(0);
      return combo;
    }
  }

  /**
   * Panel displaying the actual memory data.
   */
  private static class MemoryPanel implements InfiniteScrolledComposite.Scrollable {
    public final int lineHeight;
    public final BigInteger lineHeightBig;
    private final int[] charOffset = new int[256];

    private final Loadable loadable;
    private final Theme theme;
    protected final CopyPaste copyPaste;
    private final Font font;
    protected MemoryModel model;
    protected Selection selection;

    public MemoryPanel(Composite parent, Loadable loadable, Widgets widgets) {
      this.loadable = loadable;
      this.theme = widgets.theme;
      this.copyPaste = widgets.copypaste;
      font = theme.monoSpaceFont();
      GC gc = new GC(parent);
      gc.setFont(font);
      lineHeight = gc.getFontMetrics().getHeight();
      lineHeightBig = BigInteger.valueOf(lineHeight);
      StringBuilder sb = new StringBuilder();
      for (int i = 1; i < charOffset.length; i++) {
        charOffset[i] = gc.stringExtent(sb.append('0').toString()).x;
      }
      gc.dispose();
    }

    public void registerMouseEvents(InfiniteScrolledComposite parent) {
      parent.addContentListener(new MouseAdapter() {
        private final LongPoint selectionPoint = new LongPoint(0, 0);
        private boolean selecting;

        @Override
        public void mouseDown(MouseEvent e) {
          parent.setFocus();
          if (isSelectionButton(e)) {
            if ((e.stateMask & SWT.SHIFT) != 0 && selection != null) {
              selecting = true;
              updateSelection(parent.getLocation(e));
            } else {
              startSelecting(parent.getLocation(e));
            }
            parent.redraw();
          }
        }

        @Override
        public void mouseUp(MouseEvent e) {
          if (isSelectionButton(e)) {
            selecting = false;
            if (selection != null && selection.isEmpty()) {
              selection = null;
              parent.redraw();
            }
          }
        }

        @Override
        public void mouseMove(MouseEvent e) {
          if (isSelectionButtonDown(e)) {
            if (selection == null) {
              startSelecting(parent.getLocation(e));
            } else {
              updateSelection(parent.getLocation(e));
            }
          }
        }

        @Override
        public void widgetSelected(SelectionEvent e) {
          // Scrollbar was moved / mouse wheel caused scrolling.
          if (selecting) {
            if (selection == null) {
              startSelecting(parent.getMouseLocation());
            } else {
              updateSelection(parent.getMouseLocation());
            }
          }
        }

        private void startSelecting(BigPoint location) {
          selecting = true;
          long y = location.y.divide(lineHeightBig).longValueExact();
          if (y < 0 || y >= model.getLineCount()) {
            selection = null;
            copyPaste.updateCopyState();
            return;
          }
          selectionPoint.set(getCharColumn(location.x.intValue()), y);
          IntRange range = model.getSelectableRegion((int)selectionPoint.x);
          if (range != null) {
            selection = new Selection(range, selectionPoint, selectionPoint);
            copyPaste.updateCopyState();
          }
        }

        private void updateSelection(BigPoint location) {
          int x = Math.max(selection.range.from,
              Math.min(selection.range.to, getCharColumn(location.x.intValueExact())));
          long y = location.y.divide(lineHeightBig).max(BigInteger.ZERO).longValueExact();
          if (y >= model.getLineCount()) {
            y = model.getLineCount() - 1;
            x = selection.range.to;
          }

          if (y < selectionPoint.y || (y == selectionPoint.y && x < selectionPoint.x)) {
            selection = new Selection(selection.range, x, y, selectionPoint);
          } else {
            selection = new Selection(selection.range, selectionPoint, x, y);
          }
          copyPaste.updateCopyState();
          parent.redraw();
        }

        private boolean isSelectionButton(MouseEvent e) {
          return e.button == 1;
        }

        private boolean isSelectionButtonDown(MouseEvent e) {
          return (e.stateMask & SWT.BUTTON1) != 0;
        }
      });
      parent.registerContentAsCopySource(copyPaste, new CopySource() {
        @Override
        public boolean hasCopyData() {
          return selection != null && !selection.isEmpty();
        }

        @Override
        public CopyData[] getCopyData() {
          return model.getCopyData(selection);
        }
      });
    }

    public void setModel(MemoryModel model) {
      this.model = model;
      selection = null;
    }

    @Override
    public BigInteger getWidth() {
      return (model == null) ?
          BigInteger.ZERO : BigInteger.valueOf(charOffset[model.getLineLength()]);
    }

    @Override
    public BigInteger getHeight() {
      return (model == null) ? BigInteger.ZERO :
        BigInteger.valueOf(model.getLineCount()).multiply(lineHeightBig);
    }

    @Override
    public void paint(BigInteger xOffset, BigInteger yOffset, GC gc) {
      if (model == null) {
        return;
      }

      gc.setFont(font);
      Rectangle clip = gc.getClipping();
      BigInteger startY = yOffset.add(BigInteger.valueOf(top(clip)));
      long startRow = startY.divide(lineHeightBig)
          .max(BigInteger.ZERO).min(BigInteger.valueOf(model.getLineCount() - 1)).longValueExact();
      long endRow = startY.add(BigInteger.valueOf(clip.height + lineHeight - 1))
          .divide(lineHeightBig)
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

      if (selection != null && selection.isSelectionVisible(startRow, endRow)) {
        gc.setBackground(theme.memorySelectionHighlight());
        highlight(gc, yOffset, selection);
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
          .multiply(lineHeightBig).subtract(yOffset).intValueExact();
    }

    private void highlight(GC gc, BigInteger yOffset, Selection range) {
      if (range.startRow == range.endRow) {
        int so = charOffset[range.startCol], eo = charOffset[range.endCol];
        gc.fillRectangle(so, getY(range.startRow, yOffset), eo - so, lineHeight);
      } else {
        int so = charOffset[range.startCol], eo = charOffset[range.endCol];
        int fo = charOffset[range.range.from], to = charOffset[range.range.to];
        gc.fillRectangle(so, getY(range.startRow, yOffset), to - so, lineHeight);
        gc.fillRectangle(fo, getY(range.endRow, yOffset), eo - fo, lineHeight);
        gc.fillRectangle(fo, getY(range.startRow + 1, yOffset), to - fo,
            (int)(range.endRow - range.startRow - 1) * lineHeight);
      }
    }

    protected int getCharColumn(int offset) {
      int idx = Arrays.binarySearch(charOffset, offset);
      return (idx < 0) ? (-idx - 1) - 1 : idx;
    }
  }

  /**
   * Model responsible for loading the observed memory data.
   */
  private static interface MemoryDataModel {
    /**
     * @return the current address to show.
     */
    public long getAddress();

    /**
     * @return the last address (highest) possible memory address.
     */
    public long getEndAddress();

    /**
     * Fetches the memory data starting at the given offset with the given length.
     */
    public ListenableFuture<MemorySegment> get(long offset, int length);

    /**
     * @return a {@link MemoryDataModel} aligned to the given number of bytes.
     */
    public MemoryDataModel align(int byteAlign);
  }

  /**
   * {@link MemoryDataModel} that requests segments as equally sized pages and maintains a cache of
   * fetched pages.
   */
  private static class PagedMemoryDataModel implements MemoryDataModel {
    private static final int PAGE_SIZE = 0x10000;

    private final MemoryFetcher fetcher;
    private final long address;
    private final long lastAddress;

    private final Map<Long, SoftReference<ListenableFuture<Service.Memory>>> cache =
        Maps.newHashMap();

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
      ListenableFuture<Service.Memory> mem = getFromCache(page);
      if (mem == null) {
        long base = address + getOffsetForPage(page);
        mem = fetcher.get(base, (int)UnsignedLongs.min(lastAddress - base, PAGE_SIZE - 1) + 1);
        addToCache(page, mem);
      }
      return Futures.transform(mem, memory -> new MemorySegment(memory).subSegment(offset, length));
    }

    private ListenableFuture<Service.Memory> getFromCache(long page) {
      ListenableFuture<Service.Memory> result = null;
      synchronized (cache) {
        SoftReference<ListenableFuture<Service.Memory>> reference = cache.get(page);
        if (reference != null) {
          result = reference.get();
          if (result == null) {
            cache.remove(page);
          }
        }
      }
      return result;
    }

    private void addToCache(long page, ListenableFuture<Service.Memory> data) {
      synchronized (cache) {
        cache.put(page, new SoftReference<ListenableFuture<Service.Memory>>(data));
      }
    }

    @Override
    public MemoryDataModel align(int byteAlign) {
      return this;
    }

    public interface MemoryFetcher {
      ListenableFuture<Service.Memory> get(long address, long count);
    }
  }

  /**
   * Model responsible for converting memory data bytes to displayable strings.
   */
  private static interface MemoryModel {
    /**
     * @return the number of lines in this models.
     */
    public long getLineCount();

    /**
     * @return the number of character in each line.
     */
    public int getLineLength();

    /**
     * @return an {@link Iterator} over the given range of lines.
     */
    public Iterator<Segment> getLines(long start, long end, Loadable loadable);

    /**
     * @return the range of columns containing the given column that can be selected as a group.
     */
    public IntRange getSelectableRegion(int column);

    /**
     * @return the read selections within the given range of rows.
     */
    public Selection[] getReads(long startRow, long endRow, Loadable loadable);

    /**
     * @return the write selections within the given range of rows.
     */
    public Selection[] getWrites(long startRow, long endRow, Loadable loadable);

    /**
     * @return the given selected memory area as copy-paste content.
     */
    public CopyData[] getCopyData(Selection selection);
  }

  /**
   * {@link MemoryModel} displaying a fixed amount of bytes per line.
   */
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
      MemorySegment result = null;
      if (future.isDone()) {
        try {
          result = Rpc.get(future, 0, TimeUnit.MILLISECONDS);
          loadable.stopLoading();
        } catch (RpcException e) {
          if (e instanceof DataUnavailableException) {
            loadable.showMessage(info(e));
          } else {
            loadable.showMessage(error(e));
          }
        } catch (TimeoutException e) {
          throw new AssertionError(); // Should not happen.
        } catch (ExecutionException e) {
          throttleLogRpcError(LOG, "Unexpected error fetching memory", e);
          loadable.showMessage(error("Unexpected error: " + e.getCause()));
        }
      } else {
        loadable.startLoading();
        future.addListener(loadable::stopLoading, MoreExecutors.directExecutor());
      }
      return result;
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

  /**
   * A {@link MemoryModel} that uses a char array as a buffer to compute the displayed strings.
   */
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

    @Override
    public CopyData[] getCopyData(Selection selection) {
      return Futures.getUnchecked(Futures.transform(data.get(selection.startRow * BYTES_PER_ROW,
          (int)(selection.endRow - selection.startRow + 1) * BYTES_PER_ROW), memory -> {
            StringBuilder buffer = new StringBuilder();
            Iterator<Segment> lines = getLines(selection.startRow, selection.endRow + 1, memory);
            if (lines.hasNext()) {
              Segment segment = lines.next();
              if (selection.startRow == selection.endRow) {
                buffer.append(segment.array,
                    segment.offset + selection.startCol, selection.endCol - selection.startCol);
              } else {
                buffer.append(segment.array,
                    segment.offset + selection.startCol, selection.range.to - selection.startCol)
                    .append('\n');
              }
            }
            int rangeWidth = selection.range.to - selection.range.from;
            for (long line = selection.startRow + 1;
                lines.hasNext() && line < selection.endRow; line++) {
              Segment segment = lines.next();
              buffer.append(segment.array, segment.offset + selection.range.from, rangeWidth)
                  .append('\n');
            }
            if (lines.hasNext()) {
              Segment segment = lines.next();
              buffer.append(segment.array,
                  segment.offset + selection.range.from, selection.endCol - selection.range.from)
                  .append('\n');
            }
            return new CopyData[] { CopyData.text(buffer.toString()) };
          }));
    }
  }

  /**
   * {@link MemoryModel} formatting the data as bytes.
   */
  private static class ByteMemoryModel extends CharBufferMemoryModel {
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

    public ByteMemoryModel(MemoryDataModel data) {
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
    public CopyData[] getCopyData(Selection selection) {
      if (selection.range == ASCII_RANGE) {
        // Copy the actual myData, rather than the display.
        return Futures.getUnchecked(Futures.transform(
            data.get(selection.startRow * BYTES_PER_ROW,
                (int)(selection.endRow - selection.startRow + 1) * BYTES_PER_ROW),
                s -> new CopyData[] {
                  CopyData.text(s.asString(selection.startCol - ASCII_RANGE.from,
                      s.length - selection.startCol + ASCII_RANGE.from -
                      ASCII_RANGE.to + selection.endCol))
                  }));
      } else {
        return super.getCopyData(selection);
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

  /**
   * {@link MemoryModel} formatting the data as multi-byte integers.
   */
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

  /**
   * {@link IntegersMemoryModel} for 2 byte integers.
   */
  private static class Int16MemoryModel extends IntegersMemoryModel {
    public Int16MemoryModel(MemoryDataModel data) {
      super(data, 2);
    }
  }

  /**
   * {@link IntegersMemoryModel} for 4 byte integers.
   */
  private static class Int32MemoryModel extends IntegersMemoryModel {
    public Int32MemoryModel(MemoryDataModel data) {
      super(data, 4);
    }
  }

  /**
   * {@link IntegersMemoryModel} for 8 byte integers.
   */
  private static class Int64MemoryModel extends IntegersMemoryModel {
    public Int64MemoryModel(MemoryDataModel data) {
      super(data, 8);
    }
  }

  /**
   * {@link MemoryModel} displaying the data as 16bit floating point values.
   */
  private static class Float16MemoryModel extends CharBufferMemoryModel {
    private static final int FLOATS_PER_ROW = BYTES_PER_ROW / 2;
    private static final int CHARS_PER_FLOAT = 15;

    private static final int FLOAT_SEPARATOR = 1;

    private static final int FLOATS_CHARS = (CHARS_PER_FLOAT + FLOAT_SEPARATOR) * FLOATS_PER_ROW;
    private static final int CHARS_PER_ROW = ADDRESS_CHARS + FLOATS_CHARS;

    private static final IntRange FLOATS_RANGE =
        new IntRange(ADDRESS_CHARS + FLOAT_SEPARATOR, ADDRESS_CHARS + FLOATS_CHARS);

    public Float16MemoryModel(MemoryDataModel data) {
      super(data.align(4), CHARS_PER_ROW, FLOATS_RANGE);
    }

    @Override
    protected void formatMemory(char[] buffer, MemorySegment memory) {
      StringBuilder sb = new StringBuilder(50);
      for (int i = 0, j = ADDRESS_CHARS; i + 1 < memory.length;
          i += 2, j += CHARS_PER_FLOAT + FLOAT_SEPARATOR) {
        sb.setLength(0);
        if (memory.getShortKnown(i)) {
          sb.append(Float16.shortBitsToFloat(memory.getShort(i)));
        } else {
          appendUnknown(sb, CHARS_PER_FLOAT);
        }
        int count = Math.min(CHARS_PER_FLOAT, sb.length());
        sb.getChars(0, count, buffer, j + CHARS_PER_FLOAT - count + 1);
      }
    }
  }

  /**
   * {@link MemoryModel} displaying the data as 32bit floating point values.
   */
  private static class Float32MemoryModel extends CharBufferMemoryModel {
    private static final int FLOATS_PER_ROW = BYTES_PER_ROW / 4;
    private static final int CHARS_PER_FLOAT = 15;

    private static final int FLOAT_SEPARATOR = 1;

    private static final int FLOATS_CHARS = (CHARS_PER_FLOAT + FLOAT_SEPARATOR) * FLOATS_PER_ROW;
    private static final int CHARS_PER_ROW = ADDRESS_CHARS + FLOATS_CHARS;

    private static final IntRange FLOATS_RANGE =
        new IntRange(ADDRESS_CHARS + FLOAT_SEPARATOR, ADDRESS_CHARS + FLOATS_CHARS);

    public Float32MemoryModel(MemoryDataModel data) {
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

  /**
   * {@link MemoryModel} displaying the data as 64bit floating point values.
   */
  private static class Float64MemoryModel extends CharBufferMemoryModel {
    private static final int DOUBLES_PER_ROW = BYTES_PER_ROW / 8;
    private static final int CHARS_PER_DOUBLE = 24;

    private static final int DOUBLE_SEPARATOR = 1;

    private static final int DOUBLES_CHARS =
        (CHARS_PER_DOUBLE + DOUBLE_SEPARATOR) * DOUBLES_PER_ROW;
    private static final int CHARS_PER_ROW = ADDRESS_CHARS + DOUBLES_CHARS;

    private static final IntRange DOUBLES_RANGE =
        new IntRange(ADDRESS_CHARS + DOUBLE_SEPARATOR, ADDRESS_CHARS + DOUBLES_CHARS);

    public Float64MemoryModel(MemoryDataModel data) {
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

  /**
   * A text selection range.
   */
  private static class Selection {
    public final IntRange range;

    public final int startCol;
    public final long startRow;

    public final int endCol;
    public final long endRow;

    public Selection(IntRange range, LongPoint start, LongPoint end) {
      this(range, start.x, start.y, end.x, end.y);
    }

    public Selection(IntRange range, long startCol, long startRow, LongPoint end) {
      this(range, startCol, startRow, end.x, end.y);
    }

    public Selection(IntRange range, LongPoint start, long endCol, long endRow) {
      this(range, start.x, start.y, endCol, endRow);
    }

    public Selection(IntRange range, long startCol, long startRow, long endCol, long endRow) {
      this.range = range;
      this.startCol = (int)startCol;
      this.startRow = startRow;
      this.endCol = (int)endCol;
      this.endRow = endRow;
    }

    public boolean isEmpty() {
      return startCol == endCol && startRow == endRow;
    }

    public boolean isSelectionVisible(long fromRow, long toRow) {
      return fromRow <= endRow && startRow <= toRow;
    }

    @Override
    public String toString() {
      return range + " (" + startCol + "," + startRow + ") -> (" + endCol + "," + endRow + ")";
    }
  }

  /**
   * A segment of memory data.
   */
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

    public MemorySegment(Service.Memory info) {
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

    public String asString(int start, int count) {
      return new String(
          data, offset + start, Math.min(count, length - start), Charset.forName("US-ASCII"));
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

    private static BitSet computeKnown(Service.Memory data) {
      BitSet known = new BitSet(data.getData().size());
      for (Service.MemoryRange rng : data.getObservedList()) {
        known.set((int)rng.getBase(), (int)rng.getBase() + (int)rng.getSize());
      }
      return known;
    }
  }

  /**
   * A segment of character data.
   */
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
