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
package com.google.gapid.widgets;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.gapid.widgets.CopyPaste.CopyData;
import com.google.gapid.widgets.CopyPaste.CopySource;

import org.eclipse.jface.viewers.TreePath;
import org.eclipse.jface.viewers.TreeViewer;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;

public class CopySources {
  private CopySources() {
  }

  public static void registerTreeAsCopySource(
      CopyPaste cp, TreeViewer tree, ColumnTextProvider columnProvider) {
    cp.registerCopySource(tree.getControl(), new CopySource() {
      @Override
      public boolean hasCopyData() {
        return !tree.getSelection().isEmpty();
      }

      @Override
      public CopyData[] getCopyData() {
        // Create rows from all the paths.
        List<Node> roots = Lists.newArrayList();
        Map<TreePath, Node> pathToNode = Maps.newHashMap();
        for (TreePath path : tree.getStructuredSelection().getPaths()) {
          createNode(path, columnProvider, pathToNode, roots);
        }

        // Measure the column widths.
        List<Integer> maxColumnWidths = new ArrayList<>(2);
        for (Node node : roots) {
          node.measure(maxColumnWidths, 0);
        }

        // Print each of the roots and their children.
        StringBuffer plainBuf = new StringBuffer();
        for (Node node : roots) {
          node.print(plainBuf, maxColumnWidths);
        }

        return new CopyData[] { CopyData.text(plainBuf.toString()) };
      }
    });
    tree.addSelectionChangedListener(e -> cp.updateCopyState());
  }

  /**
   * Creates a new {@link Node} for the given {@link TreePath}, and any parent nodes that aren't
   * already found in pathToNode.
   * All nodes that are created are added to pathToNode.
   * Root nodes that are created are added to roots.
   *
   * @return the newly created node.
   */
  protected static Node createNode(TreePath path, ColumnTextProvider columnProvider,
      Map<TreePath, Node> pathToNode, List<Node> roots) {
    Node node = pathToNode.get(path);
    if (node != null) {
      return node;
    }

    node = new Node(columnProvider.getColumns(path.getLastSegment()));
    pathToNode.put(path, node);
    TreePath parent = path.getParentPath();
    if (parent != null && parent != TreePath.EMPTY) {
      createNode(parent, columnProvider, pathToNode, roots).addChild(node);
    } else {
      roots.add(node);
    }
    return node;
  }

  public static interface ColumnTextProvider {
    String[] getColumns(Object element);
  }

  private static class Node {
    private static final int INDENT_SIZE = 4;
    private static final String INDENT = "│   ";
    private static final String INDENT_LAST = "    ";
    private static final String BRANCH = "├── ";
    private static final String BRANCH_LAST = "└── ";

    /** All the direct descendants of this node. */
    private final List<Node> children = Lists.newArrayList();
    /** The column data for this node. */
    private final String[] columns;

    public Node(String[] columns) {
      this.columns = columns;
    }

    public void addChild(Node node) {
      children.add(node);
    }

    /**
     * Populates maxColumnWidths with the maximum column width for each column
     * of this node and all descendants.
     * @param indent the indentation of this node in number of characters.
     */
    public void measure(List<Integer> maxColumnWidths, int indent) {
      // Grow maxColumnWidths to at least as big as columns.
      while (maxColumnWidths.size() < columns.length) {
        maxColumnWidths.add(0);
      }
      for (int i = 0, c = columns.length; i < c; i++) {
        int width = columns[i].length();
        // Consider the tree as part of the first column.
        if (i == 0) { width += indent; }
        // Padding between columns.
        if (i < c - 1) { width += 1; }
        maxColumnWidths.set(i, Math.max(width, maxColumnWidths.get(i)));
      }
      // Now measure all the children.
      for (Node child : children) {
        child.measure(maxColumnWidths, indent + INDENT_SIZE);
      }
    }

    /**
     * Prints this node and all descendants to the {@link StringBuilder}.
     * This includes all tree lines, and padding for each column.
     *
     * @param maxColumnWidths the maximum column widths calculated by {@link #measure}.
     */
    public void print(StringBuffer sb, List<Integer> maxColumnWidths) {
      print(sb, maxColumnWidths, "", true, false);
    }

    private void print(
        StringBuffer sb, List<Integer> maxColumnWidths, String prefix, boolean root, boolean last) {
      StringBuffer column = new StringBuffer();
      if (!root) {
        column.append(prefix);
        column.append(last ? BRANCH_LAST : BRANCH);
        prefix += last ? INDENT_LAST : INDENT;
      }
      for (int i = 0, c = columns.length; i < c; i++) {
        column.append(columns[i]);
        if (i < c - 1) {
          // Align columns
          while (column.length() < maxColumnWidths.get(i)) {
            column.append(' ');
          }
        }
        sb.append(column.toString());
        column.setLength(0);
      }
      sb.append("\n");
      int childCount = children.size();
      for (int i = 0; i < childCount; i++) {
        children.get(i).print(sb, maxColumnWidths, prefix, false, i == childCount-1);
      }
    }
  }
}
