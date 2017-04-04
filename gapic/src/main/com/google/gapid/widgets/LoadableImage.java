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

import static java.util.logging.Level.FINE;

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.image.Images;
import com.google.gapid.rpclib.rpccore.Rpc;
import com.google.gapid.rpclib.rpccore.Rpc.Result;
import com.google.gapid.rpclib.rpccore.RpcException;
import com.google.gapid.util.Events;
import com.google.gapid.util.Events.ListenerCollection;
import com.google.gapid.util.UiErrorCallback;

import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Image;
import org.eclipse.swt.graphics.ImageData;
import org.eclipse.swt.widgets.Widget;

import java.util.concurrent.ExecutionException;
import java.util.function.Supplier;
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
  private State state;
  private Image image;

  protected LoadableImage(Widget widget, Supplier<ListenableFuture<Object>> futureSupplier,
      LoadingIndicator loading, LoadingIndicator.Repaintable repaintable) {
    this.widget = widget;
    this.futureSupplier = futureSupplier;
    this.loading = loading;
    this.repaintable = repaintable;

    state = State.NOT_STARTED;
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
    Rpc.listen(future, new UiErrorCallback<Object, Object, Void>(widget, LOG) {
      @Override
      protected ResultOrError<Object, Void> onRpcThread(Result<Object> result)
          throws RpcException, ExecutionException {
        try {
          return success(result.get());
        } catch (RpcException | ExecutionException e) {
          if (!widget.isDisposed()) {
            LOG.log(FINE, "Failed to load image", e);
          }
          return error(null);
        }
      }

      @Override
      protected void onUiThreadSuccess(Object result) {
        if (result instanceof Image) {
          updateImage((Image)result);
        } else {
          updateImage(Images.createNonScaledImage(widget.getDisplay(), (ImageData)result));
        }
      }

      @Override
      protected void onUiThreadError(Void error) {
        updateImage(null);
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

  // Factory methods: those taking in a future are assumed to already be loading, while those
  // taking a supplier will need their load() method to be invoked to start the loading process.

  public static LoadableImage forImageData(Widget widget, ListenableFuture<ImageData> future,
      LoadingIndicator loading, LoadingIndicator.Repaintable repaintable) {
    return forImageData(widget, supplier(future), loading, repaintable).load();
  }

  public static LoadableImage forImage(Widget widget, ListenableFuture<Image> future,
      LoadingIndicator loading, LoadingIndicator.Repaintable repaintable) {
    return forImage(widget, supplier(future), loading, repaintable).load();
  }

  public static LoadableImage forSmallImageData(Widget widget, ListenableFuture<ImageData> future,
      LoadingIndicator loading, LoadingIndicator.Repaintable repaintable) {
    return forSmallImageData(widget, supplier(future), loading, repaintable).load();
  }

  public static LoadableImage forSmallImage(Widget widget, ListenableFuture<Image> future,
      LoadingIndicator loading, LoadingIndicator.Repaintable repaintable) {
    return forSmallImage(widget, supplier(future), loading, repaintable).load();
  }

  public static LoadableImage forImageData(Widget widget,
      Supplier<ListenableFuture<ImageData>> future, LoadingIndicator loading,
      LoadingIndicator.Repaintable repaintable) {
    return new LoadableImage(widget, cast(future), loading, repaintable);
  }

  public static LoadableImage forImage(Widget widget, Supplier<ListenableFuture<Image>> future,
      LoadingIndicator loading, LoadingIndicator.Repaintable repaintable) {
    return new LoadableImage(widget, cast(future), loading, repaintable);
  }

  public static LoadableImage forSmallImageData(Widget widget,
      Supplier<ListenableFuture<ImageData>> future, LoadingIndicator loading,
      LoadingIndicator.Repaintable repaintable) {
    return new LoadableImage(widget, cast(future), loading, repaintable) {
      @Override
      protected Image getLoadingImage() {
        return loading.getCurrentSmallFrame();
      }
    };
  }

  public static LoadableImage forSmallImage(Widget widget, Supplier<ListenableFuture<Image>> future,
      LoadingIndicator loading, LoadingIndicator.Repaintable repaintable) {
    return new LoadableImage(widget, cast(future), loading, repaintable) {
      @Override
      protected Image getLoadingImage() {
        return loading.getCurrentSmallFrame();
      }
    };
  }

  private static <T> Supplier<ListenableFuture<T>> supplier(ListenableFuture<T> future) {
    return () -> future;
  }

  @SuppressWarnings("unchecked")
  private static Supplier<ListenableFuture<Object>> cast(Supplier<?> future) {
    return (Supplier<ListenableFuture<Object>>)future;
  }

  public Image getImage() {
    switch (state) {
      case NOT_STARTED: return getLoadingImage();
      case LOADING: loading.scheduleForRedraw(repaintable); return getLoadingImage();
      case LOADED: return image;
      case FAILED: return loading.getErrorImage();
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

  protected void updateImage(Image result) {
    if (state == State.LOADING) {
      state = (result == null) ? State.FAILED : State.LOADED;
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

  private static enum State {
    NOT_STARTED, LOADING, LOADED, FAILED, DISPOSED;
  }
}
