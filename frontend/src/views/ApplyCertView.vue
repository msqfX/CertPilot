<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { api, type AcmeAccount, type CertOrder, type DNSProvider } from '@/api'
import { useUiStore } from '@/stores/ui'

const ui = useUiStore()

const accounts = ref<AcmeAccount[]>([])
const providers = ref<DNSProvider[]>([])
const orders = ref<CertOrder[]>([])
const submitting = ref(false)

// 表单
const domainsText = ref('')
const accountId = ref<number | null>(null)
const providerId = ref<number | null>(null)
const showNewAccount = ref(false)
const showNewProvider = ref(false)
const newEmail = ref('')
const newCA = ref('letsencrypt')
const newProviderType = ref<'cloudflare' | 'alidns' | 'tencentcloud'>('cloudflare')
const newProviderName = ref('')
const cfToken = ref('')
const aliKey = ref('')
const aliSecret = ref('')
const tcSecretId = ref('')
const tcSecretKey = ref('')

let timer: number | undefined

async function load() {
  const [a, p, o] = await Promise.all([api.acmeAccounts(), api.dnsProviders(), api.certOrders()])
  accounts.value = a.data
  providers.value = p.data
  orders.value = o.data
  if (!accountId.value && accounts.value.length) accountId.value = accounts.value[0].id
  if (!providerId.value && providers.value.length) providerId.value = providers.value[0].id
}

async function addAccount() {
  try {
    await api.createAcmeAccount(newEmail.value, newCA.value)
    showNewAccount.value = false
    newEmail.value = ''
    ui.success('ACME 账户注册成功')
    await load()
  } catch (e: any) {
    ui.error(e.response?.data?.error || '注册失败')
  }
}

async function addProvider() {
  const config: Record<string, string> = {}
  if (newProviderType.value === 'cloudflare') config.auth_token = cfToken.value
  else if (newProviderType.value === 'alidns') {
    config.access_key = aliKey.value
    config.secret_key = aliSecret.value
  } else {
    config.secret_id = tcSecretId.value
    config.secret_key = tcSecretKey.value
  }
  try {
    await api.createDNSProvider({ provider: newProviderType.value, name: newProviderName.value, config })
    showNewProvider.value = false
    newProviderName.value = ''
    cfToken.value = aliKey.value = aliSecret.value = tcSecretId.value = tcSecretKey.value = ''
    ui.success('DNS 服务商已保存，凭证已加密存储')
    await load()
  } catch (e: any) {
    ui.error(e.response?.data?.error || '保存失败')
  }
}

async function removeProvider(p: DNSProvider) {
  const ok = await ui.confirm('删除 DNS 服务商', `确定删除「${p.name || p.provider}」？依赖它的自动续期任务将失败。`)
  if (!ok) return
  await api.deleteDNSProvider(p.id)
  ui.success('已删除')
  await load()
}

async function submitOrder() {
  if (!accountId.value || !providerId.value) {
    ui.error('请先选择 ACME 账户和 DNS 服务商')
    return
  }
  const domains = domainsText.value
    .split(/[\n,]/)
    .map((s) => s.trim())
    .filter(Boolean)
  if (domains.length === 0) {
    ui.error('请填写至少一个域名')
    return
  }
  submitting.value = true
  try {
    await api.createCertOrder({ domains, account_id: accountId.value, provider_id: providerId.value })
    domainsText.value = ''
    ui.success('订单已提交，DNS-01 验证通常需要 1~2 分钟')
    await load()
  } catch (e: any) {
    ui.error(e.response?.data?.error || '提交失败')
  } finally {
    submitting.value = false
  }
}

const providerLabel: Record<string, string> = {
  cloudflare: 'Cloudflare',
  alidns: '阿里云 DNS',
  tencentcloud: '腾讯云 DNS',
}

const statusStyle: Record<string, string> = {
  pending: 'bg-sky-50 text-sky-600',
  success: 'bg-emerald-50 text-emerald-600',
  fail: 'bg-red-50 text-red-600',
}
const statusLabel: Record<string, string> = { pending: '签发中', success: '成功', fail: '失败' }

onMounted(() => {
  load()
  timer = window.setInterval(load, 5000) // 订单状态轮询
})
onUnmounted(() => clearInterval(timer))
</script>

