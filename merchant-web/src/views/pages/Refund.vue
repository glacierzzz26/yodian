<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { fetchRefunds } from '@/api/mock'
import { useAuthStore } from '@/stores/auth'
import type { RefundReq } from '@/types'

const auth = useAuthStore()
const list = ref<RefundReq[]>([])
const status = ref('全部')
const loading = ref(false)
const current = ref<RefundReq | null>(null)

const ST_META: Record<string, { label: string; color: string }> = {
  pending: { label: '待审批', color: 'gold' },
  approved: { label: '已批准（退款中）', color: 'blue' },
  done: { label: '已退款', color: 'green' },
  rejected: { label: '已驳回', color: 'default' },
}
const PAY_LABEL: Record<string, string> = { wechat: '微信', alipay: '支付宝', cash: '现金', pos: 'POS' }

const cols = [
  { title: '退款单号', dataIndex: 'orderNo', width: 150 },
  { title: '来源桌台', dataIndex: 'tableNo', width: 90 },
  { title: '金额', dataIndex: 'amount', width: 100, align: 'right' as const },
  { title: '原因', dataIndex: 'reason', ellipsis: true },
  { title: '渠道', dataIndex: 'channel', width: 80 },
  { title: '状态', dataIndex: 'status', width: 120 },
  { title: '申请人', dataIndex: 'requestedBy', width: 90 },
  { title: '操作', key: 'op', width: 130 },
]

onMounted(load)
async function load() {
  loading.value = true
  try {
    list.value = await fetchRefunds()
  } catch {
    message.error('加载退款单失败')
  } finally {
    loading.value = false
  }
}

const filtered = computed(() => (status.value === '全部' ? list.value : list.value.filter((r) => r.status === status.value)))
const pending = computed(() => list.value.filter((r) => r.status === 'pending').length)

function decide(r: RefundReq, approve: boolean) {
  Modal.confirm({
    title: approve ? `批准退款 ¥${r.amount.toFixed(2)}` : '驳回退款申请',
    content: approve
      ? `原路退回 ${PAY_LABEL[r.channel]}。退款动作留痕，计入当日冲减；店主双人复核需二次确认。`
      : `驳回后申请单关闭，如需退款请重新提交。`,
    okText: approve ? '确认退款' : '驳回',
    okButtonProps: { danger: !approve },
    onOk() {
      r.status = approve ? 'approved' : 'rejected'
      r.handledBy = auth.operator?.name
      r.handledAt = '刚刚'
      current.value = null
      message.success(approve ? '已批准，原路退款处理中（模拟）' : '已驳回')
    },
  })
}

function submitNew() {
  Modal.confirm({
    title: '提交退款申请',
    content: '新退款需填写订单号与金额，提交后由店长审批。此入口为演示，详细表单见阶段 4 联调。',
    okText: '去提交',
    onOk() { message.info('请从订单详情发起退款（演示）') },
  })
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1>退款与核销</h1>
        <p>待审批 {{ pending }} 笔 · 退款动作留痕并冲减营业额（六口径对账）</p>
      </div>
      <div class="page-head-right">
        <a-button @click="submitNew">＋ 提交退款</a-button>
        <a-button @click="load">↻ 刷新</a-button>
      </div>
    </div>

    <a-alert v-if="auth.role === 'owner'" type="info" show-icon style="margin-bottom:16px">
      <template #message>店长审批遵循<b>双人复核</b>原则（申请人 ≠ 审批人）；退款原路退回，现金单需当面退款并留签收。</template>
    </a-alert>
    <a-alert v-else type="warning" show-icon style="margin-bottom:16px">
      <template #message>收银员可发起退款申请，<b>审批与核销需店长操作</b>；异常单请当面转交店长。</template>
    </a-alert>

    <div class="card">
      <div class="card-body">
        <a-table :loading="loading" :data-source="filtered" :columns="cols" size="middle"
          :row-key="(r: RefundReq) => r.id" :pagination="false">
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'amount'"><b class="num" style="color:var(--er)">-¥{{ (record as RefundReq).amount.toFixed(2) }}</b></template>
            <template v-else-if="column.dataIndex === 'channel'"><a-tag>{{ PAY_LABEL[(record as RefundReq).channel] || (record as RefundReq).channel }}</a-tag></template>
            <template v-else-if="column.dataIndex === 'status'">
              <a-tag :color="ST_META[(record as RefundReq).status].color">{{ ST_META[(record as RefundReq).status].label }}</a-tag>
            </template>
            <template v-else-if="column.key === 'op'">
              <template v-if="(record as RefundReq).status === 'pending'">
                <template v-if="auth.role === 'owner'">
                  <a-button size="small" type="link" @click="decide(record as RefundReq, true)">批准</a-button>
                  <a-button size="small" type="link" danger @click="decide(record as RefundReq, false)">驳回</a-button>
                </template>
                <span v-else style="color:var(--t3);font-size:12px">待店长审批</span>
              </template>
              <span v-else style="color:var(--t3);font-size:12px">
                {{ (record as RefundReq).handledBy || '—' }}{{ (record as RefundReq).handledAt ? ` · ${(record as RefundReq).handledAt}` : '' }}
              </span>
            </template>
          </template>
        </a-table>
      </div>
    </div>
  </div>
</template>
