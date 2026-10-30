<script setup lang="ts">
import { onMounted, ref } from 'vue'
import * as echarts from 'echarts'
import { api, type Certificate } from '@/api'

const stats = ref({ domains: 0, certificates: 0, ok: 0, expiring: 0, expired: 0 })
const upcoming = ref<Certificate[] & any[]>([])
const pieEl = ref<HTMLElement>()

onMounted(async () => {
  const [s, u] = await Promise.all([api.stats(), api.upcoming()])
  stats.value = s.data
  upcoming.value = u.data

  if (pieEl.value) {
    const chart = echarts.init(pieEl.value)
    chart.setOption({
      tooltip: { trigger: 'item' },
      legend: { bottom: 0, itemWidth: 12, itemHeight: 12, textStyle: { fontSize: 12 } },
      series: [
        {
          type: 'pie',
          radius: ['45%', '70%'],
          center: ['50%', '42%'],
          label: { show: false },
          itemStyle: { borderRadius: 6, borderColor: '#fff', borderWidth: 2 },
          data: [
            // 状态语义配色：正常=绿、临期=橙、过期=红
            { name: '正常', value: stats.value.ok, itemStyle: { color: '#22c55e' } },
            { name: '即将过期（≤30 天）', value: stats.value.expiring, itemStyle: { color: '#f59e0b' } },
            { name: '已过期', value: stats.value.expired, itemStyle: { color: '#ef4444' } },
          ].filter((d) => d.value > 0),
        },
      ],
    })
    window.addEventListener('resize', () => chart.resize())
  }
})

function daysColor(days: number) {
  if (days < 0) return 'text-red-600'
  if (days <= 30) return 'text-amber-600'
  return 'text-emerald-600'
}

function fmtDate(s: string) {
  return new Date(s).toLocaleDateString('zh-CN')
}
</script>

<template>
  <div>
    <h1 class="text-xl font-bold mb-6">仪表盘</h1>

    <!-- 统计卡片 -->
    <div class="grid grid-cols-2 md:grid-cols-5 gap-4">
      <div class="bg-white rounded-xl border border-gray-100 p-5 shadow-sm">
        <div class="text-xs text-gray-500">域名总数</div>
        <div class="text-2xl font-bold mt-1">{{ stats.domains }}</div>
      </div>
      <div class="bg-white rounded-xl border border-gray-100 p-5 shadow-sm">
        <div class="text-xs text-gray-500">证书总数</div>
        <div class="text-2xl font-bold mt-1">{{ stats.certificates }}</div>
      </div>
      <div class="bg-white rounded-xl border border-gray-100 p-5 shadow-sm">
        <div class="text-xs text-gray-500">正常</div>
        <div class="text-2xl font-bold mt-1 text-emerald-600">{{ stats.ok }}</div>
      </div>
      <div class="bg-white rounded-xl border border-gray-100 p-5 shadow-sm">
        <div class="text-xs text-gray-500">即将过期</div>
        <div class="text-2xl font-bold mt-1 text-amber-600">{{ stats.expiring }}</div>
      </div>
      <div class="bg-white rounded-xl border border-gray-100 p-5 shadow-sm">
        <div class="text-xs text-gray-500">已过期</div>
        <div class="text-2xl font-bold mt-1 text-red-600">{{ stats.expired }}</div>
      </div>
    </div>

    <div class="grid md:grid-cols-2 gap-6 mt-6">
      <!-- 状态分布 -->
      <div class="bg-white rounded-xl border border-gray-100 p-5 shadow-sm">
        <h2 class="text-sm font-semibold text-gray-700 mb-2">证书状态分布</h2>
        <div ref="pieEl" class="h-64"></div>
      </div>

      <!-- 即将到期 -->
      <div class="bg-white rounded-xl border border-gray-100 p-5 shadow-sm">
        <h2 class="text-sm font-semibold text-gray-700 mb-3">近期到期（60 天内）</h2>
        <div v-if="upcoming.length === 0" class="text-sm text-gray-400 py-8 text-center">
          暂无即将到期的证书
        </div>
        <ul v-else class="divide-y divide-gray-50">
          <li v-for="c in upcoming" :key="c.id" class="flex items-center justify-between py-2.5">
            <div>
              <div class="text-sm font-medium">{{ c.name }}</div>
              <div class="text-xs text-gray-400">{{ fmtDate(c.not_after) }} 到期</div>
            </div>
            <div class="text-sm font-semibold" :class="daysColor(c.days_left)">
              {{ c.days_left < 0 ? '已过期' : `剩 ${c.days_left} 天` }}
            </div>
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>
