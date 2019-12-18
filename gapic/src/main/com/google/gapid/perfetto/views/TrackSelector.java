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
package com.google.gapid.perfetto.views;

import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createGroup;

import com.google.gapid.perfetto.models.TrackConfig;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.viewers.CheckboxTreeViewer;
import org.eclipse.jface.viewers.DelegatingStyledCellLabelProvider;
import org.eclipse.jface.viewers.DelegatingStyledCellLabelProvider.IStyledLabelProvider;
import org.eclipse.jface.viewers.ICheckStateProvider;
import org.eclipse.jface.viewers.ITreeContentProvider;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.StyledString;
import org.eclipse.jface.viewers.Viewer;
import org.eclipse.jface.viewers.ViewerFilter;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Group;
import org.eclipse.swt.widgets.Text;

/**
 * Shows a searchable tree of all the tracks in the UI.
 */
public class TrackSelector extends Composite implements State.Listener {
  private final State.ForSystemTrace state;
  protected final CheckboxTreeViewer tree;

  public TrackSelector(Composite parent, State.ForSystemTrace state, Theme theme) {
    super(parent, SWT.NONE);
    this.state = state;

    setLayout(new FillLayout());

    Group group = createGroup(this, "Tracks");
    Composite container = createComposite(group, new GridLayout(1, false));

    Text search = new Text(container, SWT.SINGLE | SWT.SEARCH | SWT.ICON_SEARCH | SWT.ICON_CANCEL);
    search.setLayoutData(new GridData(SWT.FILL, SWT.TOP, true, false));

    tree = createTree(container, theme);
    tree.getTree().setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

    state.addListener(this);

    search.addListener(SWT.Modify, e -> {
      String query = search.getText().trim().toLowerCase();
      if (query.isEmpty()) {
        tree.resetFilters();
        return;
      }

      tree.setFilters(new ViewerFilter() {
        @Override
        public boolean select(Viewer viewer, Object parentElement, Object element) {
          if (((TrackConfig.Element<?>)element).name.toLowerCase().contains(query)) {
            return true;
          } else if (element instanceof TrackConfig.Group) {
            for (Object child : ((TrackConfig.Group)element).tracks) {
              if (select(viewer, element, child)) {
                if (!tree.getExpandedState(element)) {
                  tree.setExpandedState(element, true);
                }
                return true;
              }
            }
          }
          return false;
        }
      });
    });
  }

  @Override
  public void onDataChanged() {
    if (!state.hasData()) {
      tree.setInput(null);
    } else {
      tree.setInput(state.getTracks());
    }
  }

  private static CheckboxTreeViewer createTree(Composite parent, Theme theme) {
    CheckboxTreeViewer tree = Widgets.createCheckboxTreeViewer(parent, SWT.NONE);
    tree.setContentProvider(new ITreeContentProvider() {
      @Override
      public Object[] getElements(Object inputElement) {
        return ((TrackConfig)inputElement).elements.toArray();
      }

      @Override
      public boolean hasChildren(Object element) {
        return element instanceof TrackConfig.Group;
      }

      @Override
      public Object[] getChildren(Object element) {
        if (!(element instanceof TrackConfig.Group)) {
          return null;
        }
        return ((TrackConfig.Group)element).tracks.toArray();
      }

      @Override
      public Object getParent(Object element) {
        return null;
      }
    });
    tree.setCheckStateProvider(new ICheckStateProvider() {
      @Override
      public boolean isGrayed(Object element) {
        return false;
      }

      @Override
      public boolean isChecked(Object element) {
        return true;
      }
    });
    tree.setLabelProvider(new DelegatingStyledCellLabelProvider(new ElementLabelProvider(theme)));
    return tree;
  }

  private static class ElementLabelProvider extends LabelProvider implements IStyledLabelProvider {
    private final Theme theme;

    public ElementLabelProvider(Theme theme) {
      this.theme = theme;
    }

    @Override
    public StyledString getStyledText(Object e) {
      boolean bold = (e instanceof TrackConfig.LabelGroup);
      return new StyledString(((TrackConfig.Element<?>)e).name, bold ? theme.labelStyler() : null);
    }
  }
}
