// Copyright 2019 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

import { applyAlias } from '../alias'
import { botListLink, humanDuration, sanitizeAndHumanizeTime, timeDiffExact } from '../util'


/** mpLink produces a machine provider link for this bot
 *  @param {Object} bot - The bot object
 *  @param {Object} serverDetails - The server details returned via the API.
 */
export function mpLink(bot, serverDetails) {
  const template = serverDetails.machine_provider_template
  if (!bot.lease_id || !template) {
    return undefined;
  }
  return template.replace('%s', bot.lease_id);
}

/** parseBotData pre-processes any data in the bot data object.
 *  @param {Object} bot - The raw bot object
 */
export function parseBotData(bot) {
  if (!bot) {
    return {};
  }
  bot.state = bot.state || '{}';
  bot.state = JSON.parse(bot.state) || {};

  bot.dimensions = bot.dimensions || [];
  for (const dim of bot.dimensions) {
    dim.value.forEach(function(value, i) {
      dim.value[i] = applyAlias(value, dim.key);
    });
  }

  bot.device_list = [];
  const devices = bot.state.devices;
  if (devices) {
    for (const id in devices) {
      if (devices.hasOwnProperty(id)) {
        const device = devices[id];
        device.id = id;
        bot.device_list.push(device);
        let count = 0;
        let total = 0;
        // device.temp is a map of zone: 'value'
        device.temp = device.temp || {};
        for (const t in device.temp) {
          total += parseFloat(device.temp[t]);
          count++;
        }
        if (count) {
          device.averageTemp = (total/count).toFixed(1);
        } else {
          device.averageTemp = '???';
        }
      }
    }
  }

  for (const time of BOT_TIMES) {
    sanitizeAndHumanizeTime(bot, time);
  }
  return bot;
}

/** parseBotData pre-processes the events to get them ready to display.
 *  @param {Array<Object>} events - The raw event objects
 */
export function parseEvents(events) {
  if (!events) {
    return [];
  }
  for (const event of events) {
    sanitizeAndHumanizeTime(event, 'ts');
  }

  // Sort the most recent events first.
  events.sort((a, b) => {
    return b.ts - a.ts;
  });
  return events;
}

/** parseTasks pre-processes the tasks to get them ready to display.
 *  @param {Array<Object>} tasks - The raw task objects
 */
export function parseTasks(tasks) {
  if (!tasks) {
    return [];
  }
  for (const task of tasks) {
    for (const time of TASK_TIMES) {
      sanitizeAndHumanizeTime(task, time);
    }
    if (task.duration) {
      // Task is finished
      task.human_duration = humanDuration(task.duration);
    } else {
      const end = task.completed_ts || task.abandoned_ts || task.modified_ts || new Date();
      task.human_duration = timeDiffExact(task.started_ts, end);
      task.duration = (end.getTime() - task.started_ts) / 1000;
    }
    const total_overhead = (task.performance_stats &&
                            task.performance_stats.bot_overhead) || 0;
    // total_duration includes overhead, to give a better sense of the bot
    // being 'busy', e.g. when uploading isolated outputs.
    task.total_duration = task.duration + total_overhead;
    task.human_total_duration = humanDuration(task.total_duration);
    task.total_overhead = total_overhead;

    task.human_state = task.state || 'UNKNOWN';
    if (task.state === 'COMPLETED') {
      // use SUCCESS or FAILURE in ambiguous COMPLETED case.
      if (task.failure) {
        task.human_state = 'FAILURE';
      } else if (task.state !== 'RUNNING') {
        task.human_state = 'SUCCESS';
      }
    }
  }
  tasks.sort((a, b) => {
    return b.started_ts - a.started_ts;
  });
  return tasks;
}

/** quarantineMessage produces a quarantined message for this bot.
 *  @param {Object} bot - The bot object
 */
export function quarantineMessage(bot) {
  if (bot && bot.quarantined) {
    let msg = bot.state.quarantined;
    // Sometimes, the quarantined message is actually in 'error'.  This
    // happens when the bot code has thrown an exception.
    if (msg === undefined || msg === 'true' || msg === true) {
      msg = bot.state && bot.state.error;
    }
    return msg || 'True';
  }
  return '';
}

// Hand-picked list of dimensions that can vary a lot machine to machine,
// that is, dimensions that can be 'too unique'.
const dimensionsToStrip = ['id', 'caches', 'server_version'];

/** siblingBotsLink returns a url to a bot-list that has similar
 *  dimensions to the ones passed in
 *  @param {Array<Object>} dimensions - have 'key' and 'value'. To be
 *                         matched against.
 */
export function siblingBotsLink(dimensions) {
  const cols = ['id', 'os', 'task', 'status'];
   if (!dimensions) {
    return botListLink([], cols);
  }

  dimensions = dimensions.filter((dim) => {
    return dimensionsToStrip.indexOf(dim.key) === -1;
  });

  for (const dim of dimensions) {
    if (cols.indexOf(dim.key) === -1) {
      cols.push(dim.key);
    }
  }

  return botListLink(dimensions, cols);
}

const BOT_TIMES = ['first_seen_ts', 'last_seen_ts', 'lease_expiration_ts'];
const TASK_TIMES = ['started_ts', 'completed_ts', 'abandoned_ts', 'modified_ts'];

// These field filters trim down the data we get per task, which
// may speed up the server time and should speed up the network time.
export const TASKS_QUERY_PARAMS = 'include_performance_stats=true&limit=30&fields=cursor%2Citems(state%2Cbot_version%2Ccompleted_ts%2Ccreated_ts%2Cduration%2Cexit_code%2Cfailure%2Cinternal_failure%2Cmodified_ts%2Cname%2Cperformance_stats(bot_overhead%2Cisolated_download(duration%2Cinitial_number_items%2Cinitial_size%2Cnum_items_cold%2Cnum_items_hot%2Ctotal_bytes_items_cold%2Ctotal_bytes_items_hot)%2Cisolated_upload(duration%2Cinitial_number_items%2Cinitial_size%2Cnum_items_cold%2Cnum_items_hot%2Ctotal_bytes_items_cold%2Ctotal_bytes_items_hot))%2Cserver_versions%2Cstarted_ts%2Ctask_id)';

export const EVENTS_QUERY_PARAMS = 'limit=50&fields=cursor%2Citems(event_type%2Cmaintenance_msg%2Cmessage%2Cquarantined%2Ctask_id%2Cts%2Cversion)';
