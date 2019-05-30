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
package com.google.gapid;

import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;

import com.google.common.collect.Lists;

import org.eclipse.jface.action.MenuManager;
import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.GC;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.graphics.Transform;
import org.eclipse.swt.widgets.Canvas;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Event;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.OutputStream;
import java.util.Base64;
import java.util.BitSet;
import java.util.List;
import java.util.Random;
import java.util.zip.InflaterOutputStream;

public class SnakeView extends Canvas implements MainWindow.MainView {
  private static final int MARGIN = 5;
  private static final int SCORE_MARGIN = 20 + MARGIN;
  private static final int WIDTH = 60;
  private static final int HEIGHT = 40;
  private static final int START_SPEED = 125;
  private static final double SPEED_MULTIPLIER = 0.95;
  private static final int APPLES_PER_LEVEL_BASE = 4; // 5 for level 1, 6 for level 2, etc.

  private final Random rnd;
  private int tileSize;
  private State state = State.STARTING;
  private Point origin = new Point(0, 0);
  private Level level = Level.getFirst();
  private Snake snake;
  private Point apple;
  private boolean growing = false;
  private int speed = START_SPEED;
  private int score = 0;
  private int progress = 0;

  public SnakeView(Composite parent) {
    super(parent, SWT.NO_BACKGROUND | SWT.DOUBLE_BUFFERED);
    rnd = new Random();

    snake = new Snake(level);
    newApple();

    addListener(SWT.Paint, e -> paint(e.gc));

    addListener(SWT.KeyDown, e -> {
      switch (state) {
        case STARTING:
        case RUNNING:
          if (snake.updateDirection(e)) {
            start();
          }
          break;
        case DEAD:
          if (e.keyCode == SWT.SPACE) {
            level = Level.getFirst();
            snake = new Snake(level);
            newApple();
            state = State.STARTING;
            speed = START_SPEED;
            score = 0;
            progress = 0;
            paint();
          }
      }
    });

    addListener(SWT.Resize, e -> {
      Rectangle size = getClientArea();
      tileSize = Math.min((size.width - 2 * MARGIN) / (WIDTH + 2),
          (size.height - 2 * MARGIN - SCORE_MARGIN) / (HEIGHT + 2));
      origin = new Point((size.width - tileSize * WIDTH) / 2,
          (size.height - SCORE_MARGIN - tileSize * HEIGHT) / 2);
    });
  }

  private void paint() {
    GC gc = new GC(this);
    paint(gc);
    gc.dispose();
  }

  private void paint(GC gc) {
    gc.fillRectangle(getClientArea());

    StringBuilder status = new StringBuilder()
        .append("Level: ").append(level.id)
        .append(" Score: ").append(score);
    switch (state) {
      case STARTING:
        status.append(" - Press a direction key to start!");
        break;
      case DEAD:
        status.append(" - Game over, you died! - Press space to start a new game!");
        break;
      default:
    }
    gc.drawText(status.toString(), origin.x, origin.y + (HEIGHT + 1) * tileSize + MARGIN);

    Transform transform = new Transform(getDisplay());
    transform.translate(origin.x, origin.y);
    transform.scale(tileSize, tileSize);
    gc.setTransform(transform);

    gc.setBackground(getDisplay().getSystemColor(SWT.COLOR_BLACK));
    gc.fillRectangle(-1, -1, WIDTH + 2, 1);
    gc.fillRectangle(-1, HEIGHT, WIDTH + 2, 1);
    gc.fillRectangle(-1, -1, 1, HEIGHT + 2);
    gc.fillRectangle(WIDTH, -1, 1, HEIGHT + 2);
    level.paint(gc);

    gc.setBackground(getDisplay().getSystemColor(SWT.COLOR_RED));
    gc.fillRectangle(apple.x, apple.y, 1, 1);

    gc.setBackground(getDisplay().getSystemColor(SWT.COLOR_GREEN));
    snake.paint(gc);

    gc.setTransform(null);
    transform.dispose();
  }

  private void start() {
    if (state == State.STARTING) {
      state = State.RUNNING;
      scheduleIfNotDisposed(this, speed, this::updateState);
    }
  }

  protected void updateState() {
    Point newHead = snake.move(growing);
    if (newHead == null) {
      state = State.DEAD;
      paint();
      return;
    }

    growing = newHead.equals(apple);
    if (growing) {
      score++;
      if (++progress >= APPLES_PER_LEVEL_BASE + level.id) {
        state = State.STARTING;
        level = level.nextLevel();
        snake = new Snake(level);
        progress = 0;
        speed = (int)(START_SPEED * Math.pow(SPEED_MULTIPLIER, level.id - 1));
        growing = false;
        newApple();
        paint();
        return;
      }

      newApple();
      speed = (int)(speed * SPEED_MULTIPLIER);
    }
    scheduleIfNotDisposed(this, speed, this::updateState);
    paint();
  }

  private void newApple() {
    do {
      apple = new Point(rnd.nextInt(WIDTH), rnd.nextInt(HEIGHT));
    } while (snake.contains(apple) || !level.isSafe(apple));
  }

