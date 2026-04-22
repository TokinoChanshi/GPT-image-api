<template>
  <div id="evo-app" class="industrial-theme">
    <div class="scanlines"></div>
    <div class="noise"></div>
    <router-view />
  </div>
</template>

<style>
:root {
  --bg-color: #050505;
  --panel-bg: rgba(15, 15, 15, 0.8);
  --accent-primary: #00ffa3;
  --accent-secondary: #f59e0b;
  --text-main: #e0e0e0;
  --text-muted: #666;
  --border-dim: rgba(255, 255, 255, 0.07);
  --font-mono: 'JetBrains Mono', 'Fira Code', monospace;
}

body {
  margin: 0;
  background-color: var(--bg-color);
  color: var(--text-main);
  font-family: var(--font-mono);
  overflow-x: hidden;
}

#evo-app {
  min-height: 100vh;
  position: relative;
}

/* Background Effects */
.scanlines {
  position: fixed;
  top: 0; left: 0; width: 100%; height: 100%;
  background: linear-gradient(
    rgba(18, 16, 16, 0) 50%,
    rgba(0, 0, 0, 0.05) 50%
  );
  background-size: 100% 2px;
  pointer-events: none;
  z-index: 100;
  opacity: 0.3;
}

.noise {
  position: fixed;
  top: 0; left: 0; width: 100%; height: 100%;
  background-image: url("https://grainy-gradients.vercel.app/noise.svg");
  opacity: 0.02;
  pointer-events: none;
  z-index: 101;
}

/* Global Industrial UI Components */
.el-card {
  background: var(--panel-bg) !important;
  border: 1px solid var(--border-dim) !important;
  border-radius: 0 !important;
}

.el-button {
  border-radius: 0 !important;
  text-transform: uppercase;
  font-weight: bold;
  letter-spacing: 1px;
}

.el-tag {
  border-radius: 0 !important;
  font-family: var(--font-mono);
  font-weight: 800;
}

/* Scrollbar styling */
::-webkit-scrollbar { width: 6px; }
::-webkit-scrollbar-track { background: #0a0a0a; }
::-webkit-scrollbar-thumb { background: #222; }
::-webkit-scrollbar-thumb:hover { background: var(--accent-primary); }
</style>
