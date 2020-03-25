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
import static com.google.gapid.widgets.Widgets.createDropDown;
import static com.google.gapid.widgets.Widgets.createDropDownViewer;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createStandardTabFolder;
import static com.google.gapid.widgets.Widgets.createStandardTabItem;
import static com.google.gapid.widgets.Widgets.createTreeColumn;
import static com.google.gapid.widgets.Widgets.createTreeViewer;
import static com.google.gapid.widgets.Widgets.ifNotDisposed;
import static com.google.gapid.widgets.Widgets.packColumns;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;
import static java.util.Collections.emptyList;

import com.google.common.collect.Lists;
import com.google.common.primitives.UnsignedLong;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.common.util.concurrent.MoreExecutors;
import com.google.gapid.models.Analytics;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.Follower;
import com.google.gapid.models.Memory;
import com.google.gapid.models.Memory.Observation;
import com.google.gapid.models.Memory.StructNode;
import com.google.gapid.models.Memory.StructObservation;
import com.google.gapid.models.Models;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.proto.service.Service.MemoryRange;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.proto.service.types.TypeInfo.Type.TyCase;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.Rpc.Result;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.server.Client.DataUnavailableException;
import com.google.gapid.util.BigPoint;
import com.google.gapid.util.Float16;
import com.google.gapid.util.IntRange;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.LongPoint;
import com.google.gapid.util.Messages;
import com.google.gapid.util.MoreFutures;
import com.google.gapid.util.MouseAdapter;
import com.google.gapid.widgets.CopyPaste;
import com.google.gapid.widgets.CopyPaste.CopyData;
import com.google.gapid.widgets.CopyPaste.CopySource;
import com.google.gapid.widgets.InfiniteScrolledComposite;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.viewers.ArrayContentProvider;
import org.eclipse.jface.viewers.ComboViewer;
import org.eclipse.jface.viewers.ITreeContentProvider;
import org.eclipse.jface.viewers.ITreeViewerListener;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.StructuredSelection;
import org.eclipse.jface.viewers.StyledCellLabelProvider;
import org.eclipse.jface.viewers.TreeExpansionEvent;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.jface.viewers.TreeViewerColumn;
import org.eclipse.jface.viewers.Viewer;
import org.eclipse.jface.viewers.ViewerCell;
import org.eclipse.jface.viewers.ViewerComparator;
import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.StyleRange;
import org.eclipse.swt.events.MouseEvent;
import org.eclipse.swt.events.SelectionEvent;
import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.graphics.Font;
import org.eclipse.swt.graphics.GC;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Combo;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Display;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.TabFolder;
import org.eclipse.swt.widgets.TreeItem;

import java.math.BigInteger;
import java.nio.ByteOrder;
import java.util.Arrays;
import java.util.Collections;
import java.util.Iterator;
import java.util.List;
import java.util.NoSuchElementException;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;
import java.util.function.Consumer;
import java.util.logging.Logger;

/**
 * View that has two tabs, displaying block-styled or struct-styled memory after loading.
 */
