// Copyright 2018 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

import 'elements-sk/error-toast-sk'

import './index.js'
import { mockAuthdAppGETs } from '../test_util'

(function(){

const fetchMock = require('fetch-mock');

mockAuthdAppGETs(fetchMock, {
  can_pet_dogs: true
});

const btn = document.getElementById('test-button');
btn.addEventListener('click', () => {
  const swapp = document.getElementsByTagName('swarming-app');
  swapp[0].addBusyTasks(1);
});

})();