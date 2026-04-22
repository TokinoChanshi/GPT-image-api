<template>
  <div class="accounts-container">
    <!-- Toolbar -->
    <div class="toolbar">
      <div class="left">
        <el-input
          v-model="searchKeyword"
          placeholder="Filter by ID or Email | 搜索识别号或邮箱"
          class="search-input"
          :prefix-icon="Search"
          clearable
        />
        <el-select v-model="statusFilter" placeholder="Status | 状态" class="status-select" clearable>
          <el-option label="Active | 活跃" value="active" />
          <el-option label="Banned | 隔离" value="banned" />
        </el-select>
        <el-radio-group v-model="viewMode" size="small" class="view-radio">
          <el-radio-button label="list">TABLE | 列表</el-radio-button>
          <el-radio-button label="grid">CLUSTER | 阵列</el-radio-button>
        </el-radio-group>
      </div>
      <div class="right">
        <el-button-group>
          <el-button type="primary" :icon="Refresh" @click="fetchAccounts">SYNC | 同步</el-button>
          <el-button type="success" :icon="Upload" @click="importDialogVisible = true">IMPORT | 导入</el-button>
          <el-button type="warning" :icon="Download" @click="exportAccounts">EXPORT | 导出</el-button>
        </el-button-group>
      </div>
    </div>

    <!-- Main Cluster Table -->
    <el-card shadow="never" class="table-card">
      <el-table 
        v-loading="loading"
        :data="filteredAccounts" 
        style="width: 100%" 
        class="evo-table"
      >
        <el-table-column type="selection" width="55" />
        
        <el-table-column prop="Email" label="NODE IDENTIFIER | 节点标识" width="280">
          <template #default="{ row }">
            <div class="account-cell">
              <div class="acc-avatar">{{ row.Email[0].toUpperCase() }}</div>
              <div class="acc-info">
                <span class="email">{{ row.Email }}</span>
                <div class="meta-tags">
                  <el-tag size="small" class="mini-tag">ID:{{ row.ID }}</el-tag>
                  <el-tag size="small" type="info" class="mini-tag">SECURE</el-tag>
                </div>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="FIRMWARE | 固件" width="120">
          <template #default="{ row }">
            <div class="firmware-tag">{{ row.AccountType || 'Plus' }}</div>
          </template>
        </el-table-column>

        <el-table-column label="STATUS | 状态" width="120">
          <template #default="{ row }">
            <div class="status-pill" :class="row.Status.toLowerCase()">
              {{ row.Status.toUpperCase() }}
            </div>
          </template>
        </el-table-column>

        <el-table-column label="CAPABILITY | 核心特征" width="160">
          <template #default="{ row }">
            <div v-if="row.HasIMG2" class="cap-pill img2">
              <span class="pulse"></span>
              IMG2_READY
            </div>
            <div v-else class="cap-pill legacy">
              LEGACY_D3
            </div>
          </template>
        </el-table-column>

        <el-table-column label="LOAD | 负载" width="180">
          <template #default="{ row }">
            <div class="load-cell">
              <div class="bar-bg">
                <div class="bar-fill" :style="{ width: (row.UsageLimit > 0 ? (row.UsageCount / row.UsageLimit * 100) : 0) + '%' }"></div>
              </div>
              <span class="load-text">{{ row.UsageCount }} / {{ row.UsageLimit }} REQ</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="LAST_HEARTBEAT | 最后心跳">
          <template #default="{ row }">
            <span class="time-text">{{ new Date(row.UpdatedAt).toLocaleTimeString() }}</span>
          </template>
        </el-table-column>

        <el-table-column label="OPERATIONS | 操作" width="180" align="right" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" class="op-btn" @click="probeAccount(row)">PROBE</el-button>
            <el-divider direction="vertical" />
            <el-dropdown trigger="click">
              <el-button link type="info" class="op-btn">MORE</el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="editAccount(row)">Edit | 编辑</el-dropdown-item>
                  <el-dropdown-item>Reset | 重置</el-dropdown-item>
                  <el-dropdown-item divided style="color: #ef4444;">Eject | 移除</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          layout="total, sizes, prev, pager, next"
          :total="accounts.length"
          :page-size="15"
          background
        />
      </div>
    </el-card>

    <!-- Import Dialog -->
    <el-dialog
      v-model="importDialogVisible"
      title="MASS CREDENTIAL IMPORT | 批量凭证导入"
      width="600px"
      class="evo-dialog"
    >
      <el-upload
        drag
        action="http://localhost:8080/v1/admin/accounts/import"
        multiple
        :on-success="handleImportSuccess"
        class="evo-uploader"
      >
        <el-icon class="el-icon--upload"><upload-filled /></el-icon>
        <div class="el-upload__text">
          Drag 'n' Drop JSON shards here or <em>click to browse</em>
          <div class="zh">将 JSON 凭证文件拖拽至此或 <em>点击浏览</em></div>
        </div>
        <template #tip>
          <div class="upload-tip">
            Supports: <b>Standard / RT-Only / Session-Only</b> formats.
          </div>
        </template>
      </el-upload>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Search, Refresh, Upload, Download, UploadFilled } from '@element-plus/icons-vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'

const accounts = ref([])
const loading = ref(false)
const searchKeyword = ref('')
const statusFilter = ref('')
const viewMode = ref('list')
const importDialogVisible = ref(false)

