// ═══════════════════════════════════════════════════
// ESLint — Flat Config (v9+)
// ═══════════════════════════════════════════════════
import js from '@eslint/js';
import tseslint from 'typescript-eslint';

export default tseslint.config(
  // ── Ignore ──
  {
    ignores: ['dist/**', 'node_modules/**', 'coverage/**', 'test-results/**', 'playwright-report/**'],
  },

  // ── Base ──
  js.configs.recommended,
  ...tseslint.configs.recommendedTypeChecked,

  // ── Project-wide rules ──
  {
    languageOptions: {
      parserOptions: {
        project: './tsconfig.json',
        tsconfigRootDir: import.meta.dirname,
      },
    },
    rules: {
      // ── Code quality ──
      'no-console': ['warn', { allow: ['warn', 'error'] }],
      'no-unused-vars': 'off',
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_', varsIgnorePattern: '^_' }],

      // ── TypeScript strictness ──
      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/no-unsafe-assignment': 'warn',
      '@typescript-eslint/no-unsafe-member-access': 'warn',
      '@typescript-eslint/prefer-optional-chain': 'error',
      '@typescript-eslint/prefer-nullish-coalescing': 'error',
      '@typescript-eslint/strict-boolean-expressions': 'off',

      // ── Style ──
      '@typescript-eslint/explicit-function-return-type': 'off',
      '@typescript-eslint/no-non-null-assertion': 'warn',
      'object-shorthand': ['error', 'all'],
      curly: ['error', 'all'],
    },
  },

  // ── Relax rules for test files ──
  {
    files: ['**/*.test.ts', '**/*.spec.ts', '**/tests/**'],
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-unsafe-assignment': 'off',
      '@typescript-eslint/no-non-null-assertion': 'off',
      'no-console': 'off',
    },
  },

  // ── Relax rules for main.ts (event listeners use `this`) ──
  {
    files: ['src/main.ts'],
    rules: {
      '@typescript-eslint/no-this-alias': 'off',
    },
  },

  // ── Relax rules for config files ──
  {
    files: ['vite.config.ts', 'eslint.config.js', 'playwright.config.ts'],
    rules: {
      'no-console': 'off',
      '@typescript-eslint/no-explicit-any': 'off',
    },
  },
);
