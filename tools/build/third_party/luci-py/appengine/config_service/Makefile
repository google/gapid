# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

build: compile-proto

compile-proto:
	cd components/config/proto && make

test: build
	tools/run_coverage.py

deploy_dev: build
	tools/gae upload -x -A luci-config-dev

upload_dev: build
	tools/gae upload -A luci-config-dev

deploy_staging: build
	tools/gae upload -x -A luci-config-staging

upload_staging: build
	tools/gae upload -A luci-config-staging

deploy: build
	tools/gae upload -x -A luci-config

deploy-spin: build
	$(eval GCBDIR := $(shell mktemp -d -p /tmp luci-config.XXXX))
	# The protoc symlink points to nowhere.
	# Is that expected?
	rsync -aLK --exclude="tools/protoc" . $(GCBDIR)
	cd $(GCBDIR) && gcloud builds submit --timeout 3600 --config cloudbuild.yaml . --project chrome-infra-spinnaker
	rm -fr $(GCBDIR)

upload: build
	tools/gae upload -A luci-config

external_deps:
	cd ui && make
