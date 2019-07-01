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
package com.google.gapid.views;

import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createTreeForViewer;
import static java.util.stream.StreamSupport.stream;

import com.google.common.base.CharMatcher;
import com.google.common.base.Splitter;
import com.google.common.collect.Maps;
import com.google.gapid.image.Images;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Models;
import com.google.gapid.models.TraceTargets;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Loadable.Message;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.DialogBase;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.dialogs.IDialogConstants;
import org.eclipse.jface.resource.JFaceResources;
import org.eclipse.jface.resource.LocalResourceManager;
import org.eclipse.jface.viewers.ITreeContentProvider;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.jface.viewers.Viewer;
import org.eclipse.jface.viewers.ViewerFilter;
import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.graphics.ImageLoader;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.internal.DPIUtil;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Shell;
import org.eclipse.swt.widgets.Text;
import org.eclipse.swt.widgets.Tree;
import org.eclipse.swt.widgets.TreeItem;

import java.util.Map;

/**
 * Dialog to allow the user to pick a trace target for tracing.
 */
public class TraceTargetPickerDialog extends DialogBase implements TraceTargets.Listener {
  private static final int ICON_SIZE_DIP = 24;
  private static final int INITIAL_HEIGHT = 600;
  private static final CharMatcher SEARCH_SEPARATOR = CharMatcher.anyOf("/\\");
  private static final Splitter SEARCH_SPLITTER = Splitter.on(SEARCH_SEPARATOR)
      .trimResults()
      .omitEmptyStrings();

  private final Models models;
  private final Widgets widgets;
  protected final TraceTargets targets;
  protected final ImageLoader imageLoader = new ImageLoader();

  private LoadablePanel<Tree> loading;
  protected TreeViewer tree;
  protected LocalResourceManager resources;

  private Loadable.Message lastLoadError;
  private TraceTargets.Node selected;

  public TraceTargetPickerDialog(Shell shell, Models models, TraceTargets targets, Widgets widgets) {
    super(shell, widgets.theme);
    this.models = models;
    this.targets = targets;
    this.widgets = widgets;
  }

  public TraceTargets.Node getSelected() {
    return selected;
  }

  @Override
  public void onTreeRootLoaded(Message error) {
    if (loading != null) {
      if (error == null) {
        loading.stopLoading();
        tree.setInput(targets.getData());
      } else {
        loading.showMessage(error);
      }
    }
  }

  @Override
  public int open() {
    models.analytics.postInteraction(View.Trace, ClientAction.ShowActivityPicker);

    lastLoadError = null;
    targets.load();
    targets.addListener(this);
    try {
      return super.open();
    } finally {
      targets.removeListener(this);
    }
  }

  @Override
  public String getTitle() {
    return Messages.SELECT_ACTIVITY;
  }

  @Override
  public void create() {
    super.create();
  }

  @Override
  protected Point getInitialSize() {
    Point size = super.getInitialSize();
    size.y = INITIAL_HEIGHT;
    return size;
  }

