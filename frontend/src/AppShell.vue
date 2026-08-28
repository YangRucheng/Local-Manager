<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

const menuOptions = [
  { label: '配电室', key: 'rooms' },
  { label: '配电柜', key: 'cabinets' },
  { label: '元器件', key: 'components' },
  {
    label: '系统管理',
    key: 'system',
    type: 'submenu' as const,
    children: [{ label: '附件管理', key: 'annex' }],
  },
]

const activeKey = computed(() => {
  const path = route.path
  if (path.startsWith('/rooms')) return 'rooms'
  if (path.startsWith('/cabinets')) return 'cabinets'
  if (path.startsWith('/system/annex')) return 'annex'
  return 'components'
})

const sectionTitle = computed(
  () =>
    ({
      rooms: '配电室管理',
      cabinets: '配电柜管理',
      components: '元器件台账',
      annex: '附件管理',
    })[activeKey.value] ?? '本地电气台账',
)

function handleMenu(key: string) {
  if (key === 'annex') {
    router.push('/system/annex')
  } else {
    router.push(`/${key}`)
  }
}
</script>

<template>
  <n-layout class="app-shell" position="absolute" has-sider>
    <n-layout-sider bordered :width="176" :native-scrollbar="false" class="sider">
      <div class="brand">本地电气台账</div>
      <n-menu
        :value="activeKey"
        :options="menuOptions"
        :root-indent="14"
        :indent="14"
        :expanded-keys="['system']"
        @update:value="handleMenu"
      />
    </n-layout-sider>

    <n-layout class="main">
      <n-layout-header bordered class="topbar">
        <div class="topbar-title">{{ sectionTitle }}</div>
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
  height: 56px;
  padding: 0 18px;
  font-size: 15px;
  font-weight: 600;
  color: #1e293b;
  border-bottom: 1px solid #eef1f5;
  white-space: nowrap;
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
