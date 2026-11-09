import { globalIgnores } from 'eslint/config'
import { defineConfigWithVueTs, vueTsConfigs } from '@vue/eslint-config-typescript'
import pluginVue from 'eslint-plugin-vue'
import skipFormatting from '@vue/eslint-config-prettier/skip-formatting'

// To allow more languages other than `ts` in `.vue` files, uncomment the following lines:
// import { configureVueProject } from '@vue/eslint-config-typescript'
// configureVueProject({ scriptLangs: ['ts', 'tsx'] })
// More info at https://github.com/vuejs/eslint-config-typescript/#advanced-setup

export default defineConfigWithVueTs(
  {
    name: 'app/files-to-lint',
    files: ['**/*.{ts,mts,tsx,vue}'],
  },

  globalIgnores(['**/dist/**', '**/dist-ssr/**', '**/coverage/**']),

  pluginVue.configs['flat/essential'],
  vueTsConfigs.recommended,
  skipFormatting,

  {
    // Links must go through openInBrowser, which only opens http(s). Banning the
    // runtime's Browser import (renamed or not) and namespace imports of it
    // closes every route to Browser.OpenURL.
    name: 'app/browser-links',
    rules: {
      'no-restricted-imports': [
        'error',
        {
          paths: [
            {
              name: '@wailsio/runtime',
              importNames: ['Browser'],
              message: "Use openInBrowser from '@/utils/openInBrowser': it only opens http(s) links.",
            },
          ],
        },
      ],
      'no-restricted-syntax': [
        'error',
        {
          selector: "ImportDeclaration[source.value='@wailsio/runtime'] > ImportNamespaceSpecifier",
          message: "Import named exports from '@wailsio/runtime'; links go through openInBrowser from '@/utils/openInBrowser'.",
        },
      ],
    },
  },
)
