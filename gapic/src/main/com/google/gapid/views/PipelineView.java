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

import static com.google.gapid.proto.service.api.API.ResourceType.PipelineResource;
import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Loadable.MessageType.Info;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createTextarea;
import static com.google.gapid.widgets.Widgets.createStandardTabFolder;
import static com.google.gapid.widgets.Widgets.createStandardTabItem;
import static com.google.gapid.widgets.Widgets.createTableColumn;
import static com.google.gapid.widgets.Widgets.createTableViewer;
import static com.google.gapid.widgets.Widgets.createGroup;
import static com.google.gapid.widgets.Widgets.disposeAllChildren;
import static com.google.gapid.widgets.Widgets.packColumns;
import static com.google.gapid.widgets.Widgets.sorting;
import static com.google.gapid.widgets.Widgets.withAsyncRefresh;
import static com.google.gapid.widgets.Widgets.ColumnAndComparator;
import static java.util.logging.Level.FINE;

import com.google.gapid.models.Models;
import com.google.gapid.models.Resources;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.SingleInFlight;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.viewers.TableViewer;
import org.eclipse.jface.viewers.ArrayContentProvider;
import org.eclipse.jface.viewers.ViewerComparator;
import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.SashForm;
import org.eclipse.swt.widgets.TabFolder;
import org.eclipse.swt.widgets.TabItem;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Text;
import org.eclipse.swt.widgets.TableItem;
import org.eclipse.swt.widgets.Group;
import org.eclipse.swt.widgets.Widget;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;

import java.util.Arrays;
import java.util.List;
import java.util.ArrayList;
import java.util.Collections;
import java.util.Comparator;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

/**
 * View the displays the information for each stage of the pipeline.
 */
public class PipelineView extends Composite
    implements Tab, Capture.Listener, Resources.Listener, CommandStream.Listener {
  protected static final Logger LOG = Logger.getLogger(ShaderView.class.getName());

  private Models models;
  private Widgets widgets;

  private final LoadablePanel<Composite> loading;
  private final TableViewer pipelineTable;

  private List<Data> pipelines = Collections.emptyList();

  public PipelineView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;
    this.widgets = widgets;

    setLayout(new FillLayout());

    loading = LoadablePanel.create(this, widgets,
        panel -> createComposite(panel, new FillLayout(SWT.VERTICAL)));

    SashForm splitter = new SashForm(loading.getContents(), SWT.HORIZONTAL);

    Composite pipelineContainer = createComposite(splitter, new GridLayout(1, false), SWT.BORDER);
    Composite stagesContainer = createComposite(splitter, new FillLayout());
    splitter.setWeights(models.settings.pipelineSplitterWeights);

    pipelineTable = createTableViewer(pipelineContainer, SWT.BORDER | SWT.SINGLE | SWT.FULL_SELECTION);
    pipelineTable.getTable().setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
    pipelineTable.setContentProvider(ArrayContentProvider.getInstance());

    ColumnAndComparator[] columns = new ColumnAndComparator[]{
      createTableColumn(pipelineTable, "Pipelines", Data::getId,
          Comparator.comparingLong(Data::getSortId)),
      createTableColumn(pipelineTable, "Bound", Data::getBound,
          Comparator.comparingInt(Data::getSortBound))
    };

    sorting(pipelineTable, Arrays.asList(columns));
    pipelineTable.getTable().setSortColumn(columns[1].getTableColumn());
    pipelineTable.getTable().setSortDirection(SWT.UP);
    pipelineTable.setComparator(columns[1].getComparator(false));

    pipelineTable.setInput(pipelines);
    packColumns(pipelineTable.getTable());

    pipelineTable.getTable().addListener(SWT.Selection, e -> {
      disposeAllChildren(stagesContainer);
      TabFolder folder = createStandardTabFolder(stagesContainer);
      createPipelineTabs(folder);
      stagesContainer.requestLayout();
    });

    models.resources.addListener(this);
    models.commands.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.resources.removeListener(this);
      models.commands.removeListener(this);
    });
  }

  @Override
  public void reinitialize() {

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
 
  }

  @Override
  public void onCommandsLoaded() {
    if (!models.commands.isLoaded()) {
      loading.showMessage(Info, Messages.CAPTURE_LOAD_FAILURE);
    } else {
      if (models.commands.getSelectedCommands() == null) {
        loading.showMessage(Info, Messages.SELECT_COMMAND);
      } else {
        loading.stopLoading();
      }
    }
  }

  @Override
  public void onCommandsSelected(CommandIndex path) {
    loading.stopLoading();
    updatePipelines();
  }


  public void updatePipelines() {
    pipelines.clear();
    Widgets.Refresher refresher = withAsyncRefresh(pipelineTable);
    Resources.ResourceList resources = models.resources.getResources(API.ResourceType.PipelineResource);
    pipelines = new ArrayList<Data>();
    if (!resources.isEmpty()) {
      resources.stream()
        .map(r -> new Data(r.resource))
        .forEach(data -> {
          pipelines.add(data);
          data.load(models, pipelineTable.getTable(), refresher);
        });
    }
    
    pipelineTable.setInput(pipelines);
    packColumns(pipelineTable.getTable());
  }

  public void createPipelineTabs(TabFolder folder) {
    int selection = pipelineTable.getTable().getSelectionIndex();
    if (selection >= 0) {
      Data data = (Data)pipelineTable.getElementAt(selection);
      Rpc.listen(models.resources.loadResource(data.info),
          new UiCallback<API.ResourceData, API.Pipeline>(this, LOG) {
        @Override
        protected API.Pipeline onRpcThread(Rpc.Result<API.ResourceData> result)
            throws RpcException, ExecutionException {
          API.Pipeline pipeline = result.get().getPipeline();
          data.resource = pipeline;
          return pipeline;
        }
  
        @Override
        protected void onUiThread(API.Pipeline result) {
          List<API.Stage> stages = result.getStagesList();

          for (API.Stage stage : stages) {
            TabItem item = createStandardTabItem(folder, stage.getDebugName());

            Group group = createGroup(folder, stage.getStageName());

            item.setControl(group);
          }
        }
      });
    }
  }

  @Override
  public Control getControl() {
    return this;
  }

  private static class Data {
    public final Service.Resource info;
    public Object resource;
    public boolean bound;

    public Data(Service.Resource info) {
      this.info = info;
    }

    public String getId() {
      return info.getHandle();
    }

    public long getSortId() {
      return info.getOrder();
    }

    public String getLabel() {
      return info.getLabel();
    }

    public String getBound() {
      return (bound ? "Yes" : "No");
    }

    public int getSortBound() {
      return (bound ? 0 : 1);
    }

    public void load(Models models, Widget widget, Widgets.Refresher refresher) {
      Rpc.listen(models.resources.loadResource(info),
          new UiCallback<API.ResourceData, API.Pipeline>(widget, LOG) {
            @Override
            protected API.Pipeline onRpcThread(Rpc.Result<API.ResourceData> result)
                throws RpcException, ExecutionException {
              return result.get().getPipeline();
            }
      
            @Override
            protected void onUiThread(API.Pipeline result) {
              bound = result.getBound();
              refresher.refresh();
            }
          });
    }
  }
}
