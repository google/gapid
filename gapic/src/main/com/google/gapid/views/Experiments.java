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
import com.google.gapid.proto.service.path.Path;
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

import java.util.List;
import java.util.stream.Collectors;

public class Experiments {
  private final Models models;
  private final Theme theme;
  private ProfileExperiments experiment;

  public Experiments(Models models, Theme theme) {
    this.models = models;
    this.theme = theme;
    this.experiment = new ProfileExperiments();
  }

  public void showExperimentsPopup(Shell parent) {
    models.analytics.postInteraction(View.Commands, ClientAction.ShowExperiments);
    ExperimentsDialog dialog = new ExperimentsDialog(parent, models, theme);
    if (dialog.open() == Window.OK) {
      models.analytics.postInteraction(View.Commands, ClientAction.UpdateExperiments);
      experiment = new ProfileExperiments(dialog.disableAnisotropicFiltering, experiment.disabledCommands);
      models.profile.updateExperiments(experiment);
    }
  }

  public ProfileExperiments getExperiments() {
    return this.models.profile.getExperiments();
  }

  public boolean isAnyCommandDisabled(List<Path.Command> commands) {
    return commands.stream().anyMatch(c -> experiment.disabledCommands.contains(c));
  }

  public boolean areAllCommandsDisabled(List<Path.Command> commands) {
    return commands.stream().allMatch(c -> experiment.disabledCommands.contains(c));
  }

  public void disableCommands(List<Path.Command> commands) {
    models.analytics.postInteraction(View.Commands, ClientAction.DisableCommand);
    List <Path.Command> disabledCommands = experiment.disabledCommands
        .stream()
        .filter(c -> !commands.contains(c))
        .collect(Collectors.toList());
    disabledCommands.addAll(commands);
    experiment = new ProfileExperiments(experiment.disableAnisotropicFiltering, disabledCommands);
    models.profile.updateExperiments(experiment);
  }

  public void enableCommands(List<Path.Command> commands) {
    models.analytics.postInteraction(View.Commands, ClientAction.EnableCommand);
    experiment = new ProfileExperiments(experiment.disableAnisotropicFiltering,
        experiment.disabledCommands
        .stream()
        .filter(c-> !commands.contains(c))
        .collect(Collectors.toList()));
    models.profile.updateExperiments(experiment);
  }

  private static class ExperimentsDialog extends DialogBase {
    private final Models models;
    private Button disableAnisotropicFilteringButton;
    public boolean disableAnisotropicFiltering;

    public ExperimentsDialog(Shell parentShell, Models models, Theme theme) {
      super(parentShell, theme);
      this.models = models;
      disableAnisotropicFiltering = false;
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
      disableAnisotropicFilteringButton = Widgets.withLayoutData(
        Widgets.createCheckbox(container, "Disable Anisotropic Filtering",
          models.profile.getExperiments().disableAnisotropicFiltering),
        withSpans(new GridData(), 2, 1));
      return area;
    }

    @Override
    protected void okPressed() {
      disableAnisotropicFiltering = disableAnisotropicFilteringButton.getSelection();
      super.okPressed();
    }
  }
}
