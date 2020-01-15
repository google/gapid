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

import static com.google.gapid.widgets.Widgets.createTreeColumn;
import static com.google.gapid.widgets.Widgets.createTreeViewer;
import static com.google.gapid.widgets.Widgets.packColumns;

import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.models.CpuTrack;
import com.google.gapid.perfetto.models.ProcessInfo;
import com.google.gapid.perfetto.models.ThreadInfo;

import org.eclipse.jface.viewers.ITreeContentProvider;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.TreeViewer;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.FillLayout;
import org.eclipse.swt.widgets.Composite;

/**
 * Displays information about a list of selected CPU slices.
 */
public class CpuSlicesSelectionView extends Composite {
  public CpuSlicesSelectionView(Composite parent, State state, CpuTrack.Slices sel) {
    super(parent, SWT.NONE);
    setLayout(new FillLayout());

    TreeViewer viewer = createTreeViewer(this, SWT.NONE);
    viewer.getTree().setHeaderVisible(true);
    viewer.setContentProvider(new ITreeContentProvider() {
      @Override
      public Object[] getElements(Object inputElement) {
        return sel.processes.toArray();
      }

      @Override
      public boolean hasChildren(Object element) {
        return (element instanceof CpuTrack.ByProcess) ||
            (element instanceof CpuTrack.ByThread);
      }

      @Override
      public Object getParent(Object element) {
        return null;
      }

      @Override
      public Object[] getChildren(Object element) {
        if (element instanceof CpuTrack.ByProcess) {
          return ((CpuTrack.ByProcess)element).threads.toArray();
        } else if (element instanceof CpuTrack.ByThread) {
          return ((CpuTrack.ByThread)element).slices.toArray();
        }
        return null;
      }
    });
    viewer.setLabelProvider(new LabelProvider());

    createTreeColumn(viewer, "Name", el -> {
      if (el instanceof CpuTrack.ByProcess) {
        long pid = ((CpuTrack.ByProcess)el).pid;
        ProcessInfo pi = state.getProcessInfo(pid);
        return (pi == null) ? "<unknown process> [" + pid + "]" : pi.getDisplay();
      } else if (el instanceof CpuTrack.ByThread) {
        long tid = ((CpuTrack.ByThread)el).tid;
        ThreadInfo ti = state.getThreadInfo(tid);
        return (ti == null) ? "<unknown thread> [" + tid + "]" : ti.getDisplay();
      } else {
        return "Slice " + ((CpuTrack.Slice)el).id;
      }
    });
    createTreeColumn(viewer, "Slice Duration", el -> {
      if (el instanceof CpuTrack.ByProcess) {
        return TimeSpan.timeToString(((CpuTrack.ByProcess)el).dur);
      } else if (el instanceof CpuTrack.ByThread) {
        return TimeSpan.timeToString(((CpuTrack.ByThread)el).dur);
      } else {
        return TimeSpan.timeToString(((CpuTrack.Slice)el).dur);
      }
    });
    createTreeColumn(viewer, "Slice Start Time", el -> {
      if (el instanceof CpuTrack.Slice) {
        return TimeSpan.timeToString(((CpuTrack.Slice)el).time - state.getTraceTime().start);
      } else {
        return "";
      }
    });
    createTreeColumn(viewer, "End State", el -> {
      if (el instanceof CpuTrack.Slice) {
        return ((CpuTrack.Slice)el).endState.label;
      } else {
        return "";
      }
    });
    createTreeColumn(viewer, "Priority", el -> {
      if (el instanceof CpuTrack.Slice) {
        return String.valueOf(((CpuTrack.Slice)el).priority);
      } else {
        return "";
      }
    });
    viewer.setInput(sel);
    packColumns(viewer.getTree());
  }
}
