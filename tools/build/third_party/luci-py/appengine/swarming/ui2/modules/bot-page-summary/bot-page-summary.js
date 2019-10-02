// Copyright 2019 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

import { html, render } from 'lit-html'
import { humanDuration, initPropertyFromAttrOrProperty } from '../util'

import 'elements-sk/checkbox-sk'
import '../sort-toggle'

/**
 * @module swarming-ui/modules/bot-page-summary
 * @description <h2><code>bot-page-summary<code></h2>
 *
 * <p>
 *   Bot Page Summary shows a table of tasks with some aggregate time
 * </p>
 */

const SHORT_NAME_LENGTH = 50;
const SHOW_LIMIT = 15;

const taskRow = (task, ele) => html`
<tr>
  <td title=${task.full_name} class=break-all>${ele._shortenName(task.full_name)}</td>
  <td>${task.total}</td>
  <td>${task.success}</td>
  <td>${task.failed}</td>
  <td>${task.bot_died}</td>
  <td>${humanDuration(task.avg_duration)}</td>
  <td>${humanDuration(task.avg_overhead)}</td>
  <td>${task.total_time_percent}%</td>
</tr>
`;

const template = (ele) => html`
<table>
  <thead>
    <tr>
      <th>
        <span>Name</span>
        <sort-toggle
            key=full_name
            .currentKey=${ele._sort}
            .direction=${ele._dir}>
        </sort-toggle>
      </th>
      <th>
        <span>Total</span>
        <sort-toggle
            key=total
            .currentKey=${ele._sort}
            .direction=${ele._dir}>
        </sort-toggle>
      </th>
      <th>
        <span>Success</span>
        <sort-toggle
            key=success
            .currentKey=${ele._sort}
            .direction=${ele._dir}>
        </sort-toggle>
      </th>
      <th>
        <span>Failed</span>
        <sort-toggle
            key=failed
            .currentKey=${ele._sort}
            .direction=${ele._dir}>
        </sort-toggle>
      </th>
      <th>
        <span>Died</span>
        <sort-toggle
            key=bot_died
            .currentKey=${ele._sort}
            .direction=${ele._dir}>
        </sort-toggle>
      </th>
      <th>
        <span>Average Duration</span>
        <sort-toggle
            key=avg_duration
            .currentKey=${ele._sort}
            .direction=${ele._dir}>
        </sort-toggle>
      </th>
      <th>
        <span>Average Overhead</span>
        <sort-toggle
            key=avg_overhead
            .currentKey=${ele._sort}
            .direction=${ele._dir}>
        </sort-toggle>
      </th>
      <th>Percent of Total</th>
    </tr>
  </thead>
  <tbody>
    ${ele._sortAndLimitTasks().map((task) => taskRow(task, ele))}

    <tr class=thick>
      <td>Total</td>
      <td>${ele._totalStats.total}</td>
      <td>${ele._totalStats.success}</td>
      <td>${ele._totalStats.failed}</td>
      <td>${ele._totalStats.bot_died}</td>
      <td>${humanDuration(ele._totalStats.avg_duration)}</td>
      <td>${humanDuration(ele._totalStats.avg_overhead)}</td>
      <td>100.0%</td>
    </tr>
  </tbody>
</table>

<div>
  <table>
    <thead>
      <tr>
        <th title="How much time passed between the oldest task fetched and now.">
          Total Wall Time
        </th>
        <th title="How much of the wall time this bot was busy with a task.">
          Wall Time Utilization
        </th>
      </tr>
    </thead>
    <tbody>
      <tr>
        <td>${humanDuration(ele._totalStats.wall_time)}</td>
        <td>${ele._totalStats.wall_time_utilization}%</td>
      </tr>
    </tbody>
  </table>

  <div class=controls>
    <checkbox-sk
        ?checked=${ele._fullNames}
        @click=${ele._toggleName}>
    </checkbox-sk>
    <span>Show Full Names</span>

    <checkbox-sk
        ?hidden=${ele._summarized.length <= SHOW_LIMIT}
        ?checked=${ele._showAllTasks}
        @click=${ele._toggleShow}>
    </checkbox-sk>
    <span ?hidden=${ele._summarized.length <= SHOW_LIMIT}>Show All Tasks</span>
  </div>
</div>
`;

function chromiumNameRules(name) {
  const pieces = name.split('/');
  if (pieces.length === 5) {
    // this appears to be a buildbot name
    // piece 0 is tag "name", piece 3 is "buildername"
    // We throw the rest away (OS, commit hash, build number) so we
    // can identify the "true name".
    name = pieces[0] + '/' + pieces[3];
  }
  name = name.replace(' (with patch)', '');
  return name;
}

