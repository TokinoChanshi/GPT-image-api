<template>
  <div class="dashboard-container">
    <!-- Top Stats Row -->
    <el-row :gutter="20" class="stat-cards">
      <el-col :span="6">
        <div class="stat-card total">
          <div class="card-icon"><el-icon><Histogram /></el-icon></div>
          <div class="card-content">
            <div class="label">TOTAL NODES | 总节点</div>
            <div class="value">{{ stats.total }}</div>
          </div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card active">
          <div class="card-icon"><el-icon><Connection /></el-icon></div>
          <div class="card-content">
            <div class="label">ACTIVE KERNELS | 活跃内核</div>
            <div class="value">{{ stats.active }}</div>
          </div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card img2">
          <div class="card-icon"><el-icon><Picture /></el-icon></div>
          <div class="card-content">
            <div class="label">IMG2 ACTIVATED | 激活 IMG2</div>
            <div class="value">{{ stats.img2 }}</div>
          </div>
        </div>
      </el-col>
      <el-col :span="6">
        <div class="stat-card danger">
          <div class="card-icon"><el-icon><Warning /></el-icon></div>
          <div class="card-content">
            <div class="label">OFFLINE | 隔离离线</div>
            <div class="value">{{ stats.banned }}</div>
          </div>
        </div>
      </el-col>
    </el-row>

    <!-- Main Content Area -->
    <el-row :gutter="20" class="main-grid">
      <!-- Event Stream -->
      <el-col :span="16">
        <div class="terminal-window">
          <div class="terminal-header">
            <div class="buttons">
              <span class="dot red"></span>
              <span class="dot yellow"></span>
              <span class="dot green"></span>
            </div>
            <div class="title">LIVE EVENT STREAM | 实时内核事件流</div>
          </div>
          <div class="terminal-body" ref="terminalBody">
            <div v-for="(log, idx) in logs" :key="idx" class="log-line">
              <span class="time">[{{ new Date().toLocaleTimeString() }}]</span>
              <span class="tag">[KERNEL]</span>
              <span class="content">{{ log }}</span>
            </div>
            <div class="log-line active">
              <span class="cursor">█</span>
              <span class="typing">Awaiting next operator command...</span>
            </div>
          </div>
        </div>
      </el-col>

      <!-- System Health -->
      <el-col :span="8">
        <el-card class="health-card" header="SYSTEM STATUS | 系统运行状态">
          <div class="health-item">
            <span class="label">KERNEL VERSION</span>
            <span class="val">Singularity 4.2 Stable</span>
          </div>
          <div class="health-item">
            <span class="label">CPU LOAD | 处理器负载</span>
            <el-progress :percentage="12" status="success" stroke-width="12" />
          </div>
          <div class="health-item">
            <span class="label">MEMORY USAGE | 内存占用</span>
            <el-progress :percentage="34" stroke-width="12" />
          </div>
          <div class="health-item">
            <span class="label">UPSTREAM LATENCY | 上游延迟</span>
            <span class="val">142ms</span>
          </div>
          <div class="health-item">
            <span class="label">IMG2 SUCCESS RATE | 生成成功率</span>
            <span class="val success">98.2%</span>
          </div>
          <div class="health-item">
            <span class="label">SENTINEL STATUS</span>
            <el-tag type="success" size="small">CONNECTED</el-tag>
          </div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import axios from 'axios'
import { Histogram, Connection, Picture, Warning } from '@element-plus/icons-vue'

const stats = ref({ total: 0, active: 0, banned: 0, img2: 0 })
const logs = ref([
  "Singularity Kernel Booting...",
  "Loading TLS Fingerprints: Chrome 131 Ready.",
  "Fetching upstream model mappings...",
  "51 high-res pipelines verified.",
  "Internal Load Balancer: Online.",
  "System fully synchronized with OpenAI-Mesh."
])

const terminalBody = ref<HTMLElement | null>(null)

const fetchStats = async () => {
  try {
    const res = await axios.get('http://localhost:8080/v1/admin/stats')
    stats.value = res.data
  } catch (e) {
    console.error('API Offline')
  }
}

let timer: any = null

