<template>
  <div class="apikeys">
    <el-card shadow="never" class="table-card">
      <template #header>
        <div class="card-header">
          <span>API Key Management</span>
          <el-button type="primary" :icon="Plus">Generate New Key</el-button>
        </div>
      </template>
      <el-table :data="keys" style="width: 100%" class="custom-table">
        <el-table-column prop="name" label="Name" width="180" />
        <el-table-column prop="key" label="API Key">
          <template #default="{ row }">
            <code>{{ row.key }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="balance" label="Balance" width="120" />
        <el-table-column prop="status" label="Status" width="120">
          <template #default="{ row }">
            <el-switch v-model="row.status" />
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="120">
          <template #default>
            <el-button link type="danger">Delete</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { Plus } from '@element-plus/icons-vue'
import { ref } from 'vue'
const keys = ref([
  { name: 'Default Key', key: 'sk-evo-1234...abcd', balance: 500, status: true },
  { name: 'Production App', key: 'sk-evo-9876...xyz1', balance: 12400, status: true },
])
</script>

<style scoped>
.table-card { background: #0d0d0d; border-color: #222; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
code {
  background: #222;
  padding: 4px 8px;
  border-radius: 4px;
  color: #00ffa3;
  font-family: 'JetBrains Mono', monospace;
}
</style>
