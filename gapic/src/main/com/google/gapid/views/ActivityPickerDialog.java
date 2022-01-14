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

import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createTreeForViewer;
import static org.eclipse.jface.dialogs.IDialogConstants.OK_ID;

import com.google.common.util.concurrent.FutureCallback;
import com.google.gapid.image.Images;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Models;
import com.google.gapid.proto.device.Device;
import com.google.gapid.proto.pkginfo.PkgInfo;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.server.GapitPkgInfoProcess;
import com.google.gapid.util.Events;
import com.google.gapid.util.Loadable.MessageType;
import com.google.gapid.util.Messages;
import com.google.gapid.util.MoreFutures;
import com.google.gapid.widgets.DialogBase;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.SearchBox;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.dialogs.IDialogConstants;
import org.eclipse.jface.resource.JFaceResources;
import org.eclipse.jface.resource.LocalResourceManager;
import org.eclipse.jface.viewers.DelegatingStyledCellLabelProvider;
import org.eclipse.jface.viewers.DelegatingStyledCellLabelProvider.IStyledLabelProvider;
import org.eclipse.jface.viewers.ITreeContentProvider;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.StyledString;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.jface.viewers.Viewer;
import org.eclipse.jface.viewers.ViewerComparator;
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
import org.eclipse.swt.widgets.Tree;
import org.eclipse.swt.widgets.TreeItem;

import java.util.logging.Level;
import java.util.logging.Logger;
import java.util.regex.Pattern;

/**
 * Dialog to allow the user to pick which application and activity to trace.
 */
public class ActivityPickerDialog extends DialogBase {
  protected static final Logger LOG = Logger.getLogger(ActivityPickerDialog.class.getName());
  private static final int ICON_SIZE_DIP = 24;
  private static final int INITIAL_MIN_HEIGHT = 600;

  private final Models models;
  private final Widgets widgets;
  private final Device.Instance device;
  private LoadablePanel<Tree> loading;
  private TreeViewer tree;
  private PackageLabelProvider labelProvider;
  private LocalResourceManager resources;
  private PkgInfo.PackageList packageList;
  private Action selected;

  /**
   * The selected application and activity.
   */
  public static class Action {
    public final PkgInfo.Package pkg;
    public final PkgInfo.Activity activity;
    public final PkgInfo.Action action;

    private Action(PkgInfo.Package pkg, PkgInfo.Activity activity, PkgInfo.Action action) {
      this.pkg = pkg;
      this.activity = activity;
      this.action = action;
    }

    @Override
    public String toString() {
      return action.getName() + ":" + pkg.getName() + "/" + activity.getName();
    }

    public static Action getFor(TreeItem item) {
      Object data = item.getData();
      if (data instanceof PkgInfo.Action) {
        TreeItem activity = item.getParentItem();
        TreeItem pkg = activity.getParentItem();
        return new Action((PkgInfo.Package)pkg.getData(), (PkgInfo.Activity)activity.getData(),
            (PkgInfo.Action)data);
      } else if (data instanceof PkgInfo.Activity) {
        TreeItem pkg = item.getParentItem();
        PkgInfo.Activity activity = (PkgInfo.Activity)data;
        PkgInfo.Action action = findLaunchAction(activity);
        if (action == null && activity.getActionsCount() == 1) {
          action = activity.getActions(0);
        }
        return (action == null) ? null :
            new Action((PkgInfo.Package)pkg.getData(), activity, action);
      } else if (data instanceof PkgInfo.Package) {
        return getFor((PkgInfo.Package)data);
      } else {
        return null;
      }
    }

    private static Action getFor(PkgInfo.Package pkg) {
      for (PkgInfo.Activity activity : pkg.getActivitiesList()) {
        PkgInfo.Action action = findLaunchAction(activity);
        if (action != null) {
          return new Action(pkg, activity, action);
        }
      }

      if (pkg.getActivitiesCount() == 1 && pkg.getActivities(0).getActionsCount() == 1) {
        return new Action(pkg, pkg.getActivities(0), pkg.getActivities(0).getActions(0));
      }
      return null;
    }

