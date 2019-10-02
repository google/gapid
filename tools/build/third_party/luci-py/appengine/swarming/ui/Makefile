# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed by the Apache v2.0 license that can be
# found in the LICENSE file.

# Set up the local directory to run the demo pages.

default:
	bower install

node_modules:
	npm install

# Run a local HTTP server for the demo pages.
.PHONY: run
run: default common_strings
	-wget -nc --output-document=./res/imp/botlist/bot-list-demo.json https://raw.githubusercontent.com/wiki/luci/luci-py/bot-list-demo.json
	-wget -nc --output-document=./res/imp/botlist/bot-list-fleet-data-demo.json https://raw.githubusercontent.com/wiki/luci/luci-py/bot-list-fleet-data-demo.json
	-wget -nc --output-document=./res/imp/tasklist/task-list-demo.json https://raw.githubusercontent.com/wiki/luci/luci-py/task-list-demo.json
	-wget -nc --output-document=./res/imp/tasklist/task-list-fleet-data-demo.json https://raw.githubusercontent.com/wiki/luci/luci-py/task-list-fleet-data-demo.json
	-wget -nc --output-document=./res/imp/tasklist/task-tags-demo.json https://raw.githubusercontent.com/wiki/luci/luci-py/task-tags-demo.json

	-wget -nc --output-document=./res/imp/botpage/bot-demo.json https://raw.githubusercontent.com/wiki/luci/luci-py/bot-demo.json
	-wget -nc --output-document=./res/imp/botpage/bot-events-demo.json https://raw.githubusercontent.com/wiki/luci/luci-py/bot-events-demo.json
	-wget -nc --output-document=./res/imp/botpage/bot-tasks-demo.json https://raw.githubusercontent.com/wiki/luci/luci-py/bot-tasks-demo.json

	-wget -nc --output-document=./res/imp/taskpage/task-request-demo.json https://raw.githubusercontent.com/wiki/luci/luci-py/task-request-demo.json
	-wget -nc --output-document=./res/imp/taskpage/task-result-demo.json https://raw.githubusercontent.com/wiki/luci/luci-py/task-result-demo.json
	-wget -nc --output-document=./res/imp/taskpage/task-stdout-demo.json https://raw.githubusercontent.com/wiki/luci/luci-py/task-stdout-demo.json
	-wget -nc --output-document=./res/imp/taskpage/milo-sample.html https://raw.githubusercontent.com/wiki/luci/luci-py/milo-sample.html
	-wget -nc --output-document=./res/imp/taskpage/new-task-demo.json https://raw.githubusercontent.com/wiki/luci/luci-py/new-task-demo.json

	-wget -nc --output-document=./res/imp/index/index-overview-stats-demo.json https://raw.githubusercontent.com/wiki/luci/luci-py/index-overview-stats-demo.json
	-wget -nc --output-document=./res/imp/stats/stats-overview-demo.json https://raw.githubusercontent.com/wiki/luci/luci-py/stats/stats-overview-demo.json

	node_modules/.bin/http-server -p 9050

.PHONY: clean
clean:
	rm -rf ./build/*
	mkdir -p ./build/js

.PHONY: vulcanize
vulcanize: clean default common_strings natural_sort
	./node_modules/.bin/vulcanize --inline-css --inline-scripts --strip-comments --abspath=./ elements.html | ./node_modules/.bin/html-minifier -o ./build/elements.html --minify-css --minify-js --remove-comments --collapse-whitespace --conservative-collapse
	cat ./res/imp/bower_components/webcomponentsjs/webcomponents-lite.min.js ./build/js/common-strings.js ./node_modules/skia-common-js/common.js ./build/js/natural-sort.js ./res/js/common.js ./res/js/alias.js | ./node_modules/.bin/uglifyjs -o ./build/js/js.js

.PHONY: debug_build
debug_build: clean default common_strings natural_sort
	cp elements.html ./build/elements.html
	cat ./res/imp/bower_components/webcomponentsjs/webcomponents-lite.min.js ./build/js/common-strings.js ./node_modules/skia-common-js/common.js ./build/js/natural-sort.js ./res/js/common.js ./res/js/alias.js > ./build/js/js.js
	ln -s ../res/imp/ ./build/

.PHONY: common_strings
common_strings: node_modules
	echo "(function(){var customUtils = {};" > ./build/js/common-strings.js
	cat node_modules/common-substrings/lib/customUtils.js node_modules/common-substrings/lib/bst.js node_modules/common-substrings/lib/commonSubstrings.js >> ./build/js/common-strings.js
	echo "window.Substrings = SuffixTrie;})();" >> ./build/js/common-strings.js
	# Remove require-js related things.
	sed -i -e 's/module.exports.getRandomArray/customUtils.getRandomArray/g' ./build/js/common-strings.js
	sed -i -e 's/module.exports.defaultCompareKeysFunction/customUtils.defaultCompareKeysFunction/g' ./build/js/common-strings.js
	sed -i -e 's/module.exports.defaultCheckValueEquality/customUtils.defaultCheckValueEquality/g' ./build/js/common-strings.js
	sed -i -e 's/var customUtils = require.*/ /g' ./build/js/common-strings.js
	sed -i -e 's/module.exports = BinarySearchTree;/ /g' ./build/js/common-strings.js
	sed -i -e 's/var BinarySearchTree = require.*/ /g' ./build/js/common-strings.js
	sed -i -e 's/module.exports = SuffixTrie;/ /g' ./build/js/common-strings.js

.PHONY: natural_sort
natural_sort: node_modules
	cat node_modules/javascript-natural-sort/naturalSort.js > ./build/js/natural-sort.js
	sed -i -e 's/module.exports = function naturalSort/window.naturalSort = function/g' ./build/js/natural-sort.js

.PHONY: dev_deploy
dev_deploy: vulcanize
	../tools/gae upl -A chromium-swarm-dev

.PHONY: deploy
deploy: vulcanize
	../tools/gae upl -A chromium-swarm

.PHONY: local_deploy
local_deploy: vulcanize
	../tools/start_servers.py

.PHONY: debug_local_deploy
debug_local_deploy: debug_build
	../tools/start_servers.py

.PHONY: debug_dev_deploy
debug_dev_deploy: debug_build
	../tools/gae upl -A chromium-swarm-dev
