<script setup lang="ts">
import { reactive, ref } from 'vue'
import { message } from 'ant-design-vue'
import dayjs from 'dayjs'

// 门店信息
const store = reactive({
  name: '悦点',
  address: '示例路 88 号',
  phone: '0571-88888888',
})
const lunch = ref<[dayjs.Dayjs, dayjs.Dayjs]>([dayjs('11:00', 'HH:mm'), dayjs('14:00', 'HH:mm')])
const dinner = ref<[dayjs.Dayjs, dayjs.Dayjs]>([dayjs('17:00', 'HH:mm'), dayjs('21:30', 'HH:mm')])

// 支付渠道（密钥永不入库：12.6 凭据处理）
const pay = reactive({
  wechat: { enabled: true, appId: 'wx********8888', mchId: '19******001' },
  alipay: { enabled: true, appId: '2026**********', partnerId: '2088******' },
  pos: { enabled: true },
})

// 打印机档口
const printers = reactive({
  stations: [
    { name: '热菜档', sn: 'FE-8021', enabled: true },
    { name: '凉菜档', sn: 'FE-8022', enabled: true },
    { name: '汤羹档', sn: 'FE-8024', enabled: false },
    { name: '主食档', sn: 'FE-8025', enabled: false },
    { name: '吧台', sn: 'FE-8023', enabled: true },
    { name: '收银台', sn: 'FE-8026', enabled: true },
  ],
})

// 经营开关
const biz = reactive({
  scanOrder: true, // 顾客扫码下单
  autoSettle: false, // 打烊自动结清
  qrTtlDays: 365,
  refreshSec: 30,
  lowStockWarn: true,
})

function save(scope: string) {
  setTimeout(() => {
    message.success(`${scope}设置已保存（mock）· 需重启对应端生效`)
  }, 300)
}

function openCred() {
  message.info('密钥/证书由店长在部署环境 .env 中配置（12.6），本页只读展示商户号，不保存密钥。')
}
</script>

<template>
  <div class="page">
    <div class="page-head">
      <div>
        <h1>系统设置</h1>
        <p>门店 / 支付 / 打印机 / 经营开关 · 敏感凭据按设计 12.6 分离存储</p>
      </div>
    </div>

    <div class="card">
      <div class="card-body" style="padding:0">
        <a-tabs default-active-key="store" tab-position="left" style="min-height:420px">
          <!-- 门店信息 -->
          <a-tab-pane key="store" tab="门店信息">
            <div class="tab-body">
              <div class="f-l">店名</div>
              <a-input v-model:value="store.name" style="max-width:320px" />
              <div class="f-l">地址</div>
              <a-input v-model:value="store.address" style="max-width:320px" />
              <div class="f-l">联系电话</div>
              <a-input v-model:value="store.phone" style="max-width:320px" />
              <div class="f-l">午市营业时段</div>
              <a-time-range-picker v-model:value="lunch" style="max-width:320px" />
              <div class="f-l">晚市营业时段</div>
              <a-time-range-picker v-model:value="dinner" style="max-width:320px" />
              <div style="margin-top:18px"><a-button type="primary" @click="save('门店信息')">保存</a-button></div>
            </div>
          </a-tab-pane>

          <!-- 支付渠道 -->
          <a-tab-pane key="pay" tab="支付渠道">
            <div class="tab-body">
              <div class="ch-row" v-for="c in ['微信支付', '支付宝', 'POS 刷卡']" :key="c">
                <div class="ch-name">
                  <b>{{ c }}</b>
                  <small v-if="c === '微信支付'">appId {{ pay.wechat.appId }} · 商户号 {{ pay.wechat.mchId }}</small>
                  <small v-else-if="c === '支付宝'">appId {{ pay.alipay.appId }} · 合作者 {{ pay.alipay.partnerId }}</small>
                  <small v-else>收银台终端接入</small>
                </div>
                <a-switch
                  :checked="c === '微信支付' ? pay.wechat.enabled : c === '支付宝' ? pay.alipay.enabled : pay.pos.enabled"
                  @change="(v: boolean) => (c === '微信支付' ? pay.wechat.enabled = v : c === '支付宝' ? pay.alipay.enabled = v : pay.pos.enabled = v)"
                />
              </div>
              <div class="cred-note">ⓘ 应用密钥/证书不在本页展示，部署时写入服务器环境变量（12.6）。配置入口：<a-button type="link" size="small" @click="openCred">查看凭据处理约定</a-button></div>
              <div style="margin-top:14px"><a-button type="primary" @click="save('支付渠道')">保存</a-button></div>
            </div>
          </a-tab-pane>

          <!-- 打印机 -->
          <a-tab-pane key="printer" tab="打印机">
            <div class="tab-body">
              <div class="ch-row" v-for="p in printers.stations" :key="p.name">
                <div class="ch-name"><b>{{ p.name }}</b><small>SN {{ p.sn }}</small></div>
                <a-switch v-model:checked="p.enabled" />
              </div>
              <div style="color:var(--t3);font-size:12px;margin-top:10px">打印任务与 KDS 档口同步；离线/缺纸告警阈值见运维监控。</div>
              <div style="margin-top:14px"><a-button type="primary" @click="save('打印机')">保存</a-button></div>
            </div>
          </a-tab-pane>

          <!-- 经营开关 -->
          <a-tab-pane key="biz" tab="经营开关">
            <div class="tab-body">
              <div class="ch-row">
                <div class="ch-name"><b>顾客扫码下单</b><small>关闭即进入降级模式（新单走补录）</small></div>
                <a-switch v-model:checked="biz.scanOrder" />
              </div>
              <div class="ch-row">
                <div class="ch-name"><b>打烊自动结清提醒</b><small>到点提醒未结账桌台，不自动收款</small></div>
                <a-switch v-model:checked="biz.autoSettle" />
              </div>
              <div class="ch-row">
                <div class="ch-name"><b>低库存告警</b><small>沽清菜品达阈值时推送店长</small></div>
                <a-switch v-model:checked="biz.lowStockWarn" />
              </div>
              <div class="ch-row">
                <div class="ch-name"><b>桌贴二维码有效期</b><small>过期后扫码提示联系服务员</small></div>
                <a-input-number v-model:value="biz.qrTtlDays" :min="30" :max="730" addon-after="天" style="width:140px" />
              </div>
              <div class="ch-row">
                <div class="ch-name"><b>数据刷新间隔</b><small>营业概览自动刷新秒数</small></div>
                <a-input-number v-model:value="biz.refreshSec" :min="10" :max="300" addon-after="秒" style="width:140px" />
              </div>
              <div style="margin-top:14px"><a-button type="primary" @click="save('经营开关')">保存</a-button></div>
            </div>
          </a-tab-pane>
        </a-tabs>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tab-body { padding: 4px 8px 20px; max-width: 560px }
.f-l { font-size: 13px; color: var(--t2); margin: 14px 0 6px; font-weight: 500 }
.ch-row { display: flex; align-items: center; justify-content: space-between; padding: 12px 0; border-bottom: 1px dashed var(--split) }
.ch-name b { display: block; font-size: 14px }
.ch-name small { font-size: 12px; color: var(--t3); display: block; margin-top: 2px }
.cred-note { margin-top: 12px; padding: 9px 12px; background: var(--wn-bg); border-radius: 6px; font-size: 12px; color: var(--wn); display: flex; align-items: center; gap: 6px }
</style>