onMounted(() => {
  fetchStats()
  timer = setInterval(fetchStats, 5000)
  
  // Fake some logs
  setInterval(() => {
    if (logs.value.length > 50) logs.value.shift()
    const possibleLogs = [
      "Request dispatched to node #E312",
      "Sentinel 2.0 finalized for account: *****@mail.com",
      "IMG2 asset captured: file-service://res_82k",
      "Load balancing active: shifting weight to cluster B",
      "Token refreshed for node #Elite-09"
    ]
    logs.value.push(possibleLogs[Math.floor(Math.random() * possibleLogs.length)])
    if (terminalBody.value) {
      terminalBody.value.scrollTop = terminalBody.value.scrollHeight
    }
  }, 3000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped lang="scss">
.dashboard-container {
  max-width: 1600px;
  margin: 0 auto;
}

.stat-cards {
  margin-bottom: 24px;
}

.stat-card {
  background: #0d0d0d;
  border: 1px solid rgba(255, 255, 255, 0.05);
  padding: 24px;
  display: flex;
  align-items: center;
  gap: 20px;
  position: relative;
  transition: transform 0.2s, border-color 0.2s;
  
  &:hover {
    transform: translateY(-2px);
    border-color: var(--accent-primary);
  }

  .card-icon {
    width: 48px;
    height: 48px;
    background: rgba(255, 255, 255, 0.02);
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 24px;
    color: #888;
  }

  .card-content {
    .label {
      font-size: 11px;
      font-weight: 700;
      color: #666;
      letter-spacing: 1px;
      margin-bottom: 4px;
    }
    .value {
      font-size: 28px;
      font-weight: 900;
      font-family: var(--font-mono);
      color: #fff;
    }
  }

  &.active .card-icon { color: var(--accent-primary); }
  &.img2 .card-icon { color: var(--accent-secondary); }
  &.danger .card-icon { color: #ef4444; }
}

.terminal-window {
  background: #000;
  border: 1px solid #1a1a1a;
  border-radius: 8px;
  overflow: hidden;
  height: 500px;
  display: flex;
  flex-direction: column;

  .terminal-header {
    background: #111;
    padding: 10px 16px;
    display: flex;
    align-items: center;
    border-bottom: 1px solid #1a1a1a;
    .buttons {
      display: flex;
      gap: 6px;
      .dot {
        width: 10px; height: 10px; border-radius: 50%;
        &.red { background: #ff5f56; }
        &.yellow { background: #ffbd2e; }
        &.green { background: #27c93f; }
      }
    }
    .title {
      flex: 1;
      text-align: center;
      font-size: 10px;
      font-weight: 800;
      color: #555;
      letter-spacing: 2px;
    }
  }

  .terminal-body {
    flex: 1;
    padding: 20px;
    font-family: var(--font-mono);
    font-size: 12px;
    overflow-y: auto;
    background: linear-gradient(to bottom, transparent 0%, rgba(0, 255, 163, 0.01) 100%);
    
    .log-line {
      margin-bottom: 6px;
      display: flex;
      gap: 12px;
      .time { color: #333; }
      .tag { color: var(--accent-primary); font-weight: bold; }
      .content { color: #888; }
      
      &.active {
        margin-top: 15px;
        .cursor { color: var(--accent-primary); animation: blink 1s infinite; }
        .typing { color: #aaa; font-style: italic; }
      }
    }
  }
}

.health-card {
  background: #0d0d0d !important;
  border: 1px solid rgba(255, 255, 255, 0.05) !important;
  :deep(.el-card__header) {
    border-bottom: 1px solid rgba(255, 255, 255, 0.05);
    font-weight: 800;
    font-size: 12px;
    letter-spacing: 1px;
    color: #666;
  }
}

.health-item {
  margin-bottom: 20px;
  .label {
    display: block;
    font-size: 10px;
    font-weight: 700;
    color: #444;
    margin-bottom: 8px;
  }
  .val {
    font-size: 20px;
    font-weight: 900;
    font-family: var(--font-mono);
    &.success { color: var(--accent-primary); }
  }
}

@keyframes blink {
  50% { opacity: 0; }
}
</style>
