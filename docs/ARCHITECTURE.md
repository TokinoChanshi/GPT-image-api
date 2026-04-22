# 🧬 Evo-ImageAPI Architecture: The Singularity Kernel

The **Singularity Kernel** is the high-performance heart of Evo-ImageAPI, designed to bridge the gap between volatile web sessions and stable server-side APIs.

## 1. Dialectic Detection (辩证探测)
Unlike traditional "static" model selection, the Singularity Kernel performs real-time auditing of the upstream response:
- **Phase 1: Probing**: Initiates a conversation with specific system hints (`picture_v2`).
- **Phase 2: Dialectic Analysis**: Parses the `resolved_model_slug` and `asset_pointer` protocol.
- **Phase 3: Decision**: If the upstream routes to a legacy pipeline, the kernel automatically executes a "follow-up" escalation to force the IMG2 bucket.

## 2. Sentinel 2.0 Protocol Stack
The 2026 OpenAI security update introduced a two-step handshake. The kernel implements this natively:
1.  **Prepare**: Handshakes with `/sentinel/chat-requirements/prepare` to receive a challenge.
2.  **Solve**: An optimized Go-based PoW (Proof of Work) solver handles the difficulty locally.
3.  **Finalize**: Submits the solution to `/finalize` to obtain a high-entropy session token.
4.  **Traceability**: Synchronizes `X-Oai-Turn-Trace-Id` across all fragmented requests to simulate a seamless browser session.

## 3. TLS Fingerprint Singularity (utls)
To bypass JA4H (HTTP/2 fingerprinting) and Cloudflare's strict WAF:
- **HelloChrome_131**: Every request uses the `utls` library to mimic Chrome 131's exact extension order and GREASE values.
- **H1 Downgrade Mixin**: Intentionally downgrades to HTTP/1.1 with specific header ordering to bypass the most aggressive H2-based bot detection patterns.

## 4. Secure Asset Delivery (HMAC Proxy)
OpenAI's image CDN (`oaiusercontent.com`) often returns 403 when hotlinked. 
- The kernel generates a one-time-use **HMAC signature** for each image.
- The local endpoint `/v1/p/img` verifies the signature and streams the data through the server's authorized proxy tunnel.
- This ensures 100% image accessibility without exposing account tokens to the client.
