<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, type Certificate } from '@/api'

const certs = ref<Certificate[]>([])
const showUpload = ref(false)
const certName = ref('')
const pemText = ref('')
const error = ref('')
const uploading = ref(false)

async function load() {
  certs.value = (await api.certificates()).data
}

async function upload() {
  error.value = ''
  uploading.value = true
  try {
    await api.uploadCert(certName.value, pemText.value)
    showUpload.value = false
    certName.value = ''
    pemText.value = ''
    await load()
  } catch (e: any) {
    error.value = e.response?.data?.error || '上传失败'
  } finally {
    uploading.value = false
  }
}

async function download(c: Certificate, type: 'pem' | 'key') {
  const res = await api.downloadCert(c.id, type)
  const url = URL.createObjectURL(res.data)
  const a = document.createElement('a')
  a.href = url
  a.download = `${c.common_name}.${type}`
  a.click()
  URL.revokeObjectURL(url)
}

async function remove(c: Certificate) {
  if (!confirm(`确定删除证书 ${c.common_name}？`)) return
  await api.deleteCert(c.id)
  await load()
}

function fmtDate(s: string) {
  return new Date(s).toLocaleDateString('zh-CN')
}

onMounted(load)
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-xl font-bold">SSL 证书</h1>
      <div class="flex gap-2">
        <button
          class="rounded-lg border border-gray-200 bg-white px-3.5 py-2 text-sm text-gray-400 cursor-not-allowed"
          disabled
          title="V1.5 提供 Let's Encrypt 自动申请"
        >
          申请证书（V1.5）
        </button>
        <button
          class="rounded-lg bg-slate-900 text-white px-3.5 py-2 text-sm hover:bg-slate-800"
          @click="showUpload = true"
        >
          上传证书
        </button>
      </div>
    </div>

    <div class="bg-white rounded-xl border border-gray-100 shadow-sm overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="text-left text-xs text-gray-500 bg-gray-50/60">
            <th class="px-5 py-3 font-medium">证书</th>
            <th class="px-5 py-3 font-medium">覆盖域名</th>
            <th class="px-5 py-3 font-medium">颁发机构</th>
            <th class="px-5 py-3 font-medium">有效期</th>
            <th class="px-5 py-3 font-medium">状态</th>
            <th class="px-5 py-3 font-medium text-right">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-50">
          <tr v-if="certs.length === 0">
            <td colspan="6" class="px-5 py-10 text-center text-gray-400">暂无证书</td>
          </tr>
          <tr v-for="c in certs" :key="c.id" class="hover:bg-gray-50/50">
            <td class="px-5 py-3">
              <div class="font-medium">{{ c.name }}</div>
              <div class="text-xs text-gray-400">
                {{ c.source === 'upload' ? '手动上传' : c.source === 'monitor' ? '监控发现' : c.source }}
              </div>
            </td>
            <td class="px-5 py-3 text-gray-600 max-w-56">
              <div class="truncate" :title="c.domains.join(', ')">
                {{ c.domains.join(', ') || c.common_name }}
              </div>
            </td>
            <td class="px-5 py-3 text-gray-600">{{ c.issuer }}</td>
            <td class="px-5 py-3 text-gray-600 text-xs">
              {{ fmtDate(c.not_before) }} ~ {{ fmtDate(c.not_after) }}
            </td>
            <td class="px-5 py-3">
              <span
                class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium"
                :class="
                  c.expired
                    ? 'bg-red-50 text-red-600'
                    : c.expiring_soon
                      ? 'bg-amber-50 text-amber-600'
                      : 'bg-emerald-50 text-emerald-600'
                "
              >
                {{ c.expired ? `已过期 ${-c.days_left} 天` : `剩 ${c.days_left} 天` }}
              </span>
            </td>
            <td class="px-5 py-3 text-right space-x-2 whitespace-nowrap">
              <button class="text-sky-600 hover:underline" @click="download(c, 'pem')">PEM</button>
              <button v-if="c.has_key" class="text-sky-600 hover:underline" @click="download(c, 'key')">
                KEY
              </button>
              <button class="text-red-500 hover:underline" @click="remove(c)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 上传证书 -->
    <div v-if="showUpload" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50" @click.self="showUpload = false">
      <div class="bg-white rounded-2xl p-6 w-[560px] shadow-xl">
        <h2 class="font-semibold mb-4">上传证书（PEM：证书链 + 私钥）</h2>
        <div class="space-y-3">
          <input
            v-model="certName"
            placeholder="证书名称（可选，默认主域名）"
            class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400"
          />
          <textarea
            v-model="pemText"
            rows="12"
            placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----BEGIN PRIVATE KEY-----&#10;..."
            class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-xs font-mono focus:outline-none focus:border-slate-400"
          ></textarea>
          <p class="text-xs text-gray-400">私钥将使用 AES-GCM 加密存储，不会明文落库。</p>
          <p v-if="error" class="text-sm text-red-600">{{ error }}</p>
          <div class="flex justify-end gap-2 pt-1">
            <button class="rounded-lg px-3.5 py-2 text-sm hover:bg-gray-100" @click="showUpload = false">
              取消
            </button>
            <button
              class="rounded-lg bg-slate-900 text-white px-3.5 py-2 text-sm hover:bg-slate-800 disabled:opacity-50"
              :disabled="uploading"
              @click="upload"
            >
              {{ uploading ? '解析中…' : '上传' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
