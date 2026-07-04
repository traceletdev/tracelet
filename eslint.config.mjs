import js from '@eslint/js';
import globals from 'globals';
import tseslint from 'typescript-eslint';
import prettier from 'eslint-config-prettier';

export default [
  {
    ignores: [
      'node_modules/',
      'dist/',
      'out/',
      'binaries/',
      'ui/vscode-extension/out/',
      // Browser instrumentation, kept byte-identical with the Go-embedded copy
      // and validated by browser tests rather than lint.
      'packages/tracelet-react/index.js',
    ],
  },

  // Node CommonJS sources (adapters, framework plugins, scripts).
  {
    files: ['adapters/**/*.{js,cjs}', 'packages/**/*.{js,cjs}', 'scripts/**/*.js'],
    ...js.configs.recommended,
    languageOptions: {
      sourceType: 'commonjs',
      globals: { ...globals.node },
    },
    rules: {
      // Allow intentionally-unused identifiers when prefixed with `_`.
      'no-unused-vars': ['error', { argsIgnorePattern: '^_', varsIgnorePattern: '^_' }],
    },
  },

  // VS Code extension (TypeScript).
  ...tseslint.configs.recommended.map(cfg => ({
    ...cfg,
    files: ['ui/vscode-extension/src/**/*.ts'],
    languageOptions: {
      ...cfg.languageOptions,
      globals: { ...globals.node },
    },
  })),

  // Turn off rules that conflict with Prettier (Prettier owns formatting).
  prettier,
];
