// Copyright 2019 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

import * as human from 'common-sk/modules/human'

import { humanDuration, sanitizeAndHumanizeTime, timeDiffExact } from '../util'


/** canRetry returns if the given task can be retried.
 *  See https://crbug.com/936530 for one case in which it should not.
 */
export function canRetry(request) {
  return request && request.properties && request.properties.idempotent;
}

/** cipdLink constructs a URL to a CIPD resource given a version string
 *  and a CIPD server URL.
 */
export function cipdLink(actualVersion, server) {
   // actualVersion looks like infra/python/cpython/windows-amd64:1ba7...
  if (!actualVersion || !server) {
    return undefined;
  }
  const splits = actualVersion.split(':');
  if (splits.length !== 2) {
    return undefined;
  }
  const pkg = splits[0];
  const version = splits[1];
  return `${server}/p/${pkg}/+/${version}`;
}

/** durationChart returns an array of times to be displayed in a chart.
 *  These times are pending time, overhead before, task duration, and
 *  overhead after. We truncate them to 1 decimal place for display.
 */
export function durationChart(result) {
  const oneDecimalPlace = function(a) {
    if (!a) {
      return 0.0;
    }
    return Math.round(a * 10)/10;
  }
  let preOverhead = 0;
  let postOverhead = 0;
  // These are only put in upon task completion.
  if (result.performance_stats) {
    postOverhead = (result.performance_stats.isolated_upload &&
                    result.performance_stats.isolated_upload.duration) || 0;
    // We only know the certain timings of isolating. To get
    // close enough (tm) overhead timings, we assume CIPD is the only
    // other source of overhead and all of CIPD's overhead is done pre-task.
    preOverhead = result.performance_stats.bot_overhead - postOverhead;
  }
  return [result.pending, preOverhead,
         result.duration, postOverhead].map(oneDecimalPlace);
}

/** firstDimension returns the first entry in an array of dimensions for
 *  a given key or null if the dimension array is malformed or key is not found.
 */
export function firstDimension(dimensionArr, key) {
  const dimensions = dimensionArr.filter(function(d) {
    return d.key === key;
  });
  if (!dimensions.length) {
    return null;
  }
  const values = dimensions[0].value;
  if (!values.length) {
    return null;
  }
  return values[0];
}

/** hasRichOutput returns true if a task supports a rich logs representation
 *  (e.g. Milo), false otherwise.
 */
export function hasRichOutput(ele) {
  if (!ele || !ele._request || !ele._request.tagMap) {
    return false;
  }
  const tagMap = ele._request.tagMap;
  return tagMap['allow_milo'] || tagMap['luci_project'];
}

/** humanState returns a human readable string corresponding to
 *  the task's state. It takes into account what slice this is
 *  and what slice ran, so as not to confuse the user.
 */
export function humanState(result, currentSliceIdx) {
  if (!result || !result.state) {
    return '';
  }
  if (currentSliceIdx !== undefined && result.current_task_slice !== currentSliceIdx) {
    return 'THIS SLICE DID NOT RUN. Select another slice above.';
  }
  const state = result.state;
  if (state === 'COMPLETED') {
    if (result.failure) {
      return 'COMPLETED (FAILURE)';
    }
    if (wasDeduped(result)) {
      return 'COMPLETED (DEDUPED)';
    }
    return 'COMPLETED (SUCCESS)';
  }
  return state;
}

/** isolateLink constructs a URL to a isolate resource given an
 *  isolate ref object.
 */
export function isolateLink(ref) {
  return ref.isolatedserver + '/browse?namespace='+ref.namespace +
         '&hash=' + ref.isolated;
}

/** isSummaryTask returns true if this task is a summary taskID
 *  and false otherwise.  See taskPageLink for more details.
 */
export function isSummaryTask(id) {
  return id && id.endsWith(0);
}

/** parseRequest pre-processes any data in the task request object.
 */
export function parseRequest(request) {
  if (!request) {
    return {};
  }
  request.tagMap = {};
  request.tags = request.tags || [];
  for (const tag of request.tags) {
    const split = tag.split(':', 1);
    const key = split[0];
    const rest = tag.substring(key.length + 1);
    request.tagMap[key] = rest;
  };

  TASK_TIMES.forEach((time) => {
    sanitizeAndHumanizeTime(request, time);
  });
  return request;
}

/** parseRequest pre-processes any data in the task result object.
 */
