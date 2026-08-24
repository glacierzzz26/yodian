<script setup lang="ts">
import { ref } from 'vue'
import { message } from 'ant-design-vue'

interface ImportLog {
  id: number
  time: string
  operator: string
  file: string
  total: number
  ok: number
  fail: number
  status: 'success' | 'partial' | 'failed'
}
const logs = ref<ImportLog[]>([
  { id: 1, time: '昨天 21:00', operator: '张店长', file: '菜品导入模板_v2.xlsx', total: 42, ok: 40, fail: 2, status: 'partial' },
  { id: 2, time: '08-18 15:30', operator: '张店长', file: '新菜品.xlsx', total: 8, ok: 8, fail: 0, status: 'success' },
  { id: 3, time: '08-15 10:20', operator: '李收银', file: '临时导入.csv', total: 12, ok: 0, fail: 12, status: 'failed' },
])
const dragging = ref(false)

function onFile(file: File) {
  const ok = file.name.endsWith('.xlsx') || file.name.endsWith('.csv')
  message.success(ok ? `已接收「${file.name}」，解析中…（mock）` : `「${file.name}」格式不支持，请上传 .xlsx / .csv`)
  if (ok) {
    logs.value.unshift({ id: Date.now(), time: '刚刚', operator: '当前账号', file: file.name, total: 30, ok: 28, fail: 2, status: 'partial' })
  }
  return false // 阻止真实上传
}

function downloadTpl() {
  message.success('模板已生成（mock）：含 分类/菜品名/价格/单位/计费方式/档口/规格 列')
}

const cols = [
  { title: '时间', dataIndex: 'time', width: 120 },
  { title: '操作人', dataIndex: 'operator', width: 100 },
  { title: '文件', dataIndex: 'file', ellipsis: true },
  { title: '总数', dataIndex: 'total', width: 70, align: 'center' as const },
  { title: '成功', dataIndex: 'ok', width: 70, align: 'center' as const },
  { title: '失败', dataIndex: 'fail', width: 70, align: 'center' as const },
  { title: '结果', dataIndex: 'status', width: 90 },
]
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1>批量导入</h1>
        <p>按模板批量导入菜品（.xlsx / .csv）· 导入前自动校验，失败行生成报告</p>
      </div>
    </div>

    <div class="card">
      <div class="card-body">
        <div class="up" :class="{ drag: dragging }" @dragover.prevent="dragging = true" @dragleave="dragging = false" @drop.prevent="dragging = false; $event.dataTransfer?.files.length && onFile($event.dataTransfer.files[0])">
          <a-upload :before-upload="onFile" :show-upload-list="false" accept=".xlsx,.csv">
            <div class="up-in">
              <div class="up-ic">⬆</div>
              <b>点击选择文件，或将文件拖拽到此处</b>
              <p>支持 .xlsx / .csv，单文件 ≤ 5MB，最多 200 行</p>
            </div>
          </a-upload>
        </div>
        <div style="margin-top:14px;display:flex;justify-content:space-between;align-items:center">
          <span style="font-size:13px;color:var(--t3)">模板列：分类 · 菜品名 · 价格 · 单位 · 计费方式(份/人) · 档口 · 规格 · 上架状态</span>
          <a-button @click="downloadTpl">下载导入模板</a-button>
        </div>
      </div>
    </div>

    <div class="card">
      <div class="card-head"><div><h3>导入记录</h3><p>最近导入 {{ logs.length }} 次</p></div></div>
      <div class="card-body">
        <a-table :data-source="logs" :columns="cols" size="middle" :row-key="(r: ImportLog) => r.id" :pagination="false">
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'status'">
              <a-tag :color="(record as ImportLog).status === 'success' ? 'green' : (record as ImportLog).status === 'partial' ? 'gold' : 'red'">
                {{ (record as ImportLog).status === 'success' ? '全部成功' : (record as ImportLog).status === 'partial' ? '部分失败' : '失败' }}
              </a-tag>
            </template>
            <template v-else-if="column.dataIndex === 'fail'">
              <span :style="{ color: (record as ImportLog).fail ? 'var(--er)' : 'inherit' }">{{ (record as ImportLog).fail }}</span>
            </template>
          </template>
        </a-table>
      </div>
    </div>
  </div>
</template>

<style scoped>
.up { border: 1.5px dashed var(--bd); border-radius: 10px; background: var(--bg); transition: .15s }
.up.drag { border-color: var(--brand); background: var(--brand-bg) }
.up-in { padding: 46px 20px; text-align: center; cursor: pointer }
.up-ic { font-size: 34px; color: var(--brand); margin-bottom: 6px }
.up-in b { font-size: 15px }
.up-in p { font-size: 12px; color: var(--t3); margin: 6px 0 0 }
</style>
