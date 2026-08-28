<script setup lang="ts">
import { ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { cabinetApi } from '@/api/cabinets'
import { errorMessage } from '@/api/client'
import type { Cabinet, CabinetInput, Room } from '@/types'

const props = defineProps<{
  show: boolean
  cabinet?: Cabinet | null
  rooms: Room[]
  defaultRoomId?: number
}>()
const emit = defineEmits<{
  'update:show': [v: boolean]
  saved: [cabinet: Cabinet]
}>()

const message = useMessage()
const submitting = ref(false)

const form = ref<CabinetInput>({ room_id: 0, name: '', remark: '', image_ids: [] })

watch(
  () => props.show,
  (open) => {
    if (!open) return
    if (props.cabinet) {
      form.value = {
        room_id: props.cabinet.room_id,
        name: props.cabinet.name,
        remark: props.cabinet.remark,
        image_ids: [...(props.cabinet.image_ids ?? [])],
      }
    } else {
      form.value = {
        room_id: props.defaultRoomId && props.rooms.some((r) => r.id === props.defaultRoomId)
          ? props.defaultRoomId
          : (props.rooms[0]?.id ?? 0),
        name: '',
        remark: '',
        image_ids: [],
      }
    }
  },
)

async function submit() {
  const name = form.value.name.trim()
  if (!name) {
    message.error('请填写配电柜名称')
    return
  }
  if (!form.value.room_id) {
    message.error('请选择所属配电室')
    return
  }
  submitting.value = true
  try {
    const payload: CabinetInput = {
      room_id: form.value.room_id,
      name,
      remark: form.value.remark ?? '',
      image_ids: form.value.image_ids ?? [],
    }
    const saved = props.cabinet
      ? await cabinetApi.update(props.cabinet.id, payload)
      : await cabinetApi.create(payload)
    message.success(props.cabinet ? '配电柜已更新' : '配电柜已创建')
    emit('saved', saved)
    emit('update:show', false)
  } catch (err) {
    message.error(errorMessage(err))
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <n-modal
    :show="show"
    preset="card"
    :title="cabinet ? '编辑配电柜' : '新建配电柜'"
    style="width: 560px"
    :mask-closable="false"
    @update:show="(v: boolean) => emit('update:show', v)"
  >
    <n-form label-placement="top">
      <n-form-item label="所属配电室" required>
        <n-select
          v-model:value="form.room_id"
          :options="rooms.map((r) => ({ label: r.name, value: r.id }))"
          placeholder="选择配电室"
        />
      </n-form-item>
      <n-form-item label="名称" required>
        <n-input v-model:value="form.name" placeholder="例如：G01" maxlength="50" />
      </n-form-item>
      <n-form-item label="备注">
        <n-input
          v-model:value="form.remark"
          type="textarea"
          :rows="2"
          placeholder="配电柜说明（可选）"
        />
      </n-form-item>
      <n-form-item label="图片（最多 9 张）">
        <ImageUploader v-model="form.image_ids" />
      </n-form-item>
    </n-form>
    <template #footer>
      <div class="footer">
        <n-button @click="emit('update:show', false)">取消</n-button>
        <n-button type="primary" :loading="submitting" @click="submit">
          {{ cabinet ? '保存' : '创建' }}
        </n-button>
      </div>
    </template>
  </n-modal>
</template>

<style scoped>
.footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
