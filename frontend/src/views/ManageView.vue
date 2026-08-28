<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useDialog, useMessage } from 'naive-ui'
import { roomApi } from '@/api/rooms'
import { cabinetApi } from '@/api/cabinets'
import { annexApi } from '@/api/annex'
import { errorMessage } from '@/api/client'
import type { Cabinet, Room } from '@/types'
import ImageThumbnails from '@/components/ImageThumbnails.vue'
import RoomFormModal from '@/components/RoomFormModal.vue'
import CabinetFormModal from '@/components/CabinetFormModal.vue'

const router = useRouter()
const message = useMessage()
const dialog = useDialog()

const rooms = ref<Room[]>([])
const cabinets = ref<Cabinet[]>([])
const loading = ref(false)

// 弹窗
const roomModalShow = ref(false)
const editingRoom = ref<Room | null>(null)
const cabinetModalShow = ref(false)
const editingCabinet = ref<Cabinet | null>(null)
const cabinetDefaultRoom = ref<number>(0)

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

function cabinetsOf(roomId: number) {
  return cabinets.value.filter((c) => c.room_id === roomId)
}

onMounted(() => void load())

function openCreateRoom() {
  editingRoom.value = null
  roomModalShow.value = true
}

function openEditRoom(room: Room) {
  editingRoom.value = room
  roomModalShow.value = true
}

function openCreateCabinet(roomId: number) {
  editingCabinet.value = null
  cabinetDefaultRoom.value = roomId
  cabinetModalShow.value = true
}

function openEditCabinet(cab: Cabinet) {
  editingCabinet.value = cab
  cabinetDefaultRoom.value = cab.room_id
  cabinetModalShow.value = true
}

function onRoomSaved() {
  void load()
}

function onCabinetSaved() {
  void load()
}

