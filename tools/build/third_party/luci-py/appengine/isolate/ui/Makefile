# Copyright 2017 The LUCI Authors. All rights reserved.
# Use of this source code is governed by the Apache v2.0 license that can be
# found in the LICENSE file.

# Set up the local directory to run the demo pages.

default:
	bower install

node_modules:
	npm install

# Run a local HTTP server for the demo pages.
.PHONY: run
run: default
	node_modules/.bin/http-server -p 9050

.PHONY: clean
clean:
	rm -rf ./build/*
	mkdir -p ./build/js

.PHONY: vulcanize
vulcanize: clean default
	./node_modules/.bin/vulcanize --inline-css --inline-scripts --strip-comments --abspath=./ elements.html | ./node_modules/.bin/html-minifier -o ./build/elements.html --minify-css --minify-js --remove-comments --collapse-whitespace --conservative-collapse
	cat ./res/imp/bower_components/webcomponentsjs/webcomponents-lite.min.js ./node_modules/skia-common-js/common.js ./res/js/common.js | ./node_modules/.bin/uglifyjs -o ./build/js/js.js

.PHONY: debug_build
debug_build: clean default
	cp elements.html ./build/elements.html
	cat ./res/imp/bower_components/webcomponentsjs/webcomponents-lite.min.js ./node_modules/skia-common-js/common.js ./res/js/common.js > ./build/js/js.js
	ln -s ../res/imp/ ./build/

.PHONY: dev_deploy
dev_deploy: vulcanize
	../tools/gae upl -A isolateserver-dev

.PHONY: deploy
deploy: vulcanize
	../tools/gae upl -A isolateserver

.PHONY: local_deploy
local_deploy: vulcanize
	../tools/start_servers.py

.PHONY: debug_local_deploy
debug_local_deploy: debug_build
	../tools/start_servers.py

.PHONY: debug_dev_deploy
debug_dev_deploy: debug_build
	../tools/gae upl -A isolateserver-dev
