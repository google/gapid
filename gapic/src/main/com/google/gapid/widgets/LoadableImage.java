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

import com.google.common.base.Preconditions;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.image.Images;
import com.google.gapid.rpclib.rpccore.Rpc;
import com.google.gapid.rpclib.rpccore.Rpc.Result;
import com.google.gapid.rpclib.rpccore.RpcException;
import com.google.gapid.server.Client.DataUnavailableException;
import com.google.gapid.util.Events;
import com.google.gapid.util.Events.ListenerCollection;
import com.google.gapid.util.UiErrorCallback;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.widgets.Widget;

import java.util.concurrent.ExecutionException;
import java.util.function.Supplier;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * A widget that displays an {@link Image} that may need to be loaded. While the image is being
 * loaded, a loading indicator is drawn.
 */
public class LoadableImage {
  protected static final Logger LOG = Logger.getLogger(LoadableImage.class.getName());

  private final ListenerCollection<Listener> listeners = Events.silentListeners(Listener.class);
  private int loadCount = 0;
  protected final Widget widget;
  private final Supplier<ListenableFuture<Object>> futureSupplier;
  private ListenableFuture<Object> future;
  protected final LoadingIndicator loading;
  private final LoadingIndicator.Repaintable repaintable;
  protected final ErrorStrategy errorStrategy;
  private State state;
  private Image image;

  protected LoadableImage(Widget widget, Supplier<ListenableFuture<Object>> futureSupplier,
      LoadingIndicator loading, LoadingIndicator.Repaintable repaintable,
      ErrorStrategy errorStrategy) {
    this.widget = widget;
    this.futureSupplier = futureSupplier;
    this.loading = loading;
    this.repaintable = repaintable;
    this.errorStrategy = errorStrategy;

    state = State.NOT_STARTED;
  }

  public static Builder newBuilder(LoadingIndicator loading) {
    return new Builder(loading);
  }

  public LoadableImage load() {
    loadCount++;
    if (loadCount++ != 1 || state != State.NOT_STARTED) {
      return this;
    }
    state = State.LOADING;
    listeners.fire().onLoadingStart();
    loading.scheduleForRedraw(repaintable);

    future = futureSupplier.get();
    Rpc.listen(future, new UiErrorCallback<Object, Object, Image>(widget, LOG) {
      @Override
      protected ResultOrError<Object, Image> onRpcThread(Result<Object> result)
          throws RpcException, ExecutionException {
        try {
          return success(result.get());
        } catch (RpcException | ExecutionException e) {
          if (!widget.isDisposed()) {
            return error(errorStrategy.handleError(e));
          }
          return error(null);
        }
      }

      @Override
      protected void onUiThreadSuccess(Object result) {
        if (result instanceof Image) {
          updateImage((Image)result, true);
        } else {
          updateImage(Images.createNonScaledImage(widget.getDisplay(), (ImageData)result), true);
        }
      }

      @Override
      protected void onUiThreadError(Image errorIcon) {
        updateImage(errorIcon, false);
      }
    });
    return this;
  }

  public LoadableImage unload() {
    loadCount--;
    if (loadCount != 0 || state != State.LOADING) {
      return this;
    }
    future.cancel(true);
    state = State.NOT_STARTED;
    return this;
  }

  public Image getImage() {
    switch (state) {
      case NOT_STARTED: return getLoadingImage();
      case LOADING: loading.scheduleForRedraw(repaintable); return getLoadingImage();
      case LOADED:
      case FAILED: return image;
      case DISPOSED: SWT.error(SWT.ERROR_WIDGET_DISPOSED); return null;
      default: throw new AssertionError();
    }
  }

  public boolean hasFinished() {
    return (state != State.NOT_STARTED) && (state != State.LOADING);
  }

  protected Image getLoadingImage() {
    return loading.getCurrentFrame();
  }

  public void dispose() {
    if (image != null) {
      image.dispose();
      image = null;
      state = State.DISPOSED;
    }
  }

