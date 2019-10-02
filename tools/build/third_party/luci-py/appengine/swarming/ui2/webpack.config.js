// Copyright 2018 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

const commonBuilder = require('pulito');
const path = require('path');
const CopyWebpackPlugin = require('copy-webpack-plugin');

module.exports = (env, argv) => {
  let config = commonBuilder(env, argv, __dirname);
  // Make all CSS/JS files appear at the /newres location.
  config.output.publicPath='/newres/';
  config.module.rules.push({
    test: /.js$/,
    use: 'html-template-minifier-webpack',
  });
  config.plugins.push(
    new CopyWebpackPlugin([
        { from: 'node_modules/@webcomponents/custom-elements/custom-elements.min.js' },
    ])
  );

  return config;
}
