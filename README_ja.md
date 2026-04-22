# 🚀 Evo-ImageAPI: Industrial-Grade DALL-E 3 / IMG2 Singularity Gateway
# 🚀 Evo-ImageAPI: 産業グレード DALL-E 3 / IMG2 シンギュラリティ画像生成ゲートウェイ

[English](README.md) | [日本語](#japanese-readme)

---

<a name="japanese-readme"></a>

## 🌟 概要

**Evo-ImageAPI** は、ChatGPT Web (chatgpt.com) の画像生成機能を標準化された OpenAI 互換の SaaS API に変換するために設計された、高性能なリバースエンジニアリング API ゲートウェイです。

高度な **Singularity Kernel (シンギュラリティ・カーネル)** を搭載しており、Plus アカウントと無料アカウントの両方で Image 2 (IMG2) パイプラインの安定したトリガー、自動弁証法的検出、およびバイパスを提供します。

### ✨ 主な機能

*   **🔥 IMG2 弁証法エンジン**: `resolved_model_slug` (gpt-5-3) と `file-service://` プロトコルをリアルタイムで分析し、本物の Image 2 高解像度クオリティを保証します。
*   **🛡️ Sentinel 2.0 2段階プロトコル**: 最新の `/prepare` + `/finalize` プロトコルシーケンス (2026年標準) を実装し、単一ステップへの自動フォールバック機能を備えています。
*   **🎭 TLS Singularity (utls)**: Chrome 131 指紋をシミュレートし、HTTP/1.1 ダウンレグレードミキシンを使用して Cloudflare 429/403 および JA4H 検出をバイパスします。
*   **🔒 HMAC 署名付きプロキシ**: HMAC 署名検証付きのローカル画像プロキシ (`/v1/p/img`) を統合し、OpenAI の 403 ホットリンク問題を解決します。
*   **🏗️ 産業用 SaaS ダッシュボード**: リアルタイムのクラスター監視とバッチ資格情報管理を備えた、プロフェッショナルグレードの Vue 3 + TypeScript 管理 UI。
*   **🔋 大規模プール管理**: 自動トークン復活 (RT -> AT) とヘルス監査により、10,000 以上のアカウントをサポートします。

---

## 📸 インターフェースプレビュー

![Dashboard Preview](https://github.com/TokinoChanshi/GPT-image-api/blob/main/docs/preview_dashboard.png?raw=true)
*図1: Singularity Terminal - リアルタイムのクラスターステータスとイベントストリーム。*

![Account Cluster Preview](https://github.com/TokinoChanshi/GPT-image-api/blob/main/docs/preview_cluster.png?raw=true)
*図2: Node Cluster - プロフェッショナルなバッチ管理と IMG2 機能監査。*

---

## 🛠️ 技術スタック

| モジュール | テクノロジー |
| :--- | :--- |
| **Backend** | Go 1.21+ / Gin / GORM (SQLite) |
| **Frontend** | Vue 3 / TS / Element Plus / Pinia |
| **Security** | utls (TLS Fingerprinting) / Sentinel PoW v1 |
| **Transport** | HTTP/1.1 Downgrade Mixin (Bypass CF) |

---

## 🚀 クイックスタート

### 1. 前提条件
*   Go 1.21+
*   Node.js 20+
*   動作するプロキシ (グローバル/US 推奨)

### 2. インストール手順

```bash
# リポジトリをクローン
git clone https://github.com/TokinoChanshi/GPT-image-api.git
cd GPT-image-api

# バックエンドの設定
cd backend
go mod tidy
cp .env.example .env # 資格情報を更新
go run scripts/setup_init.go # 管理者アカウントの初期化
go run main.go

# フロントエンドの設定
cd ../frontend
npm install
npm run dev
```

---

## 🕵️ IMG2 監査標準

従来のプロジェクトとは異なり、Evo-ImageAPI は以下の3つの指標で **真の IMG2** を識別します：

1.  **`resolved_model_slug`**: **`gpt-5-3`** または **`i-5-mini-m`** であることを検証済み。
2.  **`asset_pointer`**: **`file-service://`** (高解像度ソースプロトコル) であることを検証済み。
3.  **ペイロードサイズ**: ファイルサイズが常に **3.0MB - 3.8MB** (オリジナル品質) であること。

---

## 📄 ライセンス
MIT ライセンス。オープンソースコミュニティのために szqs.site によって作成されました。

## ⚠️ 免責事項
このプロジェクトは教育および研究目的のみに使用されます。作者は誤用について一切の責任を負いません。