export function parseResult(result) {
  if (!result) {
    return {};
  }
  TASK_TIMES.forEach((time) => {
    sanitizeAndHumanizeTime(result, time);
  });

  // In the JSON, this is a string
  if (result.try_number) {
    result.try_number = +result.try_number;
  }

  const now = new Date();
  // Running and bot_died tasks have no duration set, so we can figure it out.
  if (!result.duration && result.state === 'RUNNING' && result.started_ts) {
    result.duration = (now - result.started_ts) / 1000;
  } else if (!result.duration && result.state === 'BOT_DIED' &&
              result.started_ts && result.abandoned_ts) {
    result.duration = (result.abandoned_ts - result.started_ts) / 1000;
  }
  // Make the duration human readable
  result.human_duration = humanDuration(result.duration);
  if (result.state === 'RUNNING') {
    result.human_duration += '*';
  } else if (result.state === 'BOT_DIED') {
    result.human_duration += ' -- died';
  }

  const end = result.started_ts || result.abandoned_ts || new Date();
  if (!result.created_ts) {
    // This should never happen
    result.pending = 0;
    result.human_pending = '';
  } else if (end <= result.created_ts) {
    // In the case of deduplicated tasks, started_ts comes before the task.
    result.pending = 0;
    result.human_pending = '0s';
  } else {
    result.pending = (end - result.created_ts) / 1000; // convert to seconds.
    result.human_pending = timeDiffExact(result.created_ts, end);
  }
  result.current_task_slice = parseInt(result.current_task_slice) || 0;
  return result;
}

/** richLogsLink returns a URL to a rich logs representation (e.g. Milo)
 *  given information in the request/server_details of ele. If the data
 *  is not there (e.g. the task doesn't support it), undefined will be returned.
 */
export function richLogsLink(ele) {
  if (!ele || !ele._request || !ele._request.tagMap) {
    return undefined;
  }
  const tagMap = ele._request.tagMap;
  const miloHost = tagMap['milo_host'];
  let logs = tagMap['log_location'];
  if (logs && miloHost) {
    logs = logs.replace('logdog://', '');
    return miloHost.replace('%s', logs);
  }
  const displayTemplate = ele.server_details.display_server_url_template;
  if (!displayTemplate) {
    return undefined;
  }
  return displayTemplate.replace('%s', ele._taskId);
}

/** sliceExpires returns a human readable time stamp of when a task slice expires.
 */
export function sliceExpires(slice, request) {
  if (!request.created_ts) {
    return '';
  }
  const delta = slice.expiration_secs * 1000;
  return human.localeTime(new Date(request.created_ts.getTime() + delta));
}

/** stateClass returns a class corresponding to the task's state.
 */
export function stateClass(result) {
  if (!result || !result.state) {
    return '';
  }
  const state = result.state;
  if (state === 'CANCELED' || state === 'TIMED_OUT' || state === 'EXPIRED' ||
      state === 'NO_RESOURCE') {
    return 'exception';
  }
  if (state === 'BOT_DIED') {
    return 'bot_died';
  }
  if (state === 'RUNNING' || state === 'PENDING') {
    return 'pending_task';
  }
  if (state === 'COMPLETED') {
    if (result.failure) {
      return 'failed_task';
    }
  }
  return '';
}

/** taskCost returns a human readable cost in USD for a task.
 */
export function taskCost(result) {
  if (!result || !result.costs_usd || !result.costs_usd.length) {
    return 0;
  }
  return result.costs_usd[0].toFixed(4);
}

/** sliceExpires returns a human readable time stamp of when a task expires,
 *  which is after any and all slices expire.
 */
export function taskExpires(request) {
  if (!request.created_ts) {
    return '';
  }
  const delta = request.expiration_secs * 1000;
  return human.localeTime(new Date(request.created_ts.getTime() + delta));
}

export function taskInfoClass(ele, result) {
  // Prevents a flash of grey while request and result load.
  if (!ele || !result || ele._currentSliceIdx === -1) {
    return '';
  }
  if (ele._currentSliceIdx !== result.current_task_slice) {
    return 'inactive';
  }
  return '';
}

/** wasDeduped returns true or false depending on if this task was de-duped.
 */
export function wasDeduped(result) {
  return result.try_number === 0;
}

/** wasPickedUp returns true iff a task was started.
 */
export function wasPickedUp(result) {
  return result && result.state !== 'PENDING' && result.state !== 'NO_RESOURCE' &&
         result.state !== 'CANCELED' && result.state !== 'EXPIRED';
}

const TASK_TIMES = ['abandoned_ts', 'completed_ts', 'created_ts', 'modified_ts',
                    'started_ts'];