function removeRoom(room: Room) {
  const n = cabinetsOf(room.id).length
  dialog.warning({
    title: '删除配电室',
    content: `确定删除「${room.name}」吗？其下 ${n} 个配电柜及其全部台账记录将一并删除，图片变为未引用状态。此操作不可恢复。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await roomApi.remove(room.id)
        message.success('已删除配电室')
        void load()
      } catch (err) {
        message.error(errorMessage(err))
      }
    },
  })
}

function removeCabinet(cab: Cabinet) {
  dialog.warning({
    title: '删除配电柜',
    content: `确定删除「${cab.name}」吗？其下的台账记录将保留但不再归属该配电柜。`,
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

async function cleanupAnnex() {
  try {
    const res = await annexApi.cleanup()
    message.success(`已清理 ${res.count} 个未引用图片`)
    if (res.count > 0) void load()
  } catch (err) {
    message.error(errorMessage(err))
  }
}

async function recompute() {
  try {
    await annexApi.recompute()
    message.success('引用次数已重新计算')
  } catch (err) {
    message.error(errorMessage(err))
  }
}
</script>

<template>
  <n-layout position="absolute" class="page">
    <n-layout-header class="header" bordered>
      <div class="header-inner">
        <div class="title">
          <n-button size="small" quaternary @click="router.push('/')">← 返回台账</n-button>
          <span class="heading">配电室 / 配电柜管理</span>
        </div>
        <n-space :size="8">
          <n-button size="small" secondary @click="recompute">重算引用次数</n-button>
          <n-button size="small" secondary @click="cleanupAnnex">清理未引用图片</n-button>
          <n-button size="small" type="primary" @click="openCreateRoom">＋ 新建配电室</n-button>
        </n-space>
      </div>
    </n-layout-header>

    <n-layout-content class="content" content-style="padding: 16px 20px 40px">
      <n-grid v-if="rooms.length" :cols="1" :x-gap="12" :y-gap="12" responsive="screen">
        <n-grid-item v-for="room in rooms" :key="room.id">
          <n-card :bordered="true" class="room-card">
            <template #header>
              <div class="room-header">
                <div class="room-title">
                  <span class="room-name">🏢 {{ room.name }}</span>
                  <n-tag size="small" type="info" :bordered="false">
                    {{ room.cabinet_count }} 个配电柜
                  </n-tag>
                </div>
                <n-space :size="6">
                  <n-button size="tiny" secondary @click="openCreateCabinet(room.id)">
                    ＋ 配电柜
                  </n-button>
                  <n-button size="tiny" @click="openEditRoom(room)">编辑</n-button>
                  <n-button size="tiny" type="error" secondary @click="removeRoom(room)">
                    删除
                  </n-button>
                </n-space>
              </div>
            </template>

            <div class="room-body">
              <div class="room-images">
                <span class="field-label">图片：</span>
                <ImageThumbnails :image-ids="room.image_ids" :size="56" :max-show="6" />
              </div>
              <div class="room-remark">
                <span class="field-label">备注：</span>
                <n-text depth="3">{{ room.remark || '—' }}</n-text>
              </div>

              <n-divider v-if="cabinetsOf(room.id).length" style="margin: 12px 0" />
              <div v-if="cabinetsOf(room.id).length" class="cabinet-list">
                <div
                  v-for="cab in cabinetsOf(room.id)"
                  :key="cab.id"
                  class="cabinet-item"
                >
                  <div class="cabinet-main">
                    <span class="cabinet-name">🔌 {{ cab.name }}</span>
                    <span class="cabinet-remark">{{ cab.remark || '—' }}</span>
                    <ImageThumbnails :image-ids="cab.image_ids" :size="36" />
                  </div>
                  <n-space :size="6">
                    <n-button size="tiny" @click="openEditCabinet(cab)">编辑</n-button>
                    <n-button size="tiny" type="error" secondary @click="removeCabinet(cab)">
                      删除
                    </n-button>
                  </n-space>
                </div>
              </div>
              <n-empty
                v-else
                size="small"
                description="暂无配电柜，点击「＋ 配电柜」添加"
                style="padding: 8px 0"
              />
            </div>
          </n-card>
        </n-grid-item>
      </n-grid>

      <n-card v-else :bordered="true" class="empty-card">
        <n-empty description="暂无配电室，点击右上角「＋ 新建配电室」创建">
          <template #extra>
            <n-button size="small" type="primary" @click="openCreateRoom">新建配电室</n-button>
          </template>
        </n-empty>
      </n-card>
    </n-layout-content>

    <RoomFormModal v-model:show="roomModalShow" :room="editingRoom" @saved="onRoomSaved" />
    <CabinetFormModal
      v-model:show="cabinetModalShow"
      :cabinet="editingCabinet"
      :rooms="rooms"
      :default-room-id="cabinetDefaultRoom || undefined"
      @saved="onCabinetSaved"
    />
  </n-layout>
</template>

<style scoped>
.page {
  background: #f5f7fa;
}
.header {
  background: #fff;
}
.header-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 56px;
  padding: 0 20px;
}
.title {
  display: flex;
  align-items: center;
  gap: 8px;
}
.heading {
  font-size: 16px;
  font-weight: 600;
}
.content {
  max-width: 1080px;
  margin: 0 auto;
}
.room-card {
  border-radius: 10px;
}
.room-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.room-title {
  display: flex;
  align-items: center;
  gap: 10px;
}
.room-name {
  font-weight: 600;
}
.room-body {
  padding: 0 4px;
}
.room-images,
.room-remark {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}
.field-label {
  color: #8492a6;
  font-size: 12px;
  flex-shrink: 0;
}
.cabinet-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.cabinet-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 12px;
  background: #fafbfd;
  border: 1px solid #eff1f5;
  border-radius: 8px;
}
.cabinet-main {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.cabinet-name {
  font-weight: 500;
  flex-shrink: 0;
}
.cabinet-remark {
  color: #8492a6;
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.empty-card {
  border-radius: 10px;
}
</style>
