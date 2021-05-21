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

import static java.util.Arrays.stream;
import static java.util.stream.Collectors.toSet;

import com.google.common.collect.Lists;
import com.google.common.collect.Sets;
import com.google.gapid.image.Image.PixelInfo;
import com.google.gapid.proto.stream.Stream;
import com.google.gapid.proto.stream.Stream.Channel;
import com.google.gapid.util.Range;

import java.util.Collections;
import java.util.List;
import java.util.Set;
import java.util.stream.DoubleStream;
import java.util.stream.IntStream;

/**
 * Histogram calculates the number of pixel components across a list of images that land into a
 * set of different ranges (bins). This can be used for calculating min / max limits for HDR images,
 * or displaying the brightness values of each separate channel.
 *
 * As many high-dynamic-range images are typically non-linear and have bright 'speckles' orders of
 * magnitude higher than the average value, the histogram supports non-linear bin ranges.
 */
public class Histogram {
  private final Set<Channel> channels;
  private final Mapper mapper;
  private final Bins bins;

  private final boolean isCount;

  public Histogram(Image[] images, int numBins) {
    boolean logFit = stream(images).anyMatch(i -> i.getType() == Image.ImageType.HDR);
    this.isCount = stream(images).allMatch(i -> i.getType() == Image.ImageType.COUNT);

    this.channels = getChannels(images);
    this.mapper = Mapper.get(images, logFit);
    this.bins = Bins.get(images, mapper, numBins);
  }

  private static Set<Stream.Channel> getChannels(Image[] images) {
    return Sets.immutableEnumSet(stream(images)
        .flatMap(i -> i.getChannels().stream())
        .collect(toSet()));
  }

  /**
   * Returns the a good default starting range to use for tone mapping.
   */
  public Range getInitialRange(double snapThreshold) {
    // Don't leave data out if it's a count
    if (isCount) {
      // Limit the upper bound to 10 initially so details can be discerned
      return new Range(0., Math.min(10.0 / 255.0, mapper.limits.max));
    }
    if (isLinear()) {
      return Range.IDENTITY;
    }

    double rangeMin = getPercentile(1, false);
    double rangeMax = getPercentile(99, true);

    // Snap the range to the limits if they're close enough.
    if (mapper.normalize(rangeMin) < snapThreshold) {
      rangeMin = mapper.limits.min;
    }
    if (mapper.normalize(rangeMax) > 1.0 - snapThreshold) {
      rangeMax = mapper.limits.max;
    }

    return new Range(rangeMin, rangeMax);
  }

  /**
   * @param percentile the percentile value ranging from 0 to 100.
   * @param lastBin if true, for all the percentile bins from different channels, choose the last
   * bin, otherwise choose the first bin.
   * @return the absolute pixel value at the specified percentile in the histogram.
   */
  private double getPercentile(int percentile, boolean lastBin) {
    List<Integer> percentileBins = Lists.newArrayList();
    for (Stream.Channel c : channels) {
      int bin = bins.getPercentileBin(percentile, c);
      if (bin >= 0) {
        percentileBins.add(bin);
      }
    }
    if (percentileBins.isEmpty()) {
      return mapper.limits.max;
    } else {
      int chosenBin = lastBin ? Collections.max(percentileBins) : Collections.min(percentileBins);
      // If to choose lastBin, return the upper limit of the bin, otherwise the lower limit.
      return getValueFromNormalizedX((chosenBin + (lastBin ? 1 : 0)) / (double)bins.count());
    }
  }

  /**
   * @return the absolute value as a normalized [0, 1] point on the (possibly) non-linear histogram.
   */
  public double getNormalizedXFromValue(double value) {
    return Range.IDENTITY.clamp(mapper.map(value));
  }

  /**
   * @return the absolute value from a normalized [0, 1] point on the (possibly) non-linear
   * histogram.
   */
  public double getValueFromNormalizedX(double normalizedX) {
    return mapper.unmap(Range.IDENTITY.clamp(normalizedX));
  }

  /**
   * @return the normalized channel value for the given channel and bin.
   */
  public float get(Channel channel, int bin) {
    return bins.getNormalized(channel, bin);
  }

  /**
   * @return the union of all channels across all images.
   */
  public Set<Channel> getChannels() {
    return channels;
  }

  /**
   * Returns the number of bins.
   */
  public int getNumBins() {
    return bins.count();
  }

  public boolean isLinear() {
    return !(mapper instanceof ExpMapper);
  }

  /**
   * Returns a {@link DoubleStream} of <code>count</code> values evenly spaced over the [0, 1]
   * interval. If this is a linear histogram, the returned values will be spread linearly,
   * otherwise they will be spread exponentially with the same exponent.
   */
  public DoubleStream range(int count) {
    return mapper.range(count);
  }

  /**
   * Helper to build {@link Bins} instances with a given {@link Mapper}.
   */
  public static class Binner {
    private Mapper mapper;
    private final int numBins;
    private final int[][] bins;

    public Binner(Mapper mapper, int numBins) {
      this.mapper = mapper;
      this.numBins = numBins;
      this.bins = new int[numBins][Stream.Channel.values().length];
    }

    /**
     * Adds the given value as a data point for the given channel, incrementing it's bin count.
     */
    public void bin(float value, Stream.Channel channel) {
      int binIdx = (int)(mapper.map(value) * (numBins - 1));
      binIdx = Math.max(0, Math.min(numBins - 1, binIdx));
      bins[binIdx][getChannelIdx(channel)]++;
    }

    /**
     * Returns a new {@link Bins} instance with the binned counts computed so far.
     */
    public Bins getBins() {
      return new Bins(bins);
    }
  }

