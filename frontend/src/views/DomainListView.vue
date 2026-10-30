<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, type Domain } from '@/api'

const domains = ref<Domain[]>([])
const loading = ref(false)
const showAdd = ref(false)
const showBatch = ref(false)
const newName = ref('')
const newRemark = ref('')
const newCheck = ref(true)
const batchNames = ref('')
const error = ref('')

async function load() {
  loading.value = true
  try {
    domains.value = (await api.domains()).data
  } finally {
    loading.value = false
  }
}

async function add() {
  error.value = ''
  try {
    await api.createDomain({ name: newName.value, remark: newRemark.value, check: newCheck.value })
    showAdd.value = false
    newName.value = ''
    newRemark.value = ''
    await load()
  } catch (e: any) {
    error.value = e.response?.data?.error || '添加失败'
  }
}

async function batch() {
  error.value = ''
  try {
    const res = await api.batchImport(batchNames.value)
    showBatch.value = false
    batchNames.value = ''
    await load()
    alert(`导入完成：新增 ${res.data.created} 个，跳过 ${res.data.skipped} 个`)
  } catch (e: any) {
    error.value = e.response?.data?.error || '导入失败'
  }
}

async function check(d: Domain) {
  d.last_error = ''
  try {
    const res = await api.checkDomain(d.id)
    await load()
    const c = res.data.cert
    alert(`检测成功：${c.issuer} 颁发，${c.days_left} 天后过期`)
  } catch (e: any) {
    alert('检测失败：' + (e.response?.data?.error || e.message))
    await load()
  }
}

async function remove(d: Domain) {
  if (!confirm(`确定删除域名 ${d.name}？`)) return
  await api.deleteDomain(d.id)
  await load()
}

onMounted(load)
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-xl font-bold">域名列表</h1>
      <div class="flex gap-2">
        <button
          class="rounded-lg border border-gray-200 bg-white px-3.5 py-2 text-sm hover:bg-gray-50"
          @click="showBatch = true"
        >
          批量导入
        </button>
        <button
          class="rounded-lg bg-slate-900 text-white px-3.5 py-2 text-sm hover:bg-slate-800"
          @click="showAdd = true"
        >
          添加域名
        </button>
      </div>
    </div>

    <div class="bg-white rounded-xl border border-gray-100 shadow-sm overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="text-left text-xs text-gray-500 bg-gray-50/60">
            <th class="px-5 py-3 font-medium">域名</th>
            <th class="px-5 py-3 font-medium">颁发机构</th>
            <th class="px-5 py-3 font-medium">到期时间</th>
            <th class="px-5 py-3 font-medium">剩余天数</th>
            <th class="px-5 py-3 font-medium">最近检测</th>
            <th class="px-5 py-3 font-medium text-right">操作</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-50">
          <tr v-if="domains.length === 0">
            <td colspan="6" class="px-5 py-10 text-center text-gray-400">
              暂无域名，点击右上角添加
            </td>
          </tr>
          <tr v-for="d in domains" :key="d.id" class="hover:bg-gray-50/50">
            <td class="px-5 py-3">
              <div class="font-medium">{{ d.name }}</div>
              <div v-if="d.last_error" class="text-xs text-red-500">{{ d.last_error }}</div>
            </td>
            <td class="px-5 py-3 text-gray-600">{{ d.cert?.issuer || '—' }}</td>
            <td class="px-5 py-3 text-gray-600">
              {{ d.cert ? new Date(d.cert.not_after).toLocaleDateString('zh-CN') : '—' }}
            </td>
            <td class="px-5 py-3">
              <span
                v-if="d.cert"
                class="font-semibold"
                :class="d.cert.days_left < 0 ? 'text-red-600' : d.cert.days_left <= 30 ? 'text-amber-600' : 'text-emerald-600'"
              >
                {{ d.cert.days_left < 0 ? '已过期' : `${d.cert.days_left} 天` }}
              </span>
              <span v-else class="text-gray-400">—</span>
            </td>
            <td class="px-5 py-3 text-xs text-gray-400">
              {{ d.last_check_at ? new Date(d.last_check_at).toLocaleString('zh-CN') : '未检测' }}
            </td>
            <td class="px-5 py-3 text-right space-x-2">
              <button class="text-sky-600 hover:underline" @click="check(d)">检测</button>
              <button class="text-red-500 hover:underline" @click="remove(d)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- 添加域名 -->
    <div v-if="showAdd" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50" @click.self="showAdd = false">
      <div class="bg-white rounded-2xl p-6 w-96 shadow-xl">
        <h2 class="font-semibold mb-4">添加域名</h2>
        <div class="space-y-3">
          <input
            v-model="newName"
            placeholder="example.com"
            class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400"
          />
          <input
            v-model="newRemark"
            placeholder="备注（可选）"
            class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400"
          />
          <label class="flex items-center gap-2 text-sm text-gray-600">
            <input v-model="newCheck" type="checkbox" class="rounded" />
            添加后立即检测 HTTPS
          </label>
          <p v-if="error" class="text-sm text-red-600">{{ error }}</p>
          <div class="flex justify-end gap-2 pt-2">
            <button class="rounded-lg px-3.5 py-2 text-sm hover:bg-gray-100" @click="showAdd = false">
              取消
            </button>
            <button
              class="rounded-lg bg-slate-900 text-white px-3.5 py-2 text-sm hover:bg-slate-800"
              @click="add"
            >
              添加
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 批量导入 -->
    <div v-if="showBatch" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50" @click.self="showBatch = false">
      <div class="bg-white rounded-2xl p-6 w-96 shadow-xl">
        <h2 class="font-semibold mb-4">批量导入域名</h2>
        <textarea
          v-model="batchNames"
          rows="8"
          placeholder="每行一个域名，如：&#10;example.com&#10;api.example.com"
          class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm font-mono focus:outline-none focus:border-slate-400"
        ></textarea>
        <p v-if="error" class="text-sm text-red-600 mt-2">{{ error }}</p>
        <div class="flex justify-end gap-2 pt-3">
          <button class="rounded-lg px-3.5 py-2 text-sm hover:bg-gray-100" @click="showBatch = false">
            取消
          </button>
          <button
            class="rounded-lg bg-slate-900 text-white px-3.5 py-2 text-sm hover:bg-slate-800"
            @click="batch"
          >
            导入
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