  @Override
  public void updateViewMenu(MenuManager manager) {
    // Do nothing.
  }

  private static enum State {
    STARTING, RUNNING, DEAD;
  }

  private static class Level {
    private static final LevelDefinition[] LEVELS = {
        level("eJwDAAAAAAE=", WIDTH / 2 - 1, HEIGHT / 2),
        level("eJxjYBiU4M9/IGAGABR6BPw=", WIDTH / 2 - 1, HEIGHT / 2 - 5),
        level("eJxjYBgg8Oc/EDBTohcAU1gJ9w==", WIDTH / 2 - 2, HEIGHT / 2),
        level("eJxjYKAX+PMfCNgZHEBsIMEColnQuTDQAFQrj9CLXTGMCzEZAAQND7s=",
            WIDTH / 2 - 2, HEIGHT / 2 + 10),
        level("eJxjYKAc/GBg/5fwh+GBbMED+2P2DOzN7GzJPOwGBhIJDMcYGhgbeNnSeNgkGHoSDY4daHh4WOJbMg+PhF" +
            "1PIj+Qy/zYji2Bh83A4F0yw7EEhuZiAyCXfYE8cxrDMTuGCoMEtoQ3ACMvH3o=",
            WIDTH / 2 - 3, HEIGHT / 2 + 10),
    };

    public final int id;
    public final int start;
    private final BitSet set;

    public Level(int id, int start, BitSet set) {
      this.id = id;
      this.start = start;
      this.set = set;
    }

    public boolean isSafe(int pos) {
      return !set.get(pos);
    }

    public boolean isSafe(Point pos) {
      return isSafe(pos.x + pos.y * WIDTH);
    }

    public Level nextLevel() {
      return LEVELS[Math.min(id, LEVELS.length - 1)].get(id + 1);
    }

    public void paint(GC gc) {
      set.stream().forEach(pos -> gc.fillRectangle(pos % WIDTH, pos / WIDTH, 1, 1));
    }

    public static Level getFirst() {
      return LEVELS[0].get(1);
    }

    private static LevelDefinition level(String bits, int x, int y) {
      return new LevelDefinition(bits, x + y * WIDTH);
    }

    private static class LevelDefinition {
      private final String bits;
      private final int start;

      public LevelDefinition(String bits, int start) {
        this.bits = bits;
        this.start = start;
      }

      public Level get(int id) {
        return new Level(id, start, fromString(bits));
      }

      private static BitSet fromString(String s) {
        ByteArrayOutputStream baos = new ByteArrayOutputStream();
        try (OutputStream out = new InflaterOutputStream(baos)) {
          out.write(Base64.getDecoder().decode(s));
        } catch (IOException e) {
          throw new RuntimeException(e);
        }
        return BitSet.valueOf(baos.toByteArray());
      }
    }
  }

  private static class Snake {
    private static final int[] DX = { 1, -1, 0, 0 };
    private static final int[] DY = { 0, 0, 1, -1 };
    private static final int[] OPPOSITE = { 1, 0, 3, 2 };

    private final Level level;
    private final List<Integer> snake = Lists.newArrayList();
    private int head;
    private int direction = 0;
    private int newDirection = -1;

    public Snake(Level level) {
      this.level = level;
      for (int i = 0; i < 2 + level.id; i++) {
        snake.add(level.start + i);
      }
      head = snake.size() - 1;
    }

    public void paint(GC gc) {
      for (int pos : snake) {
        int x = pos % WIDTH, y = pos / WIDTH;
        gc.fillRectangle(x, y, 1, 1);
      }
    }

    public boolean updateDirection(Event e) {
      switch (e.keyCode) {
        case SWT.ARROW_RIGHT: newDirection = 0; break;
        case SWT.ARROW_LEFT: newDirection = 1; break;
        case SWT.ARROW_DOWN: newDirection = 2; break;
        case SWT.ARROW_UP: newDirection = 3; break;
        default: return false;
      }
      return true;
    }

    public Point move(boolean grow) {
      if (newDirection >= 0 && newDirection != OPPOSITE[direction]) {
        direction = newDirection;
      }
      newDirection = -1;

      int curPos = snake.get(head);
      int newX = curPos % WIDTH + DX[direction], newY = curPos / WIDTH + DY[direction];
      int newPos = newX + newY * WIDTH;
      if (newX < 0 || newX >= WIDTH || newY < 0 || newY >= HEIGHT || !level.isSafe(newPos)) {
        return null; // He's dead, Jim!
      }
      head = (head + 1) % snake.size();
      int index = snake.indexOf(newPos);
      if (index >= 0 && (grow || index != head)) {
        return null; // He ate himself.
      }
      if (grow) {
        snake.add(head, newPos);
      } else {
        snake.set(head, newPos);
      }
      return new Point(newX, newY);
    }

    public boolean contains(Point p) {
      return snake.contains(p.x + p.y * WIDTH);
    }
  }
}
