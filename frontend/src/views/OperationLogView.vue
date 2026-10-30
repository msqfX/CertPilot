<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { api, type OperationLog } from '@/api'

const logs = ref<OperationLog[]>([])

onMounted(async () => {
  logs.value = (await api.operationLogs()).data
})
</script>

<template>
  <div>
    <h1 class="text-xl font-bold mb-6">操作日志</h1>
    <div class="bg-white rounded-xl border border-gray-100 shadow-sm overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="text-left text-xs text-gray-500 bg-gray-50/60">
            <th class="px-5 py-3 font-medium">时间</th>
            <th class="px-5 py-3 font-medium">操作</th>
            <th class="px-5 py-3 font-medium">对象</th>
            <th class="px-5 py-3 font-medium">IP</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-50">
          <tr v-if="logs.length === 0">
            <td colspan="4" class="px-5 py-10 text-center text-gray-400">暂无日志</td>
          </tr>
          <tr v-for="l in logs" :key="l.id" class="hover:bg-gray-50/50">
            <td class="px-5 py-3 text-xs text-gray-400 whitespace-nowrap">
              {{ new Date(l.created_at).toLocaleString('zh-CN') }}
            </td>
            <td class="px-5 py-3">
              <code class="text-xs bg-gray-100 rounded px-1.5 py-0.5">{{ l.action }}</code>
            </td>
            <td class="px-5 py-3 text-gray-600">{{ l.target }}</td>
            <td class="px-5 py-3 text-xs text-gray-400">{{ l.ip }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>
