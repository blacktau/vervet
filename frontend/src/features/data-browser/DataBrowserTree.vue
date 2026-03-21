<script lang="ts" setup>
import { h, watch } from 'vue'
import { NEllipsis, NIcon } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import CollectionIcon from '@/features/icon/CollectionIcon.vue'
import { CircleStackIcon, EyeIcon, FolderIcon, FolderOpenIcon } from '@heroicons/vue/24/outline'
import { DataNodeType, type DataTreeNode } from '@/features/data-browser/types.ts'
import { useDataTreeContextMenu } from '@/features/data-browser/useDataTreeContextMenu.ts'
import { useDataBrowserStore } from '@/features/data-browser/browserStore.ts'
import { useTabStore } from '@/features/tabs/tabs.ts'
import DataTreeContextMenu from '@/features/data-browser/DataTreeContextMenu.vue'
import { useDialogStore } from '@/stores/dialog.ts'
import { useNotifier } from '@/utils/dialog.ts'
import * as collectionsProxy from 'wailsjs/go/api/CollectionsProxy'
import * as databasesProxy from 'wailsjs/go/api/DatabasesProxy'

const tabStore = useTabStore()
const browserStore = useDataBrowserStore()
const contextMenu = useDataTreeContextMenu()
const dialogStore = useDialogStore()
const dialog = useDialog()
const notifier = useNotifier()
const { t } = useI18n()

const renderPrefix = ({ option }: { option: DataTreeNode }) => {
  if (option.type === DataNodeType.Database) {
    return h(NIcon, { size: 18 }, () => h(CircleStackIcon))
  }
  if (option.type === DataNodeType.Folder) {
    const isExpanded = browserStore.currentExpandedKeys.includes(option.key as string)
    const Icon = isExpanded ? FolderOpenIcon : FolderIcon
    return h(NIcon, { size: 18 }, () => h(Icon))
  }
  if (option.type === DataNodeType.Collection) {
    return h(NIcon, { size: 18 }, () => h(CollectionIcon))
  }
  if (option.type === DataNodeType.View) {
    return h(NIcon, { size: 18 }, () => h(EyeIcon))
  }
  return null
}

