<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { roomApi } from '@/api/rooms'
import { cabinetApi } from '@/api/cabinets'
import { equipmentApi } from '@/api/equipment'
import { errorMessage } from '@/api/client'
import type { Cabinet, Equipment, Room } from '@/types'
import ImageThumbnails from '@/components/ImageThumbnails.vue'
import RecordFormModal from '@/components/RecordFormModal.vue'
import RoomFormModal from '@/components/RoomFormModal.vue'
import CabinetFormModal from '@/components/CabinetFormModal.vue'

const message = useMessage()

const NEW_ROOM = '__new_room__'
const NEW_CABINET = '__new_cabinet__'

// 基础数据
const rooms = ref<Room[]>([])
const cabinets = ref<Cabinet[]>([])
const baseLoading = ref(false)

// 筛选
const roomFilter = ref<number | null>(null)
const cabinetFilter = ref<number | null>(null)
const keyword = ref('')
const keywordDebounced = ref('')

// 列表
const items = ref<Equipment[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const loading = ref(false)

// 弹窗
const recordModalShow = ref(false)
const editingRecord = ref<Equipment | null>(null)
const roomModalShow = ref(false)
const cabinetModalShow = ref(false)

const roomOptions = computed(() => [
  { label: '＋ 新建配电室…', value: NEW_ROOM },
  ...rooms.value.map((r) => ({ label: r.name, value: r.id })),
])

const cabinetOptions = computed(() => {
  const list = roomFilter.value
    ? cabinets.value.filter((c) => c.room_id === roomFilter.value)
    : cabinets.value
  const data = list.map((c) => ({
    label: roomFilter.value ? c.name : `${c.room_name} / ${c.name}`,
    value: c.id,
  }))
  return [{ label: '＋ 新建配电柜…', value: NEW_CABINET }, ...data]
})

async function loadBase() {
  baseLoading.value = true
  try {
    const [r, c] = await Promise.all([roomApi.list(), cabinetApi.list()])
    rooms.value = r
    cabinets.value = c
  } catch (err) {
    message.error(errorMessage(err))
  } finally {
    baseLoading.value = false
  }
}

async function loadData() {
  loading.value = true
  try {
    const res = await equipmentApi.list({
      room_id: roomFilter.value ?? undefined,
      cabinet_id: cabinetFilter.value ?? undefined,
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

// 关键字防抖
let debounceTimer: number | undefined
watch(keyword, (v) => {
  window.clearTimeout(debounceTimer)
  debounceTimer = window.setTimeout(() => {
    keywordDebounced.value = v
  }, 300)
})

// 筛选/关键字变化 → 回到第一页
watch([roomFilter, cabinetFilter, keywordDebounced], () => {
  page.value = 1
  void loadData()
})
watch([page, pageSize], () => void loadData())

// 切换房间时清空柜筛选
watch(roomFilter, (rid) => {
  if (cabinetFilter.value != null) {
    const valid = cabinets.value.some((c) => c.id === cabinetFilter.value && c.room_id === rid)
    if (!valid) cabinetFilter.value = null
  }
})

onMounted(() => {
  void loadBase()
  void loadData()
})

// 下拉菜单 + 弹窗新建
function onRoomSelect(value: string | number | null) {
  if (value === NEW_ROOM) {
    roomModalShow.value = true
    roomFilter.value = null
    return
  }
  roomFilter.value = value as number | null
}

function onCabinetSelect(value: string | number | null) {
  if (value === NEW_CABINET) {
    cabinetModalShow.value = true
    cabinetFilter.value = null
    return
  }
  cabinetFilter.value = value as number | null
}

function onRoomSaved(room: Room) {
  void loadBase()
  roomFilter.value = room.id
}

function onCabinetSaved(cab: Cabinet) {
  void loadBase()
  cabinetFilter.value = cab.id
  if (!roomFilter.value) roomFilter.value = cab.room_id
}

// 元器件弹窗（新建/编辑）
function openCreate() {
  editingRecord.value = null
  recordModalShow.value = true
}

function openEdit(row: Equipment) {
  editingRecord.value = row
  recordModalShow.value = true
}

function onRecordSaved() {
  void loadBase()
  void loadData()
}

async function removeRecord(row: Equipment) {
  try {
    await equipmentApi.remove(row.id)
    message.success('已删除')
    void loadData()
  } catch (err) {
    message.error(errorMessage(err))
  }
}

const columns = [
  { title: '配电室', key: 'room_name', width: 130, ellipsis: { tooltip: true } },
  { title: '配电柜', key: 'cabinet_name', width: 100, ellipsis: { tooltip: true } },
  { title: '名称', key: 'name', width: 170, ellipsis: { tooltip: true } },
  { title: '型号', key: 'model', width: 150, ellipsis: { tooltip: true } },
  { title: '厂家', key: 'manufacturer', width: 150, ellipsis: { tooltip: true } },
  { title: '数量', key: 'quantity', width: 70, align: 'right' as const },
  { title: '备注', key: 'remark', minWidth: 160, ellipsis: { tooltip: true } },
  { title: '图片', key: 'image_ids', width: 150 },
  { title: '操作', key: 'actions', width: 130, align: 'center' as const },
]
</script>

<template>
  <div>
    <div class="toolbar">
      <n-space align="center" :size="10">
        <n-select
          :value="roomFilter"
          placeholder="配电室（全部）"
          clearable
          :options="roomOptions"
          :loading="baseLoading"
          style="width: 180px"
          @update:value="onRoomSelect"
        />
        <n-select
          :value="cabinetFilter"
          placeholder="配电柜（全部）"
          clearable
          :options="cabinetOptions"
          style="width: 180px"
          @update:value="onCabinetSelect"
        />
        <n-input v-model:value="keyword" placeholder="搜索名称 / 型号" clearable style="width: 220px">
          <template #prefix>🔍</template>
        </n-input>
        <n-button type="primary" @click="openCreate">＋ 新建元器件</n-button>
      </n-space>
    </div>

    <n-card :bordered="false" class="table-card">
      <template #header>
        <div class="card-header">
          <span>元器件列表</span>
          <n-text depth="3" style="font-size: 12px">共 {{ total }} 条</n-text>
        </div>
      </template>
      <n-data-table
        :columns="columns"
        :data="items"
        :loading="loading"
        :row-key="(row: Equipment) => row.id"
        :bordered="false"
        :scroll-x="1150"
      >
        <template #empty>
          <n-empty description="暂无元器件，点击「新建元器件」添加" />
        </template>
        <template #cell-image_ids="{ row }">
          <ImageThumbnails :image-ids="(row as Equipment).image_ids" :size="40" />
        </template>
        <template #cell-actions="{ row }">
          <n-space :size="6">
            <n-button size="tiny" @click="openEdit(row as Equipment)">编辑</n-button>
            <n-popconfirm @positive-click="removeRecord(row as Equipment)">
              <template #trigger>
                <n-button size="tiny" type="error" secondary>删除</n-button>
              </template>
              确定删除「{{ (row as Equipment).name }}」吗？
            </n-popconfirm>
          </n-space>
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

    <RecordFormModal
      v-model:show="recordModalShow"
      :record="editingRecord"
      :rooms="rooms"
      :cabinets="cabinets"
      :default-room-id="roomFilter ?? undefined"
      :default-cabinet-id="cabinetFilter ?? undefined"
      @saved="onRecordSaved"
    />
    <RoomFormModal v-model:show="roomModalShow" @saved="onRoomSaved" />
    <CabinetFormModal
      v-model:show="cabinetModalShow"
      :rooms="rooms"
      :default-room-id="roomFilter ?? undefined"
      @saved="onCabinetSaved"
    />
  </div>
</template>

<style scoped>
.toolbar {
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
</style>
