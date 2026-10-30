<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const auth = useAuthStore()

const groups = [
  {
    title: '工作台',
    items: [{ label: '仪表盘', to: '/dashboard', icon: '📊' }],
  },
  {
    title: '证书管理',
    items: [
      { label: '域名列表', to: '/domains', icon: '🌐' },
      { label: 'SSL 证书', to: '/certificates', icon: '🔒' },
      { label: '申请证书', to: '', icon: '📝', soon: 'V1.5' },
    ],
  },
  {
    title: '自动化',
    items: [
      { label: '自动续期', to: '', icon: '🔄', soon: 'V1.5' },
      { label: 'DNS 服务商', to: '', icon: '☁️', soon: 'V1.5' },
      { label: '部署管理', to: '', icon: '🚀', soon: 'V2.0' },
    ],
  },
  {
    title: '系统管理',
    items: [
      { label: '通知设置', to: '/notifications', icon: '🔔' },
      { label: '操作日志', to: '/logs', icon: '📋' },
      { label: '系统设置', to: '', icon: '⚙️', soon: 'V2.0' },
    ],
  },
]

function logout() {
  auth.logout()
  router.push('/login')
}
</script>

<template>
  <div class="flex h-full">
    <!-- 侧边栏 -->
    <aside class="w-60 shrink-0 flex flex-col bg-slate-900 text-slate-300">
      <div class="px-5 py-5 border-b border-slate-800">
        <div class="text-white font-bold text-lg tracking-wide">CertPilot</div>
        <div class="text-xs text-slate-500 mt-0.5">SSL Certificate Manager</div>
      </div>
      <nav class="flex-1 overflow-y-auto py-3">
        <div v-for="g in groups" :key="g.title" class="mb-2">
          <div class="px-5 pt-3 pb-1 text-[11px] uppercase tracking-wider text-slate-500">
            {{ g.title }}
          </div>
          <router-link
            v-for="item in g.items"
            :key="item.label"
            :to="item.to"
            class="flex items-center gap-2.5 px-5 py-2 text-sm hover:bg-slate-800 hover:text-white transition-colors"
            :class="{ 'pointer-events-none text-slate-600': !item.to }"
          >
            <span>{{ item.icon }}</span>
            <span>{{ item.label }}</span>
            <span
              v-if="item.soon"
              class="ml-auto text-[10px] px-1.5 py-0.5 rounded bg-slate-800 text-slate-500"
            >
              {{ item.soon }}
            </span>
          </router-link>
        </div>
      </nav>
      <div class="px-5 py-4 border-t border-slate-800 text-sm">
        <div class="text-slate-400 truncate">{{ auth.user?.email }}</div>
        <button class="mt-1 text-xs text-slate-500 hover:text-white" @click="logout">
          退出登录
        </button>
      </div>
    </aside>

    <!-- 主内容 -->
    <main class="flex-1 overflow-y-auto">
      <div class="max-w-6xl mx-auto p-8">
        <router-view />
      </div>
    </main>
  </div>
</template>
