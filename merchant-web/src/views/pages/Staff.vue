<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { fetchStaff } from '@/api/mock'
import type { Staff, Role } from '@/types'

const list = ref<Staff[]>([])
const loading = ref(false)
const modalOpen = ref(false)
const editing = ref<Staff | null>(null)
const form = ref<{ name: string; employeeNo: string; role: Role; phone: string }>({ name: '', employeeNo: '', role: 'cashier', phone: '' })

const ROLE_LABEL: Record<Role, string> = { owner: '店长', cashier: '收银员', kitchen: '后厨' }

onMounted(load)
async function load() {
  loading.value = true
  try {
    list.value = await fetchStaff()
  } catch {
    message.error('加载员工失败')
  } finally {
    loading.value = false
  }
}

const cols = [
  { title: '姓名', dataIndex: 'name', width: 100 },
  { title: '工号', dataIndex: 'employeeNo', width: 100 },
  { title: '角色', dataIndex: 'role', width: 100 },
  { title: '手机号', dataIndex: 'phone', width: 130 },
  { title: '状态', dataIndex: 'status', width: 90 },
  { title: '最近登录', dataIndex: 'lastLogin', width: 140 },
  { title: '操作', key: 'op', width: 170 },
]

function openAdd() {
  editing.value = null
  form.value = { name: '', employeeNo: '', role: 'cashier', phone: '' }
  modalOpen.value = true
}
function openEdit(s: Staff) {
  editing.value = s
  form.value = { name: s.name, employeeNo: s.employeeNo, role: s.role, phone: s.phone }
  modalOpen.value = true
}
function save() {
  if (!form.value.name.trim() || !form.value.employeeNo.trim()) { message.warning('请填写姓名与工号'); return }
  if (editing.value) {
    Object.assign(editing.value, form.value)
    message.success('员工已更新')
  } else {
    list.value.push({ id: Date.now(), ...form.value, status: 'active', lastLogin: '未登录' })
    message.success('员工已新增（初始密码默认 123456，首次登录需修改）')
  }
  modalOpen.value = false
}
function toggle(s: Staff) {
  s.status = s.status === 'active' ? 'disabled' : 'active'
  message.success(s.status === 'active' ? '已启用（可登录）' : '已停用（立即禁止登录）')
}
function resetPwd(s: Staff) {
  Modal.confirm({ title: `重置 ${s.name} 密码`, content: '重置为默认密码 123456，下次登录强制修改。', okText: '重置', onOk() { message.success('已重置') } })
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1>员工与权限</h1>
        <p>三角色权限矩阵见设计 16.2 · 员工 {{ list.length }} 人，在岗 {{ list.filter((s) => s.status === 'active').length }}</p>
      </div>
      <div class="page-head-right">
        <a-button @click="load">↻ 刷新</a-button>
        <a-button type="primary" @click="openAdd">＋ 新增员工</a-button>
      </div>
    </div>

    <a-alert type="info" show-icon style="margin-bottom:16px">
      <template #message>权限规则：<b>店长</b>全部功能；<b>收银员</b>收银工作台 + 报表查看（退款/核销/降级/对账需店长）；<b>后厨</b>仅 KDS 大屏 + 打印任务。店长须至少 1 人，不可停用本人。</template>
    </a-alert>

    <div class="card">
      <div class="card-body">
        <a-table :loading="loading" :data-source="list" :columns="cols" size="middle"
          :row-key="(r: Staff) => r.id" :pagination="false">
          <template #bodyCell="{ column, record }">
            <template v-if="column.dataIndex === 'role'">
              <a-tag :color="(record as Staff).role === 'owner' ? 'gold' : (record as Staff).role === 'cashier' ? 'blue' : 'default'">{{ ROLE_LABEL[(record as Staff).role] }}</a-tag>
            </template>
            <template v-else-if="column.dataIndex === 'status'">
              <a-tag :color="(record as Staff).status === 'active' ? 'green' : 'default'">{{ (record as Staff).status === 'active' ? '在岗' : '已停用' }}</a-tag>
            </template>
            <template v-else-if="column.key === 'op'">
              <a-button size="small" type="link" @click="openEdit(record as Staff)">编辑</a-button>
              <a-button size="small" type="link" @click="resetPwd(record as Staff)">重置密码</a-button>
              <a-button size="small" type="link" :danger="(record as Staff).status === 'active'" @click="toggle(record as Staff)">{{ (record as Staff).status === 'active' ? '停用' : '启用' }}</a-button>
            </template>
          </template>
        </a-table>
      </div>
    </div>

    <a-modal v-model:open="modalOpen" :title="editing ? `编辑员工 · ${editing.name}` : '新增员工'" :footer="null" width="420">
      <div class="form-grid">
        <div class="fg"><label>姓名 *</label><a-input v-model:value="form.name" placeholder="如 李收银" /></div>
        <div class="fg"><label>工号 *</label><a-input v-model:value="form.employeeNo" placeholder="如 1005" /></div>
        <div class="fg"><label>角色</label>
          <a-select v-model:value="form.role" :options="(['owner','cashier','kitchen'] as Role[]).map((r) => ({ label: ROLE_LABEL[r], value: r }))" style="width:100%" />
        </div>
        <div class="fg"><label>手机号</label><a-input v-model:value="form.phone" placeholder="用于接收告警通知" /></div>
      </div>
      <div style="margin-top:20px;text-align:right">
        <a-button style="margin-right:8px" @click="modalOpen = false">取消</a-button>
        <a-button type="primary" @click="save">保存</a-button>
      </div>
    </a-modal>
  </div>
</template>

<style scoped>
.form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 14px }
.fg { display: flex; flex-direction: column; gap: 6px }
.fg label { font-size: 13px; color: var(--t2) }
</style>
