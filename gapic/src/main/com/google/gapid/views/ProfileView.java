/*
 * Copyright (C) 2019 Google Inc.
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

import static com.google.gapid.util.Loadable.MessageType.Error;
import static com.google.gapid.util.Loadable.MessageType.Loading;

import com.google.common.collect.Lists;
import com.google.gapid.models.Capture;
import com.google.gapid.models.Models;
import com.google.gapid.models.Profile;
import com.google.gapid.models.Settings;
import com.google.gapid.perfetto.canvas.Panel;
import com.google.gapid.perfetto.canvas.RenderContext;
import com.google.gapid.perfetto.models.CpuInfo;
import com.google.gapid.perfetto.models.ProcessInfo;
import com.google.gapid.perfetto.models.ThreadInfo;
import com.google.gapid.perfetto.views.RootPanel;
import com.google.gapid.perfetto.views.State;
import com.google.gapid.perfetto.views.TraceComposite;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.LoadablePanel;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;

import java.util.List;

public class ProfileView extends Composite implements Tab, Capture.Listener, Profile.Listener {
  private final Models models;

  private final LoadablePanel<TraceUi> loading;
  private final TraceUi traceUi;

  public ProfileView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new FillLayout(SWT.VERTICAL));

    loading = new LoadablePanel<TraceUi>(this, widgets, p -> new TraceUi(p, models, widgets.theme) {
      @Override
      protected Settings settings() {
        return models.settings;
      }
    });
    traceUi = loading.getContents();

    models.capture.addListener(this);
    models.profile.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.capture.removeListener(this);
      models.profile.removeListener(this);
    });
  }

  @Override
  public Control getControl() {
    return this;
  }

  @Override
  public void reinitialize() {
    if (models.profile.isLoaded()) {
      updateProfile(models.profile.getData());
    } else {
      loading.showMessage(
          Loading, models.capture.isLoaded() ? Messages.LOADING_CAPTURE : Messages.LOADING_PROFILE);
    }
  }

  @Override
  public void onCaptureLoadingStart(boolean maintainState) {
    loading.showMessage(Loading, Messages.LOADING_CAPTURE);
  }

  @Override
  public void onCaptureLoaded(Loadable.Message error) {
    if (error != null) {
      loading.showMessage(Error, Messages.CAPTURE_LOAD_FAILURE);
    }
  }

  @Override
  public void onProfileLoadingStart() {
    loading.showMessage(Loading, Messages.LOADING_PROFILE);
  }

  @Override
  public void onProfileLoaded(Loadable.Message error) {
    if (error != null) {
      loading.showMessage(error);
    } else {
      loading.stopLoading();
      updateProfile(models.profile.getData());
    }
  }

  private void updateProfile(Profile.Data data) {
    traceUi.update(data);
  }

  private abstract static class TraceUi extends TraceComposite<State> {
    protected final List<Panel> panels = Lists.newArrayList();

    public TraceUi(Composite parent, Models models, Theme theme) {
      super(parent, models.analytics, theme);
    }

    public void update(Profile.Data data) {
      panels.clear();
    }

    @Override
    protected State createState() {
      return new State(this) {
        @Override
        public CpuInfo getCpuInfo() {
          return CpuInfo.NONE;
        }

        @Override
        public ProcessInfo getProcessInfo(long id) {
          return null;
        }

        @Override
        public ThreadInfo getThreadInfo(long id) {
          return null;
        }
      };
    }

    @Override
    protected RootPanel<State> createRootPanel() {
      return new RootPanel<State>(state, settings()) {
        @Override
        protected void createUi() {
          top.add(timeline);
          for (Panel panel : panels) {
            bottom.add(panel);
          }
        }

        @Override
        protected void preTopUiRender(RenderContext ctx, Repainter repainter) {
          // Do nothing.
        }

        @Override
        protected void preMainUiRender(RenderContext ctx, Repainter repainter) {
          // Do nothing.
        }
      };
    }

    protected abstract Settings settings();
  }
}
