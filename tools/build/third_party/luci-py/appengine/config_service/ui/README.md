# LUCI Config UI

This is a UI for the configuration service.


## Setting up

*	First, make sure you have the [Polymer CLI](https://www.polymer-project.org/2.0/docs/tools/polymer-cli) installed.

*   Install [Google App Engine SDK](https://cloud.google.com/appengine/downloads).

*	Run `bower install` in the ui directory to make sure you have all the dependencies installed.


## Running locally

*	First, change all the URLs in the iron-ajax elements. Simply add "https://luci-config.appspot.com" before each URL.
	*	One in src/config-ui/front-page.html
	*	Two in src/config-ui/config-set.html
	*	One in src/config-ui/config-ui.html

*	In the config-service folder run `dev_appserver.py app.yaml`

*	Visit [http://localhost:8080](http://localhost:8080)


## Running Tests

*	Your application is already set up to be tested via [web-component-tester](https://github.com/Polymer/web-component-tester). 
	Run `wct`, `wct -p` or `polymer test` inside ui folder to run your application's test suites locally. 
	These commands will run tests for all browsers installed on your computer.

## Third Party Files

In order to use proper authentication, the google-signin-aware element was needed. However, this element has not been updated to 
Polymer 2.0, so edits were made to the current version to ensure compatibility.
The modified google-signin-aware element can be found in the ui/common/third_party/google-signin folder.