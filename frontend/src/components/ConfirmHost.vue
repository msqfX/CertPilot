<script setup lang="ts">
import { useUiStore } from '@/stores/ui'

const ui = useUiStore()
</script>

<template>
  <div
    v-if="ui.confirmState"
    class="fixed inset-0 bg-black/40 flex items-center justify-center z-[90]"
    @click.self="ui.resolveConfirm(false)"
  >
    <div class="bg-white rounded-2xl p-6 w-80 shadow-xl">
      <h2 class="font-semibold text-base">{{ ui.confirmState.title }}</h2>
      <p class="text-sm text-gray-500 mt-2 leading-relaxed">{{ ui.confirmState.message }}</p>
      <div class="flex justify-end gap-2 mt-5">
        <button
          class="rounded-lg px-3.5 py-2 text-sm hover:bg-gray-100"
          @click="ui.resolveConfirm(false)"
        >
          取消
        </button>
        <button
          class="rounded-lg bg-red-600 text-white px-3.5 py-2 text-sm hover:bg-red-500"
          @click="ui.resolveConfirm(true)"
        >
          确定
        </button>
      </div>
    </div>
  </div>
</template>
