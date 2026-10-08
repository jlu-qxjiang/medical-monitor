/**
 * 创建家庭组页
 * 输入家庭组名称，创建后自动成为 owner
 */
const api = require('../../../utils/api');
const app = getApp();

Page({
  data: {
    familyName: '',
    isSubmitting: false,
    errorMsg: '',
    createdFamily: null,
    noFamily: false // 是否从无家庭组状态进入
  },

  onLoad(options) {
    if (!app.checkLogin()) return;

    this.setData({
      noFamily: options.no_family === 'true'
    });
  },

  /**
   * 输入家庭组名称
   */
  handleNameInput(e) {
    this.setData({
      familyName: e.detail.value,
      errorMsg: ''
    });
  },

  /**
   * 提交创建
   */
  async handleSubmit() {
    const name = this.data.familyName.trim();

    if (!name) {
      this.setData({ errorMsg: '请输入家庭组名称' });
      return;
    }

    if (name.length > 20) {
      this.setData({ errorMsg: '家庭组名称不超过20个字符' });
      return;
    }

    if (this.data.isSubmitting) return;

    this.setData({ isSubmitting: true, errorMsg: '' });

    try {
      const result = await api.post('/families', { name }, {
        showLoading: true,
        loadingText: '创建中...',
        showError: false
      });

      this.setData({
        isSubmitting: false,
        createdFamily: result
      });

    } catch (err) {
      this.setData({
        isSubmitting: false,
        errorMsg: err.message || '创建失败，请重试'
      });
    }
  },

  /**
   * 创建成功后的操作
   */
  handleGoHome() {
    const family = this.data.createdFamily;
    if (family) {
      app.switchFamily({
        id: family.id,
        name: family.name,
        my_role: 'owner',
        invite_code: family.invite_code
      });
    }
    wx.switchTab({ url: '/pages/index/index' });
  },

  /**
   * 复制邀请码
   */
  handleCopyInviteCode() {
    const code = this.data.createdFamily ? this.data.createdFamily.invite_code : '';
    if (!code) return;

    wx.setClipboardData({
      data: code,
      success: () => {
        wx.showToast({ title: '邀请码已复制', icon: 'success' });
      }
    });
  },

  /**
   * 分享邀请码给微信好友
   */
  handleShareInvite() {
    const code = this.data.createdFamily ? this.data.createdFamily.invite_code : '';
    // 触发分享（需要页面定义 onShareAppMessage）
    wx.showShareMenu({
      withShareTicket: true,
      menus: ['shareAppMessage']
    });
    // 提示用户点击右上角分享
    wx.showToast({ title: '点击右上角分享给家人', icon: 'none', duration: 3000 });
  }
});
