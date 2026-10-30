<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, type Channel, type NotificationLog } from '@/api'

const channels = ref<Channel[]>([])
const logs = ref<NotificationLog[]>([])
const showAdd = ref(false)
const type = ref<'email' | 'webhook' | 'wecom'>('email')
const name = ref('')
const to = ref('')
const url = ref('')
const error = ref('')

async function load() {
  const [c, l] = await Promise.all([api.channels(), api.notificationLogs()])
  channels.value = c.data
  logs.value = l.data
}

async function add() {
  error.value = ''
  const config: Record<string, string> = {}
  if (type.value === 'email') config.to = to.value
  else config.url = url.value
  try {
    await api.createChannel({ type: type.value, name: name.value, config })
    showAdd.value = false
    name.value = ''
    to.value = ''
    url.value = ''
    await load()
  } catch (e: any) {
    error.value = e.response?.data?.error || '创建失败'
  }
}

async function test(ch: Channel) {
  try {
    await api.testChannel(ch.id)
    alert('测试通知已发送，请查收')
  } catch (e: any) {
    alert('发送失败：' + (e.response?.data?.error || e.message))
  }
  await load()
}

async function remove(ch: Channel) {
  if (!confirm(`确定删除渠道 ${ch.name || ch.type}？`)) return
  await api.deleteChannel(ch.id)
  await load()
}

const typeLabel: Record<string, string> = { email: '邮件', webhook: 'Webhook', wecom: '企业微信' }
const typeLabel2: Record<string, string> = {
  expiry_remind: '到期提醒',
  renew_ok: '续期成功',
  renew_fail: '续期失败',
  test: '测试通知',
}

onMounted(load)
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-xl font-bold">通知设置</h1>
      <button
        class="rounded-lg bg-slate-900 text-white px-3.5 py-2 text-sm hover:bg-slate-800"
        @click="showAdd = true"
      >
        添加渠道
      </button>
    </div>

    <!-- 渠道 -->
    <div class="bg-white rounded-xl border border-gray-100 shadow-sm overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="text-left text-xs text-gray-500 bg-gray-50/60">
            <th class="px-5 py-3 font-medium">渠道</th>
            <th class="px-5 py-3 font-medium">类型</th>
            <th class="px-5 py-3 font-medium">状态</th>
            <th class="px-5 py-3 font-medium text-right">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-50">
          <tr v-if="channels.length === 0">
            <td colspan="4" class="px-5 py-10 text-center text-gray-400">
              暂无通知渠道。证书到期前 30 / 15 / 7 / 3 天会向已启用的渠道发送提醒。
            </td>
          </tr>
          <tr v-for="ch in channels" :key="ch.id" class="hover:bg-gray-50/50">
            <td class="px-5 py-3 font-medium">{{ ch.name || typeLabel[ch.type] }}</td>
            <td class="px-5 py-3 text-gray-600">{{ typeLabel[ch.type] }}</td>
            <td class="px-5 py-3">
              <span
                class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium"
                :class="ch.enabled ? 'bg-emerald-50 text-emerald-600' : 'bg-gray-100 text-gray-500'"
              >
                {{ ch.enabled ? '已启用' : '已停用' }}
              </span>
            </td>
            <td class="px-5 py-3 text-right space-x-2">
              <button class="text-sky-600 hover:underline" @click="test(ch)">测试</button>
              <button class="text-red-500 hover:underline" @click="remove(ch)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 通知记录 -->
    <h2 class="text-sm font-semibold text-gray-700 mt-8 mb-3">通知记录</h2>
    <div class="bg-white rounded-xl border border-gray-100 shadow-sm overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="text-left text-xs text-gray-500 bg-gray-50/60">
            <th class="px-5 py-3 font-medium">时间</th>
            <th class="px-5 py-3 font-medium">类型</th>
            <th class="px-5 py-3 font-medium">标题</th>
            <th class="px-5 py-3 font-medium">状态</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-50">
          <tr v-if="logs.length === 0">
            <td colspan="4" class="px-5 py-10 text-center text-gray-400">暂无记录</td>
          </tr>
          <tr v-for="l in logs" :key="l.id">
            <td class="px-5 py-3 text-xs text-gray-400 whitespace-nowrap">
              {{ new Date(l.created_at).toLocaleString('zh-CN') }}
            </td>
            <td class="px-5 py-3 text-gray-600">{{ typeLabel2[l.type] || l.type }}</td>
            <td class="px-5 py-3">
              {{ l.title }}
              <span v-if="l.error" class="text-xs text-red-500 ml-1">{{ l.error }}</span>
            </td>
            <td class="px-5 py-3">
              <span :class="l.status === 'ok' ? 'text-emerald-600' : 'text-red-600'">
                {{ l.status === 'ok' ? '成功' : '失败' }}
              </span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 添加渠道 -->
    <div v-if="showAdd" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50" @click.self="showAdd = false">
      <div class="bg-white rounded-2xl p-6 w-96 shadow-xl">
        <h2 class="font-semibold mb-4">添加通知渠道</h2>
        <div class="space-y-3">
          <select
            v-model="type"
            class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400"
          >
            <option value="email">邮件</option>
            <option value="webhook">Webhook</option>
            <option value="wecom">企业微信群机器人</option>
          </select>
          <input
            v-model="name"
            placeholder="渠道名称（可选）"
            class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400"
          />
          <input
            v-if="type === 'email'"
            v-model="to"
            placeholder="收件邮箱，多个用英文逗号分隔"
            class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400"
          />
          <input
            v-else
            v-model="url"
            :placeholder="type === 'wecom' ? '企业微信群机器人 Webhook 地址' : 'Webhook URL（POST JSON）'"
            class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400"
          />
          <p class="text-xs text-gray-400">邮件发送使用服务端 SMTP 配置（见后端 .env）。</p>
          <p v-if="error" class="text-sm text-red-600">{{ error }}</p>
          <div class="flex justify-end gap-2 pt-2">
            <button class="rounded-lg px-3.5 py-2 text-sm hover:bg-gray-100" @click="showAdd = false">
              取消
            </button>
            <button
              class="rounded-lg bg-slate-900 text-white px-3.5 py-2 text-sm hover:bg-slate-800"
              @click="add"
            >
              保存
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
