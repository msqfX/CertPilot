import http from './http'

export interface CertInfo {
  common_name: string
  dns_names?: string[]
  issuer: string
  not_before: string
  not_after: string
  fingerprint: string
  days_left: number
}

export interface Domain {
  id: number
  name: string
  remark: string
  last_check_at: string | null
  last_cert_id: number | null
  last_error: string
  cert?: CertInfo
}

export interface Certificate {
  id: number
  name: string
  common_name: string
  issuer: string
  not_before: string
  not_after: string
  domains: string[]
  days_left: number
  has_key: boolean
  expired: boolean
  expiring_soon: boolean
  source: string
  auto_renew: boolean
}

export interface AcmeAccount {
  id: number
  email: string
  ca: string
  reg_uri: string
}

export interface DNSProvider {
  id: number
  provider: 'cloudflare' | 'alidns' | 'tencentcloud'
  name: string
}

export interface CertOrder {
  id: number
  domain_list: string[]
  status: 'pending' | 'success' | 'fail'
  error: string
  certificate_id: number | null
  auto: boolean
  created_at: string
  provider?: string
}

export interface Channel {
  id: number
  type: 'email' | 'webhook' | 'wecom'
  name: string
  enabled: boolean
}

export interface NotificationLog {
  id: number
  type: string
  title: string
  content: string
  status: string
  error: string
  created_at: string
}

export interface OperationLog {
  id: number
  action: string
  target: string
  detail: string
  ip: string
  created_at: string
}

export const api = {
  login: (email: string, password: string) => http.post('/auth/login', { email, password }),
  register: (email: string, password: string, nickname: string) =>
    http.post('/auth/register', { email, password, nickname }),

  stats: () => http.get('/dashboard/stats'),
  upcoming: () => http.get('/dashboard/upcoming'),

  domains: () => http.get<Domain[]>('/domains'),
  createDomain: (data: { name: string; remark?: string; check?: boolean }) =>
    http.post('/domains', data),
  batchImport: (names: string) => http.post('/domains/batch', { names }),
  checkDomain: (id: number) => http.post(`/domains/${id}/check`),
  deleteDomain: (id: number) => http.delete(`/domains/${id}`),

  certificates: () => http.get<Certificate[]>('/certificates'),
  uploadCert: (name: string, pem: string) => http.post('/certificates', { name, pem }),
  updateCert: (id: number, data: { auto_renew: boolean }) =>
    http.patch(`/certificates/${id}`, data),
  downloadCert: (id: number, type: 'pem' | 'key') =>
    http.get(`/certificates/${id}/download`, { params: { type }, responseType: 'blob' }),
  deleteCert: (id: number) => http.delete(`/certificates/${id}`),

  acmeAccounts: () => http.get<AcmeAccount[]>('/acme/accounts'),
  createAcmeAccount: (email: string, ca: string) =>
    http.post('/acme/accounts', { email, ca }),

  dnsProviders: () => http.get<DNSProvider[]>('/dns-providers'),
  createDNSProvider: (data: { provider: string; name: string; config: Record<string, string> }) =>
    http.post('/dns-providers', data),
  deleteDNSProvider: (id: number) => http.delete(`/dns-providers/${id}`),

  certOrders: () => http.get<CertOrder[]>('/certificates/orders'),
  createCertOrder: (data: { domains: string[]; account_id: number; provider_id: number }) =>
    http.post('/certificates/orders', data),

  channels: () => http.get<Channel[]>('/notifications/channels'),
  createChannel: (data: { type: string; name: string; config: Record<string, string> }) =>
    http.post('/notifications/channels', data),
  testChannel: (id: number) => http.post(`/notifications/channels/${id}/test`),
  deleteChannel: (id: number) => http.delete(`/notifications/channels/${id}`),
  notificationLogs: () => http.get<NotificationLog[]>('/notifications/logs'),

  operationLogs: () => http.get<OperationLog[]>('/operation-logs'),
}
