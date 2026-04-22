# 🚀 Evo-ImageAPI: Industrial-Grade DALL-E 3 / IMG2 Singularity Gateway
# 🚀 Evo-ImageAPI: Pasarela de generación de imágenes DALL-E 3 / IMG2 de grado industrial

[English](README.md) | [Español](#spanish-readme)

---

<a name="spanish-readme"></a>

## 🌟 Resumen

**Evo-ImageAPI** es una pasarela API de alto rendimiento diseñada mediante ingeniería inversa para transformar las capacidades de generación de imágenes de ChatGPT Web (chatgpt.com) en una API SaaS estandarizada y compatible con OpenAI.

Cuenta con el avanzado **Singularity Kernel**, que proporciona la activación del flujo de trabajo Image 2 (IMG2) más estable del mundo, con detección dialéctica automática y evasión tanto para cuentas Plus como para cuentas gratuitas.

### ✨ Características Principales

*   **🔥 Motor Dialéctico IMG2**: Análisis en tiempo real de los protocolos `resolved_model_slug` (gpt-5-3) y `file-service://` para garantizar una calidad auténtica de alta resolución Image 2.
*   **🛡️ Sentinel 2.0 de dos pasos**: Implementa la última secuencia de protocolo `/prepare` + `/finalize` (estándar 2026) con retroceso automático a un solo paso.
*   **🎭 TLS Singularity (utls)**: Simula huellas digitales de Chrome 131 con mixins de degradación a HTTP/1.1 para evadir la detección de Cloudflare 429/403 y JA4H.
*   **🔒 Proxy firmado con HMAC**: Proxy de imagen local integrado (`/v1/p/img`) con verificación de firma HMAC para resolver problemas de hotlinking 403 de OpenAI.
*   **🏗️ Panel SaaS Industrial**: Interfaz de usuario de gestión profesional en Vue 3 + TypeScript con monitoreo de clúster en tiempo real y gestión de credenciales por lotes.
*   **🔋 Gestión de Pool Masivo**: Soporte para más de 10,000 cuentas con reactivación automática de tokens (RT -> AT) y auditoría de estado.

---

## 📸 Vista previa de la interfaz

![Dashboard Preview](https://github.com/TokinoChanshi/GPT-image-api/blob/main/docs/preview_dashboard.png?raw=true)
*Figura 1: Terminal Singularity - Estado del clúster y flujo de eventos en tiempo real.*

![Account Cluster Preview](https://github.com/TokinoChanshi/GPT-image-api/blob/main/docs/preview_cluster.png?raw=true)
*Figura 2: Clúster de nodos - Gestión profesional por lotes y auditoría de capacidad IMG2.*

---

## 🛠️ Stack Tecnológico

| Módulo | Tecnología |
| :--- | :--- |
| **Backend** | Go 1.21+ / Gin / GORM (SQLite) |
| **Frontend** | Vue 3 / TS / Element Plus / Pinia |
| **Seguridad** | utls (TLS Fingerprinting) / Sentinel PoW v1 |
| **Transporte** | HTTP/1.1 Downgrade Mixin (Bypass CF) |

---

## 🚀 Inicio Rápido

### 1. Requisitos previos
*   Go 1.21+
*   Node.js 20+
*   Proxy funcional (se recomienda Global/US)

### 2. Instrucciones de instalación

```bash
# Clonar el repositorio
git clone https://github.com/TokinoChanshi/GPT-image-api.git
cd GPT-image-api

# Configuración del Backend
cd backend
go mod tidy
cp .env.example .env # Actualice sus credenciales
go run scripts/setup_init.go # Inicializar cuenta de administrador
go run main.go

# Configuración del Frontend
cd ../frontend
npm install
npm run dev
```

---

## 🕵️ Estándar de auditoría IMG2

A diferencia de los proyectos heredados, Evo-ImageAPI identifica **IMG2 auténtico** mediante tres indicadores:

1.  **`resolved_model_slug`**: Verificado como **`gpt-5-3`** o **`i-5-mini-m`**.
2.  **`asset_pointer`**: Verificado como **`file-service://`** (protocolo de origen de alta resolución).
3.  **Tamaño del Payload**: Tamaños de archivo consistentes entre **3.0MB y 3.8MB** (calidad original).

---

## 📄 Licencia
Licencia MIT. Creado por szqs.site para la comunidad de código abierto.

## ⚠️ Descargo de responsabilidad
Este proyecto es solo para fines educativos y de investigación. Los autores no asumen responsabilidad por cualquier mal uso.
