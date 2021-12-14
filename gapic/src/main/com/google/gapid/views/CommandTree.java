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

import static com.google.gapid.image.Images.noAlpha;
import static com.google.gapid.models.ImagesModel.THUMB_SIZE;
import static com.google.gapid.util.Colors.getRandomColor;
import static com.google.gapid.util.Colors.lerp;
import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.widgets.Widgets.createBaloonToolItem;
import static com.google.gapid.widgets.Widgets.createButton;
import static com.google.gapid.widgets.Widgets.createCheckbox;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.withIndents;
import static com.google.gapid.widgets.Widgets.withLayoutData;

import com.google.common.collect.Maps;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Capture;
import com.google.gapid.models.CommandStream;
import com.google.gapid.models.CommandStream.CommandIndex;
import com.google.gapid.models.CommandStream.Node;
import com.google.gapid.models.Follower;
import com.google.gapid.models.Models;
import com.google.gapid.models.Profile;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.proto.service.api.API;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.util.Paths;
import com.google.gapid.util.SelectionHandler;
import com.google.gapid.views.Formatter.Style;
import com.google.gapid.views.Formatter.StylingString;
import com.google.gapid.widgets.LinkifiedTreeWithImages;
import com.google.gapid.widgets.LoadableImage;
import com.google.gapid.widgets.LoadableImageWidget;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.viewers.TreePath;
import org.eclipse.jface.viewers.TreeViewerColumn;
import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Color;
import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.RGBA;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Shell;
import org.eclipse.swt.widgets.ToolBar;
import org.eclipse.swt.widgets.TreeItem;

import java.util.List;
import java.util.Map;
import java.util.logging.Logger;

/**
 * API command view displaying the commands with their hierarchy grouping in a tree.
 */
