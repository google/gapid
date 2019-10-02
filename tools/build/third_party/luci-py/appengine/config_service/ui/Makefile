# Copyright 2017 The LUCI Authors. All rights reserved.
# Use of this source code is governed by the Apache v2.0 license that can be
# found in the LICENSE file.

default:
	bower install

.PHONY: run
run: default
	polymer serve

.PHONY: clean
clean:
	rm -rf bower_components
