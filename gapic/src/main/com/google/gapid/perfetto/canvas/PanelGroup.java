package com.google.gapid.perfetto.canvas;

import com.google.common.collect.Lists;

import java.util.Collections;
import java.util.List;

/**
 * Contains a list of child {@link Panel panels}, which are laid out vertically.
 */
public class PanelGroup extends Panel.Base {
  private final List<PanelGroup.Child> panels = Lists.newArrayList();

  public PanelGroup() {
  }

  public int size() {
    return panels.size();
  }

  public Panel getPanel(int idx) {
    return panels.get(idx).panel;
  }

  public void add(Panel panel) {
    double y = panels.isEmpty() ? 0 : panels.get(panels.size() - 1).getNextY();
    panels.add(new Child(panel, y, 0));
  }

  public void clear() {
    panels.clear();
  }

  public void setVisible(int idx, boolean visible) {
    panels.get(idx).visible = visible;
  }

  @Override
  public double getPreferredHeight() {
    double y = 0;
    for (Child child : panels) {
      double want = child.visible ? child.panel.getPreferredHeight() : 0;
      child.y = y;
      child.h = want;
      y += want;
    }
    return y;
  }

  @Override
  public void setSize(double w, double h) {
    super.setSize(w, h);
    for (PanelGroup.Child child : panels) {
      if (child.visible) {
        child.panel.setSize(w, child.h);
      }
    }
  }

  @Override
  public void render(RenderContext ctx, Repainter repainter) {
    Area clip = ctx.getClip();
    int first = findPanelIdx(clip.y);

    for (int i = first; i < panels.size(); i++) {
      PanelGroup.Child child = panels.get(i);
      if (!child.visible) {
        continue;
      }

      ctx.withTranslation(0, child.y, () -> {
        ctx.withClip(clip.x, 0, clip.w, child.h, () -> {
          child.panel.render(ctx, repainter.translated(0, child.y));
        });
      });
      if (child.getNextY() >= clip.y + clip.h) {
        break;
      }
    }
  }

  @Override
  public void visit(Visitor v, Area area) {
    super.visit(v, area);
    int first = findPanelIdx(area.y);
    for (int i = first; i < panels.size(); i++) {
      PanelGroup.Child child = panels.get(i);
      if (!child.visible) {
        continue;
      }
      area.intersect(0, child.y, width, child.h)
          .ifNotEmpty(a -> child.panel.visit(v, a.translate(0, -child.y)));
    }
  }

  @Override
  public Dragger onDragStart(double x, double y, int mods, double scrollTop) {
    Child child = findPanel(y);
    if (child == null) {
      return Dragger.NONE;
    }
    return child.panel.onDragStart(x, y - child.y, mods, scrollTop).translated(0, child.y);
  }

  @Override
  public Hover onMouseMove(TextMeasurer m, double x, double y, double scrollTop) {
    Child child = findPanel(y);
    if (child == null) {
      return Hover.NONE;
    }
    return child.panel.onMouseMove(m, x, y - child.y, scrollTop).translated(0, child.y);
  }

  private int findPanelIdx(double y) {
    int first = Collections.binarySearch(panels, null, (c1, ign) -> {
      if (c1.visible) {
        return (y >= c1.y && y < c1.getNextY()) ? 0 : Double.compare(c1.y, y);
      } else {
        return c1.y < y ? -1 : 1;
      }
    });
    return (first < 0) ? -first - 1 : first;
  }

  private Child findPanel(double y) {
    int idx = findPanelIdx(y);
    return (idx < panels.size()) ? panels.get(idx) : null;
  }

  private static class Child {
    public final Panel panel;
    public double y, h;
    public boolean visible;

    public Child(Panel panel, double y, double h) {
      this.panel = panel;
      this.y = y;
      this.h = Math.max(0, h);
      this.visible = true;
    }

    public double getNextY() {
      return y + h;
    }
  }
}