public class CommandTree extends Composite
    implements Tab, Capture.Listener, CommandStream.Listener, Profile.Listener {
  protected static final Logger LOG = Logger.getLogger(CommandTree.class.getName());
  private static final String COMMAND_INDEX_HOVER = "Double click to copy index. Use Ctrl+G to jump to a given command index.";
  private static final String COMMAND_INDEX_DSCRP = "Command index: ";

  private final Models models;
  private final Paths.CommandFilter filter;
  private final LoadablePanel<Tree> loading;
  protected final Tree tree;
  private final Label commandIdx;
  private final SelectionHandler<Control> selectionHandler;

  public CommandTree(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;
    this.filter = models.commands.getFilter();

    setLayout(new GridLayout(1, false));

    // TODO: add search back
    ToolBar bar = new ToolBar(this, SWT.FLAT);
    createBaloonToolItem(bar, widgets.theme.filter(), bubble -> {
      bubble.setLayout(new GridLayout(1, false));
      createCheckbox(bubble, "Show Host Commands", filter.showHostCommands,
          e -> filter.showHostCommands = ((Button)e.widget).getSelection());
      createCheckbox(bubble, "Show Submit Info Nodes", filter.showSubmitInfoNodes,
          e -> filter.showSubmitInfoNodes = ((Button)e.widget).getSelection());
      createCheckbox(bubble, "Show Event/Sync Commands", filter.showSyncCommands,
          e -> filter.showSyncCommands = ((Button)e.widget).getSelection());
      createCheckbox(bubble, "Show Begin/End Commands", filter.showBeginEndCommands,
          e -> filter.showBeginEndCommands = ((Button)e.widget).getSelection());
      withLayoutData(createButton(bubble, "Apply", e -> bubble.close()),
          new GridData(SWT.RIGHT, SWT.TOP, false, false));

      bubble.addListener(SWT.Close, e -> models.commands.setFilter(filter));
    }, "Filter");

    loading = LoadablePanel.create(this, widgets, p -> new Tree(p, models, widgets));
    tree = loading.getContents();
    commandIdx = createLabel(this, COMMAND_INDEX_DSCRP);
    commandIdx.setToolTipText(COMMAND_INDEX_HOVER);
    commandIdx.addListener(SWT.MouseDoubleClick, e -> {
      if (commandIdx.getText().length() > COMMAND_INDEX_DSCRP.length()) {
        widgets.copypaste.setContents(commandIdx.getText().substring(COMMAND_INDEX_DSCRP.length()));
      }
    });

    bar.setLayoutData(new GridData(SWT.RIGHT, SWT.TOP, false, false));
    loading.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
    commandIdx.setLayoutData(withIndents(new GridData(SWT.FILL, SWT.FILL, true, false), 3, 0));

    models.capture.addListener(this);
    models.commands.addListener(this);
    models.profile.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.commands.removeListener(this);
      models.profile.removeListener(this);
    });

    selectionHandler = new SelectionHandler<Control>(LOG, tree.getControl()) {
      @Override
      protected void updateModel(Event e) {
        models.analytics.postInteraction(View.Commands, ClientAction.Select);
        CommandStream.Node node = tree.getSelection();
        if (node != null) {
          CommandIndex index = node.getIndex();
          if (index == null) {
            models.commands.load(node, () -> models.commands.selectCommands(node.getIndex(), false));
          } else {
            commandIdx.setText(COMMAND_INDEX_DSCRP + node.getIndexString());
            models.commands.selectCommands(index, false);
          }
          models.profile.linkCommandToGpuGroup(node.getCommandStart());
        }
      }
    };

    CommandOptions.CreateCommandOptionsMenu(tree.getControl(), widgets, tree, this.models);

    tree.registerAsCopySource(widgets.copypaste, node -> {
      models.analytics.postInteraction(View.Commands, ClientAction.Copy);
      Service.CommandTreeNode data = node.getData();
      if (data == null) {
        // Copy before loaded. Not ideal, but this is unlikely.
        return new String[] { "Loading..." };
      }

      StringBuilder result = new StringBuilder();
      if (data.getGroup().isEmpty() && data.hasCommands()) {
        result.append(data.getCommands().getTo(0)).append(": ");
        API.Command cmd = node.getCommand();
        if (cmd == null) {
          // Copy before loaded. Not ideal, but this is unlikely.
          result.append("Loading...");
        } else {
          result.append(Formatter.toString(cmd, models.constants::getConstants));
        }
      } else {
        result.append(data.getCommands().getFrom(0)).append(": ").append(data.getGroup());
      }
      return new String[] { result.toString() };
    }, true);
  }

  protected void select(TreePath path) {
    models.commands.selectCommands(((CommandStream.Node)path.getLastSegment()).getIndex(), true);
  }

  @Override
  public Control getControl() {
    return this;
  }

  @Override
  public void reinitialize() {
    updateTree(false);
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    updateTree(true);
  }

  @Override
  public void onCaptureLoaded(Loadable.Message error) {
    if (error != null) {
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    }
  }

  @Override
  public void onCommandsLoadingStart() {
    updateTree(false);
  }

  @Override
  public void onCommandsLoaded() {
    updateTree(false);
  }

  @Override
  public void onCommandsSelected(CommandIndex index) {
    selectionHandler.updateSelectionFromModel(() -> models.commands.getTreePath(index.getNode()).get(), tree::setSelection);
  }

  @Override
  public void onProfileLoadingStart() {
    tree.profileLoadingError = null;
  }

  @Override
  public void onProfileLoaded(Loadable.Message error) {
    tree.profileLoadingError = error;
    tree.refresh();
  }

  private void updateTree(boolean assumeLoading) {
    if (assumeLoading || !models.commands.isLoaded()) {
      loading.startLoading();
      tree.setInput(null);
      commandIdx.setText(COMMAND_INDEX_DSCRP);
      return;
    }

    loading.stopLoading();
    tree.setInput(models.commands.getData());
    if (models.commands.getSelectedCommands() != null) {
      onCommandsSelected(models.commands.getSelectedCommands());
    }
  }

  protected static class Tree extends LinkifiedTreeWithImages<CommandStream.Node, String> {
    private static final float COLOR_INTENSITY = 0.15f;
    private static final int DURATION_WIDTH = 95;

    protected final Models models;
    private final Widgets widgets;
    private final Map<Long, Color> threadBackgroundColors = Maps.newHashMap();
    private Loadable.Message profileLoadingError;

    public Tree(Composite parent, Models models, Widgets widgets) {
      super(parent, SWT.H_SCROLL | SWT.V_SCROLL | SWT.MULTI, widgets);
      this.models = models;
      this.widgets = widgets;

      addGpuPerformanceColumn();
    }

    protected void addGpuPerformanceColumn() {
      // The command tree's GPU performances are calculated from client's side.
      setUpStateForColumnAdding();
      addColumn("GPU Time", false);
    }

    private void addColumn(String title, boolean wallTime) {
      TreeViewerColumn column = addColumn(title, node -> {
        Service.CommandTreeNode data = node.getData();
        if (data == null) {
          return "";
        } else if (profileLoadingError != null) {
          return "Profiling failed.";
        } else if (!models.profile.isLoaded()) {
          return "Profiling...";
        } else {
          Profile.Duration duration = models.profile.getData().getDuration(data.getCommands());
          return wallTime ? duration.formatWallTime() : duration.formatGpuTime();
        }
      }, DURATION_WIDTH);
      column.getColumn().setAlignment(SWT.RIGHT);
    }

    public void refresh() {
      refresher.refresh();
    }

    public void updateTree(TreeItem item) {
      labelProvider.updateHierarchy(item);
    }

    @Override
    protected ContentProvider<Node> createContentProvider() {
      return new ContentProvider<CommandStream.Node>() {
        @Override
        protected boolean hasChildNodes(CommandStream.Node element) {
          return element.getChildCount() > 0;
        }

        @Override
        protected CommandStream.Node[] getChildNodes(CommandStream.Node node) {
          return node.getChildren();
        }

        @Override
        protected CommandStream.Node getParentNode(CommandStream.Node child) {
          return child.getParent();
        }

        @Override
        protected boolean isLoaded(CommandStream.Node element) {
          return element.getData() != null;
        }

        @Override
        protected boolean isDefaultExpanded(CommandStream.Node element) {
          Service.CommandTreeNode data = element.getData();
          return data != null && data.getExpandByDefault();
        }

        @Override
        protected void load(CommandStream.Node node, Runnable callback) {
          models.commands.load(node, callback);
        }
      };
    }

    private Style getCommandStyle(Service.CommandTreeNode node, StylingString string) {
      if (node.getExperimentalCommandsCount() == 0) {
        return string.labelStyle();
      }

      final List<Path.Command> experimentCommands = node.getExperimentalCommandsList();
      if (widgets.experiments.areAllCommandsDisabled(experimentCommands)) {
        return string.disabledLabelStyle();
      }

      if (widgets.experiments.isAnyCommandDisabled(experimentCommands)) {
        return string.semiDisabledLabelStyle();
      }

      return string.labelStyle();
    }

    @Override
    protected <S extends StylingString> S format(
        CommandStream.Node element, S string, Follower.Prefetcher<String> follower) {
      Service.CommandTreeNode data = element.getData();
      if (data == null) {
        string.append("Loading...", string.structureStyle());
      } else {
        if (data.getGroup().isEmpty() && data.hasCommands()) {
          API.Command cmd = element.getCommand();
          if (cmd == null) {
            string.append("Loading...", string.structureStyle());
          } else {
            Formatter.format(cmd, models.constants::getConstants, follower::canFollow,
                string, getCommandStyle(data, string), string.identifierStyle());
          }
        } else {
          string.append(data.getGroup(), getCommandStyle(data, string));
          long count = data.getNumCommands();
          string.append(
              " (" + count + " command" + (count != 1 ? "s" : "") + ")", string.structureStyle());
        }
      }
      return string;
    }

    @Override
    protected Color getBackgroundColor(CommandStream.Node node) {
      API.Command cmd = node.getCommand();
      if (cmd == null) {
        return null;
      }

      long threadId = cmd.getThread();
      Color color = threadBackgroundColors.get(threadId);
      if (color == null) {
        Control control = getControl();
        RGBA bg = control.getBackground().getRGBA();
        color = new Color(control.getDisplay(),
            lerp(getRandomColor(getColorIndex(threadId)), bg.rgb, COLOR_INTENSITY), bg.alpha);
        threadBackgroundColors.put(threadId, color);
      }
      return color;
    }

    private static int getColorIndex(long threadId) {
      // TODO: The index should be the i'th thread in use by the capture, not a hash of the
      // thread ID. This requires using the list of threads exposed by the service.Capture.
      int hash = (int)(threadId ^ (threadId >>> 32));
      hash = hash ^ (hash >>> 16);
      hash = hash ^ (hash >>> 8);
      return hash & 0xff;
    }

    @Override
    protected boolean shouldShowImage(CommandStream.Node node) {
      return models.images.isReady() &&
          node.getData() != null && !node.getData().getGroup().isEmpty();
    }

    @Override
    protected ListenableFuture<ImageData> loadImage(CommandStream.Node node, int size) {
      return noAlpha(models.images.getThumbnail(
          node.getPath(Path.CommandTreeNode.newBuilder()).build(), size, i -> { /*noop*/ }));
    }

    @Override
    protected void createImagePopupContents(Shell shell, CommandStream.Node node) {
      LoadableImageWidget.forImage(
          shell, LoadableImage.newBuilder(widgets.loading)
              .forImageData(loadImage(node, THUMB_SIZE))
              .onErrorShowErrorIcon(widgets.theme))
      .withImageEventListener(new LoadableImage.Listener() {
        @Override
        public void onLoaded(boolean success) {
          if (success) {
            Widgets.ifNotDisposed(shell,() -> {
              Point oldSize = shell.getSize();
              Point newSize = shell.computeSize(SWT.DEFAULT, SWT.DEFAULT);
              shell.setSize(newSize);
              if (oldSize.y != newSize.y) {
                Point location = shell.getLocation();
                location.y += (oldSize.y - newSize.y) / 2;
                shell.setLocation(location);
              }
            });
          }
        }
      });
    }

    @Override
    protected Follower.Prefetcher<String> prepareFollower(CommandStream.Node node, Runnable cb) {
      return models.follower.prepare(node, cb);
    }

    @Override
    protected void follow(Path.Any path) {
      models.follower.onFollow(path);
    }

    @Override
    public void reset() {
      super.reset();
      for (Color color : threadBackgroundColors.values()) {
        color.dispose();
      }
      threadBackgroundColors.clear();
    }
  }
}