public class MemoryView extends Composite
    implements Tab, Capture.Listener, CommandStream.Listener, Follower.Listener, Memory.Listener {
  protected static final Logger LOG = Logger.getLogger(MemoryView.class.getName());

  protected final Models models;
  protected final Widgets widgets;
  protected final LoadablePanel<TabFolder> loading;
  protected final TabFolder folder;
  protected final BlockMemoryPanel blockPanel;
  private final StructMemoryPanel structPanel;

  public MemoryView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;
    this.widgets = widgets;

    setLayout(new GridLayout(1, true));

    loading = LoadablePanel.create(this, widgets, panel -> createStandardTabFolder(panel));
    folder = loading.getContents();
    blockPanel = new BlockMemoryPanel(folder);
    structPanel = new StructMemoryPanel(folder);
    createStandardTabItem(folder, Messages.MEMORY_BLOCK_TAB_TEXT, blockPanel);
    createStandardTabItem(folder, Messages.MEMORY_STRUCT_TAB_TEXT, structPanel);

    loading.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
    folder.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    models.capture.addListener(this);
    models.commands.addListener(this);
    models.follower.addListener(this);
    models.memory.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.commands.removeListener(this);
      models.follower.removeListener(this);
      models.memory.removeListener(this);
    });
  }

  @Override
  public Control getControl() {
    return this;
  }

  @Override
  public void reinitialize() {
    if (!models.capture.isLoaded() || !models.commands.isLoaded()) {
      onCaptureLoadingStart(false);
    } else if (models.commands.getSelectedCommands() == null) {
      onCommandsLoaded();
    } else {
      onCommandsSelected(models.commands.getSelectedCommands());
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
  public void onCommandsLoaded() {
    if (!models.commands.isLoaded()) {
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    } else if (models.commands.getSelectedCommands() == null) {
      loading.showMessage(Info, Messages.SELECT_MEMORY);
    }
  }

  @Override
  public void onCommandsSelected(CommandIndex range) {
    if (range == null) {
      loading.showMessage(Info, Messages.SELECT_MEMORY);
    }
  }

  @Override
  public void onMemoryLoadingStart() {
    loading.startLoading();
  }

  @Override
  public void onMemoryLoaded() {
    updateUi();
  }

  @Override
  public void onMemoryFollowed(Path.Memory path) {
    blockPanel.updateState(path);
    models.memory.setPool(path.getPool());
    updateUi();
    structPanel.reorderStruct(path.getAddress());
  }

  private void updateUi() {
    if (!models.memory.isLoaded()) {
      return;
    }
    blockPanel.updateUi();
    structPanel.updateUi();
    loading.stopLoading();
  }

  /**
   * View that displays the observed block-styled memory contents in an infinite scrolling panel.
   */
  private class BlockMemoryPanel extends Composite{
    protected final Selections selections;
    private final BlockMemoryScrollable memoryPanel;
    protected final InfiniteScrolledComposite memoryScroll;
    private final State uiState = new State();

    public BlockMemoryPanel(Composite parent) {
      super(parent, SWT.NONE);
      setLayout(new GridLayout());

      memoryPanel = new BlockMemoryScrollable(this, widgets);
      selections = new Selections(this, this::setDataType, this::setObservation);
      memoryScroll = new InfiniteScrolledComposite(this, SWT.H_SCROLL | SWT.V_SCROLL, memoryPanel);
      memoryPanel.registerMouseEvents(memoryScroll, models.analytics);

      selections.setLayoutData(new GridData(SWT.FILL, SWT.TOP, true, false));
      memoryScroll.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
    }

    public void updateUi() {
      Memory.Data memory = models.memory.getData();
      if (memory.getObservations().length > 0 && !uiState.isComplete()) {
        // If the memory view is not showing anything yet, show the first observation.
        uiState.update(memory.getObservations()[0].getPath());
      }

      if (!uiState.isComplete()) {
        loading.showMessage(Info, Messages.SELECT_MEMORY);
        return;
      }

      long address = getCurrentAddress();
      uiState.address = -1; // The memoryScroll will now control the currently selected address.

      selections.setPool(memory.getPool());
      selections.setDataType(uiState.dataType);
      selections.setObservations(memory.getObservations());

      memoryPanel.setModel(uiState.getMemoryModel(memory, new Loadable() {
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
      }));
      memoryScroll.updateMinSize();
      goToAddress(address);
      selections.updateSelectedObservation(address);
    }

    public void updateState(Path.Memory path) {
      uiState.update(path);
    }

    public void goToObservation(long address) {
      selections.updateSelectedObservation(address);
      goToAddress(address);
    }

    private void setDataType(DataType dataType) {
      if (uiState.setDataType(dataType)) {
        updateUi();
      }
    }

    private void setObservation(Observation obs) {
      models.analytics.postInteraction(View.Memory, ClientAction.SelectObservation);
      Path.Memory memoryPath = obs.getPath();
      uiState.update(memoryPath);
      updateUi();
    }

    private void goToAddress(long address) {
      scheduleIfNotDisposed(memoryScroll, () -> memoryScroll.scrollTo(BigInteger.ZERO,
          UnsignedLong.fromLongBits(address).bigIntegerValue()
              .divide(BigInteger.valueOf(FixedMemoryModel.BYTES_PER_ROW))
              .multiply(BigInteger.valueOf(memoryPanel.lineHeight))));
    }

    private long getCurrentAddress() {
      if (uiState.address >= 0) {
        return uiState.address;
      }

      return memoryScroll.getScrollLocation().y
          .divide(BigInteger.valueOf(memoryPanel.lineHeight))
          .multiply(BigInteger.valueOf(FixedMemoryModel.BYTES_PER_ROW))
          .add(BigInteger.valueOf(uiState.offset))
          .longValue();
    }
  }

  /**
   * Bookkeeping of the current user selections in BlockMemoryPanel.
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
    public int offset = -1;
    public long address = -1;

    public State() {
    }

    public void update(Path.Memory path) {
      if (path != null) {
        offset = (int)Long.remainderUnsigned(path.getAddress(), FixedMemoryModel.BYTES_PER_ROW);
        address = path.getAddress();
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
      return offset >= 0;
    }

    public MemoryModel getMemoryModel(Memory.Data data, Loadable loadable) {
      return dataType.getMemoryModel(data, loadable, offset);
    }
  }

  /**
   * The memory data can be visualized as different atomic data types to ease buffer inspection.
   */
  private static enum DataType {
    Byte() {
      @Override
      public MemoryModel getMemoryModel(Memory.Data memory, Loadable loadable, long offset) {
        return new ByteMemoryModel(memory, loadable, offset);
      }
    }, Int16() {
      @Override
      public MemoryModel getMemoryModel(Memory.Data memory, Loadable loadable, long offset) {
        return new Int16MemoryModel(memory, loadable, offset);
      }
    }, Int32() {
      @Override
      public MemoryModel getMemoryModel(Memory.Data memory, Loadable loadable, long offset) {
        return new Int32MemoryModel(memory, loadable, offset);
      }
    }, Int64() {
      @Override
      public MemoryModel getMemoryModel(Memory.Data memory, Loadable loadable, long offset) {
        return new Int64MemoryModel(memory, loadable, offset);
      }
    }, Float16() {
      @Override
      public MemoryModel getMemoryModel(Memory.Data memory, Loadable loadable, long offset) {
        return new Float16MemoryModel(memory, loadable, offset);
      }
    }, Float32() {
      @Override
      public MemoryModel getMemoryModel(Memory.Data memory, Loadable loadable, long offset) {
        return new Float32MemoryModel(memory, loadable, offset);
      }
    }, Float64() {
      @Override
      public MemoryModel getMemoryModel(Memory.Data memory, Loadable loadable, long offset) {
        return new Float64MemoryModel(memory, loadable, offset);
      }
    };

    public abstract MemoryModel getMemoryModel(Memory.Data memory, Loadable loadable, long offset);

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
  private static class BlockMemoryScrollable implements InfiniteScrolledComposite.Scrollable {
    public final int lineHeight;
    public final BigInteger lineHeightBig;
    private final int[] charOffset = new int[256];

    private final Theme theme;
    protected final CopyPaste copyPaste;
    private final Font font;
    protected MemoryModel model;
    protected Selection selection;

    public BlockMemoryScrollable(Composite parent, Widgets widgets) {
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

    public void registerMouseEvents(InfiniteScrolledComposite parent, Analytics analytics) {
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
            } else {
              analytics.postInteraction(View.Memory, ClientAction.Select);
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
          analytics.postInteraction(View.Memory, ClientAction.Copy);
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
      for (Selection read : model.getReads(startRow, endRow)) {
        highlight(gc, yOffset, read);
      }

      gc.setBackground(theme.memoryWriteHighlight());
      for (Selection write : model.getWrites(startRow, endRow)) {
        highlight(gc, yOffset, write);
      }

      if (selection != null && selection.isSelectionVisible(startRow, endRow)) {
        gc.setBackground(theme.memorySelectionHighlight());
        highlight(gc, yOffset, selection);
      }
      gc.setBackground(background);

      int y = getY(startRow, yOffset);
      Iterator<Segment> it = model.getLines(startRow, endRow);
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
   * View that displays the observed struct-styled memory contents.
   */
  private class StructMemoryPanel extends Composite {
    protected final TreeViewer treeViewer;

    public StructMemoryPanel(Composite parent) {
      super(parent, SWT.NONE);
      setLayout(new GridLayout());

      treeViewer = createStructTreeViewer();
      treeViewer.getTree().setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
    }

    public void updateUi() {
      treeViewer.setComparator(null);
      loadAndSetStructs(models.memory.getData().getStructObservations());
    }

    public void reorderStruct(long interestedStructRoot) {
      // Reorder to make the interested observation listed on top.
      // If not found, by default the observations with type TypeInfo.Type.STRUCT are listed on top.
      treeViewer.setComparator(new ViewerComparator() {
        @Override
        public int compare(Viewer viewer, Object e1, Object e2) {
          if (!(e1 instanceof StructNode) || !(e2 instanceof StructNode)) {
            return 0;
          }
          StructNode n1 = (StructNode) e1;
          StructNode n2 = (StructNode) e2;
          if (n1.getRootAddress() == interestedStructRoot && n2.getRootAddress() == interestedStructRoot) {
            return 0;
          } else if (n1.getRootAddress() == interestedStructRoot) {
            return -1;
          } else if (n2.getRootAddress() == interestedStructRoot) {
            return 1;
          } else if (n1.getTypeCase() == TyCase.STRUCT && n2.getTypeCase() != TyCase.STRUCT) {
            return -1;
          } else if (n1.getTypeCase() != TyCase.STRUCT && n2.getTypeCase() == TyCase.STRUCT) {
            return 1;
          }
          return 0;
        }
      });
    }

    private void loadAndSetStructs(StructObservation[] structs) {
      if (structs == null || structs.length == 0) {
        treeViewer.setInput(emptyList());
        packColumns(treeViewer.getTree());
        return;
      }

      Rpc.listen(models.types.loadStructNodes(structs),
          new UiCallback<List<StructNode>, List<StructNode>>(this, LOG) {
        @Override
        protected List<StructNode> onRpcThread(Result<List<StructNode>> result)
            throws RpcException, ExecutionException {
          return StructNode.simplifyTrees(result.get());
        }

        @Override
        protected void onUiThread(List<StructNode> result) {
          treeViewer.setInput(result);
          // Expand the TypeInfo.StructType nodes by default.
          for (StructNode node : result) {
            if (node.getTypeCase() == TyCase.STRUCT) {
              treeViewer.expandToLevel(node, 1);
            }
          }
          for (TreeItem item : treeViewer.getTree().getItems()) {
            // Give visual hint to the elements of level 1.
            item.setBackground(widgets.theme.memoryFirstLevelBackground());
          }
          packColumns(treeViewer.getTree());
        }
      });
    }

    private TreeViewer createStructTreeViewer() {
      TreeViewer tree = createTreeViewer(this, SWT.NONE);
      tree.getTree().setHeaderVisible(true);
      tree.setLabelProvider(new LabelProvider());
      tree.setContentProvider(new ITreeContentProvider() {
        @SuppressWarnings("unchecked")
        @Override
        // Please pass argument of type List<StructNode> to setInput(argument) method for this tree.
        public Object[] getElements(Object inputElement) {
          return ((List<StructNode>)inputElement).toArray();
        }

        @Override
        public boolean hasChildren(Object element) {
          return element instanceof StructNode && ((StructNode)element).hasChildren();
        }

        @Override
        public Object getParent(Object element) {
          return null;
        }

        @Override
        public Object[] getChildren(Object element) {
          if (element instanceof StructNode) {
            return ((StructNode)element).getChildren().toArray();
          } else {
            return new Object[0];
          }
        }
      });

      // Adjusts tree column width when expanding and collapsing.
      tree.addTreeListener(new ITreeViewerListener() {
        @Override
        public void treeExpanded(TreeExpansionEvent event) {
          // Add some delay here to avoid calling too early and no pack is done.
          scheduleIfNotDisposed(tree.getTree(), () -> packColumns(tree.getTree()));
        }

        @Override
        public void treeCollapsed(TreeExpansionEvent event) {
          // Add some delay here to avoid calling too early and no pack is done.
          scheduleIfNotDisposed(tree.getTree(), () -> packColumns(tree.getTree()));
        }
      });
      tree.addDoubleClickListener(e -> Display.getDefault().asyncExec(() -> packColumns(tree.getTree())));

      // Add link action
      tree.getTree().addListener(SWT.MouseUp, e -> {
        if (isOnLink(tree, new Point(e.x, e.y))) {
          StructNode node = ((StructNode)tree.getTree().getItem(new Point(e.x, e.y)).getData());
          if (node.isLargeArray()) {
            // Large array links are synthetic on the client side
            long address = node.getRootAddress();
            blockPanel.goToObservation(address);
            folder.setSelection(0);
          } else if (node.getValue().hasLink()) {
            // Other links come from the server, and are paths to arbitrary places
            models.follower.onFollow(node.getValue().getLink());
          }
        }
      });
      tree.getTree().addListener(SWT.MouseMove, e -> {
        if (isOnLink(tree, new Point(e.x, e.y))) {
          setCursor(getDisplay().getSystemCursor(SWT.CURSOR_HAND));
        } else {
          setCursor(null);
        }
      });

      createTreeColumn(tree, "Type", e -> ((StructNode)e).getTypeFormatted());
      createTreeColumn(tree, "Name", e -> ((StructNode)e).getStructName());
      TreeViewerColumn valueColumn = createTreeColumn(tree, "Value");
      valueColumn.setLabelProvider(new StyledCellLabelProvider(){
        @Override
        public void update(ViewerCell cell) {
          StructNode e = (StructNode)cell.getElement();
          cell.setText(e.getValueFormatted());
          if (e.getValue().hasLink() || e.isLargeArray()) {
            StyleRange style = new StyleRange();
            widgets.theme.linkStyler().applyStyles(style);
            style.length = cell.getText().length();
            cell.setStyleRanges(new StyleRange[] { style });
          }

          super.update(cell);
        }
      });

      return tree;
    }

    // Return true if the cursor is on the value column of a link (either link from the server, or a large array)
    private boolean isOnLink(TreeViewer tree, Point point) {
      TreeItem item = tree.getTree().getItem(point);
      return item != null && item.getData() instanceof StructNode
          && (((StructNode)item.getData()).isLargeArray() || ((StructNode)item.getData()).getValue().hasLink())
          && item.getBounds(2).contains(point);
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
    public Iterator<Segment> getLines(long start, long end);

    /**
     * @return the range of columns containing the given column that can be selected as a group.
     */
    public IntRange getSelectableRegion(int column);

    /**
     * @return the read selections within the given range of rows.
     */
    public List<Selection> getReads(long startRow, long endRow);

    /**
     * @return the write selections within the given range of rows.
     */
    public List<Selection> getWrites(long startRow, long endRow);

    /**
     * @return the given selected memory area as copy-paste content.
     */
    public CopyData[] getCopyData(Selection selection);
  }

  private static class SegmentLoader {
    private final Memory.Data data;
    private final Loadable loadable;
    private long offset;
    private int length;
    private ListenableFuture<Memory.Segment> segment;

    public SegmentLoader(Memory.Data data, Loadable loadable) {
      this.data = data;
      this.loadable = loadable;
    }

    public Memory.Segment load(long off, int len) {
      if (segment == null || offset != off || length != len) {
        segment = data.load(off, len);
        offset = off;
        length = len;

        if (!segment.isDone()) {
          loadable.startLoading();
          segment.addListener(loadable::stopLoading, MoreExecutors.directExecutor());
        }
      }

      if (segment.isDone()) {
        try {
          return Rpc.get(segment, 0, TimeUnit.MILLISECONDS);
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
      }

      return null;
    }
  }

  /**
   * {@link MemoryModel} displaying a fixed amount of bytes per line.
   */
  private static abstract class FixedMemoryModel implements MemoryModel {
    protected static final char UNKNOWN_CHAR = '?';
    protected static final int BYTES_PER_ROW = 16;

    protected final Memory.Data data;
    private final SegmentLoader loader;
    protected final long alignOffset;
    private final long rows;

    public FixedMemoryModel(Memory.Data data, Loadable loadable, long alignOffset) {
      this.data = data;
      this.loader = new SegmentLoader(data, loadable);
      this.alignOffset = alignOffset;
      this.rows = UnsignedLong.fromLongBits(data.getEndAddress() - alignOffset).bigIntegerValue()
          .add(BigInteger.valueOf(BYTES_PER_ROW))
          .divide(BigInteger.valueOf(BYTES_PER_ROW))
          .longValueExact();
    }

    /**
     * Returns the physical address of the first byte in the given row.
     */
    protected long getAddress(long row) {
      return alignOffset + row * BYTES_PER_ROW;
    }

    @Override
    public long getLineCount() {
      return rows;
    }

    private Memory.Segment getMemorySegment(long startRow, long endRow) {
      if (startRow < 0 || endRow < startRow || endRow > getLineCount()) {
        throw new IndexOutOfBoundsException(
            "[" + startRow + ", " + endRow + ") outside of [0, " + getLineCount() + ")");
      }
      return loader.load(getAddress(startRow), (int)(endRow - startRow) * BYTES_PER_ROW);
    }

    @Override
    public Iterator<Segment> getLines(long startRow, long endRow) {
      Memory.Segment segment = getMemorySegment(startRow, endRow);
      if (segment != null) {
        return getLines(startRow, endRow, segment);
      } else {
        return Collections.emptyIterator();
      }
    }

    protected Iterator<Segment> getLines(long startRow, long endRow, Memory.Segment memory) {
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

    protected abstract void getLine(Segment segment, Memory.Segment memory, long line);

    protected abstract IntRange[] getDataRanges();

    @Override
    public List<Selection> getReads(long startRow, long endRow) {
      Memory.Segment memory = getMemorySegment(startRow, endRow);
      return memory == null || !memory.hasReads() ? Collections.emptyList() :
          getSelections(memory.getReads(), startRow);
    }

    @Override
    public List<Selection> getWrites(long startRow, long endRow) {
      Memory.Segment memory = getMemorySegment(startRow, endRow);
      return memory == null || !memory.hasWrites() ? Collections.emptyList() :
          getSelections(memory.getWrites(), startRow);
    }

    private List<Selection> getSelections(Iterator<Service.MemoryRange> ranges, long startRow) {
      List<Selection> result = Lists.newArrayList();
      //IntRange[] ranges = getDataRanges();
      while (ranges.hasNext()) {
        MemoryRange memRange = ranges.next();
        for (IntRange colRange : getDataRanges()) {
          long start = memRange.getBase(), end = start + memRange.getSize();
          result.add(new Selection(colRange,
              getColForOffset(colRange, start, true), startRow + getRowForOffset(start),
              getColForOffset(colRange, end, false), startRow + getRowForOffset(end)));
        }
      }
      return result;
    }

    private static long getRowForOffset(long offset) {
      return Long.divideUnsigned(offset, BYTES_PER_ROW);
    }

    // TODO: this fails miserably for multi-byte views (e.g float/double, short/int/long, etc),
    //       because it may stop/start selecting in the middle of a value.
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

    public CharBufferMemoryModel(
        Memory.Data data, Loadable loadable, long offset, int charsPerRow, IntRange memoryRange) {
      super(data, loadable, offset);
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
    protected void getLine(Segment segment, Memory.Segment memory, long line) {
      segment.array = new char[charsPerRow];
      segment.offset = 0;
      segment.count = charsPerRow;
      formatLine(segment.array, memory, line);
    }

    private void formatLine(char[] array, Memory.Segment memory, long line) {
      Arrays.fill(array, ' ');
      long address = getAddress(line);
      for (int i = CHARS_PER_ADDRESS - 1; i >= 0; i--, address >>>= 4) {
        array[i] = HEX_DIGITS[(int)address & 0xF];
      }
      array[CHARS_PER_ADDRESS] = ':';
      formatMemory(array, memory);
    }

    protected abstract void formatMemory(char[] buffer, Memory.Segment memory);

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
      return Futures.getUnchecked(MoreFutures.transform(data.load(selection.startRow * BYTES_PER_ROW + alignOffset,
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

    public ByteMemoryModel(Memory.Data data, Loadable loadable, long offset) {
      super(data, loadable, offset, CHARS_PER_ROW, BYTES_RANGE);
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
        // Copy the actual data, rather than the display.
        return Futures.getUnchecked(MoreFutures.transform(
            data.load(selection.startRow * BYTES_PER_ROW,
                (int)(selection.endRow - selection.startRow + 1) * BYTES_PER_ROW),
                s -> new CopyData[] {
                  CopyData.text(s.asString(selection.startCol - ASCII_RANGE.from,
                      s.length() - selection.startCol + ASCII_RANGE.from -
                      ASCII_RANGE.to + selection.endCol))
                  }));
      } else {
        return super.getCopyData(selection);
      }
    }

    @Override
    protected void formatMemory(char[] buffer, Memory.Segment memory) {
      for (int i = 0, j = ADDRESS_CHARS; i < memory.length();
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

      for (int i = 0, j = ADDRESS_CHARS + BYTES_CHARS + ASCII_SEPARATOR; i < memory.length();
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

    protected IntegersMemoryModel(Memory.Data data, Loadable loadable, long offset, int size) {
      super(data, loadable, offset, charsPerRow(size), itemsRange(size));
      this.size = size;
    }

    @Override
    protected void formatMemory(char[] buffer, Memory.Segment memory) {
      final int charsPerItem = charsPerItem(size);
      for (int i = 0, j = ADDRESS_CHARS; i + size <= memory.length();
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
    public Int16MemoryModel(Memory.Data data, Loadable loadable, long offset) {
      super(data, loadable, offset, 2);
    }
  }

  /**
   * {@link IntegersMemoryModel} for 4 byte integers.
   */
  private static class Int32MemoryModel extends IntegersMemoryModel {
    public Int32MemoryModel(Memory.Data data, Loadable loadable, long offset) {
      super(data, loadable, offset, 4);
    }
  }

  /**
   * {@link IntegersMemoryModel} for 8 byte integers.
   */
  private static class Int64MemoryModel extends IntegersMemoryModel {
    public Int64MemoryModel(Memory.Data data, Loadable loadable, long offset) {
      super(data, loadable, offset, 8);
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

    public Float16MemoryModel(Memory.Data data, Loadable loadable, long offset) {
      super(data, loadable, offset, CHARS_PER_ROW, FLOATS_RANGE);
    }

    @Override
    protected void formatMemory(char[] buffer, Memory.Segment memory) {
      StringBuilder sb = new StringBuilder(50);
      for (int i = 0, j = ADDRESS_CHARS; i + 1 < memory.length();
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

    public Float32MemoryModel(Memory.Data data, Loadable loadable, long offset) {
      super(data, loadable, offset, CHARS_PER_ROW, FLOATS_RANGE);
    }

    @Override
    protected void formatMemory(char[] buffer, Memory.Segment memory) {
      StringBuilder sb = new StringBuilder(50);
      for (int i = 0, j = ADDRESS_CHARS; i + 3 < memory.length();
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

    public Float64MemoryModel(Memory.Data data, Loadable loadable, long offset) {
      super(data, loadable, offset, CHARS_PER_ROW, DOUBLES_RANGE);
    }

    @Override
    protected void formatMemory(char[] buffer, Memory.Segment memory) {
      StringBuilder sb = new StringBuilder(50);
      for (int i = 0, j = ADDRESS_CHARS; i + 7 < memory.length();
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