  /**
   * Maps values in the given range to a normalized [0, 1] range.
   */
  private static class Mapper {
    protected final Range limits;

    public Mapper(Range limits) {
      this.limits = limits;
    }

    /**
     * Returns a mapper whose range will encompass all values from the given images.
     */
    public static Mapper get(Image[] images, boolean logFit) {
      // Get the limits and average value.
      double min = Double.POSITIVE_INFINITY;
      double max = Double.NEGATIVE_INFINITY;
      double average = 0.0;
      for (Image image : images) {
        PixelInfo info = image.getInfo();
        switch (image.getType()) {
        case LDR:
          min = Math.min(0.0, min);
          max = Math.max(1.0, max);
          break;
        case HDR:
          min = Math.min(info.getMin(), min);
          max = Math.max(info.getMax(), max);
          break;
        case COUNT:
          min = Math.min(0.0, min);
          max = Math.max(info.getMax(), max);
          break;
        }
        average += info.getAverage();
      }

      Range limits = new Range(min, max);

      // This is an average-of-averages, which is OK, because we only ever compute a histogram across
      // multiple images that have the same size. The only reasons the weights of the averages would
      // be different is because we skip the infinite and NaN values when computing the average.
      // These are, though, the exception and would technically make the average be undefined anyways.
      average /= images.length;

      double exponent = 1.0;
      if (logFit) {
        // We want the average in the middle of the histogram.
        // Calculate the non-linear power from this.
        // limits.frac(average) ^ P == 0.5
        // P * log(limits.frac(average)) == log(0.5)
        // P = log(0.5) / log(limits.frac(average))
        exponent = Math.log(0.5) / Math.log(limits.frac(average));

        // Don't go non-linear if it isn't necessary.
        if (exponent > 0.95 && exponent < 1.05) {
          exponent = 1.0;
        }
      }
      return (exponent == 1) ? new Mapper(limits) : new ExpMapper(limits, exponent);
    }

    /**
     * Normalizes the given value to the [0, 1] range. For linear mappings, this is equivalent to
     * the {@link #map(double)} function.
     */
    public double normalize(double value) {
      return limits.frac(value);
    }

    /**
     * Maps the given value to the [0, 1] range. For exponential mappings, the returned values are
     * adjusted to fit the exponential curve.
     */
    public double map(double value) {
      return limits.frac(value);
    }

    /**
     * Returns the inverse of the {@link #map(double)} function.
     */
    public double unmap(double value) {
      return limits.lerp(value);
    }

    /**
     * See {@link Histogram#range(int)}.
     */
    public DoubleStream range(int count) {
      return IntStream.range(1, count).mapToDouble(i -> (double)i / (count - 1));
    }
  }

  /**
   * A {@link Mapper} that fits the range onto an exponential curve.
   */
  private static class ExpMapper extends Mapper {
    /**
     * The exponential power used to transform a normalized linear [0, 1] range where 0 represents
     * {@code limits.min}, and 1 represents {@code limits.max} to a normalized bin range [0, 1]
     * where 0 is the first and 1 is the last bin.
     */
    private final double power;

    public ExpMapper(Range limits, double power) {
      super(limits);
      this.power = power;
    }

    @Override
    public double map(double value) {
      return Math.pow(limits.frac(value), power);
    }

    @Override
    public double unmap(double value) {
      return limits.lerp(Math.pow(value, 1 / power));
    }

    @Override
    public DoubleStream range(int count) {
      return IntStream.range(1, count).mapToDouble(i -> Math.pow((double)i / (count - 1), power));
    }
  }

  /**
   * Holds the histogram's binned data, created with the {@link Binner} class.
   */
  private static class Bins {
    private final int[][] bins;
    private final int[] max, total;

    public Bins(int[][] bins) {
      this.bins = bins;
      this.max = new int[Stream.Channel.values().length];
      this.total = new int[Stream.Channel.values().length];
      computeMaxAndTotals();
    }

    private void computeMaxAndTotals() {
      for (int channel = 0; channel < max.length; channel++) {
        int curMax = 0;
        for (int bin = 0; bin < bins.length; bin++) {
          int value = bins[bin][channel];
          total[channel] += value;
          curMax = Math.max(curMax, value);
        }
        max[channel] = curMax;
      }
    }

    /**
     * Returns the binned data for the given images using the given mapper.
     */
    public static Bins get(Image[] images, Mapper mapper, int numBins) {
      Binner binner = new Binner(mapper, numBins);
      for (Image image : images) {
        image.bin(binner);
      }
      return binner.getBins();
    }

    /**
     * Returns the count for the given channel in the given bin, normalized to a [0, 1] range.
     */
    public float getNormalized(Stream.Channel channel, int bin) {
      int cIdx = getChannelIdx(channel);
      return (float)bins[bin][cIdx] / max[cIdx];
    }

    public int count() {
      return bins.length;
    }

    /**
     * Returns the index of the bin which matches the given percentile, or -1.
     */
    public int getPercentileBin(int percentile, Stream.Channel channel) {
      int cIdx = getChannelIdx(channel);
      int threshold = percentile * total[getChannelIdx(channel)] / 100;
      int sum = 0;
      for (int b = 0; b < bins.length; b++) {
        sum += bins[b][cIdx];
        if (sum >= threshold) {
          return b;
        }
      }
      return -1;
    }
  }

  @SuppressWarnings("ProtocolBufferOrdinal")
  public static int getChannelIdx(Stream.Channel channel) {
    return channel.ordinal();
  }
}
