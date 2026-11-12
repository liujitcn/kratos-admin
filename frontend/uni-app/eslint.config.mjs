import pluginVue from 'eslint-plugin-vue'
import tsParser from '@typescript-eslint/parser'
import prettier from '@vue/eslint-config-prettier'
import vueParser from 'vue-eslint-parser'
import {
  configureVueProject,
  defineConfigWithVueTs,
  vueTsConfigs,
} from '@vue/eslint-config-typescript'

const projectFiles = ['**/*.{vue,uvue,js,jsx,cjs,mjs,ts,tsx,cts,mts}']

configureVueProject({
  scriptLangs: ['ts', 'js'],
  rootDir: import.meta.dirname,
})

export default defineConfigWithVueTs(
  {
    ignores: [
      'logs',
      '*.log',
      'npm-debug.log*',
      'yarn-debug.log*',
      'yarn-error.log*',
      'pnpm-debug.log*',
      'lerna-debug.log*',
      'node_modules',
      '**/dist/**',
      '.DS_Store',
      '*.local',
      '.eslintrc.cjs',
      '**/src/rpc/**',
      'apps/uni-app/pages/**',
      'apps/uni-app/pagesMember/**',
      'packages/core/src/module.mjs',
      'packages/core/src/vite.mjs',
      'packages/modules/system/src/index.mjs',
    ],
  },
  pluginVue.configs['flat/essential'],
  vueTsConfigs.recommended,
  prettier,
  {
    files: ['**/*.uvue'],
    languageOptions: {
      parser: vueParser,
      parserOptions: {
        parser: tsParser,
        extraFileExtensions: ['.uvue'],
      },
    },
  },
  {
    files: projectFiles,
    languageOptions: {
      ecmaVersion: 'latest',
      parserOptions: {
        extraFileExtensions: ['.uvue'],
      },
      globals: {
        uni: 'readonly',
        wx: 'readonly',
        WechatMiniprogram: 'readonly',
        getCurrentPages: 'readonly',
        UniApp: 'readonly',
        UniHelper: 'readonly',
        Page: 'readonly',
        AnyObject: 'readonly',
      },
    },
    rules: {
      'prettier/prettier': [
        'warn',
        {
          singleQuote: true,
          semi: false,
          printWidth: 100,
          trailingComma: 'all',
          endOfLine: 'auto',
        },
      ],
      'vue/multi-word-component-names': 'off',
      'vue/no-setup-props-reactivity-loss': 'off',
      'vue/no-deprecated-html-element-is': 'off',
      '@typescript-eslint/no-unused-vars': 'off',
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-empty-object-type': 'off',
    },
  },
  {
    files: ['**/*.uvue'],
    rules: {
      'prettier/prettier': 'off',
    },
  },
)
