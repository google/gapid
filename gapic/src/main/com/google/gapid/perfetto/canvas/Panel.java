package com.google.gapid.perfetto.canvas;

import org.eclipse.swt.graphics.Cursor;
import org.eclipse.swt.widgets.Display;

import java.util.function.BiConsumer;

/**
 * A {@link Panel} is a UI control handling painting and user interactions.
 */
public interface Panel {
  public double getPreferredHeight();
  public void setSize(double w, double h);

  public void render(RenderContext ctx, Repainter repainter);

  @SuppressWarnings("unused")
  public default Panel.Dragger onDragStart(double x, double y, int mods, double scrollTop) {
    return Dragger.NONE;
  }

  @SuppressWarnings("unused")
  public default Panel.Hover onMouseMove(TextMeasurer m, double x, double y, double scrollTop) {
    return Hover.NONE;
  }

  public void visit(Visitor v, Area area);

  public static interface Repainter {
    public void repaint(Area area);

    public default Repainter translated(double dx, double dy) {
      return new Repainter() {
        @Override
        public void repaint(Area area) {
          Repainter.this.repaint(area.translate(dx, dy));
        }
      };
    }
  }

  public static interface TextMeasurer {
    public Size measure(String text);
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
    };

    /** Returns the area requiring to be redrawn **/
    public Area onDrag(double x, double y);

    /** Returns the area requiring to be redrawn **/
    public Area onDragEnd(double x, double y);

    public default Cursor getCursor(@SuppressWarnings("unused") Display display) { return null; }

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
      public Panel.Hover translated(double dx, double dy) {
        return NONE;
      }
    };

    public default Area getRedraw() { return Area.NONE; }
    public default Cursor getCursor(@SuppressWarnings("unused") Display display) { return null; }
    public default void stop() { /* empty */ }

    /** Returns whether the screen should be redrawn. */
    public default boolean click() { return false; }

    public default Panel.Hover translated(double dx, double dy) {
      Area redraw = getRedraw();
      if (redraw == Area.NONE) {
        return this;
      }

      return new Hover() {
        @Override
        public Area getRedraw() {
          return redraw.translate(dx,  dy);
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
        public boolean click() {
          return Hover.this.click();
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
