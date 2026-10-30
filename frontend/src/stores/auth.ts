import { defineStore } from 'pinia'

const TOKEN_KEY = 'certpilot_token'
const USER_KEY = 'certpilot_user'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem(TOKEN_KEY) || '',
    user: JSON.parse(localStorage.getItem(USER_KEY) || 'null') as
      | { id: number; email: string; nickname?: string }
      | null,
  }),
  getters: {
    isLoggedIn: (s) => !!s.token,
  },
  actions: {
    setSession(token: string, user: { id: number; email: string; nickname?: string }) {
      this.token = token
      this.user = user
      localStorage.setItem(TOKEN_KEY, token)
      localStorage.setItem(USER_KEY, JSON.stringify(user))
    },
    logout() {
      this.token = ''
      this.user = null
      localStorage.removeItem(TOKEN_KEY)
      localStorage.removeItem(USER_KEY)
    },
  },
})
