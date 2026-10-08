/**
 * 药品详情页（子女端）
 * 展示药品完整信息，支持编辑/删除
 */
const api = require('../../../utils/api');
const app = getApp();

Page({
  data: {
    medicineId: null,
    familyId: null,
    medicine: null,
    isLoading: true,
    isDeleting: false
  },

  onLoad(options) {
    if (!app.checkLogin()) return;

    const { id, family_id } = options;
    this.setData({
      medicineId: id,
      familyId: family_id || app.globalData.currentFamilyId
    });
  },

  onShow() {
    if (this.data.medicineId) {
      this.loadData();
    }
  },

  /**
   * 加载药品详情
   */
  async loadData() {
    this.setData({ isLoading: true });

    try {
      const data = await api.get(
        `/families/${this.data.familyId}/medicines/${this.data.medicineId}`,
        {},
        { showLoading: false, showError: false }
      );

      this.setData({
        medicine: data,
        isLoading: false
      });
    } catch (err) {
      console.error('加载药品详情失败:', err);
      this.setData({ isLoading: false });
      wx.showToast({ title: '加载失败', icon: 'none' });
    }
  },

  /**
   * 编辑药品
   */
  handleEdit() {
    wx.navigateTo({
      url: `/pages/medicine/add/add?family_id=${this.data.familyId}&id=${this.data.medicineId}`
    });
  },

  /**
   * 删除药品
   */
  handleDelete() {
    if (this.data.isDeleting) return;

    wx.showModal({
      title: '确认删除',
      content: `确定要删除「${this.data.medicine.name}」吗？删除后不可恢复。`,
      confirmText: '删除',
      confirmColor: '#FA5151',
      success: async (res) => {
        if (res.confirm) {
          await this._doDelete();
        }
      }
    });
  },

  /**
   * 执行删除
   */
  async _doDelete() {
    this.setData({ isDeleting: true });

    try {
      await api.del(
        `/families/${this.data.familyId}/medicines/${this.data.medicineId}`,
        {},
        { showLoading: true, loadingText: '删除中...', showError: false }
      );

      wx.showToast({ title: '已删除', icon: 'success' });
      wx.navigateBack();

    } catch (err) {
      this.setData({ isDeleting: false });

      // 处理 3003 错误：药品已关联用药计划
      if (err.message && err.message.includes('3003')) {
        wx.showToast({ title: '该药品已关联用药计划，无法删除', icon: 'none', duration: 3000 });
      } else {
        wx.showToast({ title: '删除失败，请重试', icon: 'none' });
      }
    }
  }
});
