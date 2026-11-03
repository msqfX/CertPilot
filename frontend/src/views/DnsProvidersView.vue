<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, type DNSProvider } from '@/api'
import { useUiStore } from '@/stores/ui'

const ui = useUiStore()
const providers = ref<DNSProvider[]>([])
const showAdd = ref(false)
const type = ref<'cloudflare' | 'alidns' | 'tencentcloud'>('cloudflare')
const name = ref('')
const cfToken = ref('')
const aliKey = ref('')
const aliSecret = ref('')
const tcSecretId = ref('')
const tcSecretKey = ref('')

const providerLabel: Record<string, string> = {
  cloudflare: 'Cloudflare',
  alidns: '阿里云 DNS',
  tencentcloud: '腾讯云 DNS（DNSPod）',
}

async function load() {
  providers.value = (await api.dnsProviders()).data
}

async function add() {
  const config: Record<string, string> = {}
  if (type.value === 'cloudflare') config.auth_token = cfToken.value
  else if (type.value === 'alidns') {
    config.access_key = aliKey.value
    config.secret_key = aliSecret.value
  } else {
    config.secret_id = tcSecretId.value
    config.secret_key = tcSecretKey.value
  }
  try {
    await api.createDNSProvider({ provider: type.value, name: name.value, config })
    showAdd.value = false
    name.value = ''
    cfToken.value = aliKey.value = aliSecret.value = tcSecretId.value = tcSecretKey.value = ''
    ui.success('DNS 服务商已保存，凭证已加密存储')
    await load()
  } catch (e: any) {
    ui.error(e.response?.data?.error || '保存失败')
  }
}

async function remove(p: DNSProvider) {
  const ok = await ui.confirm('删除 DNS 服务商', `确定删除「${p.name || providerLabel[p.provider]}」？依赖它的自动续期任务将失败。`)
  if (!ok) return
  await api.deleteDNSProvider(p.id)
  ui.success('已删除')
  await load()
}

onMounted(load)
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h1 class="text-xl font-bold">DNS 服务商</h1>
        <p class="text-xs text-gray-400 mt-1">用于 ACME DNS-01 验证，签发和自动续期泛域名证书的必要条件</p>
      </div>
      <button
        class="rounded-lg bg-slate-900 text-white px-3.5 py-2 text-sm hover:bg-slate-800"
        @click="showAdd = true"
      >
        添加服务商
      </button>
    </div>

    <div class="bg-white rounded-xl border border-gray-100 shadow-sm overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="text-left text-xs text-gray-500 bg-gray-50/60">
            <th class="px-5 py-3 font-medium">名称</th>
            <th class="px-5 py-3 font-medium">类型</th>
            <th class="px-5 py-3 font-medium text-right">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-50">
          <tr v-if="providers.length === 0">
            <td colspan="3" class="px-5 py-10 text-center text-gray-400">
              暂无 DNS 服务商，添加后即可在「申请证书」中签发泛域名证书
            </td>
          </tr>
          <tr v-for="p in providers" :key="p.id" class="hover:bg-gray-50/50">
            <td class="px-5 py-3 font-medium">{{ p.name || providerLabel[p.provider] }}</td>
            <td class="px-5 py-3 text-gray-600">{{ providerLabel[p.provider] }}</td>
            <td class="px-5 py-3 text-right">
              <button class="text-red-500 hover:underline" @click="remove(p)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="showAdd" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50" @click.self="showAdd = false">
      <div class="bg-white rounded-2xl p-6 w-96 shadow-xl">
        <h2 class="font-semibold mb-4">添加 DNS 服务商</h2>
        <div class="space-y-3">
          <select
            v-model="type"
            class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400"
          >
            <option value="cloudflare">Cloudflare</option>
            <option value="alidns">阿里云 DNS</option>
            <option value="tencentcloud">腾讯云 DNS（DNSPod）</option>
          </select>
          <input
            v-model="name"
            placeholder="名称（可选）"
            class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400"
          />
          <input
            v-if="type === 'cloudflare'"
            v-model="cfToken"
            placeholder="API Token（Zone.DNS 编辑权限）"
            class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400"
          />
          <template v-if="type === 'alidns'">
            <input v-model="aliKey" placeholder="AccessKey ID" class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400" />
            <input v-model="aliSecret" placeholder="AccessKey Secret" class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400" />
          </template>
          <template v-if="type === 'tencentcloud'">
            <input v-model="tcSecretId" placeholder="SecretId" class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400" />
            <input v-model="tcSecretKey" placeholder="SecretKey" class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400" />
          </template>
          <p class="text-xs text-gray-400">凭证使用 AES-GCM 加密存储，不会明文落库。</p>
          <div class="flex justify-end gap-2 pt-2">
            <button class="rounded-lg px-3.5 py-2 text-sm hover:bg-gray-100" @click="showAdd = false">取消</button>
            <button class="rounded-lg bg-slate-900 text-white px-3.5 py-2 text-sm hover:bg-slate-800" @click="add">
              保存
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
