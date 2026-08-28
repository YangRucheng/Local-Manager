<script setup lang="ts">
import { computed } from 'vue'
import { annexFileUrl } from '@/types'

const props = withDefaults(
  defineProps<{ imageIds: number[]; size?: number; maxShow?: number }>(),
  { size: 40, maxShow: 3 },
)

const shown = computed(() => props.imageIds.slice(0, props.maxShow))
const hiddenCount = computed(() => Math.max(0, props.imageIds.length - props.maxShow))
</script>

<template>
  <div v-if="imageIds.length" class="thumbs">
    <n-image
      v-for="(id, i) in shown"
      :key="id"
      :src="annexFileUrl(id)"
      :preview-src="annexFileUrl(id)"
      :alt="`图片${i + 1}`"
      object-fit="cover"
      :width="size"
      :height="size"
      class="thumb"
    />
    <n-tooltip v-if="hiddenCount > 0" trigger="hover">
      <template #trigger>
        <span class="more" :style="{ width: size + 'px', height: size + 'px' }">
          +{{ hiddenCount }}
        </span>
      </template>
      共 {{ imageIds.length }} 张图片，点击图片可预览
    </n-tooltip>
  </div>
  <span v-else class="empty">—</span>
</template>

<style scoped>
.thumbs {
  display: flex;
  align-items: center;
  gap: 4px;
}
.thumb {
  border-radius: 4px;
  overflow: hidden;
  cursor: pointer;
}
.more {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  background: #f0f2f5;
  color: #606984;
  font-size: 12px;
  flex-shrink: 0;
}
.empty {
  color: #c0c4cc;
}
</style>
