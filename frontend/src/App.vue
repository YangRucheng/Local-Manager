<script setup lang="ts">
import { dateZhCN, zhCN } from 'naive-ui'
import AppShell from './AppShell.vue'

// 蓝色系主题（语义色 success/error/warning 保持默认）
const themeOverrides = {
  common: {
    primaryColor: '#2563EB',
    primaryColorHover: '#3B82F6',
    primaryColorPressed: '#1D4ED8',
    primaryColorSuppl: '#3B82F6',
    infoColor: '#2563EB',
    infoColorHover: '#3B82F6',
    infoColorPressed: '#1D4ED8',
    infoColorSuppl: '#3B82F6',
  },
}
</script>

<template>
  <n-config-provider :locale="zhCN" :date-locale="dateZhCN" :theme-overrides="themeOverrides">
    <n-message-provider>
      <n-dialog-provider>
        <AppShell />
      </n-dialog-provider>
    </n-message-provider>
  </n-config-provider>
</template>

<style>
html,
body,
#app {
  height: 100%;
  margin: 0;
  padding: 0;
}
body {
  font-family:
    -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Hiragino Sans GB',
    'Microsoft YaHei', sans-serif;
  background: #f5f7fa;
}
.app-shell {
  background: #f5f7fa;
}
</style>
