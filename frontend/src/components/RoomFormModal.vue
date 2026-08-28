<script setup lang="ts">
import { ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { roomApi } from '@/api/rooms'
import { errorMessage } from '@/api/client'
import type { Room, RoomInput } from '@/types'

const props = defineProps<{ show: boolean; room?: Room | null }>()
const emit = defineEmits<{
  'update:show': [v: boolean]
  saved: [room: Room]
}>()

const message = useMessage()
const submitting = ref(false)

const form = ref<RoomInput>({ name: '', remark: '', image_ids: [] })

watch(
  () => props.show,
  (open) => {
    if (!open) return
    if (props.room) {
      form.value = {
        name: props.room.name,
        remark: props.room.remark,
        image_ids: [...(props.room.image_ids ?? [])],
      }
    } else {
      form.value = { name: '', remark: '', image_ids: [] }
    }
  },
)

async function submit() {
  const name = form.value.name.trim()
  if (!name) {
    message.error('请填写配电室名称')
    return
  }
  submitting.value = true
  try {
    const payload: RoomInput = {
      name,
      remark: form.value.remark ?? '',
      image_ids: form.value.image_ids ?? [],
    }
    const saved = props.room ? await roomApi.update(props.room.id, payload) : await roomApi.create(payload)
    message.success(props.room ? '配电室已更新' : '配电室已创建')
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
    :title="room ? '编辑配电室' : '新建配电室'"
    style="width: 560px"
    :mask-closable="false"
    @update:show="(v: boolean) => emit('update:show', v)"
  >
    <n-form label-placement="top">
      <n-form-item label="名称" required>
        <n-input v-model:value="form.name" placeholder="例如：一号配电室" maxlength="50" />
      </n-form-item>
      <n-form-item label="备注">
        <n-input
          v-model:value="form.remark"
          type="textarea"
          :rows="2"
          placeholder="配电室说明（可选）"
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
          {{ room ? '保存' : '创建' }}
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
