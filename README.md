# 🚀 Evo-ImageAPI: Industrial-Grade DALL-E 3 / IMG2 Singularity Gateway

Evo-ImageAPI is a high-performance, reverse-engineered API gateway designed to transform ChatGPT Web (chatgpt.com) image generation capabilities into a standardized, OpenAI-compatible API. 

It features advanced **Dialectic Detection** to identify and force-trigger the latest **Image 2 (IMG2)** pipelines on both Plus and Ordinary accounts.

## ✨ Core Features

*   **IMG2 Dialectic Engine**: Real-time analysis of `resolved_model_slug` and `file-service://` protocols to guarantee Image 2 quality.
*   **Two-Step Sentinel Protocol**: Implements the latest `/prepare` + `/finalize` sequence with automatic fallback to single-step, matching 2026 OpenAI standards.
*   **HMAC Signed Image Proxy**: Integrated local image proxy (`/v1/p/img`) with HMAC signature verification to bypass OpenAI's 403 hotlinking protection.
*   **Multi-Turn Escalation**: Automatically retries in the same conversation (up to 3 turns) to force the backend into the high-res IMG2 bucket.
*   **Singularity Kernel**: Go-based kernel using `utls` to simulate Chrome 131 fingerprints and HTTP/1.1 downgrade mixins to bypass Cloudflare 429/403.
*   **Industrial Dashboard**: Professional-grade Vue 3 management UI with real-time cluster monitoring.
*   **Massive Account Pool**: Support for thousands of accounts with automatic token revival (RT -> AT).
*   **Zero-Config Storage**: Powered by SQLite (Pure Go implementation) for easy deployment.

## 🛠️ Technical Stack

*   **Backend**: Go 1.21+, Gin, GORM (SQLite), `utls` (TLS Fingerprinting).
*   **Frontend**: Vue 3, TypeScript, Element Plus, Axios, Pinia.
*   **Reverse Engineering**: Custom Sentinel Proof-of-Work (PoW v1) solver.

## 🚀 Quick Start

### 1. Prerequisites
*   Go 1.21+
*   Node.js 20+
*   Working Proxy (Global/US recommended)

### 2. Installation
```bash
# Clone the repository
git clone https://github.com/your-username/evo-image-api.git
cd evo-image-api

# Setup Backend
cd backend
go mod tidy
cp .env.example .env # Update your credentials
go run scripts/setup_init.go # Initialize admin
go run main.go

# Setup Frontend
cd ../frontend
npm install
npm run dev
```

### 3. Usage (OpenAI Compatible)
```bash
curl --location 'http://localhost:8080/v1/images/generations' \
--header 'Authorization: Bearer sk-evo-test-key-001' \
--header 'Content-Type: application/json' \
--data '{
    "model": "gpt-5-3",
    "prompt": "A breathtaking cinematic poster of a cyber-fantasy Beijing",
    "n": 1,
    "size": "1024x1024"
}'
```

## 🕵️ IMG2 Audit (How we identify True IMG2)
Unlike other projects, Evo-ImageAPI looks for:
1.  `resolved_model_slug`: **`gpt-5-3`** or **`i-5-mini-m`**.
2.  `asset_pointer`: **`file-service://`** (High-res原图协议).
3.  `tool_message_count`: **>= 2** (Dual pipeline output).

## 📄 License
MIT License. Created by szqs.site for the open source community.

## ⚠️ Disclaimer
This project is for educational and research purposes only. Use of reverse-engineered APIs is subject to the Terms of Service of the provider.