<template>
  <div>
    <h1 class="text-xl font-bold mb-6">申请证书</h1>

    <div class="grid md:grid-cols-2 gap-6">
      <!-- 签发表单 -->
      <div class="bg-white rounded-xl border border-gray-100 shadow-sm p-6">
        <h2 class="text-sm font-semibold text-gray-700 mb-4">签发参数</h2>
        <div class="space-y-4">
          <div>
            <label class="text-xs text-gray-500 block mb-1.5">ACME 账户（Let's Encrypt）</label>
            <div class="flex gap-2">
              <select
                v-model="accountId"
                class="flex-1 rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:border-slate-400"
              >
                <option v-for="a in accounts" :key="a.id" :value="a.id">
                  {{ a.email }}（{{ a.ca === 'letsencrypt' ? '正式' : '测试' }}）
                </option>
              </select>
              <button
                class="shrink-0 rounded-lg border border-gray-200 px-3 text-sm hover:bg-gray-50"
                @click="showNewAccount = true"
              >
                新建
              </button>
            </div>
          </div>

          <div>
            <label class="text-xs text-gray-500 block mb-1.5">DNS 服务商（DNS-01 验证）</label>
            <div class="flex gap-2">
              <select
                v-model="providerId"
                class="flex-1 rounded-lg border border-gray-200 px-3 py-2 text-sm focus:outline-none focus:border-slate-400"
              >
                <option v-for="p in providers" :key="p.id" :value="p.id">
                  {{ p.name || providerLabel[p.provider] }}
                </option>
              </select>
              <button
                class="shrink-0 rounded-lg border border-gray-200 px-3 text-sm hover:bg-gray-50"
                @click="showNewProvider = true"
              >
                新建
              </button>
            </div>
          </div>

          <div>
            <label class="text-xs text-gray-500 block mb-1.5">域名（每行一个，支持泛域名 *.example.com）</label>
            <textarea
              v-model="domainsText"
              rows="4"
              placeholder="example.com&#10;*.example.com"
              class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm font-mono focus:outline-none focus:border-slate-400"
            ></textarea>
          </div>

          <button
            class="w-full rounded-lg bg-slate-900 text-white py-2.5 text-sm font-medium hover:bg-slate-800 disabled:opacity-50"
            :disabled="submitting"
            @click="submitOrder"
          >
            {{ submitting ? '提交中…' : '提交签发' }}
          </button>
          <p class="text-xs text-gray-400 leading-relaxed">
            泛域名必须走 DNS-01 验证，需要 DNS API 权限。签发成功后证书自动入库并开启自动续期。
          </p>
        </div>
      </div>

      <!-- 订单记录 -->
      <div class="bg-white rounded-xl border border-gray-100 shadow-sm overflow-hidden self-start">
        <div class="px-5 py-3 border-b border-gray-50">
          <h2 class="text-sm font-semibold text-gray-700">签发记录</h2>
        </div>
        <table class="w-full text-sm">
          <tbody class="divide-y divide-gray-50">
            <tr v-if="orders.length === 0">
              <td class="px-5 py-10 text-center text-gray-400">暂无签发记录</td>
            </tr>
            <tr v-for="o in orders" :key="o.id">
              <td class="px-5 py-3">
                <div class="font-medium">{{ o.domain_list.join(', ') }}</div>
                <div class="text-xs text-gray-400 mt-0.5">
                  {{ o.provider }} · {{ new Date(o.created_at).toLocaleString('zh-CN') }}
                  <span v-if="o.auto" class="ml-1">（自动续期）</span>
                </div>
                <div v-if="o.error" class="text-xs text-red-500 mt-0.5">{{ o.error }}</div>
              </td>
              <td class="px-5 py-3 text-right">
                <span class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium" :class="statusStyle[o.status]">
                  {{ statusLabel[o.status] }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- 新建 ACME 账户 -->
    <div v-if="showNewAccount" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50" @click.self="showNewAccount = false">
      <div class="bg-white rounded-2xl p-6 w-96 shadow-xl">
        <h2 class="font-semibold mb-4">注册 ACME 账户</h2>
        <div class="space-y-3">
          <input
            v-model="newEmail"
            type="email"
            placeholder="邮箱（用于 Let's Encrypt 到期通知）"
            class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400"
          />
          <select
            v-model="newCA"
            class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400"
          >
            <option value="letsencrypt">Let's Encrypt 正式环境</option>
            <option value="letsencrypt-staging">Let's Encrypt 测试环境（不限流，仅调试用）</option>
          </select>
          <div class="flex justify-end gap-2 pt-2">
            <button class="rounded-lg px-3.5 py-2 text-sm hover:bg-gray-100" @click="showNewAccount = false">取消</button>
            <button class="rounded-lg bg-slate-900 text-white px-3.5 py-2 text-sm hover:bg-slate-800" @click="addAccount">
              注册
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 新建 DNS 服务商 -->
    <div v-if="showNewProvider" class="fixed inset-0 bg-black/40 flex items-center justify-center z-50" @click.self="showNewProvider = false">
      <div class="bg-white rounded-2xl p-6 w-96 shadow-xl">
        <h2 class="font-semibold mb-4">添加 DNS 服务商</h2>
        <div class="space-y-3">
          <select
            v-model="newProviderType"
            class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400"
          >
            <option value="cloudflare">Cloudflare</option>
            <option value="alidns">阿里云 DNS</option>
            <option value="tencentcloud">腾讯云 DNS（DNSPod）</option>
          </select>
          <input
            v-model="newProviderName"
            placeholder="名称（可选）"
            class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400"
          />
          <input
            v-if="newProviderType === 'cloudflare'"
            v-model="cfToken"
            placeholder="API Token（Zone.DNS 编辑权限）"
            class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400"
          />
          <template v-if="newProviderType === 'alidns'">
            <input v-model="aliKey" placeholder="AccessKey ID" class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400" />
            <input v-model="aliSecret" placeholder="AccessKey Secret" class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400" />
          </template>
          <template v-if="newProviderType === 'tencentcloud'">
            <input v-model="tcSecretId" placeholder="SecretId" class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400" />
            <input v-model="tcSecretKey" placeholder="SecretKey" class="w-full rounded-lg border border-gray-200 px-3.5 py-2.5 text-sm focus:outline-none focus:border-slate-400" />
          </template>
          <p class="text-xs text-gray-400">凭证使用 AES-GCM 加密存储，不会明文落库。</p>
          <div class="flex justify-end gap-2 pt-2">
            <button class="rounded-lg px-3.5 py-2 text-sm hover:bg-gray-100" @click="showNewProvider = false">取消</button>
            <button class="rounded-lg bg-slate-900 text-white px-3.5 py-2 text-sm hover:bg-slate-800" @click="addProvider">
              保存
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
