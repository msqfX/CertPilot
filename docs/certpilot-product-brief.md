# CertPilot 产品定位与技术架构

> CertPilot — Your SSL Certificate Control Center.
> 让每一个域名的 HTTPS 证书都尽在掌握。

## 一、产品定位

CertPilot 定位为 **SSL 证书集中管理平台（SSL Certificate Manager）**，而不是只查看证书到期时间的工具。产品形态是轻量级的 SaaS 风格运维工具，面向个人开发者、运维工程师和中小企业。

核心目标：

- 集中管理多个域名的 SSL 证书
- 自动检测证书有效期和过期时间
- 支持 Let's Encrypt 免费证书自动申请
- 支持证书自动续期
- 支持 Nginx、Docker、服务器等场景的证书部署
- 证书即将过期时发送邮件、Webhook 等通知

> 品牌提示：实际使用「CertPilot」名称前，还需检查域名可注册性和商标情况。

## 二、技术架构

| 层 | 技术栈 |
| --- | --- |
| 前端管理平台 | Vue 3 · TypeScript · Vite · Tailwind CSS · shadcn-vue · Pinia · ECharts |
| Go API 服务 | Go · Gin · GORM · JWT（用户管理、域名管理、证书管理、通知管理） |

证书引擎重点推荐 **Go 的 Lego**：支持 ACME、DNS-01、HTTP-01、证书申请、续期、吊销，并覆盖大量 DNS 服务商，可直接作为证书管理核心引擎。

## 三、功能模块（六大模块）

### 1. 仪表盘

- 域名总数、证书总数
- 正常、即将过期、已过期数量
- 近期证书到期日历
- 证书状态统计

### 2. 域名管理

- 添加、删除域名
- 自动检测 HTTPS
- 查看 DNS 解析
- 检测证书颁发机构
- 批量导入域名

### 3. SSL 证书管理

- 免费申请 Let's Encrypt 证书
- 上传已有证书
- 泛域名证书
- 多域名 SAN 证书
- 证书自动续期
- 证书下载（PEM / KEY）

### 4. DNS 服务商

- Cloudflare
- 阿里云 DNS
- 腾讯云 DNS
- 华为云 DNS
- 自定义 DNS API

支持 DNS-01 自动验证。

### 5. 自动部署

- SSH 连接服务器
- 自动上传证书
- 自动替换 Nginx 证书
- 执行 Nginx reload
- 部署失败自动记录日志

### 6. 通知中心

- 到期前 30 / 15 / 7 / 3 天提醒
- 续期成功通知
- 续期失败告警
- 邮件、Webhook、企业微信

## 四、数据库核心设计

| 表名 | 用途 |
| --- | --- |
| users | 用户信息 |
| domains | 域名信息 |
| certificates | 证书信息 |
| certificate_domains | 证书与域名关联 |
| acme_accounts | ACME 账户 |
| dns_providers | DNS 服务商配置 |
| deploy_servers | 部署服务器 |
| deploy_tasks | 证书部署任务 |
| renewal_tasks | 自动续期任务 |
| notifications | 通知记录 |
| operation_logs | 操作日志 |

**安全要求**：证书私钥、DNS API Token、SSH 密钥不能明文存储。建议使用 AES-GCM 加密，主密钥从环境变量或独立密钥管理服务读取。

## 五、首页 UI 设计方向

- 使用深色科技风首页，管理页面以浅色为主
- 整体参考现代 SaaS 运维平台

侧边栏结构：

```
CertPilot
SSL Certificate Manager
├─ 工作台
│  └─ 仪表盘
├─ 证书管理
│  ├─ 域名列表
│  ├─ SSL 证书
│  └─ 申请证书
├─ 自动化
│  ├─ 自动续期
│  ├─ DNS 服务商
│  └─ 部署管理
├─ 任务记录
└─ 系统管理
   ├─ 通知设置
   ├─ 操作日志
   └─ 系统设置
```

## 六、开发路线

| 阶段 | 主要功能 |
| --- | --- |
| V1.0 | 域名添加、HTTPS 检测、证书解析、到期监控、邮件通知 |
| V1.5 | ACME 自动申请、DNS 验证、自动续期、证书下载 |
| V2.0 | SSH 自动部署、Nginx Reload、多用户、团队管理 |

## 七、关键架构决策

把**域名监控**和**证书签发**设计成两个独立能力：

- **域名监控**：只需检测公网 HTTPS，不需要用户提供服务器权限。
- **证书管理**：需要用户授权 DNS API 或配置 HTTP 验证环境，才能自动申请和续期。

这样即使用户只想监控 100 个域名，也不需要配置任何服务器。
