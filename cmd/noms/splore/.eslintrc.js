// Copyright 2017 Attic Labs, Inc. All rights reserved.
// Licensed under the Apache License, version 2.0:
// http://www.apache.org/licenses/LICENSE-2.0

module.exports = {
  parser: '@babel/eslint-parser',
  parserOptions: {
    requireConfigFile: false,
    babelOptions: {
      presets: ['@babel/preset-env', '@babel/preset-react', '@babel/preset-flow'],
    },
    ecmaVersion: 2023,
    sourceType: 'module',
    ecmaFeatures: {
      jsx: true,
    },
  },
  rules: {
    'arrow-body-style': ['error', 'as-needed'],
    camelcase: 'error',
    eqeqeq: 'error',
    'no-fallthrough': 'error',
    'no-new-wrappers': 'error',
    'no-throw-literal': 'error',
    'no-unused-vars': ['error', { argsIgnorePattern: '^_', varsIgnorePattern: '^_$' }],
    'no-var': 'error',
    'prefer-arrow-callback': 'error',
    'prefer-const': 'error',
    'require-yield': 'error',
    radix: 'error',
    'react/jsx-no-duplicate-props': 'error',
    'react/jsx-no-undef': 'error',
    'react/jsx-uses-react': 'error',
    'react/jsx-uses-vars': 'error',
    'react/react-in-jsx-scope': 'off',
  },
  env: {
    es2023: true,
    browser: true,
  },
  extends: ['eslint:recommended', 'plugin:react/recommended'],
  globals: {
    alert: 'readonly',
    console: 'readonly',
    document: 'readonly',
    fetch: 'readonly',
    window: 'readonly',
  },
  plugins: ['flowtype', 'react'],
  settings: {
    react: {
      version: 'detect',
    },
  },
};