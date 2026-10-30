# CertPilot

**CertPilot — Your SSL Certificate Control Center.**
让每一个域名的 HTTPS 证书都尽在掌握。

SSL 证书集中管理平台：集中管理多域名证书、自动检测有效期、Let's Encrypt 自动申请与续期、Nginx/Docker/服务器自动部署、到期通知（邮件 / Webhook / 企业微信）。

## 技术栈

- **前端**：Vue 3 · TypeScript · Vite · Tailwind CSS · shadcn-vue · Pinia · ECharts
- **后端**：Go · Gin · GORM · JWT
- **证书引擎**：[Lego](https://goacme.org/)（ACME · DNS-01 / HTTP-01 · 申请 / 续期 / 吊销）
- **存储**：MySQL

## 开发路线

| 阶段 | 主要功能 |
| --- | --- |
| V1.0 | 域名添加、HTTPS 检测、证书解析、到期监控、邮件通知 |
| V1.5 | ACME 自动申请、DNS 验证、自动续期、证书下载 |
| V2.0 | SSH 自动部署、Nginx Reload、多用户、团队管理 |

详细产品定位与技术架构见 [docs/certpilot-product-brief.md](docs/certpilot-product-brief.md)。
