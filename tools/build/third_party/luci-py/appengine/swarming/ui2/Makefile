.ONESHELL:

default: package-lock.json
	npx webpack --mode=development

release: package-lock.json
	npx webpack --mode=production

serve: package-lock.json
	npx webpack-dev-server --mode=development --watch-poll

test: default
	# Run the generated tests just once under Xvfb.
	xvfb-run --auto-servernum --server-args "-screen 0 1280x1024x24" npx karma start

browser_test:
	# Good for debugging/developing tests.
	# Starts up the server so one can navigate to http://0.0.0.0:9876 and run the
	# tests there, seeing the results. Unfortunately, due to how karma needs to
	# concatenate the _test.js files and run them through webpack, it doesn't
	# autodetect changes and needs to be re-run manually :(
	xvfb-run --auto-servernum --server-args "-screen 0 1280x1024x24" npx karma start --no-single-run

continuous_test:
	# Good for debugging/developing tests.
	# Runs the tests every time a file changes in ./modules/
	# Works on linux only, with an apt-get install inotify-tools
	make test
	while inotifywait -e modify ./modules/*; do
		make test
	done

.PHONY: docs
docs: package-lock.json
	npx jsdoc -c jsdoc.config.js -d docs
	xdg-open docs/index.html

package-lock.json: package.json
	npm install
	# This should tell Makefile that the "built" package-lock.json file is older than
	# the "source" package.json and thus does not require to be "rebuilt".
	touch package-lock.json
