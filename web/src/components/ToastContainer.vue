<script setup lang="ts">
import { useToast } from '../composables/useToast'

const { toasts, remove } = useToast()

const typeStyles = {
  success: 'bg-nord14/20 border-nord14 text-nord14',
  error: 'bg-nord11/20 border-nord11 text-nord11',
  info: 'bg-nord8/20 border-nord8 text-nord8',
}

const typeIcons = {
  success: 'M5 13l4 4L19 7',
  error: 'M6 18L18 6M6 6l12 12',
  info: 'M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z',
}
</script>

<template>
  <div class="fixed bottom-8 right-4 z-50 flex flex-col gap-2 pointer-events-none">
    <TransitionGroup name="toast">
      <div
        v-for="toast in toasts"
        :key="toast.id"
        class="pointer-events-auto flex items-center gap-2 px-4 py-2.5 rounded-lg border shadow-lg text-sm font-mono max-w-sm"
        :class="typeStyles[toast.type]"
        @click="remove(toast.id)"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
          <path stroke-linecap="round" stroke-linejoin="round" :d="typeIcons[toast.type]" />
        </svg>
        <span>{{ toast.message }}</span>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-enter-active {
  transition: all 0.3s ease-out;
}
.toast-leave-active {
  transition: all 0.2s ease-in;
}
.toast-enter-from {
  opacity: 0;
  transform: translateX(30px);
}
.toast-leave-to {
  opacity: 0;
  transform: translateX(30px);
}
</style>
