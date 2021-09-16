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

import static com.google.gapid.server.Client.throwIfError;
import static com.google.gapid.util.Colors.lerp;
import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.util.Logging.throttleLogRpcError;
import static com.google.gapid.widgets.Widgets.createBoldLabel;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createGroup;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createLink;
import static com.google.gapid.widgets.Widgets.createScrolledComposite;
import static com.google.gapid.widgets.Widgets.createTableColumn;
import static com.google.gapid.widgets.Widgets.createTableViewer;
import static com.google.gapid.widgets.Widgets.disposeAllChildren;
import static com.google.gapid.widgets.Widgets.packColumns;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withMargin;

import com.google.common.base.Joiner;
import com.google.common.collect.Lists;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.Models;
import com.google.gapid.models.Resources;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiErrorCallback;
import com.google.gapid.server.Client.DataUnavailableException;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.viewers.ArrayContentProvider;
import org.eclipse.jface.viewers.ColumnViewerToolTipSupport;
import org.eclipse.jface.viewers.StyledCellLabelProvider;
import org.eclipse.jface.viewers.TableViewer;
import org.eclipse.jface.viewers.TableViewerColumn;
import org.eclipse.jface.viewers.ViewerCell;
import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.ScrolledComposite;
import org.eclipse.swt.custom.StackLayout;
import org.eclipse.swt.custom.StyleRange;
import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.RGB;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.layout.RowLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Group;
import org.eclipse.swt.widgets.Label;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

/**
 * View the displays the information for each stage of the pipeline.
 */
