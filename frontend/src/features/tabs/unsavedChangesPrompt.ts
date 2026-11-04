import { h } from 'vue'
import { NButton, NSpace } from 'naive-ui'
import { i18nGlobal } from '@/i18n'
import { useDialoger } from '@/utils/dialog'

export type UnsavedChoice = 'save' | 'discard' | 'cancel'

function renderContent(names: string[]) {
  if (names.length === 1) {
    return () => i18nGlobal.t('query.unsavedChangesMessage', { filename: names[0] })
  }
  return () =>
    h('div', [
      i18nGlobal.t('query.unsavedChangesMultipleMessage', { count: names.length }),
      h(
        'ul',
        { style: 'margin: 8px 0 0; padding-left: 20px; max-height: 200px; overflow: auto;' },
        names.map((name) => h('li', name)),
      ),
    ])
}

/** Asks once what to do with every unsaved query, however many there are. */
export function promptUnsavedChanges(names: string[]): Promise<UnsavedChoice> {
  const isSingle = names.length === 1
  return new Promise((resolve) => {
    const choose = (choice: UnsavedChoice) => {
      resolve(choice)
      dialog.destroy()
    }
    const button = (label: string, choice: UnsavedChoice, type?: 'primary') =>
      h(NButton, { size: 'small', type, onClick: () => choose(choice) }, { default: () => label })

    const dialog = useDialoger().show({
      type: 'warning',
      title: i18nGlobal.t('query.unsavedChangesTitle'),
      content: renderContent(names),
      action: () =>
        h(NSpace, { justify: 'end' }, {
          default: () => [
            button(i18nGlobal.t('query.unsavedChangesCancel'), 'cancel'),
            button(
              i18nGlobal.t(isSingle ? 'query.unsavedChangesDontSave' : 'query.unsavedChangesDiscardAll'),
              'discard',
            ),
            button(
              i18nGlobal.t(isSingle ? 'query.unsavedChangesSave' : 'query.unsavedChangesSaveAll'),
              'save',
              'primary',
            ),
          ],
        }),
      // Esc or a click outside closes it without a button: that's a cancel. After a
      // button, the promise has already resolved, so this does nothing.
      onAfterLeave: () => resolve('cancel'),
    })
  })
}
