<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useMessage } from 'naive-ui'
import { equipmentApi } from '@/api/equipment'
import { errorMessage } from '@/api/client'
import type { Cabinet, Equipment, EquipmentInput, Room } from '@/types'

const props = defineProps<{
  show: boolean
  record?: Equipment | null
  rooms: Room[]
  cabinets: Cabinet[]
  defaultRoomId?: number
  defaultCabinetId?: number
}>()
const emit = defineEmits<{
  'update:show': [v: boolean]
  saved: [record: Equipment]
}>()

const message = useMessage()
const submitting = ref(false)

const form = ref<EquipmentInput>({
  room_id: 0,
  cabinet_id: null,
  name: '',
  model: '',
  manufacturer: '',
  quantity: 0,
  remark: '',
  image_ids: [],
})

watch(
  () => props.show,
  (open) => {
    if (!open) return
    if (props.record) {
      form.value = {
        room_id: props.record.room_id,
        cabinet_id: props.record.cabinet_id,
        name: props.record.name,
        model: props.record.model,
        manufacturer: props.record.manufacturer,
        quantity: props.record.quantity,
        remark: props.record.remark,
        image_ids: [...(props.record.image_ids ?? [])],
      }
    } else {
      const defaultRoom =
        props.defaultRoomId && props.rooms.some((r) => r.id === props.defaultRoomId)
          ? props.defaultRoomId
          : (props.rooms[0]?.id ?? 0)
      form.value = {
        room_id: defaultRoom,
        cabinet_id: props.defaultCabinetId ?? null,
        name: '',
        model: '',
        manufacturer: '',
        quantity: 0,
        remark: '',
        image_ids: [],
      }
    }
  },
)

// 当前房间可选的柜
const roomCabinets = computed(() => props.cabinets.filter((c) => c.room_id === form.value.room_id))

// 切换房间时若柜不属于该房间则清空
watch(
  () => form.value.room_id,
  (rid) => {
    if (rid && form.value.cabinet_id && !props.cabinets.some((c) => c.id === form.value.cabinet_id && c.room_id === rid)) {
      form.value.cabinet_id = null
    }
  },
)

async function submit() {
  const name = form.value.name.trim()
  if (!form.value.room_id) {
    message.error('请选择配电室')
    return
  }
  if (!name) {
    message.error('请填写名称')
    return
  }
  const quantity = form.value.quantity ?? 0
  if (quantity < 0) {
    message.error('数量不能为负数')
    return
  }
  submitting.value = true
  try {
    const payload: EquipmentInput = {
      room_id: form.value.room_id,
      cabinet_id: form.value.cabinet_id ?? null,
      name,
      model: form.value.model ?? '',
      manufacturer: form.value.manufacturer ?? '',
      quantity,
      remark: form.value.remark ?? '',
      image_ids: form.value.image_ids ?? [],
    }
    const saved = props.record
      ? await equipmentApi.update(props.record.id, payload)
      : await equipmentApi.create(payload)
    message.success(props.record ? '记录已更新' : '记录已创建')
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
    :title="record ? '编辑元器件' : '新建元器件'"
    style="width: 640px"
    :mask-closable="false"
    @update:show="(v: boolean) => emit('update:show', v)"
  >
    <n-form label-placement="top">
      <n-grid :cols="2" :x-gap="16">
        <n-form-item-gi label="配电室" required>
          <n-select
            v-model:value="form.room_id"
            :options="rooms.map((r) => ({ label: r.name, value: r.id }))"
            placeholder="选择配电室"
          />
        </n-form-item-gi>
        <n-form-item-gi label="配电柜">
          <n-select
            v-model:value="form.cabinet_id"
            :options="roomCabinets.map((c) => ({ label: c.name, value: c.id }))"
            placeholder="选择配电柜（可选）"
            clearable
          />
        </n-form-item-gi>
        <n-form-item-gi label="名称" required>
          <n-input v-model:value="form.name" placeholder="设备名称" maxlength="100" />
        </n-form-item-gi>
        <n-form-item-gi label="型号">
          <n-input v-model:value="form.model" placeholder="例如 DZ47-63" maxlength="100" />
        </n-form-item-gi>
        <n-form-item-gi label="厂家">
          <n-input v-model:value="form.manufacturer" placeholder="生产厂家" maxlength="100" />
        </n-form-item-gi>
        <n-form-item-gi label="数量">
          <n-input-number v-model:value="form.quantity" :min="0" style="width: 100%" />
        </n-form-item-gi>
      </n-grid>
      <n-form-item label="备注">
        <n-input
          v-model:value="form.remark"
          type="textarea"
          :rows="2"
          placeholder="备注（可选）"
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
          {{ record ? '保存' : '创建' }}
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
