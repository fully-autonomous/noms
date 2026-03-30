// Copyright 2016 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

'use strict';

const path = require('path');

module.exports = (env, argv) => {
  const devMode = argv.mode !== 'production';

  return {
    mode: devMode ? 'development' : 'production',
    entry: './src/main.js',
    output: {
      path: path.resolve(__dirname),
      filename: 'out.js',
    },
    module: {
      rules: [{
        test: /\.js$/,
        use: {
          loader: 'babel-loader',
        },
        exclude: /node_modules/,
      }],
    },
    devtool: devMode ? 'inline-source-map' : false,
  };
};