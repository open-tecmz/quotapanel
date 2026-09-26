/* eslint-disable */
module.exports = {
  root: true,
  env: {
    browser: true,
    es2021: true,
    node: true,
  },
  extends: [
    'eslint:recommended',
    'plugin:vue/vue3-recommended',
    'plugin:@typescript-eslint/recommended',
    'prettier',
  ],
  parser: 'vue-eslint-parser',
  parserOptions: {
    parser: '@typescript-eslint/parser',
    ecmaVersion: 2021,
    sourceType: 'module',
  },
  plugins: ['@typescript-eslint'],
  rules: {
    // 允许 any 类型（项目中大量使用）
    '@typescript-eslint/no-explicit-any': 'off',
    // 未使用变量降级为警告，忽略 _ 前缀参数
    '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_', varsIgnorePattern: '^_' }],
    // 关闭 require() 限制（wailsjs 使用）
    '@typescript-eslint/no-require-imports': 'off',
    // Vue：单词组件名要求关闭（项目中 App.vue 等单词命名存在）
    'vue/multi-word-component-names': 'off',
    // Vue：不强制 prop 默认值
    'vue/require-default-prop': 'off',
    // Vue：允许 v-html（少量场景使用）
    'vue/no-v-html': 'off',
    // 允许 console
    'no-console': 'off',
    // 允许空函数
    '@typescript-eslint/no-empty-function': 'off',
    // 允许空 catch 块（项目中大量使用 catch {} 忽略错误）
    'no-empty': ['error', { allowEmptyCatch: true }],
    // 关闭 ban-types（{} 类型在 Vue 泛型参数中常用）
    '@typescript-eslint/ban-types': 'off',
    // 关闭 no-undef（TypeScript 已处理）
    'no-undef': 'off',
  },
  ignorePatterns: ['**/wailsjs/**', '**/dist/**', '**/node_modules/**', '*.cjs'],
}
