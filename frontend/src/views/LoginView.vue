<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { api } from '@/api'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()

const mode = ref<'login' | 'register'>('login')
const email = ref('')
const password = ref('')
const nickname = ref('')
const error = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  loading.value = true
  try {
    const res =
      mode.value === 'login'
        ? await api.login(email.value, password.value)
        : await api.register(email.value, password.value, nickname.value)
    auth.setSession(res.data.token, res.data.user)
    router.push('/dashboard')
  } catch (e: any) {
    error.value = e.response?.data?.error || '请求失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="h-full flex items-center justify-center bg-slate-900">
    <div class="w-96 bg-white rounded-2xl shadow-xl p-8">
      <div class="text-center mb-8">
        <div class="text-2xl font-bold text-slate-900">CertPilot</div>
        <div class="text-sm text-slate-500 mt-1">SSL Certificate Control Center</div>
      </div>

      <div class="flex gap-4 mb-6 text-sm justify-center">
        <button
          class="pb-1 border-b-2 transition-colors"
          :class="mode === 'login' ? 'border-slate-900 text-slate-900 font-medium' : 'border-transparent text-slate-400'"
          @click="mode = 'login'"
        >
          登录
        </button>
        <button
          class="pb-1 border-b-2 transition-colors"
          :class="mode === 'register' ? 'border-slate-900 text-slate-900 font-medium' : 'border-transparent text-slate-400'"
          @click="mode = 'register'"
        >
          注册
        </button>
      </div>

      <form class="space-y-4" @submit.prevent="submit">
        <input
          v-model="email"
          type="email"
          required
          placeholder="邮箱"
          class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-slate-900/10 focus:border-slate-400"
        />
        <input
          v-model="password"
          type="password"
          required
          minlength="8"
          placeholder="密码（至少 8 位）"
          class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-slate-900/10 focus:border-slate-400"
        />
        <input
          v-if="mode === 'register'"
          v-model="nickname"
          placeholder="昵称（可选）"
          class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-slate-900/10 focus:border-slate-400"
        />
        <p v-if="error" class="text-sm text-red-600">{{ error }}</p>
        <button
          type="submit"
          :disabled="loading"
          class="w-full rounded-lg bg-slate-900 text-white py-2.5 text-sm font-medium hover:bg-slate-800 disabled:opacity-50"
        >
          {{ loading ? '请稍候…' : mode === 'login' ? '登录' : '注册并登录' }}
        </button>
      </form>
    </div>
  </div>
</template>
