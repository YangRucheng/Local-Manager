<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useDialog, useMessage } from 'naive-ui'
import { roomApi } from '@/api/rooms'
import { errorMessage } from '@/api/client'
import type { Room } from '@/types'
import ImageThumbnails from '@/components/ImageThumbnails.vue'
import RoomFormModal from '@/components/RoomFormModal.vue'

const message = useMessage()
const dialog = useDialog()

const rooms = ref<Room[]>([])
const loading = ref(false)

const roomModalShow = ref(false)
const editingRoom = ref<Room | null>(null)

async function load() {
  loading.value = true
  try {
    rooms.value = await roomApi.list()
  } catch (err) {
    message.error(errorMessage(err))
  } finally {
    loading.value = false
  }
}

onMounted(() => void load())

function openCreate() {
  editingRoom.value = null
  roomModalShow.value = true
}

function openEdit(room: Room) {
  editingRoom.value = room
  roomModalShow.value = true
}

function onSaved() {
  void load()
}

function removeRoom(room: Room) {
  dialog.warning({
    title: '删除配电室',
    content: `确定删除「${room.name}」吗？其下 ${room.cabinet_count} 个配电柜及全部元器件记录将一并删除，图片变为未引用状态。此操作不可恢复。`,
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
</script>

<template>
  <div>
    <div class="toolbar">
      <n-button type="primary" @click="openCreate">＋ 新建配电室</n-button>
      <n-text depth="3" style="font-size: 12px">共 {{ rooms.length }} 个配电室</n-text>
    </div>

    <n-grid :cols="1" :x-gap="12" :y-gap="12" responsive="screen">
      <n-grid-item v-for="room in rooms" :key="room.id">
        <n-card :bordered="false" class="room-card">
          <template #header>
            <div class="room-header">
              <div class="room-title">
                <span class="room-name">🏢 {{ room.name }}</span>
                <n-tag size="small" type="info" :bordered="false">
                  {{ room.cabinet_count }} 个配电柜
                </n-tag>
              </div>
              <n-space :size="6">
                <n-button size="tiny" @click="openEdit(room)">编辑</n-button>
                <n-button size="tiny" type="error" secondary @click="removeRoom(room)">删除</n-button>
              </n-space>
            </div>
          </template>

          <div class="room-body">
            <div class="room-row">
              <span class="field-label">图片：</span>
              <ImageThumbnails :image-ids="room.image_ids" :size="56" :max-show="6" />
            </div>
            <div class="room-row">
              <span class="field-label">备注：</span>
              <n-text depth="3">{{ room.remark || '—' }}</n-text>
            </div>
          </div>
        </n-card>
      </n-grid-item>
    </n-grid>

    <n-card v-if="!rooms.length && !loading" :bordered="false" class="empty-card">
      <n-empty description="暂无配电室，点击「新建配电室」创建">
        <template #extra>
          <n-button size="small" type="primary" @click="openCreate">新建配电室</n-button>
        </template>
      </n-empty>
    </n-card>

    <RoomFormModal v-model:show="roomModalShow" :room="editingRoom" @saved="onSaved" />
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
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
.room-row {
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
.empty-card {
  border-radius: 10px;
}
</style>