    public static PkgInfo.Action findLaunchAction(PkgInfo.Activity activity) {
      for (PkgInfo.Action action : activity.getActionsList()) {
        if (action.getIsLaunch()) {
          return action;
        }
      }
      return null;
    }
  }

  public ActivityPickerDialog(
      Shell parent, Models models, Widgets widgets, Device.Instance device) {
    super(parent, widgets.theme);
    this.models = models;
    this.widgets = widgets;
    this.device = device;
  }

  public Action getSelected() {
    return selected;
  }

  @Override
  public int open() {
    models.analytics.postInteraction(View.Trace, ClientAction.ShowActivityPicker);
    return super.open();
  }

  @Override
  public String getTitle() {
    return Messages.SELECT_ACTIVITY;
  }


  @Override
  public void create() {
    super.create();
    getButton(OK_ID).setEnabled(false);
  }

  @Override
  protected void configureShell(Shell newShell) {
    super.configureShell(newShell);
    load(newShell);
  }

  @Override
  protected Point getInitialSize() {
    Point size = super.getInitialSize();
    size.y = Math.max(size.y, INITIAL_MIN_HEIGHT);
    return size;
  }

  @Override
  protected Control createDialogArea(Composite parent) {
    Composite area = (Composite)super.createDialogArea(parent);

    Composite container = createComposite(area, new GridLayout(1, false));
    container.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    SearchBox search = new SearchBox(container, "Filter activities...", true);
    search.setLayoutData(new GridData(SWT.FILL, SWT.TOP, true, false));

    loading = LoadablePanel.create(container, widgets, p -> createTreeForViewer(p, SWT.BORDER));
    loading.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
    tree = Widgets.createTreeViewer(loading.getContents());
    tree.setContentProvider(new PackageContentProvider());
    labelProvider = new PackageLabelProvider(widgets.theme);
    tree.setLabelProvider(new DelegatingStyledCellLabelProvider(labelProvider));
    tree.setComparator(new ViewerComparator()); // Sort by name.

    tree.getTree().addListener(SWT.Selection, e -> {
      selected = Action.getFor((TreeItem)e.item);
    });
    resources = new LocalResourceManager(JFaceResources.getResources(), tree.getTree());

    update();

    search.addListener(Events.Search, e -> {
      if (e.text.isEmpty()) {
        tree.resetFilters();
        return;
      }

      Pattern pattern = SearchBox.getPattern(e.text, (e.detail & Events.REGEX) != 0);
      tree.setFilters(new ViewerFilter() {
        @Override
        public boolean select(Viewer viewer, Object parentElement, Object element) {
          return !(element instanceof PkgInfo.Package) ||
              pattern.matcher(((PkgInfo.Package)element).getName()).find();
        }
      });
    });
    return area;
  }

  @Override
  protected void createButtonsForButtonBar(Composite parent) {
    Button ok = createButton(parent, IDialogConstants.OK_ID, IDialogConstants.OK_LABEL, true);
    createButton(parent, IDialogConstants.CANCEL_ID, IDialogConstants.CANCEL_LABEL, false);

    ok.setEnabled(false);
    tree.getTree().addListener(SWT.Selection, e -> ok.setEnabled(selected != null));
  }

  private void load(Shell shell) {
    float iconDensityScale = DPIUtil.getDeviceZoom() / 100.0f;
    GapitPkgInfoProcess process =
        new GapitPkgInfoProcess(models.settings, device.getSerial(), iconDensityScale);
    MoreFutures.addCallback(process.start(), new FutureCallback<PkgInfo.PackageList>() {
      @Override
      public void onFailure(Throwable t) {
        LOG.log(Level.WARNING, "Failed to read package info", t);
        Widgets.scheduleIfNotDisposed(shell, () -> showError(t.getMessage()));
      }

      @Override
      public void onSuccess(PkgInfo.PackageList result) {
        Widgets.scheduleIfNotDisposed(shell, () -> setPackageList(result));
      }
    });
  }

