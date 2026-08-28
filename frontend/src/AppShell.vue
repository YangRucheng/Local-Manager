<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { annexApi } from '@/api/annex'
import { errorMessage } from '@/api/client'

// AppShell 渲染在 <n-message-provider> 内部，此处 useMessage 可正常使用
const route = useRoute()
const router = useRouter()
const message = useMessage()

const menuOptions = [
  { label: '🏢 配电室', key: 'rooms' },
  { label: '🔌 配电柜', key: 'cabinets' },
  { label: '⚡ 元器件', key: 'components' },
]

const activeKey = computed(() => {
  const path = route.path
  if (path.startsWith('/rooms')) return 'rooms'
  if (path.startsWith('/cabinets')) return 'cabinets'
  return 'components'
})

const sectionTitle = computed(
  () =>
    ({
      rooms: '配电室管理',
      cabinets: '配电柜管理',
      components: '元器件台账',
    })[activeKey.value] ?? '本地电气台账',
)

function handleMenu(key: string) {
  router.push(`/${key}`)
}

async function recompute() {
  try {
    await annexApi.recompute()
    message.success('引用次数已重新计算')
  } catch (err) {
    message.error(errorMessage(err))
  }
}

async function cleanup() {
  try {
    const res = await annexApi.cleanup()
    message.success(res.count ? `已清理 ${res.count} 个未引用图片` : '没有需要清理的未引用图片')
  } catch (err) {
    message.error(errorMessage(err))
  }
}
</script>

<template>
  <n-layout class="app-shell" position="absolute" has-sider>
    <n-layout-sider bordered :width="176" :native-scrollbar="false" class="sider">
      <div class="brand">
        <span class="brand-icon">⚡</span>
        <span>本地电气台账</span>
      </div>
      <n-menu
        :value="activeKey"
        :options="menuOptions"
        :root-indent="14"
        :indent="14"
        @update:value="handleMenu"
      />
    </n-layout-sider>

    <n-layout class="main">
      <n-layout-header bordered class="topbar">
        <div class="topbar-title">{{ sectionTitle }}</div>
        <n-space :size="8">
          <n-button size="small" secondary @click="recompute">重算引用次数</n-button>
          <n-button size="small" secondary @click="cleanup">清理未引用图片</n-button>
        </n-space>
      </n-layout-header>
      <n-layout-content class="content" :native-scrollbar="false">
        <div class="content-inner">
          <router-view />
        </div>
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<style scoped>
.sider {
  background: #fff;
}
.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 56px;
  padding: 0 18px;
  font-size: 15px;
  font-weight: 600;
  color: #1e293b;
  border-bottom: 1px solid #eef1f5;
  white-space: nowrap;
}
.brand-icon {
  font-size: 18px;
}
.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 56px;
  padding: 0 20px;
  background: #fff;
}
.topbar-title {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
}
.main {
  background: #f5f7fa;
}
.content {
  background: #f5f7fa;
}
.content-inner {
  max-width: 1280px;
  margin: 0 auto;
  padding: 16px 20px 40px;
}
</style>