  protected void updateImage(Image result, boolean success) {
    if (state == State.LOADING) {
      state = (result == null || !success) ? State.FAILED : State.LOADED;
      image = result;
      listeners.fire().onLoaded(result != null);
    } else if (result != null) {
      result.dispose();
    }
  }

  public void addListener(Listener listener) {
    listeners.addListener(listener);
  }

  public void removeListener(Listener listener) {
    listeners.removeListener(listener);
  }

  public static interface Listener extends Events.Listener {
    /**
     * Event indicating that the image has started to load.
     */
    public default void onLoadingStart() { /* empty */ }

    /**
     * Event indicating that the image has finished loading.
     * @param success whether the image was loaded successfully
     */
    public default void onLoaded(boolean success) { /* empty */ }
  }

  /**
   * Determines how to deal with image loading errors.
   */
  public static interface ErrorStrategy {
    public Image handleError(Exception e);
  }

  /**
   * Builder for {@link LoadableImage}. If built using a future, it is assumed
   * to already be loading, while if built with a supplier, the {@link #load()}
   * method needs to be invoked to start the loading process.
   */
  public static class Builder {
    private final LoadingIndicator loading;
    private Supplier<ListenableFuture<Object>> futureSupplier;
    private ErrorStrategy errorStrategy;
    private boolean small;
    // True if build() should call load, because the future was wrapped via supplier().
    private boolean shouldLoad;

    protected Builder(LoadingIndicator loading) {
      this.loading = loading;
    }

    public Builder small() {
      this.small = true;
      return this;
    }

    public Builder large() {
      this.small = false;
      return this;
    }

    public Builder forImageData(ListenableFuture<ImageData> future) {
      this.shouldLoad = true;
      this.futureSupplier = cast(supplier(future));
      return this;
    }

    public Builder forImageData(Supplier<ListenableFuture<ImageData>> future) {
      this.shouldLoad = false;
      this.futureSupplier = cast(future);
      return this;
    }

    public Builder forImage(ListenableFuture<Image> future) {
      this.shouldLoad = true;
      this.futureSupplier = cast(supplier(future));
      return this;
    }

    public Builder forImage(Supplier<ListenableFuture<Image>> future) {
      this.shouldLoad = false;
      this.futureSupplier = cast(future);
      return this;
    }

    public Builder onErrorReturnNull() {
      this.errorStrategy = e -> {
        logImageError(e);
        return null;
      };
      return this;
    }

    public Builder onErrorShowErrorIcon(Theme theme) {
      this.errorStrategy = e -> {
        logImageError(e);
        return theme.error();
      };
      return this;
    }

    private static void logImageError(Exception e) {
      if (!(e instanceof DataUnavailableException)) {
        LOG.log(Level.WARNING, "Failed to load image", e);
      }
    }

    public LoadableImage build(Widget widget, LoadingIndicator.Repaintable repaintable) {
      Preconditions.checkState(futureSupplier != null);
      Preconditions.checkState(errorStrategy != null);

      LoadableImage result;
      if (small) {
        result = new LoadableImage(widget, futureSupplier, loading, repaintable, errorStrategy) {
          @Override
          protected Image getLoadingImage() {
            return loading.getCurrentSmallFrame();
          }
        };
      } else {
        result = new LoadableImage(widget, futureSupplier, loading, repaintable, errorStrategy);
      }
      return shouldLoad ? result.load() : result;
    }

    private static <T> Supplier<ListenableFuture<T>> supplier(ListenableFuture<T> future) {
      return () -> future;
    }

    @SuppressWarnings("unchecked")
    private static Supplier<ListenableFuture<Object>> cast(Supplier<?> future) {
      return (Supplier<ListenableFuture<Object>>)future;
    }
  }

  private static enum State {
    NOT_STARTED, LOADING, LOADED, FAILED, DISPOSED;
  }
}
