// Copyright 2019 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.
import './index.js'

import { tasksMap } from '../bot-page/test_data'
import { $$ } from 'common-sk/modules/dom'
import { parseTasks } from '../bot-page/bot-page-helpers'

let tasks = tasksMap['SkiaGPU'];

$$('bot-page-summary').tasks = parseTasks(tasks);