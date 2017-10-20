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
package com.google.gapid.image;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.collect.Sets;
import com.google.gapid.proto.stream.Stream.Channel;
import java.nio.DoubleBuffer;
import java.util.List;
import java.util.Map;
import java.util.Map.Entry;
import java.util.Set;

/**
 * Histogram calculates the number of pixel components across a list of images that land into a
 * set of different ranges (bins). This can be used for calculating min / max limits for HDR images,
 * or displaying the brightness values of each separate channel.
 *
 * As many high-dynamic-range images are typically non-linear and have bright 'speckles' orders of
 * magnitude higher than the average value, the histogram supports non-linear bin ranges.
 */
public class Histogram {
  /**
   * The minimum and maximum values across all images and their components.
   */
  public final Range limits;

  /**
   * The exponential power used to transform a normalized linear [0, 1] range where 0 represents
   * {@code limits.min}, and 1 represents {@code limits.max} to a normalized bin range [0, 1] where
   * 0 is the first and 1 is the last bin.
   */
  public final double power;

  private final Bin[] bins;
  private final Set<Channel> channels;
  private final Map<Channel, Integer> highestCounts; // Across all bins and channels.

  /**
   * Bin holds the number of pixel values that fall within a fixed range.
   */
  public static class Bin {
    private final Map<Channel, Integer> channels = Maps.newHashMap();

    /**
     * @return the bin count for the given channel.
     */
    public int get(Channel channel) {
      return channels.getOrDefault(channel, 0);
    }

    /**
     * Increments the bin count by one for the given channel.
     * @return the new bin count.
     */
    protected int inc(Channel channel) {
      int count = channels.getOrDefault(channel, 0);
      count++;
      channels.put(channel, count);
      return count;
    }
  }

  /**
   * Range defines an immutable min-max interval of doubles.
   */
  public static class Range {
    public static final Range IDENTITY = new Range(0.0, 1.0);

    public final double min;
    public final double max;

    public Range(double min, double max) {
      this.min = min;
      this.max = max;
    }

    /**
     * @return the value limited to the min and max values of this range.
     */
    public double clamp(double value) {
      return Math.max(Math.min(value, max), min);
    }

    /**
     * @return the linear interpolated value between min and max by frac.
     */
    public double lerp(double frac) {
      return min + (max - min) * frac;
    }

    /**
     * @return the inverse of {@link #lerp}, where X = frac(lerp(X)).
     */
    public double frac(double value) {
      return (value - min) / (max - min);
    }

    /**
     * @return the size of the range interval.
     */
    public double range() {
      return max - min;
    }
  }

