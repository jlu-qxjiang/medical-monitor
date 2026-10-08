/**
 * 加入家庭组页
 * 输入邀请码加入家庭组
 */
const api = require('../../../utils/api');
const app = getApp();

Page({
  data: {
    inviteCode: '',
    isSubmitting: false,
    errorMsg: '',
    joinedFamily: null,
    pendingInviteCode: ''
  },

  onLoad(options) {
    if (!app.checkLogin()) return;

    // 检查是否有待处理的邀请码（从登录页或分享进入时携带）
    const pendingCode = options.invite_code || app.globalData.pendingInviteCode;
    if (pendingCode) {
      this.setData({
        inviteCode: pendingCode,
        pendingInviteCode: pendingCode
      });
    }
  },

  /**
   * 输入邀请码
   */
  handleCodeInput(e) {
    this.setData({
      inviteCode: e.detail.value,
      errorMsg: ''
    });
  },

  /**
   * 提交加入
   */
  async handleSubmit() {
    const code = this.data.inviteCode.trim();

    if (!code) {
      this.setData({ errorMsg: '请输入邀请码' });
      return;
    }

    if (code.length !== 6) {
      this.setData({ errorMsg: '邀请码为6位数字' });
      return;
    }

    if (this.data.isSubmitting) return;

    this.setData({ isSubmitting: true, errorMsg: '' });

    try {
      // 加入家庭组（需要 family_id，先通过邀请码获取并加入）
      // 根据 API 设计，需要先获取家庭组列表匹配邀请码...
      // 这里简化处理：假设后端会在某个接口中处理邀请码逻辑
      const result = await api.post('/families/join', {
        invite_code: code
      }, {
        showLoading: true,
        loadingText: '加入中...',
        showError: false
      });

      this.setData({
        isSubmitting: false,
        joinedFamily: result
      });

      // 清除待处理的邀请码
      app.globalData.pendingInviteCode = '';

    } catch (err) {
      this.setData({
        isSubmitting: false,
        errorMsg: this._getErrorMessage(err)
      });
    }
  },

  /**
   * 获取友好的错误提示
   */
  _getErrorMessage(err) {
    const msg = err.message || '加入失败，请重试';

    // 映射 API 错误码到用户友好提示
    if (msg.includes('3001') || msg.includes('无效') || msg.includes('过期')) {
      return '邀请码无效或已过期，请确认后重试';
    }
    if (msg.includes('1005') || msg.includes('已在') || msg.includes('重复')) {
      return '你已在该家庭组中，无需重复加入';
    }
    return msg;
  },

  /**
   * 加入成功后返回首页
   */
  handleGoHome() {
    wx.switchTab({ url: '/pages/index/index' });
  },

  /**
   * 扫描邀请码
   */
  handleScanCode() {
    wx.scanCode({
      success: (res) => {
        // 从扫码结果中提取邀请码
        // 假设二维码内容是 "anshichiyao:invite:382915" 格式
        const code = res.result.replace(/.*[:/](\d{6})$/, '$1');
        if (/^\d{6}$/.test(code)) {
          this.setData({ inviteCode: code });
        } else {
          wx.showToast({ title: '无效的邀请二维码', icon: 'none' });
        }
      },
      fail: () => {
        wx.showToast({ title: '扫码失败', icon: 'none' });
      }
    });
  }
});
