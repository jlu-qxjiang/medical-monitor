/**
 * 订阅消息授权工具
 *
 * 微信订阅消息是「一次性订阅」：用户每授权一次，只能收到一条消息。
 * 因此需要在老人/子女每次与小程序交互时，引导其授权订阅，累积额度。
 * 用户勾选「总是保持以上选择，不再询问」后，下次自动授权、不再弹窗。
 *
 * 注意：tmplIds 必须是 mp.weixin.qq.com「订阅消息」里申请到的真实模板 ID，
 * 当前为占位符，备案 + 申请模板后替换成真实值即可生效。
 */
const TEMPLATE_IDS = {
  medicationReminder: 'eX7azyEwYahM7JoFMrLOMterZzyKrvqUBjo_2Hm2d1s', // 服药提醒模板（发给吃药人）
  medicationAlert: 'OpY3dd8lVTQyq_Wh-5ow21boUYK95-KeKLreH8Snw00'     // 服药告警模板（发给子女）
};

/**
 * 请求订阅授权
 * @param {string[]} tmplIds 模板 ID 列表（单次最多 3 个）
 */
function requestSubscribe(tmplIds) {
  const ids = (tmplIds || []).filter(id => !!id);
  if (ids.length === 0) return; // 模板 ID 未配置，跳过
  if (typeof wx.requestSubscribeMessage !== 'function') return;

  wx.requestSubscribeMessage({
    tmplIds: ids,
    success(res) {
      // res[tmplId] = 'accept'（同意）| 'reject'（拒绝）| 'ban'（被禁用）
      console.log('[subscribe] 授权结果:', res);
    },
    fail(err) {
      console.log('[subscribe] 授权失败:', err);
    }
  });
}

module.exports = { TEMPLATE_IDS, requestSubscribe };