  public Histogram(Image[] images, int numBins, boolean logFit) {
    bins = new Bin[numBins];

    for (int i = 0; i < numBins; i++) {
      bins[i] = new Bin();
    }

    Map<Channel, Integer> highestCounts = Maps.newHashMap();
    for (Image image : images) {
      for (Channel channel : image.getChannels()) {
        if (!highestCounts.containsKey(channel)) {
          highestCounts.put(channel, 0);
        }
      }
    }

    // Gather all the pixel values.
    Map<Channel, DoubleBuffer> allValues = Maps.newHashMap();
    for (Channel channel : highestCounts.keySet()) {
      List<DoubleBuffer> buffers = Lists.newArrayList();
      int size = 0;
      for (Image image : images) {
        DoubleBuffer buffer = image.getChannel(channel);
        buffers.add(buffer);
        size += buffer.limit();
      }
      DoubleBuffer buffer = DoubleBuffer.allocate(size);
      for (DoubleBuffer image : buffers) {
        buffer.put(image);
      }
      allValues.put(channel, buffer);
    }

    // Get the limits and average value.
    double min = Double.POSITIVE_INFINITY;
    double max = Double.NEGATIVE_INFINITY;
    double average = 0.0;
    long numValues = 0;
    for (DoubleBuffer values : allValues.values()) {
      for (double value : values.array()) {
        if (!Double.isNaN(value) && !Double.isInfinite(value)) {
          min = Math.min(min, value);
          max = Math.max(max, value);
          average += value;
          numValues++;
        }
      }
    }

    limits = new Range(min, max);

    if (numValues > 0) {
      average /= numValues;
    }

    double P = 1.0;
    if (logFit) {
      // We want the average in the middle of the histogram.
      // Calculate the non-linear power from this.
      // limits.frac(average) ^ P == 0.5
      // P * log(limits.frac(average)) == log(0.5)
      // P = log(0.5) / log(limits.frac(average))
      P = Math.log(0.5) / Math.log(limits.frac(average));

      // Don't go non-linear if it isn't necessary.
      if (P > 0.95 && P < 1.05) {
        P = 1.0;
      }
    }


    // Bucket each of the values into the bins.
    for (Map.Entry<Channel, DoubleBuffer> entry : allValues.entrySet()) {
      Channel channel = entry.getKey();
      for (double value : entry.getValue().array()) {
        if (!Double.isNaN(value) && !Double.isInfinite(value)) {
          int binIdx = (int) (Math.pow(limits.frac(value), P) * (numBins - 1));
          binIdx = Math.max(0, binIdx);
          binIdx = Math.min(binIdx, numBins - 1);
          int count = bins[binIdx].inc(channel);
          highestCounts.put(channel, Math.max(count, highestCounts.get(channel)));
        }
      }
    }

    this.highestCounts = highestCounts;
    this.channels = Sets.immutableEnumSet(allValues.keySet());
    this.power = P;
  }

  /**
   * @param percentile the percentile value ranging from 0 to 100.
   * @param high if true, return the upper limit on the percentile's bin, otherwise the lower limit.
   * @param ignore a list of channels to ignore in the calculation.
   * @return the absolute pixel value at the specified percentile in the histogram.
   */
  public double getPercentile(int percentile, boolean high, Channel ... ignore) {
    Set<Channel> toIgnore = Sets.newHashSet(ignore);

    int highestCount = 0;
    Map<Channel, Integer> cumulative = Maps.newHashMap();
    for (int i = 0; i < bins.length; i++) {
      Bin bin = bins[i];
      for (Entry<Channel, Integer> entry : bin.channels.entrySet()) {
        Channel channel = entry.getKey();
        if (!toIgnore.contains(channel)) {
          int total = cumulative.getOrDefault(channel, 0) + entry.getValue();
          cumulative.put(channel, total);
          highestCount = Math.max(total, highestCount);
        }
      }
    }

    cumulative.clear();

    int threshold = percentile * highestCount / 100;
    for (int i = 0; i < bins.length; i++) {
      Bin bin = bins[i];
      for (Entry<Channel, Integer> entry : bin.channels.entrySet()) {
        Channel channel = entry.getKey();
        if (!toIgnore.contains(channel)) {
          int sum = cumulative.getOrDefault(channel, 0) + entry.getValue();
          if (sum >= threshold) {
            return getValueFromNormalizedX((i + (high ? 1 : 0)) / (float)bins.length);
          }
          cumulative.put(channel, sum);
        }
      }
    }
    return limits.max;
  }

  /**
   * @return the absolute value as a normalized [0, 1] point on the (possibly) non-linear histogram.
   */
  public double getNormalizedXFromValue(double value) {
    return Range.IDENTITY.clamp(Math.pow(limits.frac(value), power));
  }

  /**
   * @return the absolute value from a normalized [0, 1] point on the (possibly) non-linear
   * histogram.
   */
  public double getValueFromNormalizedX(double normalizedX) {
    return limits.lerp(Math.pow(Range.IDENTITY.clamp(normalizedX), 1.0 / power));
  }

  /**
   * @return the normalized channel value for the given channel and bin.
   */
  public float get(Channel channel, int bin) {
    return bins[bin].get(channel) / (float)highestCounts.getOrDefault(channel, 0);
  }

  /**
   * @return the union of all channels across all images.
   */
  public Set<Channel> getChannels() {
    return channels;
  }

  /**
   * @return the number of bins.
   */
  public int getNumBins() {
    return bins.length;
  }
}