// exported only for testing purposes
export function prettifyName(name) {
  name = name.trim();
  name = chromiumNameRules(name);
  // Strip out 'stop' word/phrases that are appended, but don't really
  // change the core of the task.
  name = name.replace(/ \(retry\)/g, '');
  name = name.replace(/ \(debug\)/g, '');

  return name;
}

window.customElements.define('bot-page-summary', class extends HTMLElement {

  constructor() {
    super();

    this._sort = 'total';
    this._dir = 'desc';
    this._summarized = [];
    this._totalStats = {};
    this._showAllTasks = false;
    this._fullNames = false;
  }

  connectedCallback() {
    initPropertyFromAttrOrProperty(this, 'tasks');

    this._sortEvent = (e) => {
      this._sort = e.detail.key;
      this._dir = e.detail.direction;
      this.render();
    };
    this.addEventListener('sort-change', this._sortEvent);
  }

  disconnectedCallback() {
    this.removeEventListener('sort-change', this._sortEvent);
  }

  /** @prop {Array<Object>} tasks - The tasks to summarize.
   */
  get tasks() { return this._tasks || []; }
  set tasks(val) { this._tasks = val; this.render();}

  _aggregate() {
    const totalStats = {
      total: 0,
      success: 0,
      failed: 0,
      bot_died: 0,
      avg_duration: 0,
      avg_overhead: 0,
      total_overhead: 0,
      total_time: 0
    };
    if (!this.tasks || !this.tasks.length) {
      this._totalStats = totalStats;
      this._summarized = [];
      return;
    }

    const now = new Date();
    const taskNames = [];
    const taskAgg = {};

    // to compute wall_time, we find the latest task (and assume tasks
    // come to us chronologically) and find the difference between then
    // and now.
    totalStats.wall_time = (now - this.tasks[this.tasks.length - 1].started_ts) / 1000;

    for (const t of this.tasks) {
      // TODO(kjlubick): maybe have this reason about one or more tags, like
      // if there is a tag called 'name' or something.
      const name = prettifyName(t.name);

      // don't tabulate the running task, as it throws off the averages.
      if (t.state === 'RUNNING') {
        continue;
      }
      if (!taskAgg[name]) {
        taskAgg[name] = {
          full_name: name,
          total: 0,
          success: 0,
          failed: 0,
          bot_died: 0,
          avg_duration: 0,
          total_time: 0,
          total_overhead: 0,
        }
      }
      totalStats.total++;
      taskAgg[name].total++;
      if (t.failure) {
        totalStats.failed++;
        taskAgg[name].failed++;
      } else if (t.internal_failure) {
        totalStats.bot_died++;
        taskAgg[name].bot_died++;
      }
      // Look for total_duration (which is computed in bot-page-helpers
      // to include overhead), and fall back to normal duration otherwise.
      const duration = t.total_duration || t.duration || 0;
      totalStats.total_time += duration;
      taskAgg[name].total_time += duration;
      totalStats.total_overhead += t.total_overhead || 0;
      taskAgg[name].total_overhead += t.total_overhead || 0;
    }

    const summarized = [];
    for (const name in taskAgg) {
      const taskStats = taskAgg[name];
      taskStats.avg_duration = taskStats.total_time / taskStats.total;
      taskStats.avg_overhead = taskStats.total_overhead / taskStats.total;
      taskStats.total_time_percent =
            (taskStats.total_time * 100 / totalStats.total_time).toFixed(1);
      summarized.push(taskStats);
    }

    totalStats.avg_duration = totalStats.total_time / totalStats.total;
    totalStats.avg_overhead = totalStats.total_overhead / totalStats.total;
    totalStats.wall_time_utilization =
        (totalStats.total_time * 100 / totalStats.wall_time).toFixed(1);

    this._totalStats = totalStats;
    this._summarized = summarized;
  }

  render() {
    this._aggregate();
    render(template(this), this, {eventContext: this});
  }

  _sortAndLimitTasks() {
    this._summarized.sort((a, b) => {
      if (!this._sort) {
        return 0;
      }
      let dir = 1;
      if (this._dir === 'desc') {
        dir = -1;
      }
      if (this._sort === 'full_name') {
        return dir * a.full_name.localeCompare(b.full_name);
      }
      // everything else is a number.
      return dir * (a[this._sort] - b[this._sort]);
    });
    if (this._showAllTasks) {
      return this._summarized;
    }
    return this._summarized.slice(0, Math.min(this._summarized.length, SHOW_LIMIT));
  }

  _shortenName(name) {
    if (name.length > SHORT_NAME_LENGTH && !this._fullNames) {
      return name.slice(0, SHORT_NAME_LENGTH-3) + '...';
    }
    return name;
  }

  _toggleName(e) {
    e.preventDefault();
    this._fullNames = !this._fullNames;
    this.render();
  }

  _toggleShow(e) {
    e.preventDefault();
    this._showAllTasks = !this._showAllTasks;
    this.render();
  }

});
