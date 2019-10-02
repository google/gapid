// Copyright 2019 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.
import './index.js'

import { $$ } from 'common-sk/modules/dom'

(function(){

  $$('#show').addEventListener('click', () => {
    $$('dialog-pop-over').show();
  });

  $$('dialog-pop-over button.cancel').addEventListener('click', () => {
    $$('dialog-pop-over').hide();
    $$('#output').textContent = 'Cancel was pressed';
  });

  $$('dialog-pop-over button.ok').addEventListener('click', () => {
    $$('dialog-pop-over').hide();
    $$('#output').textContent = 'OK was pressed; A client could chose to do a "are you sure" or something before hiding.';
  });
})();