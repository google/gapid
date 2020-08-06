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
package com.google.gapid.perfetto.canvas;

import org.eclipse.swt.graphics.Cursor;
import org.eclipse.swt.widgets.Display;

import java.util.function.BiConsumer;
import java.util.function.BooleanSupplier;
import java.util.function.Function;

/**
 * A {@link Panel} is a UI control handling painting and user interactions.
 */
public interface Panel {
  public double getPreferredHeight();
  public void setSize(double w, double h);

  public void render(RenderContext ctx, Repainter repainter);

  @SuppressWarnings("unused")
  public default Panel.Dragger onDragStart(double x, double y, int mods) {
    return Dragger.NONE;
  }

  @SuppressWarnings("unused")
  public default Panel.Hover onMouseMove(
      Fonts.TextMeasurer m, Repainter repainter, double x, double y, int mods) {
    return Hover.NONE;
  }

  public void visit(Visitor v, Area area);

  public static interface Repainter {
    public void repaint(Area area);

    public default Repainter translated(double dx, double dy) {
      return transformed(a -> a.translate(dx, dy));
    }

    public default Repainter transformed(Function<Area, Area> transform) {
      return new Repainter() {
        @Override
        public void repaint(Area area) {
          Repainter.this.repaint(transform.apply(area));
        }
      };
    }
  }

  public static interface Dragger {
    public static final Dragger NONE = new Dragger() {
      @Override
      public Area onDrag(double x, double y) {
        return Area.NONE;
      }

      @Override
      public Area onDragEnd(double x, double y) {
        return Area.NONE;
      }

      @Override
      public Dragger translated(double dx, double dy) {
        return Dragger.NONE;
      }

      @Override
      public Dragger or(Dragger other) {
        return other;
      }
    };

    /** Returns the area requiring to be redrawn **/
    public Area onDrag(double x, double y);

    /** Returns the area requiring to be redrawn **/
    public Area onDragEnd(double x, double y);

    public default Cursor getCursor(@SuppressWarnings("unused") Display display) { return null; }

    public default Panel.Dragger or(@SuppressWarnings("unused") Panel.Dragger other) {
      return this;
    }

    public default Panel.Dragger translated(double dx, double dy) {
      return new Dragger() {
        @Override
        public Area onDrag(double x, double y) {
          return Dragger.this.onDrag(x - dx, y - dy).translate(dx, dy);
        }

        @Override
        public Area onDragEnd(double x, double y) {
          return Dragger.this.onDragEnd(x - dx, y - dy).translate(dx, dy);
        }

        @Override
        public Cursor getCursor(Display display) {
          return Dragger.this.getCursor(display);
        }
      };
    }
  }

  public static interface Hover {
    public static final Panel.Hover NONE = new Hover() {
      @Override
      public Hover translated(double dx, double dy) {
        return NONE;
      }

      @Override
      public Hover transformed(Function<Area, Area> transform) {
        return NONE;
      }
    };

    public default Area getRedraw() { return Area.NONE; }
    public default Cursor getCursor(@SuppressWarnings("unused") Display display) { return null; }
    public default void stop() { /* empty */ }
    public default boolean isOverlay() { return false; }

    /** Returns whether the screen should be redrawn. */
    public default boolean click() { return false; }
    public default boolean rightClick() { return click(); }

    public default Panel.Hover translated(double dx, double dy) {
      return transformed(a -> a.translate(dx, dy));
    }

    public default Panel.Hover transformed(Function<Area, Area> transform) {
      Area redraw = getRedraw();
      return new Hover() {
        @Override
        public Area getRedraw() {
          return transform.apply(redraw);
        }

        @Override
        public Cursor getCursor(Display display) {
          return Hover.this.getCursor(display);
        }

        @Override
        public void stop() {
          Hover.this.stop();
        }

        @Override
        public boolean isOverlay() {
          return Hover.this.isOverlay();
        }

        @Override
        public boolean click() {
          return Hover.this.click();
        }

        @Override
        public boolean rightClick() {
          return Hover.this.rightClick();
        }
      };
    }

    public default Panel.Hover withClick(BooleanSupplier onClick) {
      return new Hover() {
        @Override
        public Area getRedraw() {
          return Hover.this.getRedraw();
        }

        @Override
        public Cursor getCursor(Display display) {
          return Hover.this.getCursor(display);
        }

        @Override
        public void stop() {
          Hover.this.stop();
        }

        @Override
        public boolean isOverlay() {
          return Hover.this.isOverlay();
        }

        @Override
        public boolean click() {
          boolean r1 = Hover.this.click(), r2 = onClick.getAsBoolean();
          return r1 || r2;
        }

        @Override
        public boolean rightClick() {
          boolean r1 = Hover.this.rightClick(), r2 = onClick.getAsBoolean();
          return r1 || r2;
        }
      };
    }

    public default Panel.Hover withRedraw(Area newRedraw) {
      return new Hover() {
        @Override
        public Area getRedraw() {
          return Hover.this.getRedraw().combine(newRedraw);
        }

        @Override
        public Cursor getCursor(Display display) {
          return Hover.this.getCursor(display);
        }

        @Override
        public void stop() {
          Hover.this.stop();
        }

        @Override
        public boolean isOverlay() {
          return Hover.this.isOverlay();
        }

        @Override
        public boolean click() {
          return Hover.this.click();
        }

        @Override
        public boolean rightClick() {
          return Hover.this.rightClick();
        }
      };
    }
  }

  public static interface Visitor {
    public void visit(Panel panel, Area area);

    /**
     * Returns a {@link Panel.Visitor} that will call the given consumer only when visited by
     * {@link Panel Panels} that are instances of the given class. Allows filtering of the visited
     * children by a class/interface, e.g. visit all {@link PanelGroup PanelGroups}.
     */
    public static <T> Visitor of(Class<? extends T> cls, BiConsumer<T, Area> c) {
      return (p, a) -> {
        if (cls.isInstance(p)) {
          c.accept(cls.cast(p), a);
        }
      };
    }
  }

  public static interface Grouper {
    public int getPanelCount();
    public Panel getPanel(int idx);
    public void setVisible(int idx, boolean visible);
    public void setFiltered(int idx, boolean filtered);

    public default boolean updateFilter(Function<Panel, FilterValue> include) {
      boolean found = false;
      for (int i = 0; i < getPanelCount(); i++) {
        Panel panel = getPanel(i);
        FilterValue filter = include.apply(panel);
        boolean keep = true;
        if (panel instanceof Grouper) {
          if (filter == FilterValue.Include) {
            ((Grouper)panel).updateFilter($ -> FilterValue.Include);
          } else {
            keep = ((Grouper)panel).updateFilter(include);
          }
        } else if (filter == FilterValue.DescendOrExclude) {
          keep = false;
        }
        setFiltered(i, !keep);
        found |= keep;
      }
      return found;
    }
  }

  public static enum FilterValue {
    Include,           // Include this panel and if it's a group, all its children.
    DescendOrInclude,  // If this panel is a group then recursively filter, otherwise include it.
    DescendOrExclude;  // If this panel is a group then recursively filter, otherwise exclude it.
  }

  public abstract static class Base implements Panel {
    protected double width;
    protected double height;

    public Base() {
    }

    @Override
    public void setSize(double w, double h) {
      width = w;
      height = h;
    }

    @Override
    public void visit(Visitor v, Area area) {
      area.intersect(0, 0, width, height).ifNotEmpty(a -> v.visit(this, area));
    }
  }
}
