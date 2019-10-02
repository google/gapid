// Copyright 2018 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

let webpackConfig = require('./webpack.config.js')
// Webpack 3+ configs can be either objects or functions that produce the
// config object. Karma currently doesn't handle the latter, so do it
// ourselves here.
if (typeof webpackConfig === 'function') {
  webpackConfig = webpackConfig({}, {mode: 'development'});
}
webpackConfig.entry = null;
webpackConfig.mode = 'development';

// Allows tests to import modules locally
webpackConfig.resolve = {
    modules: ['./node_modules', './'],
}

// https://github.com/webpack-contrib/karma-webpack/issues/322#issuecomment-417862717
webpackConfig.output = {
    filename: '[name]'
}

module.exports = function(config) {
  config.set({

    // base path that will be used to resolve all patterns (eg. files, exclude)
    basePath: '',


    // frameworks to use
    // available frameworks: https://npmjs.org/browse/keyword/karma-adapter
    frameworks: ['jasmine'],


    // list of files / patterns to load in the browser
    files: [
        'node_modules/@webcomponents/custom-elements/custom-elements.min.js',
        '_all_tests.js',
    ],


    // list of files / patterns to exclude
    exclude: [
    ],

    plugins: [
        'karma-concat-preprocessor',
        'karma-webpack',
        'karma-jasmine',
        'karma-firefox-launcher',
        'karma-chrome-launcher',
        'karma-spec-reporter'
    ],

    // preprocess matching files before serving them to the browser
    // available preprocessors: https://npmjs.org/browse/keyword/karma-preprocessor
    preprocessors: {
        'modules/**/*.js': ['concat'],
        '_all_tests.js': [ 'webpack' ]
    },

    concat: {
        // By default, concat puts everything in a function(){}, but
        // that doesn't work with imports.
        header: '',
        footer: '',
        outputs: [
            {
                file: '_all_tests.js',
                inputs: [
                    'modules/**/*_test.js',
                ],
            },
        ],
    },

    // test results reporter to use
    // possible values: 'dots', 'progress'
    // available reporters: https://npmjs.org/browse/keyword/karma-reporter
    reporters: ['spec'],


    // web server port
    port: parseInt(process.env.KARMA_PORT || '9876'),


    // enable / disable colors in the output (reporters and logs)
    colors: true,


    // level of logging
    // possible values: config.LOG_DISABLE || config.LOG_ERROR || config.LOG_WARN || config.LOG_INFO || config.LOG_DEBUG
    logLevel: config.LOG_INFO,


    // enable / disable watching file and executing tests whenever any file changes
    autoWatch: false,


    // start these browsers
    // available browser launchers: https://npmjs.org/browse/keyword/karma-launcher
    browsers: ['Chrome', 'Firefox'],


    // Continuous Integration mode
    // if true, Karma captures browsers, runs the tests and exits
    singleRun: true,

    // Concurrency level
    // how many browser should be started simultaneous
    concurrency: Infinity,

    webpack: webpackConfig,
  })
}