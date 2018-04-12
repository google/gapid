/*
 * Copyright (C) 2018 Google Inc.
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
"use strict";

// The actions module provides base functionality to access the action stashes.
define(["queriable"],
function(queriable) {
  const JOB_STATUS = {
    SCHEDULED: 0,
    RUNNING: 1,
    SUCCESSFUL: 2,
    FAILED: 3
  };

  class Action extends queriable.Base {
    constructor(json) {
      super(json);
    }

    get status() {
      return this.json_.status || 0;
    }

    get statusString() {
      switch (this.status) {
        case JOB_STATUS.SCHEDULED: return "scheduled";
        case JOB_STATUS.RUNNING: return "running";
        case JOB_STATUS.SUCCESSFUL: return "successful";
        case JOB_STATUS.FAILED: return "failed";
      }
    }

    get successful() {
      return this.status == JOB_STATUS.SUCCESSFUL;
    }

    get failed() {
      return this.status == JOB_STATUS.FAILED;
    }

    get input_() {
      return this.json_.input || {};
    }

    get package() {
      return this.json_.package || {};
    }

    get output_() {
      return this.json_.output || {};
    }

    get log() {
      return this.output_.log || "";
    }

    get err() {
      return this.output_.err || "";
    }
  }

  return {
    JOB_STATUS: JOB_STATUS,
    Action: Action,
  }
});
