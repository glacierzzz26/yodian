<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { fetchTableAreas } from '@/api/mock'
import type { TableArea } from '@/types'

const areas = ref<TableArea[]>([])
const loading = ref(false)
// 桌贴有效期（设计 4.4 QR_TTL_DAYS=365）：mock 固定
const ttlDays = 365

onMounted(load)
async function load() {
  loading.value = true
  try {
    areas.value = await fetchTableAreas()
  } catch {
    message.error('加载桌台失败')
  } finally {
    loading.value = false
  }
}

function regenerate(no: string) {
  Modal.confirm({
    title: `重新生成 ${no} 二维码`,
    content: '新码将使旧桌贴立即失效；请打印后张贴替换。历史会话不受影响。',
    okText: '重新生成',
    onOk() { message.success(`已为 ${no} 生成新码（mock），旧码已失效`) },
  })
}

function batchRegen(areaName: string) {
  Modal.confirm({
    title: `重新生成 ${areaName} 全部桌贴`,
    content: '该区域全部桌贴作废重建，请确认已准备好打印替换。',
    okText: '确认重建',
    onOk() { message.success(`已重建 ${areaName} 全部桌贴（mock）`) },
  })
}

function downloadAll() {
  message.success('已打包全部桌贴 PDF（mock）：按区域分页，含桌号与二维码')
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1>二维码管理</h1>
        <p>每桌一码 · 顾客扫码按区域/桌号落单 · 桌贴有效期 {{ ttlDays }} 天（4.4 QR_TTL_DAYS）</p>
      </div>
      <div class="page-head-right">
        <a-button @click="downloadAll">⬇ 打包全部桌贴</a-button>
        <a-button @click="load">↻ 刷新</a-button>
      </div>
    </div>

    <a-alert type="info" show-icon style="margin-bottom:16px">
      <template #message>桌贴由二维码打印页生成；一码双扫落地（微信/支付宝小程序识别）在阶段 3 顾客端实现。桌贴编号建议与桌台编号一致，张贴于桌面醒目处。</template>
    </a-alert>

    <div v-for="a in areas" :key="a.id" class="zone">
      <div class="zone-head">
        <h3>{{ a.name }}</h3>
        <span class="zl">{{ a.tables.length }} 桌</span>
        <div class="zr">
          <a-button size="small" @click="batchRegen(a.name)">重建本区</a-button>
          <a-button size="small" type="primary" ghost>下载本区 PDF</a-button>
        </div>
      </div>
      <div class="qr-grid">
        <div v-for="t in a.tables" :key="t.id" class="qr-card">
          <div class="qr" :style="{ backgroundImage: `radial-gradient(circle, var(--t1) 1.5px, transparent 1.5px)` }" />
          <div class="qr-info">
            <b>{{ t.tableNo }}</b>
            <small>{{ a.defaultPayMode === 'postpay' ? '后付 · 结账时扫码' : '先付 · 点单即付' }}</small>
          </div>
          <div class="qr-op">
            <a-button size="small" @click="regenerate(t.tableNo)">重新生成</a-button>
            <a-button size="small" type="link">下载单桌</a-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.zone { margin-bottom: 22px }
.zone-head { display: flex; align-items: baseline; gap: 10px; margin-bottom: 12px }
.zone-head h3 { margin: 0; font-size: 15px }
.zone-head .zl { font-size: 12px; color: var(--t3) }
.zone-head .zr { margin-left: auto; display: flex; gap: 8px }
.qr-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(130px, 1fr)); gap: 12px }
.qr-card { border: 1px solid var(--split); border-radius: 8px; padding: 14px; text-align: center; background: var(--card) }
.qr { width: 76px; height: 76px; margin: 0 auto 8px; background-size: 9px 9px; opacity: .9;
  outline: 3px solid var(--card); outline-offset: -3px; border: 1px solid var(--split) }
.qr-info b { display: block; font-size: 15px }
.qr-info small { font-size: 11px; color: var(--t3) }
.qr-op { margin-top: 8px; display: flex; justify-content: center }
</style>