function handleContextMenuSelect(key: string) {
  const node = contextMenu.selectedNode.value
  if (!node) return

  if (key === 'disconnect' && node.type === DataNodeType.Server) {
    const serverId = node.key as string
    browserStore.disconnect(serverId)
  }

  if (key === 'addDatabase' && node.type === DataNodeType.Server) {
    const serverId = node.key as string
    dialogStore.openAddDatabaseDialog(serverId)
  }

  if (key === 'addCollection' && node.type === DataNodeType.Folder) {
    const nodeKey = node.key as string
    const parts = nodeKey.split(':')
    const serverId = parts[0]
    const dbName = parts[1]
    if (serverId && dbName && parts[2] === 'Collections') {
      dialogStore.openAddCollectionDialog(serverId, dbName)
    }
  }

  if (key === 'openQuery') {
    const nodeKey = node.key as string
    const parts = nodeKey.split(':')

    if (node.type === DataNodeType.Database) {
      const serverId = parts[0]
      const dbName = parts[1]
      if (serverId && dbName) {
        tabStore.openQuery(serverId, dbName)
      }
    }

    if (node.type === DataNodeType.Collection || node.type === DataNodeType.View) {
      const serverId = parts[0]
      const dbName = parts[1]
      const name = parts[3]
      if (serverId && dbName && name) {
        const queryText = `db.getCollection('${name}').find({}).limit(42)`
        tabStore.openQuery(serverId, dbName, queryText, name)
      }
    }
  }

  if (key === 'viewIndexes') {
    const nodeKey = node.key as string
    const parts = nodeKey.split(':')
    if (node.type === DataNodeType.Collection) {
      const serverId = parts[0]
      const dbName = parts[1]
      const collectionName = parts[3]
      if (serverId && dbName && collectionName) {
        tabStore.openIndexTab(serverId, dbName, collectionName)
      }
    }
  }

  if (key === 'serverStatus') {
    if (node.type === DataNodeType.Server) {
      const serverId = node.key as string
      if (serverId) {
        tabStore.openStatisticsTab(serverId, '', '', 'server')
      }
    }
  }

  if (key === 'statistics') {
    if (node.type === DataNodeType.Collection || node.type === DataNodeType.View) {
      const nodeKey = node.key as string
      const parts = nodeKey.split(':')
      const serverId = parts[0]
      const dbName = parts[1]
      const collectionName = parts[3]
      if (serverId && dbName && collectionName) {
        tabStore.openStatisticsTab(serverId, dbName, collectionName, 'collection')
      }
    }
    if (node.type === DataNodeType.Database) {
      const nodeKey = node.key as string
      const parts = nodeKey.split(':')
      const serverId = parts[0]
      const dbName = parts[1]
      if (serverId && dbName) {
        tabStore.openStatisticsTab(serverId, dbName, '', 'database')
      }
    }
  }

  if (key === 'rename') {
    if (node.type === DataNodeType.Collection || node.type === DataNodeType.View) {
      const nodeKey = node.key as string
      const parts = nodeKey.split(':')
      const serverId = parts[0]
      const dbName = parts[1]
      const collectionName = parts[3]
      if (serverId && dbName && collectionName) {
        dialogStore.openRenameCollectionDialog(serverId, dbName, collectionName)
      }
    }
  }

  if (key === 'dropDatabase') {
    if (node.type === DataNodeType.Database) {
      const nodeKey = node.key as string
      const parts = nodeKey.split(':')
      const serverId = parts[0]
      const dbName = parts[1]
      if (serverId && dbName) {
        dialog.warning({
          title: t('common.warning'),
          content: t('dataBrowser.dialogs.dropDatabase.message', { name: dbName }),
          positiveText: t('common.confirm'),
          negativeText: t('common.cancel'),
          onPositiveClick: async () => {
            const result = await databasesProxy.DropDatabase(serverId, dbName)
            if (!result.isSuccess) {
              notifier.error(t(`errors.${result.errorCode}`))
              return
            }
            await browserStore.refreshServerDatabases(serverId)
          },
        })
      }
    }
  }

  if (key === 'refresh') {
    const nodeKey = node.key as string
    const parts = nodeKey.split(':')
    const serverId = parts[0]
    if (serverId) {
      if (node.type === DataNodeType.Database) {
        const dbName = parts[1]
        if (dbName) {
          browserStore.refreshDatabaseCollections(serverId, dbName)
        }
      } else if (node.type === DataNodeType.Folder || node.type === DataNodeType.Collection || node.type === DataNodeType.View) {
        const dbName = parts[1]
        if (dbName) {
          browserStore.refreshDatabaseCollections(serverId, dbName)
        }
      }
    }
  }

  if (key === 'dropCollection') {
    if (node.type === DataNodeType.Collection || node.type === DataNodeType.View) {
      const nodeKey = node.key as string
      const parts = nodeKey.split(':')
      const serverId = parts[0]
      const dbName = parts[1]
      const collectionName = parts[3]
      if (serverId && dbName && collectionName) {
        dialog.warning({
          title: t('common.warning'),
          content: t('dataBrowser.dialogs.dropCollection.message', { name: collectionName }),
          positiveText: t('common.confirm'),
          negativeText: t('common.cancel'),
          onPositiveClick: async () => {
            const result = await collectionsProxy.DropCollection(serverId, dbName, collectionName)
            if (!result.isSuccess) {
              notifier.error(t(`errors.${result.errorCode}`))
              return
            }
            await browserStore.refreshDatabaseCollections(serverId, dbName)
          },
        })
      }
    }
  }
}

const renderLabel = ({ option }: { option: DataTreeNode }) => {
  return h(NEllipsis, { tooltip: { placement: 'right' } }, () => option.label)
}

const nodeProps = ({ option }: { option: DataTreeNode }) => {
  return {
    onContextmenu(e: MouseEvent) {
      e.preventDefault()
      contextMenu.openMenu(option as DataTreeNode, e)
    },
  }
}

watch(
  () => tabStore.currentTabId,
  (serverId) => {
    browserStore.updateTreeForServer(serverId)
  },
  { immediate: true },
)
</script>

<template>
  <div class="browser-tree-wrapper" @contextmenu="(e) => e.preventDefault()">
    <n-tree
      v-if="browserStore.currentTreeData.length > 0"
      :cancelable="false"
      :data="browserStore.currentTreeData"
      :expanded-keys="browserStore.currentExpandedKeys"
      :node-props="nodeProps"
      :render-label="renderLabel"
      :render-prefix="renderPrefix"
      block-line
      block-node
      virtual-scroll
      @update:expanded-keys="browserStore.handleExpand">
      <template #empty>
        <n-empty :description="$t('dataBrowser.tree.empty')" />
      </template>
    </n-tree>
    <DataTreeContextMenu
      :options="contextMenu.contextMenuOptions.value"
      :show="contextMenu.show.value"
      :x="contextMenu.position.value.x"
      :y="contextMenu.position.value.y"
      @close="contextMenu.closeMenu"
      @select="handleContextMenuSelect" />
  </div>
</template>

<style lang="scss" scoped>
.browser-tree-wrapper {
  height: 100%;
  overflow: hidden;
}

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: var(--n-text-color-3);
  font-size: 14px;
}
</style>
