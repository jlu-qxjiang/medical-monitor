/**
 * 家庭组列表页
 * 展示用户所属的全部家庭组
 */
const api = require('../../../utils/api');
const app = getApp();

Page({
  data: {
    familyList: [],
    isLoading: true,
    loadError: false
  },

  onLoad() {
    if (!app.checkLogin()) return;
  },

  onShow() {
    this.loadData();
  },

  /**
   * 加载家庭组列表
   */
  async loadData() {
    this.setData({ isLoading: true, loadError: false });

    try {
      const data = await api.get('/families', {}, {
        showLoading: false,
        showError: false
      });

      this.setData({
        familyList: data.list || [],
        isLoading: false
      });
    } catch (err) {
      console.error('加载家庭组列表失败:', err);
      this.setData({ isLoading: false, loadError: true });
    }
  },

  /**
   * 切换到指定家庭组
   */
  handleSwitchFamily(e) {
    const family = e.currentTarget.dataset.family;
    if (!family) return;

    app.switchFamily(family);
    wx.showToast({ title: `已选中「${family.name}」`, icon: 'success' });
  },

  /**
   * 查看家庭组详情（成员管理）
   */
  handleFamilyDetail(e) {
    const family = e.currentTarget.dataset.family;
    if (!family) return;

    // 切换到该家庭组
    app.switchFamily(family);

    // TODO: 跳转到成员管理页（P1 功能）
    wx.showToast({ title: '成员管理（P1 待开发）', icon: 'none' });
  },

  /**
   * 创建家庭组
   */
  handleCreate() {
    wx.navigateTo({ url: '/pages/family/create/create' });
  },

  /**
   * 加入家庭组
   */
  handleJoin() {
    wx.navigateTo({ url: '/pages/family/join/join' });
  },

  /**
   * 重试
   */
  async refreshInvite(e) {
    const fid = e.currentTarget.dataset.fid
    try {
      const data = await api.post('/families/' + fid + '/refresh-invite', {}, { showLoading: false })
      wx.showToast({ title: '邀请码已刷新', icon: 'success' })
      this.loadData()
    } catch (err) {
      wx.showToast({ title: '刷新失败', icon: 'none' })
    }
  },

  handleRetry() {
    this.loadData()
  }
});
