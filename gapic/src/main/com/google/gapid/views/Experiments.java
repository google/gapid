/*
 * Copyright (C) 2021 Google Inc.
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

import static com.google.gapid.widgets.Widgets.withSpans;

import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Models;
import com.google.gapid.models.ProfileExperiments;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.widgets.DialogBase;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.window.Window;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Shell;

import java.util.logging.Logger;

public class Experiments {
  private static final Logger LOG = Logger.getLogger(CommandEditor.class.getName());
  private final Models models;
  private final Theme theme;

  public Experiments(Models models, Theme theme) {
    this.models = models;
    this.theme = theme;
  }

  public void showExperimentsPopup(Shell parent) {
    models.analytics.postInteraction(View.Commands, ClientAction.ShowExperiments);
    ExperimentsDialog dialog = new ExperimentsDialog(parent, models, theme);
    if (dialog.open() == Window.OK) {
      models.analytics.postInteraction(View.Commands, ClientAction.UpdateExperiments);
      models.profile.updateExperiments(dialog.experiments);
    }
  }

  public ProfileExperiments getExperiments() {
    return this.models.profile.getExperiments();
  }

  public void setDisableAnisotropicFiltering(boolean selected) {
    models.analytics.postInteraction(View.Commands, ClientAction.DisableAnisotropicFiltering);
    models.profile.updateExperiments(new ProfileExperiments(selected));
  }

  private static class ExperimentsDialog extends DialogBase {
    private final Models models;
    private Button disableAnisotropicFiltering;
    public ProfileExperiments experiments;

    public ExperimentsDialog(Shell parentShell, Models models, Theme theme) {
      super(parentShell, theme);
      this.models = models;
    }

    @Override
    public String getTitle() {
      return "Profile Experiments";
    }

    @Override
    protected Control createDialogArea(Composite parent) {
      Composite area = (Composite)super.createDialogArea(parent);
      Composite container = Widgets.createComposite(area, new GridLayout(2, false));
      container.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
      disableAnisotropicFiltering = Widgets.withLayoutData(
        Widgets.createCheckbox(container, "Disable Anisotropic Filtering",
          models.profile.getExperiments().disableAnisotropicFiltering),
        withSpans(new GridData(), 2, 1));
      return area;
    }

    @Override
    protected void okPressed() {
      experiments = new ProfileExperiments(disableAnisotropicFiltering.getSelection());
      super.okPressed();
    }
  }
}
