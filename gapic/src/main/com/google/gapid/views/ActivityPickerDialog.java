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
import static org.eclipse.jface.dialogs.IDialogConstants.OK_ID;
import static org.eclipse.swt.SWT.VERTICAL;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.FutureCallback;
import com.google.common.util.concurrent.Futures;
import com.google.gapid.image.Images;
import com.google.gapid.proto.device.Device;
import com.google.gapid.proto.pkginfo.PkgInfo;
import com.google.gapid.server.GapitPkgInfoProcess;
import com.google.gapid.util.Loadable.MessageType;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Widgets;
import com.google.protobuf.ByteString;

import org.eclipse.jface.dialogs.IDialogConstants;
import org.eclipse.jface.dialogs.TitleAreaDialog;
import org.eclipse.jface.resource.JFaceResources;
import org.eclipse.jface.resource.LocalResourceManager;
import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Font;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.graphics.ImageLoader;
import org.eclipse.swt.internal.DPIUtil;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Shell;
import org.eclipse.swt.widgets.Tree;
import org.eclipse.swt.widgets.TreeItem;

import java.util.Collections;
import java.util.List;
import java.util.logging.Level;
import java.util.logging.Logger;

public class ActivityPickerDialog extends TitleAreaDialog {
  protected static final Logger LOG = Logger.getLogger(ActivityPickerDialog.class.getName());
  private static final int ICON_SIZE_DIP = 24;

  private final Widgets widgets;
  private final Device.Instance device;
  private LoadablePanel<Tree> loading;
  private Tree tree;
  private LocalResourceManager resources;
  private PkgInfo.PackageList packageList;
  private Action selected;

  public static class Action {
    public final PkgInfo.Package pkg;
    public final PkgInfo.Activity activity;
    public final PkgInfo.Action action;

    Action(PkgInfo.Package pkg, PkgInfo.Activity activity, PkgInfo.Action action) {
      this.pkg = pkg;
      this.activity = activity;
      this.action = action;
    }
  }

  public ActivityPickerDialog(Shell parent, Widgets widgets, Device.Instance device) {
    super(parent);
    this.widgets = widgets;
    this.device = device;
  }

  public Action getSelected() {
    return selected;
  }

  @Override
  public void create() {
    super.create();
    setTitle(Messages.SELECT_ACTIVITY);
    getButton(OK_ID).setEnabled(false);
  }

  @Override
  protected void configureShell(Shell newShell) {
    super.configureShell(newShell);
    newShell.setText(Messages.TRACE);
    load(newShell);
  }

  @Override
  protected boolean isResizable() {
    return true;
  }

  @Override
  protected Control createDialogArea(Composite parent) {
    Composite area = (Composite)super.createDialogArea(parent);

    Composite container = createComposite(area, new FillLayout(VERTICAL));
    container.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    loading = LoadablePanel.create(container, widgets, panel -> new Tree(panel, SWT.BORDER));
    tree = loading.getContents();
    tree.addListener(SWT.Selection, e -> {
      Object data = e.item.getData();
      selected = (data instanceof Action) ? (Action)data : null;
    });
    resources = new LocalResourceManager(JFaceResources.getResources(), tree);

    update();
    return area;
  }

  @Override
  protected void createButtonsForButtonBar(Composite parent) {
    Button ok = createButton(parent, IDialogConstants.OK_ID, IDialogConstants.OK_LABEL, true);
    createButton(parent, IDialogConstants.CANCEL_ID, IDialogConstants.CANCEL_LABEL, false);

    ok.setEnabled(false);
    tree.addListener(SWT.Selection, e -> ok.setEnabled(selected != null));
  }

  private void load(Shell shell) {
    float iconDensityScale = DPIUtil.getDeviceZoom() / 100.0f;
    GapitPkgInfoProcess process = new GapitPkgInfoProcess(device.getSerial(), iconDensityScale);
    Futures.addCallback(process.start(), new FutureCallback<PkgInfo.PackageList>() {
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
    tree.removeAll();
    if (packageList == null) {
      loading.startLoading();
      return;
    }
    loading.stopLoading();

    float iconDensityScale = DPIUtil.getDeviceZoom() / 100.0f;
    int iconSize = (int)(ICON_SIZE_DIP * iconDensityScale);

    ImageLoader loader = new ImageLoader();
    int iconCount = packageList.getIconsCount();
    Image icons[] = new Image[iconCount];
    for (int i = 0; i < iconCount; i++) {
      ByteString data = packageList.getIcons(i);
      ImageData imageData[] = loader.load(data.newInput());
      icons[i] = Images.createNonScaledImage(resources, imageData[0].scaledTo(iconSize, iconSize));
    }

    List<PkgInfo.Package> packages = Lists.newArrayList(packageList.getPackagesList());
    Collections.sort(packages, (p1, p2) -> {
      return p1.getName().compareTo(p2.getName());
    });

    Font boldFont = JFaceResources.getFontRegistry().getBold(JFaceResources.DEFAULT_FONT);
    Image noIcon = widgets.theme.androidLogo();
    for (PkgInfo.Package pkg : packages) {
      TreeItem pkgItem = new TreeItem(tree, 0);
      pkgItem.setText(pkg.getName());
      int pkgIconIdx = pkg.getIcon();
      pkgItem.setImage((pkgIconIdx >= 0 && pkgIconIdx < iconCount) ? icons[pkgIconIdx] : noIcon);

      Action launchAction = null;

      for (PkgInfo.Activity activity : pkg.getActivitiesList()) {
        TreeItem activityItem = new TreeItem(pkgItem, 0);
        activityItem.setText(activity.getName());

        for (PkgInfo.Action action : activity.getActionsList()) {
          TreeItem actionItem = new TreeItem(activityItem, 0);
          actionItem.setText(action.getName());
          if (action.getIsLaunch() && launchAction == null) {
            actionItem.setFont(boldFont);
            launchAction = new Action(pkg, activity, action);
          }
          actionItem.setData(new Action(pkg, activity, action));
        }
        if (launchAction != null && launchAction.activity == activity) {
          activityItem.setFont(boldFont);
          activityItem.setData(launchAction);
        }
        int activityIconIdx = activity.getIcon();
        if (activityIconIdx < 0 || activityIconIdx >= iconCount) {
          activityIconIdx = pkgIconIdx;
        }
        activityItem.setImage((activityIconIdx >= 0 && activityIconIdx < iconCount) ?
            icons[activityIconIdx] : noIcon);
      }
      if (launchAction != null) {
        pkgItem.setData(launchAction);
      }
    }
  }
}
