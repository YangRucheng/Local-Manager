<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { annexApi } from '@/api/annex'
import { errorMessage } from '@/api/client'
import { annexFileUrl, formatAnnexRefs } from '@/types'
import type { Annex } from '@/types'

const message = useMessage()

const items = ref<Annex[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)
const recomputing = ref(false)
const keyword = ref('')
const keywordDebounced = ref('')

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

async function load() {
  loading.value = true
  try {
    const res = await annexApi.list({
      keyword: keywordDebounced.value,
      page: page.value,
      page_size: pageSize.value,
    })
    items.value = res.items
    total.value = res.total
  } catch (err) {
    message.error(errorMessage(err))
  } finally {
    loading.value = false
  }
}

let debounceTimer: number | undefined
watch(keyword, (v) => {
  window.clearTimeout(debounceTimer)
  debounceTimer = window.setTimeout(() => {
    keywordDebounced.value = v
  }, 300)
})

watch(keywordDebounced, () => {
  page.value = 1
  void load()
})
watch([page, pageSize], () => void load())

async function recompute() {
  recomputing.value = true
  try {
    await annexApi.recompute()
    message.success('引用次数已重新计算')
    void load()
  } catch (err) {
    message.error(errorMessage(err))
  } finally {
    recomputing.value = false
  }
}

onMounted(() => void load())

const columns = [
  { title: '预览', key: 'id', width: 76 },
  { title: '文件名', key: 'original_name', minWidth: 200, ellipsis: { tooltip: true } },
  { title: '大小', key: 'size', width: 90, align: 'right' as const },
  { title: '引用次数', key: 'ref_count', width: 100, align: 'center' as const },
  { title: '引用位置', key: 'references', minWidth: 240, ellipsis: { tooltip: true } },
  { title: '上传时间', key: 'created_at', width: 170 },
]
</script>

<template>
  <div>
    <div class="toolbar">
      <n-button type="primary" :loading="recomputing" @click="recompute">重算引用次数</n-button>
      <n-input
        v-model:value="keyword"
        placeholder="按文件名搜索"
        clearable
        style="width: 220px"
      />
      <n-text depth="3" style="font-size: 12px">共 {{ total }} 个附件</n-text>
    </div>

    <n-card :bordered="false" class="table-card">
      <template #header>
        <div class="card-header">
          <span>附件清单</span>
          <n-text depth="3" style="font-size: 12px">未引用图片（引用次数 0）会保留在系统中</n-text>
        </div>
      </template>
      <n-data-table
        :columns="columns"
        :data="items"
        :loading="loading"
        :row-key="(row: Annex) => row.id"
        :bordered="false"
        :scroll-x="900"
      >
        <template #empty>
          <n-empty description="暂无附件" />
        </template>
        <template #cell-id="{ row }">
          <n-image
            :src="annexFileUrl((row as Annex).id)"
            :preview-src="annexFileUrl((row as Annex).id)"
            :alt="(row as Annex).original_name"
            object-fit="cover"
            width="48"
            height="48"
            style="border-radius: 4px"
          />
        </template>
        <template #cell-size="{ row }">
          {{ formatSize((row as Annex).size) }}
        </template>
        <template #cell-ref_count="{ row }">
          <n-tag v-if="(row as Annex).ref_count > 0" size="small" type="info" :bordered="false">
            {{ (row as Annex).ref_count }}
          </n-tag>
          <n-tag v-else size="small" type="default" :bordered="false">未引用</n-tag>
        </template>
        <template #cell-references="{ row }">
          <span :class="{ 'no-ref': !formatAnnexRefs((row as Annex).references) }">
            {{ formatAnnexRefs((row as Annex).references) || '—' }}
          </span>
        </template>
      </n-data-table>

      <div class="pagination">
        <n-pagination
          v-model:page="page"
          v-model:page-size="pageSize"
          :item-count="total"
          :page-sizes="[20, 50, 100]"
          show-size-picker
          :page-slot="7"
        />
      </div>
    </n-card>
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}
.table-card {
  border-radius: 10px;
}
.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.pagination {
  display: flex;
  justify-content: flex-end;
  padding-top: 16px;
}
.no-ref {
  color: #c0c4cc;
}
</style>
