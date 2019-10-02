// Copyright 2019 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

import { GoogleCharts } from 'google-charts';
import { html, render } from 'lit-html'
import { initPropertyFromAttrOrProperty } from '../util'


/**
 * @module swarming-ui/modules/stacked-time-chart
 * @description <h2><code>stacked-time-chart<code></h2>
 *
 * <p>
 *   This creates a stacked bar chart assuming the passed in values are
 *   times/durations. The caller can customize the labels and colors (e.g. have
 *   a label and color repeat).
 * </p>
 *
 * <p>
 *   Why not use the <google-chart> element directly?  Unfortunately
 *   stacked-bar-charts are not one of the options.
 *   https://google-developers.appspot.com/chart/interactive/docs/gallery
 * </p>
 *
 * <p>
 *   Note: the following attributes can be supplied as a property when hard-coded.
 *   Be sure to use a JSON-ified array with single quotes on the outside, double
 *   quotes on the inside.
 *   See stacked-time-chart-demo.html.
 * </p>
 *
 * @prop {Array<String>} values - The floating point values to display, in the order
 *                       they should be displayed.
 * @prop {Array<String>} labels - The human readable labels associated with values.
 * @prop {Array<String>} colors - The hex colors that the data should be, associated
 *                       with values.
 */

const loaded = new Promise((resolve, reject) => {
  try {
    GoogleCharts.load(resolve);
  }
  catch (e) {
    reject(e);
  }
});

function makeArray(maybeString) {
  if (typeof maybeString === 'string') {
    return JSON.parse(maybeString);
  }
  return maybeString;
}

const template = (ele) => html`<div id=chart>${ele._error}</div>`;

window.customElements.define('stacked-time-chart', class extends HTMLElement {

  constructor() {
    super();
    this._loaded = false;
    this._error = '';
  }

  connectedCallback() {
    initPropertyFromAttrOrProperty(this, 'colors');
    initPropertyFromAttrOrProperty(this, 'labels');
    initPropertyFromAttrOrProperty(this, 'values');

    // When passed in as literals via HTML, they are stringified JSON
    // Arrays, so they need parsing.
    this._colors = makeArray(this._colors);
    this._labels = makeArray(this._labels);
    this._values = makeArray(this._values);

    loaded.then(() => {
      this._loaded = true;
      this.render();
    }).catch((e) => {
      console.error(e);
      this._error = 'Could not load Google Charts JS from Internet';
      this.render();
    });
    this.render();
  }

  get colors() { return this._colors; }
  set colors(val) { this._colors = val; this.render();}

  get labels() { return this._labels; }
  set labels(val) { this._labels = val; this.render();}

  get values() { return this._values; }
  set values(val) { this._values = val; this.render();}

  drawChart() {
    // Create the data table. The first row is the headers. The second row
    // (and any another rows) are the data that fill it in. This element
    // is tuned for only one stacked chart, so there will only be one data row.
    const data = GoogleCharts.api.visualization.arrayToDataTable([
        // 'Type' is just a human-friendly value to remind us what the rest
        // of the headers are. It could be empty, as it doesn't show up on the
        // chart.
        ['Type'].concat(this._labels),
        // The empty string below would be a left-hand label, but since there
        // is only one entry, the label is superfluous.
        [''].concat(this._values),
    ]);

    // Do some computation to make axis lines show up nicely for different
    // ranges of duration. ticks represents the major lines (and what) should
    // be labeled. gridCount is the number of minor lines to show up between
    // the major lines.
    let total = 0;
    for (const v of this._values) {
      total += +v;
    }
    const ticks = [{v: 0, f:''}];
    let gridCount = 0;
    if (total < 120) { // 2 min
      for (let t = 10; t < total; t+=10) {
        ticks.push({v: t, f: t+'s'});
      }
      gridCount = 5;
    } else if (total < 1500) { // 25m
      for (let t = 60; t < total; t+=60) {
        ticks.push({v: t, f: t/60+'m'});
      }
      if (total < 300) {
        gridCount = 5;
      } else if (total < 900) {
        gridCount = 1;
      } else {
        gridCount = 0;
      }
      // Prevent tasks with super long times (> 10 days) from locking up the drawing.
    } else if (total < 1000000) {
      for (let t = 600; t < total; t+=600) {
        ticks.push({v: t, f: t/60+'m'});
      }
      if (total < 6000) {
        gridCount = 10;
      } else if (total < 12000) {
        gridCount = 5;
      } else {
        gridCount = 1;
      }
    }

    // These options make a stacked bar chart, using the passed in colors
    // with the legend on top, and the configured amount of minor grid lines.
    const options = {
      width: 400,
      height: 250,
      isStacked: true,
      // chartArea is how big the chart should be in the allocated space.
      // We want it to be as wide as possible, leaving a little bit of space
      // on the top and bottom for the legend and labels.  These values
      // were found via experimentation.
      chartArea: {width: '100%', height:'65%'},
      legend: {
        position: 'top',
        // Force the legend onto one line - can be tweaked if necessary
        maxLines: 1,
        alignment: 'center',
        textStyle: {fontSize: 12}
      },
      colors: this._colors,
      hAxis: {
        title: 'Time',
        ticks: ticks,
        minorGridlines: {count: gridCount},
      }
    };

    const chart = new GoogleCharts.api.visualization.BarChart(this.firstElementChild);
    chart.draw(data, options);
  }

  render() {
    render(template(this), this, {eventContext: this});
    if (this._loaded) {
      this.drawChart();
    }
  }

});
