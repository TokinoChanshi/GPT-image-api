# 🚀 Evo-ImageAPI: Industrial-Grade DALL-E 3 / IMG2 Singularity Gateway
# 🚀 Evo-ImageAPI：工业级 DALL-E 3 / IMG2 奇点生图网关

[English](#english-readme) | [中文说明](#chinese-readme)

---

<a name="english-readme"></a>

## 🌟 Overview

**Evo-ImageAPI** is a high-performance, reverse-engineered API gateway designed to transform ChatGPT Web (chatgpt.com) image generation capabilities into a standardized, OpenAI-compatible SaaS API. 

It features the advanced **Singularity Kernel**, providing the world's most stable Image 2 (IMG2) pipeline triggering, with automatic dialectic detection and bypass for both Plus and Ordinary accounts.

### ✨ Key Features

*   **🔥 IMG2 Dialectic Engine**: Real-time analysis of `resolved_model_slug` (gpt-5-3) and `file-service://` protocols to guarantee authentic Image 2 high-res quality.
*   **🛡️ Two-Step Sentinel 2.0**: Implements the latest `/prepare` + `/finalize` protocol sequence (2026 standard) with automatic fallback to single-step.
*   **🎭 TLS Singularity (utls)**: Simulates Chrome 131 fingerprints with HTTP/1.1 downgrade mixins to bypass Cloudflare 429/403 and JA4H detection.
*   **🔒 HMAC Signed Proxy**: Integrated local image proxy (`/v1/p/img`) with HMAC signature verification to solve OpenAI's 403 hotlinking issues.
*   **🏗️ Industrial SaaS Dashboard**: A professional-grade Vue 3 + TypeScript management UI with real-time cluster monitoring and batch credential management.
*   **🔋 Massive Pool Management**: Support for 10,000+ accounts with automatic token revival (RT -> AT) and health auditing.

---

<a name="chinese-readme"></a>

## 🌟 项目简介

**Evo-ImageAPI** 是一款专为生产环境设计的逆向 API 网关，将 ChatGPT Web 端的高端生图能力转化为标准化、OpenAI 兼容的商业级接口。

项目集成了独家的 **“奇点内核 (Singularity Kernel)”**，目前是市面上极少数能稳定触发 **Image 2 (IMG2)** 高清管线的开源实现，支持 Plus 与 普通账号的全量适配。

### ✨ 核心特性

*   **🔥 IMG2 辩证引擎**：实时审计 `resolved_model_slug` (gpt-5-3) 与 `file-service://` 协议，确保每一张图都是真 IMG2 高清直出。
*   **🛡️ Sentinel 2.0 协议栈**：完美支持 2026 最新两步式认证 (`/prepare` + `/finalize`)，集成 `X-Oai-Turn-Trace-Id` 全链路追踪。
*   **🎭 奇点混淆 (utls)**：基于 Chrome 131 指纹模拟，强制 HTTP/1.1 降级混淆，百分之百绕过 Cloudflare 的 JA4H 检测。
*   **🔒 HMAC 签名代理**：内置 `/v1/p/img` 安全代理，带 HMAC 签名校验，彻底解决 OpenAI 图片 CDN 的 403 防盗链问题。
*   **🏗️ 赛博工业风面板**：Vue 3 + TS 打造的商业级管理后台，实时监控 51+ 精英节点的负载、成功率与认证状态。
*   **🔋 万级号池管控**：支持从 JSON/DB 批量导入万级账号，具备自动洗号、令牌复活（RT -> AT）与高频健康探测能力。

---

## 📸 Interface Preview | 界面预览

![Dashboard Preview](https://github.com/TokinoChanshi/GPT-image-api/blob/main/docs/preview_dashboard.png?raw=true)
*Figure 1: Singularity Terminal - Real-time cluster status and event stream.*

![Account Cluster Preview](https://github.com/TokinoChanshi/GPT-image-api/blob/main/docs/preview_cluster.png?raw=true)
*Figure 2: Node Cluster - Professional batch management and IMG2 capability auditing.*

---

## 🛠️ Technical Stack | 技术栈

| Module 模块 | Technology 技术 |
| :--- | :--- |
| **Backend 后端** | Go 1.21+ / Gin / GORM (SQLite) |
| **Frontend 前端** | Vue 3 / TS / Element Plus / Pinia |
| **Security 协议** | utls (TLS Fingerprinting) / Sentinel PoW v1 |
| **Transport 传输** | HTTP/1.1 Downgrade Mixin (Bypass CF) |

---

## 🚀 Quick Start | 快速启动

### 1. Prerequisites | 环境要求
*   Go 1.21+
*   Node.js 20+
*   Working Proxy (Global/US recommended) | 全局梯子 (建议美区)

### 2. Installation | 安装步骤

```bash
# Clone the repository
git clone https://github.com/TokinoChanshi/GPT-image-api.git
cd GPT-image-api

# Setup Backend 后端设置
cd backend
go mod tidy
cp .env.example .env # Update your credentials 配置环境变量
go run scripts/setup_init.go # Initialize admin 初始化管理账号
go run main.go

# Setup Frontend 前端设置
cd ../frontend
npm install
npm run dev
```

---

## 🕵️ IMG2 Audit Standard | IMG2 审计标准

Unlike legacy projects, Evo-ImageAPI identifies **True IMG2** via:
不同于旧版项目，我们通过以下三个核心指标判定 **真·IMG2**：

1.  **`resolved_model_slug`**: Verified as **`gpt-5-3`** or **`i-5-mini-m`**.
2.  **`asset_pointer`**: Verified as **`file-service://`** (High-res source protocol).
3.  **Payload Size**: File sizes consistently between **3.0MB - 3.8MB** (Original quality).

---

## 📄 License | 授权
MIT License. Created by szqs.site for the open-source community.

## ⚠️ Disclaimer | 免责声明
This project is for educational and research purposes only. The authors do not assume responsibility for any misuse.
本项目仅供学术交流与研究使用，严禁用于非法用途。使用本软件即代表您已知晓相关风险。
