<script setup lang="ts">
import { ref } from 'vue'
import { useMessage } from 'naive-ui'
import { annexApi } from '@/api/annex'
import { errorMessage } from '@/api/client'
import { annexFileUrl, MAX_IMAGES } from '@/types'
import { mergeImageIds, validateImageSelection } from '@/utils/image'

const props = withDefaults(
  defineProps<{ modelValue: number[]; disabled?: boolean; max?: number }>(),
  { max: MAX_IMAGES },
)
const emit = defineEmits<{ 'update:modelValue': [ids: number[]] }>()

const input = ref<HTMLInputElement | null>(null)
const uploading = ref(false)
const message = useMessage()

async function uploadSelected(selected: File[]) {
  if (!selected.length || uploading.value || props.disabled) return
  const error = validateImageSelection(props.modelValue.length, selected, props.max)
  if (error) {
    message.error(error)
    return
  }
  uploading.value = true
  try {
    const uploaded = await Promise.all(selected.map((f) => annexApi.upload(f)))
    emit('update:modelValue', mergeImageIds(props.modelValue, uploaded.map((a) => a.id)))
  } catch (err) {
    message.error(errorMessage(err))
  } finally {
    uploading.value = false
  }
}

function choose(event: Event) {
  const selected = Array.from((event.target as HTMLInputElement).files ?? [])
  void uploadSelected(selected).finally(() => {
    if (input.value) input.value.value = ''
  })
}

function pasteImages(event: ClipboardEvent) {
  if (props.disabled || uploading.value) return
  const files = Array.from(event.clipboardData?.items ?? [])
    .filter((item) => item.kind === 'file' && item.type.startsWith('image/'))
    .map((item) => item.getAsFile())
    .filter((f): f is File => f !== null)
  if (!files.length) {
    message.warning('剪贴板中没有可粘贴的图片')
    return
  }
  event.preventDefault()
  void uploadSelected(files)
}

function remove(id: number) {
  emit('update:modelValue', props.modelValue.filter((x) => x !== id))
}
</script>

<template>
  <div class="image-upload-field">
    <div
      class="image-uploader"
      :tabindex="disabled ? -1 : 0"
      aria-label="图片上传区域，支持 Ctrl+V 粘贴图片"
      @paste="pasteImages"
    >
      <div v-for="(id, i) in modelValue" :key="id" class="image-item">
        <n-image
          :src="annexFileUrl(id)"
          :preview-src="annexFileUrl(id)"
          :alt="`图片${i + 1}`"
          object-fit="cover"
          width="96"
          height="96"
        />
        <n-button
          v-if="!disabled"
          class="remove"
          size="tiny"
          circle
          type="error"
          @click="remove(id)"
        >
          ×
        </n-button>
      </div>
      <button
        v-if="!disabled && modelValue.length < max"
        type="button"
        class="upload-trigger"
        :disabled="uploading"
        @click="input?.click()"
      >
        <span class="plus">+</span>
        <span>{{ uploading ? '上传中…' : '添加图片' }}</span>
      </button>
      <input
        ref="input"
        hidden
        type="file"
        multiple
        accept="image/jpeg,image/png,image/webp,image/gif"
        @change="choose"
      />
    </div>
    <p class="image-hint">
      JPG / PNG / WebP / GIF · 单张不超过 10MB · 最多 {{ max }} 张 · 支持 Ctrl+V 粘贴
    </p>
  </div>
</template>

<style scoped>
.image-upload-field {
  width: 100%;
}
.image-uploader {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  min-height: 104px;
  padding: 12px;
  border: 1px solid #e0e6ef;
  border-radius: 10px;
  background: #fafbfd;
}
.image-uploader:focus-visible {
  outline: 2px solid rgba(37, 99, 235, 0.3);
  outline-offset: 2px;
}
.image-item {
  position: relative;
  width: 98px;
  height: 98px;
  border: 1px solid #e0e6ef;
  border-radius: 8px;
  background: #fff;
  overflow: hidden;
}
.remove {
  position: absolute;
  top: 4px;
  right: 4px;
  z-index: 2;
}
.upload-trigger {
  width: 98px;
  height: 98px;
  border: 1px dashed #b8c4df;
  border-radius: 8px;
  background: #fff;
  color: #8492a6;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  transition:
    border-color 0.2s ease,
    background-color 0.2s ease,
    color 0.2s ease;
}
.upload-trigger:hover {
  border-color: #2563eb;
  background: #eff6ff;
  color: #2563eb;
}
.upload-trigger:disabled {
  cursor: wait;
  opacity: 0.7;
}
.plus {
  font-size: 26px;
  line-height: 1;
}
.image-hint {
  margin: 7px 2px 0;
  color: #8492a6;
  font-size: 12px;
  line-height: 1.6;
}
</style>