const fetchAccounts = async () => {
  loading.value = true
  try {
    const res = await axios.get('http://localhost:8080/v1/admin/accounts')
    accounts.value = res.data
  } catch (e) {
    ElMessage.error('Cluster sync failed | 集群同步失败')
  } finally {
    loading.value = false
  }
}

const filteredAccounts = computed(() => {
  return accounts.value.filter(acc => {
    const matchesSearch = acc.Email.toLowerCase().includes(searchKeyword.value.toLowerCase()) || 
                          acc.ID.toString().includes(searchKeyword.value)
    const matchesStatus = !statusFilter.value || acc.Status === statusFilter.value
    return matchesSearch && matchesStatus
  })
})

const exportAccounts = () => {
  window.open('http://localhost:8080/v1/admin/accounts/export', '_blank')
}

const handleImportSuccess = (res) => {
  ElMessage.success(res.message || 'Import successful | 导入成功')
  importDialogVisible.value = false
  fetchAccounts()
}

const probeAccount = (row) => {
  ElMessage.info(`Probing terminal ${row.Email}...`)
}

const editAccount = (row) => {
  ElMessage.info(`Opening edit terminal for ${row.ID}`)
}

onMounted(fetchAccounts)
</script>

<style scoped lang="scss">
.accounts-container {
  max-width: 1600px;
  margin: 0 auto;
}

.toolbar {
  display: flex;
  justify-content: space-between;
  margin-bottom: 24px;
  .left {
    display: flex;
    align-items: center;
    gap: 12px;
    .search-input { width: 260px; }
    .status-select { width: 130px; }
    .view-radio { margin-left: 10px; }
  }
}

.table-card {
  background: #0d0d0d !important;
  border: 1px solid rgba(255, 255, 255, 0.05) !important;
  padding: 8px;
}

.account-cell {
  display: flex;
  align-items: center;
  gap: 15px;
  .acc-avatar {
    width: 32px; height: 32px;
    background: linear-gradient(135deg, #1a1a1a, #0a0a0a);
    color: var(--accent-primary);
    font-weight: 900;
    border: 1px solid rgba(0, 255, 163, 0.1);
    display: flex; align-items: center; justify-content: center;
    border-radius: 4px;
  }
  .acc-info {
    display: flex;
    flex-direction: column;
    gap: 4px;
    .email { color: #eee; font-size: 13px; font-weight: 600; font-family: var(--font-mono); }
    .meta-tags { display: flex; gap: 4px; }
    .mini-tag { height: 16px; font-size: 9px; padding: 0 4px; background: rgba(255,255,255,0.02); border: none; color: #555; }
  }
}

.firmware-tag {
  font-size: 10px;
  font-weight: 800;
  color: var(--accent-primary);
  background: rgba(0, 255, 163, 0.05);
  padding: 2px 8px;
  display: inline-block;
  border-radius: 2px;
}

.status-pill {
  font-size: 9px;
  font-weight: 900;
  padding: 2px 8px;
  border-radius: 10px;
  display: inline-block;
  &.active { background: #00ffa3; color: #000; box-shadow: 0 0 10px rgba(0, 255, 163, 0.2); }
  &.banned { background: #ef4444; color: #fff; }
}

.cap-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 4px 10px;
  background: #111;
  border: 1px solid #1a1a1a;
  font-size: 10px;
  font-weight: 800;
  color: #444;
  &.img2 {
    color: var(--accent-secondary);
    border-color: rgba(245, 158, 11, 0.2);
    .pulse {
      width: 6px; height: 6px; background: var(--accent-secondary); border-radius: 50%;
      animation: pulse 2s infinite;
    }
  }
}

.load-cell {
  .bar-bg { width: 100%; height: 6px; background: #1a1a1a; border-radius: 3px; overflow: hidden; margin-bottom: 6px; }
  .bar-fill { height: 100%; background: var(--accent-primary); border-radius: 3px; transition: width 0.3s; }
  .load-text { font-size: 10px; color: #555; font-weight: bold; }
}

.time-text { color: #444; font-size: 11px; font-family: var(--font-mono); }

.op-btn {
  font-size: 10px;
  font-weight: 900;
  letter-spacing: 1px;
}

.pagination {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
}

:deep(.evo-table) {
  background-color: transparent !important;
  tr { background-color: transparent !important; }
  .el-table__header th {
    background-color: #080808 !important;
    color: #444;
    border-bottom: 1px solid #111;
  }
  td { border-bottom: 1px solid #111; }
  .el-table__row:hover td { background-color: rgba(255, 255, 255, 0.02) !important; }
}

@keyframes pulse {
  0% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.5; transform: scale(1.2); }
  100% { opacity: 1; transform: scale(1); }
}

:deep(.evo-dialog) {
  background: #0a0a0a !important;
  border: 1px solid #1a1a1a !important;
  .el-dialog__title { color: #fff; font-weight: 900; }
}

.evo-uploader {
  :deep(.el-upload-dragger) {
    background: #050505;
    border: 1px dashed #222;
    &:hover { border-color: var(--accent-primary); }
  }
  .el-upload__text {
    color: #555;
    em { color: var(--accent-primary); font-style: normal; }
    .zh { font-size: 12px; margin-top: 5px; }
  }
}
</style>