public class PipelineView extends Composite
    implements Tab, Capture.Listener, CommandStream.Listener, Resources.Listener {
  protected static final Logger LOG = Logger.getLogger(PipelineView.class.getName());

  protected final Models models;
  protected final Widgets widgets;
  protected final LoadablePanel<Composite> loading;
  protected final Composite stagesContainer;
  protected final Color hoverColor;

  protected String selectedStage = null;
  protected Button hoveredButton = null;

  public PipelineView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;
    this.widgets = widgets;

    setLayout(new FillLayout());

    loading = LoadablePanel.create(this, widgets,
        panel -> createComposite(panel, new FillLayout(SWT.VERTICAL)));

    stagesContainer = createComposite(loading.getContents(), new GridLayout(1, false));

    RGB selectRGB = getDisplay().getSystemColor(SWT.COLOR_BLUE).getRGB();
    RGB unselectRGB = getDisplay().getSystemColor(SWT.COLOR_LIST_BACKGROUND).getRGB();
    hoverColor = new Color(getDisplay(), lerp(selectRGB, unselectRGB, 0.5f));

    models.capture.addListener(this);
    models.commands.addListener(this);
    models.resources.addListener(this);
    addListener(SWT.Dispose, e -> {
      hoverColor.dispose();
      models.capture.removeListener(this);
      models.commands.removeListener(this);
      models.resources.removeListener(this);
    });
  }

  @Override
  public void reinitialize() {
    updatePipelines();
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
  public void onResourcesLoaded() {
    updatePipelines();
  }

  @Override
  public void onCommandsLoaded() {
    if (!models.commands.isLoaded()) {
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    } else if (models.commands.getSelectedCommands() == null) {
      loading.showMessage(Info, Messages.SELECT_DRAW_CALL);
    }
  }

  @Override
  public void onCommandsSelected(CommandIndex path) {
    updatePipelines();
  }

  private void updatePipelines() {
    loading.startLoading();

    if (models.commands.getSelectedCommands() == null) {
      loading.showMessage(Info, Messages.SELECT_DRAW_CALL);
    } else if (models.resources.isLoaded()) {
      Rpc.listen(models.resources.loadBoundPipelines(),
          new UiErrorCallback<Service.MultiResourceData, List<API.Pipeline>, Loadable.Message>(this, LOG) {
        @Override
        protected ResultOrError<List<API.Pipeline>, Loadable.Message> onRpcThread(
            Rpc.Result<Service.MultiResourceData> result) {
          try {
            List<API.Pipeline> pipelines = Lists.newArrayList();
            for (Service.MultiResourceData.ResourceOrError resource :
                result.get().getResourcesMap().values()) {
              // TODO: Don't globally fail on the first error.
              API.ResourceData data = throwIfError(resource.getResource(), resource.getError(), null);
              if (data.hasPipeline()) {
                pipelines.add(data.getPipeline());
              }
            }
            return success(pipelines);
          } catch (DataUnavailableException e) {
            return error(Loadable.Message.error(e));
          } catch (RpcException e) {
            models.analytics.reportException(e);
            return error(Loadable.Message.error(e));
          } catch (ExecutionException e) {
            models.analytics.reportException(e);
            throttleLogRpcError(LOG, "Failed to load pipelines", e);
            return error(Loadable.Message.error(e.getCause().getMessage()));
          }
        }

        @Override
        protected void onUiThreadSuccess(List<API.Pipeline> pipelines) {
          setPipelines(pipelines);
        }

        @Override
        protected void onUiThreadError(Loadable.Message error) {
          loading.showMessage(error);
        }
      });
    }
  }

  protected void setPipelines(List<API.Pipeline> pipelines) {
    loading.stopLoading();
    disposeAllChildren(stagesContainer);
    createPipelineTabs(stagesContainer, pipelines);
    stagesContainer.requestLayout();
  }

  private void createPipelineTabs(Composite folder, List<API.Pipeline> pipelines) {
    HashMap<String, StageUI> stageMap = new HashMap<String, StageUI>();

    if (!pipelines.isEmpty()) {
      RowLayout stripLayout = new RowLayout();
      stripLayout.fill = true;
      stripLayout.spacing = 0;
      stripLayout.wrap = true;
      Composite stripComposite = withLayoutData( createComposite(folder, stripLayout),
          new GridData(SWT.FILL, SWT.TOP, true, false));
      withLayoutData( new Label(folder, SWT.SEPARATOR | SWT.HORIZONTAL),
          new GridData(SWT.FILL, SWT.TOP, true, false));

      StackLayout stageStack = new StackLayout();
      Composite stageComposite = withLayoutData( createComposite(folder, stageStack),
          new GridData(SWT.FILL, SWT.FILL, true, true));

      for (int pipeIndex = 0; pipeIndex < pipelines.size(); pipeIndex++) {
        List<API.Stage> stages = pipelines.get(pipeIndex).getStagesList();

        for (int stageIndex = 0; stageIndex < stages.size(); stageIndex++) {
          API.Stage stage = stages.get(stageIndex);
          Button stageButton = new Button(stripComposite, SWT.FLAT);
          stageMap.put(stage.getDebugName(), new StageUI(stageButton, createStage(stageComposite, stage)));

          stageButton.setText(stage.getDebugName());
          stageButton.addListener(SWT.Selection, e -> {
            stageStack.topControl = stageMap.get(stage.getDebugName()).stageComposite;
            stageComposite.layout();
            StageUI stageUI = stageMap.get(selectedStage);
            if (stageUI != null) {
              stageUI.stageButton.redraw();
            }
            selectedStage = stage.getDebugName();
            stageButton.redraw();
          });

          stageButton.setBackground(getDisplay().getSystemColor(SWT.COLOR_LIST_BACKGROUND));
          stageButton.setFont(widgets.theme.bigBoldFont());
          stageButton.setToolTipText(stage.getStageName());

          if (!stage.getEnabled()) {
            stageButton.setEnabled(false);
            stageButton.addListener(SWT.Paint, e -> {
              Rectangle areaSize = stageButton.getBounds();
              e.gc.setBackground(getDisplay().getSystemColor(SWT.COLOR_LIST_FOREGROUND));
              e.gc.setLineWidth((int)(areaSize.height * 0.1));
              e.gc.drawLine(0, areaSize.height / 2, areaSize.width, areaSize.height / 2);
            });
          } else {
            stageButton.addListener(SWT.Paint, e -> {
              Rectangle areaSize = stageButton.getBounds();
              e.gc.setLineWidth((int)(areaSize.height * 0.1));

              if (hoveredButton == stageButton || stageMap.get(selectedStage).stageButton == stageButton) {
                e.gc.setForeground(getDisplay().getSystemColor(SWT.COLOR_BLUE));
              } else {
                e.gc.setBackground(getDisplay().getSystemColor(SWT.COLOR_LIST_FOREGROUND));
              }
              e.gc.drawPolygon(new int[] {0, 0, areaSize.width - 1, 0, areaSize.width - 1, areaSize.height - 1, 0, areaSize.height - 1});
            });

            stageButton.addListener(SWT.MouseEnter, e -> {
              hoveredButton = stageButton;
              stageButton.setBackground(hoverColor);
            });

            stageButton.addListener(SWT.MouseExit, e -> {
              hoveredButton = null;
              stageButton.setBackground(getDisplay().getSystemColor(SWT.COLOR_LIST_BACKGROUND));
            });
          }

          if (stageIndex != stages.size()-1) {
            Label arrowSpace = new Label(stripComposite, SWT.FILL);
            arrowSpace.setText("     ");
            arrowSpace.setFont(widgets.theme.bigBoldFont());
            API.Stage nextStage = stages.get(stageIndex+1);

            arrowSpace.addListener(SWT.Paint, e -> {
              Rectangle areaSize = arrowSpace.getBounds();
              int lineLength = (int)(areaSize.width * 0.8);
              int triangleBaseLength = (int)(areaSize.height * 0.2);

              e.gc.setBackground(getDisplay().getSystemColor(SWT.COLOR_LIST_FOREGROUND));
              e.gc.setLineWidth((int)(areaSize.height * 0.1));
              e.gc.drawLine(0, areaSize.height / 2, lineLength, areaSize.height / 2);

              if (nextStage.getEnabled()) {
                e.gc.fillPolygon(new int[] {lineLength,  areaSize.height / 2 + triangleBaseLength, areaSize.width,  areaSize.height / 2, lineLength,  areaSize.height / 2 - triangleBaseLength});
              } else {
                e.gc.drawLine(lineLength, areaSize.height / 2, areaSize.width, areaSize.height / 2);
              }
            });
          }
        }

        if (pipeIndex != pipelines.size()-1) {
          Label pipeSeparator = new Label(stripComposite, SWT.FILL);
          pipeSeparator.setText("     ");
          pipeSeparator.setFont(widgets.theme.bigBoldFont());
        }

        if (selectedStage == null || stageMap.get(selectedStage) == null) {
          selectedStage = pipelines.get(0).getStagesList().get(0).getDebugName();
        }

        stageStack.topControl = stageMap.get(selectedStage).stageComposite;
      }
    }
  }

  private Composite createStage(Composite parent, API.Stage currentStage) {
    Composite stageGroup = createComposite(parent, new GridLayout());
    FillLayout nameLayout = new FillLayout(SWT.VERTICAL);
    nameLayout.marginHeight = 5;
    Composite nameComposite = withLayoutData( createComposite(stageGroup, nameLayout),
        new GridData(SWT.BEGINNING, SWT.BEGINNING, false, false));

    Label stageName = createLabel(nameComposite, currentStage.getStageName() + " (" + currentStage.getDebugName() + ")");
    stageName.setFont(widgets.theme.bigBoldFont());

    Composite dataComposite = withLayoutData(
        createComposite(stageGroup, withMargin(new GridLayout(1, true), 0, 5)),
        new GridData(SWT.FILL, SWT.FILL, true, true));

    for (API.DataGroup dataGroup : currentStage.getGroupsList()) {
      switch (dataGroup.getDataCase()) {
        case KEY_VALUES: {
          Group dataGroupComposite = withLayoutData(
              createGroup(dataComposite, dataGroup.getGroupName(), new GridLayout(1, false)),
              new GridData(SWT.FILL, SWT.TOP, true, false));
          dataGroupComposite.setFont(widgets.theme.subTitleFont());
          ScrolledComposite scrollComposite = withLayoutData( createScrolledComposite(dataGroupComposite,
              new FillLayout(), SWT.V_SCROLL | SWT.H_SCROLL),
              new GridData(SWT.FILL, SWT.FILL, true, true));

          GridLayout gridLayout = new GridLayout(2, false);
          gridLayout.marginWidth = 5;
          gridLayout.marginHeight = 5;
          gridLayout.horizontalSpacing = -1;
          gridLayout.verticalSpacing = -1;
          Composite contentComposite = createComposite(scrollComposite, gridLayout);

          List<API.KeyValuePair> kvpList = dataGroup.getKeyValues().getKeyValuesList();

          boolean dynamicExists = false;

          for (API.KeyValuePair kvp : kvpList) {
            GridLayout kvpLayout = new GridLayout(1, false);
            kvpLayout.marginHeight = 5;
            kvpLayout.marginWidth = 5;
            Composite keyComposite = withLayoutData( createComposite(contentComposite, kvpLayout, SWT.BORDER),
                new GridData(SWT.FILL, SWT.TOP, false, false));

            Label keyLabel = withLayoutData( createBoldLabel(keyComposite, kvp.getName() + (kvp.getDynamic() ? "*:" : ":")),
                new GridData(SWT.RIGHT, SWT.CENTER, true, true));

            if (!kvp.getDependee().equals("")) {
              if (kvp.getActive()) {
                keyLabel.setToolTipText("Activated by " + kvp.getDependee());
              } else {
                keyComposite.setToolTipText("Deactivated by " + kvp.getDependee());
                keyLabel.setEnabled(false);
              }
            }

            if (!dynamicExists && kvp.getDynamic()) {
              dataGroupComposite.setText(dataGroup.getGroupName() + " (* value set dynamically)");
              dynamicExists = true;
            }

            Composite valueComposite = withLayoutData( createComposite(contentComposite, kvpLayout, SWT.BORDER),
                new GridData(SWT.FILL, SWT.TOP, false, false));

            DataValue dv = convertDataValue(kvp.getValue());
            if (dv.link != null) {
              withLayoutData( createLink(valueComposite,"<a>" + dv.getDisplay() + "</a>", e -> {
                  for (Path.Any p : dv.link) {
                    models.follower.onFollow(p);
                  }}),
                  new GridData(SWT.LEFT, SWT.CENTER, true, true));
            } else {
              Label valueLabel = withLayoutData( createLabel(valueComposite, dv.getDisplay()),
                  new GridData(SWT.LEFT, SWT.CENTER, true, true));

              if (dv.tooltipValue == null && !kvp.getDependee().equals("")) {
                valueLabel.setToolTipText((kvp.getActive() ?  "Activated by " : "Deactivated by ") + kvp.getDependee());
                valueLabel.setEnabled(kvp.getActive());
              } else {
                valueLabel.setToolTipText(dv.tooltipValue);
              }
            }
          }

          scrollComposite.setContent(contentComposite);
          scrollComposite.setExpandVertical(true);
          scrollComposite.setExpandHorizontal(true);
          scrollComposite.addListener(SWT.Resize, event -> {
            Rectangle scrollArea = scrollComposite.getClientArea();

            int currentNumColumns = gridLayout.numColumns;
            int numChildren = contentComposite.getChildren().length;
            Point tableSize = contentComposite.computeSize(SWT.DEFAULT, SWT.DEFAULT, true);

            if (tableSize.x < scrollArea.width) {
              while (tableSize.x < scrollArea.width && gridLayout.numColumns < numChildren) {
                gridLayout.numColumns += 2;
                tableSize = contentComposite.computeSize(SWT.DEFAULT, SWT.DEFAULT, true);
              }

              if (tableSize.x > scrollArea.width) {
                gridLayout.numColumns -= 2;
              }
            } else {
              while (tableSize.x > scrollArea.width && gridLayout.numColumns >= 4) {
                gridLayout.numColumns -= 2;
                tableSize = contentComposite.computeSize(SWT.DEFAULT, SWT.DEFAULT, true);
              }
            }

            if (gridLayout.numColumns != currentNumColumns) {
              scrollComposite.layout();
            }

            scrollComposite.setMinHeight(contentComposite.computeSize(scrollArea.width, SWT.DEFAULT).y);
          });

          break;
        }
        case TABLE:
          Group dataGroupComposite = withLayoutData(
              createGroup(dataComposite, dataGroup.getGroupName()),
              new GridData(SWT.FILL, SWT.TOP, true, false));
          dataGroupComposite.setFont(widgets.theme.subTitleFont());

          API.Table dataTable = dataGroup.getTable();
          if (dataTable.getDynamic()) {
            dataGroupComposite.setText(dataGroup.getGroupName() + " (table was set dynamically)");
          }

          TableViewer groupTable = createTableViewer(dataGroupComposite, SWT.BORDER | SWT.H_SCROLL | SWT.V_SCROLL | SWT.FULL_SELECTION);
          List<API.Row> rows = dataTable.getRowsList();

          groupTable.setContentProvider(ArrayContentProvider.getInstance());

          ColumnViewerToolTipSupport.enableFor(groupTable);

          for (int i = 0; i < dataTable.getHeadersCount(); i++) {
            int col = i;
            TableViewerColumn tvc = createTableColumn(groupTable, dataTable.getHeaders(i));

            StyledCellLabelProvider cellLabelProvider = new StyledCellLabelProvider() {
              @Override
              public void update(ViewerCell cell) {
                DataValue dv = convertDataValue(((API.Row)cell.getElement()).getRowValues(col));
                String display = dv.getDisplay();

                cell.setText(display);
                if (!dataTable.getActive()) {
                  cell.setForeground(getDisplay().getSystemColor(SWT.COLOR_DARK_GRAY));
                }

                if (dv.link != null) {
                  StyleRange style = new StyleRange();
                  widgets.theme.linkStyler().applyStyles(style);
                  style.length = display.length();
                  cell.setStyleRanges(new StyleRange[] { style });
                }

                super.update(cell);
              }

              @Override
              public String getToolTipText(Object element) {
                DataValue dv = convertDataValue(((API.Row)element).getRowValues(col));
                if (dv != null) {
                  if (dv.tooltipValue == null && !dataTable.getDependee().equals("")) {
                    return ((dataTable.getActive() ? "Activated by "  : "Deactivated by ") + dataTable.getDependee());
                  } else {
                    return dv.tooltipValue;
                  }
                }

                return null;
              }
            };

            tvc.setLabelProvider(cellLabelProvider);
          }

          groupTable.setInput(rows);

          packColumns(groupTable.getTable());

          groupTable.getTable().addListener(SWT.MouseDown, e -> {
            ViewerCell cell = groupTable.getCell(new Point(e.x, e.y));

            if (cell != null) {
              DataValue dv = convertDataValue(((API.Row)cell.getElement()).getRowValues(cell.getColumnIndex()));

              if (dv.link != null) {
                for (Path.Any p : dv.link) {
                  models.follower.onFollow(p);
                }
                return;
              }
            }
          });

          break;

        case SHADER:
          ShaderView.ShaderWidget shaderView = withLayoutData(
              new ShaderView.ShaderWidget(dataComposite, false, models, widgets),
              new GridData(SWT.FILL, SWT.FILL, true, true));
          shaderView.setShader(null, dataGroup.getShader());
          break;

        case DATA_NOT_SET:
          // Ignore;
          break;
      }
    }

    stageGroup.requestLayout();
    return stageGroup;
  }

  @Override
  public Control getControl() {
    return this;
  }

  protected static DataValue convertDataValue(API.DataValue val) {
    DataValue.Builder dv = new DataValue.Builder();
    switch (val.getValCase()) {
      case VALUE:
        Joiner valueJoiner = Joiner.on(", ");
        ArrayList<String> values = new ArrayList<String>();

        switch (val.getValue().getValCase()) {
          case FLOAT32:
            dv.displayValue = Float.toString(val.getValue().getFloat32());
            break;
          case FLOAT64:
            dv.displayValue = Double.toString(val.getValue().getFloat64());
            break;
          case UINT:
            dv.displayValue = Long.toString(val.getValue().getUint());
            break;
          case SINT:
            dv.displayValue = Long.toString(val.getValue().getSint());
            break;
          case UINT8:
            dv.displayValue = Integer.toString(val.getValue().getUint8());
            break;
          case SINT8:
            dv.displayValue = Integer.toString(val.getValue().getSint8());
            break;
          case UINT16:
            dv.displayValue = Integer.toString(val.getValue().getUint16());
            break;
          case SINT16:
            dv.displayValue = Integer.toString(val.getValue().getSint16());
            break;
          case UINT32:
            dv.displayValue = Integer.toString(val.getValue().getUint32());
            break;
          case SINT32:
            dv.displayValue = Integer.toString(val.getValue().getSint32());
            break;
          case UINT64:
            dv.displayValue = Long.toString(val.getValue().getUint64());
            break;
          case SINT64:
            dv.displayValue = Long.toString(val.getValue().getSint64());
            break;
          case BOOL:
            dv.displayValue = Boolean.toString(val.getValue().getBool());
            break;
          case STRING:
            dv.displayValue = val.getValue().getString();
            break;
          case FLOAT32_ARRAY:
            for (float value : val.getValue().getFloat32Array().getValList()) {
              values.add(Float.toString(value));
            }
            dv.displayValue = valueJoiner.join(values);
            break;
          case FLOAT64_ARRAY:
            for (double value : val.getValue().getFloat64Array().getValList()) {
              values.add(Double.toString(value));
            }
            dv.displayValue = valueJoiner.join(values);
            break;
          case UINT_ARRAY:
            for (long value : val.getValue().getUintArray().getValList()) {
              values.add(Long.toString(value));
            }
            dv.displayValue = valueJoiner.join(values);
            break;
          case SINT_ARRAY:
            for (long value : val.getValue().getSintArray().getValList()) {
              values.add(Long.toString(value));
            }
            dv.displayValue = valueJoiner.join(values);
            break;
          case UINT8_ARRAY:
            for (byte value : val.getValue().getUint8Array()) {
              values.add(Byte.toString(value));
            }
            dv.displayValue = valueJoiner.join(values);
            break;
          case SINT8_ARRAY:
            for (int value : val.getValue().getSint8Array().getValList()) {
              values.add(Integer.toString(value));
            }
            dv.displayValue = valueJoiner.join(values);
            break;
          case UINT16_ARRAY:
            for (int value : val.getValue().getUint16Array().getValList()) {
              values.add(Integer.toString(value));
            }
            dv.displayValue = valueJoiner.join(values);
            break;
          case SINT16_ARRAY:
            for (int value : val.getValue().getSint16Array().getValList()) {
              values.add(Integer.toString(value));
            }
            dv.displayValue = valueJoiner.join(values);
            break;
          case UINT32_ARRAY:
            for (int value : val.getValue().getUint32Array().getValList()) {
              values.add(Integer.toString(value));
            }
            dv.displayValue = valueJoiner.join(values);
            break;
          case SINT32_ARRAY:
            for (int value : val.getValue().getSint32Array().getValList()) {
              values.add(Integer.toString(value));
            }
            dv.displayValue = valueJoiner.join(values);
            break;
          case UINT64_ARRAY:
            for (long value : val.getValue().getUint64Array().getValList()) {
              values.add(Long.toString(value));
            }
            dv.displayValue = valueJoiner.join(values);
            break;
          case SINT64_ARRAY:
            for (long value : val.getValue().getSint64Array().getValList()) {
              values.add(Long.toString(value));
            }
            dv.displayValue = valueJoiner.join(values);
            break;
          case BOOL_ARRAY:
            for (boolean value : val.getValue().getBoolArray().getValList()) {
              values.add(Boolean.toString(value));
            }
            dv.displayValue = valueJoiner.join(values);
            break;
          case STRING_ARRAY:
            dv.displayValue = valueJoiner.join(val.getValue().getStringArray().getValList());
            break;
          default:
        }
        break;

      case HANDLEVAL:
        dv.displayValue = "0x" + Long.toHexString(val.getHandleVal().getValue());
        break;

      case ENUMVAL:
        dv.displayValue = val.getEnumVal().getDisplayValue();
        dv.tooltipValue = val.getEnumVal().getStringValue();
        break;
      case BITFIELD:
        Joiner joiner = Joiner.on((val.getBitfield().getCombined()) ? "" : " | ");
        dv.displayValue = joiner.join(val.getBitfield().getSetDisplayNamesList());
        if (val.getBitfield().getCombined()) {
          joiner = Joiner.on(" | ");
        }
        dv.tooltipValue = joiner.join(val.getBitfield().getSetBitnamesList());
        break;
      case LINK:
        dv.from(convertDataValue(val.getLink().getDisplayVal()));
        dv.link = val.getLink().getLinkList();
        break;
      default:
    }

    if (!val.getLabel().isEmpty()) {
      dv.displayLabel = val.getLabel();
    }
    return dv.build();
  }

  private static class DataValue {
    public final String displayValue;
    public final String displayLabel;
    public final String tooltipValue;
    public final List<Path.Any> link;

    public DataValue(
        String displayValue, String displayLabel, String tooltipValue, List<Path.Any> link) {
      this.displayValue = displayValue;
      this.displayLabel = displayLabel;
      this.tooltipValue = tooltipValue;
      this.link = link;
    }

    public String getDisplay() {
      if (displayLabel == null || displayLabel.isEmpty()) {
        return displayValue;
      }
      return displayValue + " (" + displayLabel + ")";
    }

    public static class Builder {
      public String displayValue = "???";
      public String displayLabel;
      public String tooltipValue;
      public List<Path.Any> link;

      public void from(DataValue dv) {
        displayValue = dv.displayValue;
        displayLabel = dv.displayLabel;
        tooltipValue = dv.tooltipValue;
        link = dv.link;
      }

      public DataValue build() {
        return new DataValue(displayValue, displayLabel, tooltipValue, link);
      }
    }
  }

  private static class StageUI {
    public Button stageButton;
    public Composite stageComposite;

    public StageUI(Button stageButton, Composite stageComposite) {
      this.stageButton = stageButton;
      this.stageComposite = stageComposite;
    }
  }
}