  protected void setPackageList(PkgInfo.PackageList packageList) {
    this.packageList = packageList;
    update();
  }

  protected void showError(String message) {
    loading.showMessage(MessageType.Error, message);
  }

  private void update() {
    if (packageList == null) {
      tree.setInput(PkgInfo.PackageList.getDefaultInstance());
      loading.startLoading();
      return;
    }

    int iconSize = (int)(ICON_SIZE_DIP * DPIUtil.getDeviceZoom() / 100.0f);
    ImageLoader loader = new ImageLoader();
    int iconCount = packageList.getIconsCount();
    Image icons[] = new Image[iconCount];
    for (int i = 0; i < iconCount; i++) {
      ImageData imageData[] = loader.load(packageList.getIcons(i).newInput());
      icons[i] = Images.createNonScaledImage(resources, imageData[0].scaledTo(iconSize, iconSize));
    }

    labelProvider.setIcons(icons);
    tree.setInput(packageList);
    loading.stopLoading();
  }

  private static class PackageContentProvider implements ITreeContentProvider {
    public PackageContentProvider() {
    }

    @Override
    public Object[] getElements(Object root) {
      return ((PkgInfo.PackageList)root).getPackagesList().toArray();
    }

    @Override
    public boolean hasChildren(Object element) {
      if (element instanceof PkgInfo.PackageList) {
        return ((PkgInfo.PackageList)element).getPackagesCount() > 0;
      } else if (element instanceof PkgInfo.Package) {
        return ((PkgInfo.Package)element).getActivitiesCount() > 0;
      } else if (element instanceof PkgInfo.Activity) {
        return ((PkgInfo.Activity)element).getActionsCount() > 0;
      } else {
        return false;
      }
    }

    @Override
    public Object[] getChildren(Object element) {
      if (element instanceof PkgInfo.PackageList) {
        return ((PkgInfo.PackageList)element).getPackagesList().toArray();
      } else if (element instanceof PkgInfo.Package) {
        return ((PkgInfo.Package)element).getActivitiesList().toArray();
      } else if (element instanceof PkgInfo.Activity) {
        return ((PkgInfo.Activity)element).getActionsList().toArray();
      } else {
        return new Object[0];
      }
    }

    @Override
    public Object getParent(Object element) {
      return null;
    }
  }

  private static class PackageLabelProvider extends LabelProvider implements IStyledLabelProvider {
    private final Theme theme;
    private Image[] icons = new Image[0];

    public PackageLabelProvider(Theme theme) {
      this.theme = theme;
    }

    public void setIcons(Image[] icons) {
      this.icons = icons;
    }

    @Override
    public StyledString getStyledText(Object element) {
      if (element instanceof PkgInfo.Package) {
        return new StyledString(((PkgInfo.Package)element).getName());
      } else if (element instanceof PkgInfo.Activity) {
        PkgInfo.Activity a = (PkgInfo.Activity)element;
        return new StyledString(
            a.getName(), Action.findLaunchAction(a) != null ? theme.labelStyler() : null);
      } else if (element instanceof PkgInfo.Action){
        PkgInfo.Action a = (PkgInfo.Action)element;
        return new StyledString(a.getName(), a.getIsLaunch() ? theme.labelStyler() : null);
      } else {
        return new StyledString("");
      }
    }

    @Override
    public Image getImage(Object element) {
      if (element instanceof PkgInfo.Package) {
        return getIcon(((PkgInfo.Package) element).getIcon(), true);
      } else if (element instanceof PkgInfo.Activity) {
        return getIcon(((PkgInfo.Activity) element).getIcon(), false);
      } else {
        return null;
      }
    }

    private Image getIcon(int index, boolean fallback) {
      if (index < 0 || index >= icons.length || icons[index] == null) {
        return fallback ? theme.androidLogo() : null;
      }
      return icons[index];
    }
  }
}
