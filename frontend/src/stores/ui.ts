import { defineStore } from 'pinia'

export interface Toast {
  id: number
  type: 'success' | 'error' | 'info'
  message: string
}

interface ConfirmState {
  title: string
  message: string
  resolve: (ok: boolean) => void
}

export const useUiStore = defineStore('ui', {
  state: () => ({
    toasts: [] as Toast[],
    confirmState: null as ConfirmState | null,
  }),
  actions: {
    toast(type: Toast['type'], message: string) {
      const id = Date.now() + Math.random()
      this.toasts.push({ id, type, message })
      setTimeout(() => {
        this.toasts = this.toasts.filter((t) => t.id !== id)
      }, 3500)
    },
    success(message: string) {
      this.toast('success', message)
    },
    error(message: string) {
      this.toast('error', message)
    },
    confirm(title: string, message: string): Promise<boolean> {
      return new Promise((resolve) => {
        this.confirmState = { title, message, resolve }
      })
    },
    resolveConfirm(ok: boolean) {
      this.confirmState?.resolve(ok)
      this.confirmState = null
    },
  },
})
