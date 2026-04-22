<template>
  <el-container class="layout-root">
    <el-aside :width="collapsed ? '64px' : '240px'" class="sidebar">
      <div class="logo">
        <div class="logo-icon">Σ</div>
        <span v-if="!collapsed" class="logo-text">EVO-IMAGE</span>
      </div>
      <el-menu
        :default-active="activePath"
        :collapse="collapsed"
        background-color="transparent"
        text-color="#a3a3a3"
        active-text-color="#00ffa3"
        class="side-menu"
        router
      >
        <el-menu-item index="/">
          <el-icon><Odometer /></el-icon>
          <template #title>CORE TERMINAL | 核心终端</template>
        </el-menu-item>
        <el-menu-item index="/accounts">
          <el-icon><Cpu /></el-icon>
          <template #title>NODE CLUSTER | 节点集群</template>
        </el-menu-item>
        <el-menu-item index="/apikeys">
          <el-icon><Lock /></el-icon>
          <template #title>ACCESS KEYS | 访问令牌</template>
        </el-menu-item>
      </el-menu>
    </el-aside>

    <el-container class="main-container">
      <el-header class="topbar">
        <div class="left">
          <el-button link @click="collapsed = !collapsed" class="fold-btn">
            <el-icon :size="20">
              <component :is="collapsed ? 'Expand' : 'Fold'" />
            </el-icon>
          </el-button>
          <span class="crumb">{{ currentRouteName }}</span>
        </div>
        <div class="right">
          <el-dropdown trigger="click">
            <span class="user-entry">
              <el-avatar :size="30" class="operator-avatar">OP</el-avatar>
              <span class="nick">OPERATOR #SZQS</span>
              <el-icon><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item>
                  <el-icon><User /></el-icon> 个人中心 | Profile
                </el-dropdown-item>
                <el-dropdown-item divided>
                  <el-icon><SwitchButton /></el-icon> 退出登录 | Logout
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <el-main class="content-viewport">
        <router-view v-slot="{ Component }">
          <transition name="fade-transform" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </el-main>

      <el-footer class="footer">
        <div class="brand-line">
          <b>EVO-IMAGE-API</b>
          <span class="sep">|</span>
          <span>SINGULARITY KERNEL V4.2</span>
          <span class="sep">|</span>
          <span class="status">SYNCED | 已同步</span>
        </div>
      </el-footer>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { Odometer, Cpu, Lock, Expand, Fold, ArrowDown, User, SwitchButton } from '@element-plus/icons-vue'

const route = useRoute()
const collapsed = ref(false)
const activePath = computed(() => route.path)
const currentRouteName = computed(() => route.name)
</script>

<style scoped lang="scss">
.layout-root {
  height: 100vh;
  background: #050505;
}

.sidebar {
  background: #0a0a0a;
  border-right: 1px solid rgba(255, 255, 255, 0.05);
  transition: width 0.3s;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.logo {
  height: 64px;
  display: flex;
  align-items: center;
  padding: 0 20px;
  gap: 12px;
  .logo-icon {
    font-size: 24px;
    font-weight: 900;
    color: var(--accent-primary);
    text-shadow: 0 0 10px var(--accent-primary);
  }
  .logo-text {
    font-weight: 900;
    letter-spacing: 2px;
    font-size: 16px;
    color: #fff;
  }
}

.side-menu {
  border-right: none;
  flex: 1;
  :deep(.el-menu-item) {
    height: 50px;
    line-height: 50px;
    margin: 4px 10px;
    border-radius: 4px;
    &:hover {
      background: rgba(255, 255, 255, 0.03) !important;
    }
    &.is-active {
      background: rgba(0, 255, 163, 0.05) !important;
      color: var(--accent-primary) !important;
    }
  }
}

.main-container {
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.topbar {
  background: rgba(10, 10, 10, 0.8);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px;
  height: 56px;
  .left {
    display: flex;
    align-items: center;
    gap: 15px;
    .fold-btn {
      color: #888;
      &:hover { color: #fff; }
    }
    .crumb {
      font-weight: 700;
      letter-spacing: 2px;
      text-transform: uppercase;
      font-size: 14px;
      color: #eee;
    }
  }
  .user-entry {
    display: flex;
    align-items: center;
    gap: 10px;
    cursor: pointer;
    .nick {
      font-size: 13px;
      font-weight: 600;
      color: #aaa;
    }
    .operator-avatar {
      background: linear-gradient(135deg, #00ffa3, #00d1ff);
      color: #000;
      font-weight: 800;
      font-size: 12px;
    }
  }
}

.content-viewport {
  padding: 24px;
  background: #050505;
  overflow-y: auto;
}

.footer {
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #080808;
  border-top: 1px solid rgba(255, 255, 255, 0.03);
  font-size: 11px;
  color: #444;
  .brand-line {
    display: flex;
    align-items: center;
    gap: 10px;
    b { color: var(--accent-primary); letter-spacing: 1px; }
    .sep { color: #222; }
    .status { color: #008f5a; }
  }
}

/* Transitions */
.fade-transform-enter-active,
.fade-transform-leave-active {
  transition: all 0.3s;
}
.fade-transform-enter-from {
  opacity: 0;
  transform: translateX(-10px);
}
.fade-transform-leave-to {
  opacity: 0;
  transform: translateX(10px);
}
</style>
