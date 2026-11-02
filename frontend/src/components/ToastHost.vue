<script setup lang="ts">
import { useUiStore } from '@/stores/ui'

const ui = useUiStore()

const styles: Record<string, string> = {
  success: 'bg-emerald-600',
  error: 'bg-red-600',
  info: 'bg-slate-800',
}

const icons: Record<string, string> = {
  success: '✓',
  error: '✕',
  info: 'ℹ',
}
</script>

<template>
  <div class="fixed top-4 right-4 z-[100] space-y-2 pointer-events-none">
    <TransitionGroup
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0 translate-x-4"
      enter-to-class="opacity-100 translate-x-0"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-for="t in ui.toasts"
        :key="t.id"
        class="flex items-center gap-2.5 rounded-lg px-4 py-2.5 text-sm text-white shadow-lg max-w-sm"
        :class="styles[t.type]"
      >
        <span class="shrink-0 w-4 h-4 rounded-full bg-white/20 flex items-center justify-center text-[10px]">
          {{ icons[t.type] }}
        </span>
        <span>{{ t.message }}</span>
      </div>
    </TransitionGroup>
  </div>
</template>