  @Override
  protected Control createDialogArea(Composite parent) {
    Composite area = (Composite)super.createDialogArea(parent);

    Composite container = createComposite(area, new GridLayout(1, false));
    container.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    Text search = new Text(container, SWT.SINGLE | SWT.SEARCH | SWT.ICON_SEARCH | SWT.ICON_CANCEL);
    search.setLayoutData(new GridData(SWT.FILL, SWT.TOP, true, false));

    loading = LoadablePanel.create(container, widgets, p -> createTreeForViewer(p, SWT.BORDER));
    loading.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
    tree = Widgets.createTreeViewer(loading.getContents());
    Widgets.Refresher refresher = Widgets.withAsyncRefresh(tree);
    tree.setContentProvider(new ITreeContentProvider() {
      @Override
      public Object[] getElements(Object element) {
        return getChildren(element);
      }

      @Override
      public boolean hasChildren(Object element) {
        return cast(element).getChildCount() > 0;
      }

      @Override
      public Object[] getChildren(Object element) {
        return cast(element).getChildren();
      }

      @Override
      public Object getParent(Object element) {
        return cast(element).getParent();
      }
    });
    tree.setLabelProvider(new LabelProvider() {
      private final Map<TraceTargets.Node, Image> images = Maps.newHashMap();

      @Override
      public String getText(Object element) {
        TraceTargets.Node node = cast(element);
        Service.TraceTargetTreeNode data = node.getData();
        if (data == null) {
          targets.load(node, refresher::refresh);
          return "Loading...";
        } else {
          return data.getName();
        }
      }

      @Override
      public Image getImage(Object element) {
        TraceTargets.Node node = cast(element);
        if (!images.containsKey(node)) {
          Service.TraceTargetTreeNode data = node.getData();
          if (data != null) {
            Image image = null;
            if (!data.getIcon().isEmpty()) {
              ImageData[] imageData = imageLoader.load(data.getIcon().newInput());
              int size = (int)(ICON_SIZE_DIP * DPIUtil.getDeviceZoom() / 100f);
              image = Images.createNonScaledImage(resources, imageData[0].scaledTo(size, size));
            }
            images.put(node, image);
          }
        }
        return images.get(node);
      }
    });

    resources = new LocalResourceManager(JFaceResources.getResources(), tree.getTree());

    tree.getTree().addListener(SWT.Selection, e -> {
      selected = (e.item == null) ? null : cast(((TreeItem)e.item).getData());
      Button ok = getButton(IDialogConstants.OK_ID);
      if (ok != null) {
        ok.setEnabled(selected != null && selected.isTraceable());
      }
    });

    search.addListener(SWT.Modify, e -> {
      String query = search.getText().trim();
      if (query.isEmpty()) {
        tree.resetFilters();
        return;
      }

      String[] parts = stream(SEARCH_SPLITTER.split(query).spliterator(), false)
          .map(String::toLowerCase)
          .toArray(String[]::new);
      boolean lastWasSeparator = SEARCH_SEPARATOR.matches(query.charAt(query.length() - 1));

      tree.setFilters(new ViewerFilter() {
        @Override
        public boolean select(Viewer viewer, Object parentElement, Object element) {
          return matches(cast(element));
        }

        private boolean matches(TraceTargets.Node node) {
          Service.TraceTargetTreeNode data = node.getData();
          int depth = node.getDepth();
          if (depth > parts.length) {
            return true;
          } else if (data == null) {
            targets.load(node, refresher::refresh);
            return true;
          } else if (!data.getName().toLowerCase().contains(parts[depth - 1])) {
            return false;
          } else if (depth == parts.length) {
            if (lastWasSeparator && !tree.getExpandedState(node)) {
              tree.setExpandedState(node, true);
            }
            return true;
          } else if (node.getChildCount() == 0) {
            return false;
          }
          if (!tree.getExpandedState(node)) {
            tree.setExpandedState(node, true);
          }
          for (int i = 0; i < node.getChildCount(); i++) {
            if (matches(node.getChild(i))) {
              return true;
            }
          }
          return false;
        }
      });
    });


    if (lastLoadError != null) {
      loading.showMessage(lastLoadError);
    } else if (targets.isLoaded()) {
      loading.stopLoading();
      tree.setInput(targets.getData());
    } else {
      loading.startLoading();
    }

    return area;
  }

  @Override
  protected void createButtonsForButtonBar(Composite parent) {
    Button ok = createButton(parent, IDialogConstants.OK_ID, IDialogConstants.OK_LABEL, true);
    createButton(parent, IDialogConstants.CANCEL_ID, IDialogConstants.CANCEL_LABEL, false);
    ok.setEnabled(false);
  }

  protected static TraceTargets.Node cast(Object element) {
    return (TraceTargets.Node)element;
  }
}
