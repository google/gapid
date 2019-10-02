# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
projdir := $(patsubst %/,%,$(dir $(mkfile_path)))
deploy := $(abspath $(projdir)/../../../../infra_internal/appengine/deploy.py)

deploy-check:
ifeq (,$(wildcard $(deploy)))
	$(error $(deploy) not found, deployment is only supported inside an infra_internal checkout)
endif

up: deploy-check
	$(deploy) --group luci-prod --push --service swarming

switch: deploy-check
	$(deploy) --group luci-prod --switch --service swarming

dev:
	gae.py devserver -A swarming-local --app-dir . -- --port 9050 --admin_port 8200 --log_level debug
