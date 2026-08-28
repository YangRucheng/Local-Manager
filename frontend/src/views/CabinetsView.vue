<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useDialog, useMessage } from 'naive-ui'
import { roomApi } from '@/api/rooms'
import { cabinetApi } from '@/api/cabinets'
import { errorMessage } from '@/api/client'
import type { Cabinet, Room } from '@/types'
import ImageThumbnails from '@/components/ImageThumbnails.vue'
import CabinetFormModal from '@/components/CabinetFormModal.vue'

const message = useMessage()
const dialog = useDialog()

const rooms = ref<Room[]>([])
const cabinets = ref<Cabinet[]>([])
const loading = ref(false)
const roomFilter = ref<number | null>(null)

const cabinetModalShow = ref(false)
const editingCabinet = ref<Cabinet | null>(null)
const cabinetDefaultRoom = ref<number>(0)

const filteredCabinets = computed(() =>
  roomFilter.value ? cabinets.value.filter((c) => c.room_id === roomFilter.value) : cabinets.value,
)

async function load() {
  loading.value = true
  try {
    const [r, c] = await Promise.all([roomApi.list(), cabinetApi.list()])
    rooms.value = r
    cabinets.value = c
  } catch (err) {
    message.error(errorMessage(err))
  } finally {
    loading.value = false
  }
}

onMounted(() => void load())

function openCreate() {
  editingCabinet.value = null
  cabinetDefaultRoom.value = roomFilter.value ?? 0
  cabinetModalShow.value = true
}

function openEdit(cab: Cabinet) {
  editingCabinet.value = cab
  cabinetDefaultRoom.value = cab.room_id
  cabinetModalShow.value = true
}

function onSaved() {
  void load()
}

function removeCabinet(cab: Cabinet) {
  dialog.warning({
    title: '删除配电柜',
    content: `确定删除「${cab.name}」吗？其下的元器件记录将保留但不再归属该配电柜。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await cabinetApi.remove(cab.id)
        message.success('已删除配电柜')
        void load()
      } catch (err) {
        message.error(errorMessage(err))
      }
    },
  })
}
</script>

<template>
  <div>
    <div class="toolbar">
      <n-select
        v-model:value="roomFilter"
        placeholder="全部配电室"
        clearable
        :options="rooms.map((r) => ({ label: r.name, value: r.id }))"
        style="width: 200px"
      />
      <n-button type="primary" @click="openCreate">＋ 新建配电柜</n-button>
      <n-text depth="3" style="font-size: 12px">共 {{ filteredCabinets.length }} 个配电柜</n-text>
    </div>

    <n-card :bordered="false" class="table-card">
      <n-data-table
        :columns="[
          { title: '配电室', key: 'room_name', width: 200, ellipsis: { tooltip: true } },
          { title: '名称', key: 'name', width: 200, ellipsis: { tooltip: true } },
          { title: '备注', key: 'remark', minWidth: 200, ellipsis: { tooltip: true } },
          { title: '图片', key: 'image_ids', width: 170 },
          { title: '操作', key: 'actions', width: 130, align: 'center' as const },
        ]"
        :data="filteredCabinets"
        :loading="loading"
        :row-key="(row: Cabinet) => row.id"
        :bordered="false"
        :scroll-x="800"
      >
        <template #empty>
          <n-empty description="暂无配电柜" />
        </template>
        <template #cell-image_ids="{ row }">
          <ImageThumbnails :image-ids="(row as Cabinet).image_ids" :size="40" />
        </template>
        <template #cell-actions="{ row }">
          <n-space :size="6">
            <n-button size="tiny" @click="openEdit(row as Cabinet)">编辑</n-button>
            <n-button size="tiny" type="error" secondary @click="removeCabinet(row as Cabinet)">
              删除
            </n-button>
          </n-space>
        </template>
      </n-data-table>
    </n-card>

    <CabinetFormModal
      v-model:show="cabinetModalShow"
      :cabinet="editingCabinet"
      :rooms="rooms"
      :default-room-id="cabinetDefaultRoom || undefined"
      @saved="onSaved"
    />
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
</style>
