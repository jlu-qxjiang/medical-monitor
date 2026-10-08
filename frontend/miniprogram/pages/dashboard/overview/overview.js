/**
 * 家庭看板 — 指定老人今日用药详情
 * 子女端查看单个老人今日用药完成情况
 */
const api = require('../../../utils/api');
const dateUtil = require('../../../utils/date');

const app = getApp();

Page({
  data: {
    elderUserId: null,
    elderName: '',
    familyId: null,

    // 看板数据
    slots: [],
    summary: {},
    date: '',

    // 页面状态
    isLoading: true,
    loadError: false
  },

  onLoad(options) {
    if (!app.checkLogin()) return;

    const { elder_user_id, elder_name, family_id } = options;

    this.setData({
      elderUserId: elder_user_id,
      elderName: elder_name || '老人详情',
      familyId: family_id || app.globalData.currentFamilyId
    });

    wx.setNavigationBarTitle({
      title: `${this.data.elderName}的用药详情`
    });

    this.loadData();
  },

  async onPullDownRefresh() {
    await this.loadData();
    wx.stopPullDownRefresh();
  },

  /**
   * 加载看板数据
   */
  async loadData() {
    this.setData({ isLoading: true, loadError: false });

    try {
      const data = await api.get('/dashboard/today', {
        family_id: this.data.familyId,
        elder_user_id: this.data.elderUserId
      }, { showLoading: false, showError: false });

      // 为每个时间槽添加便于展示的字段
      const slots = (data.slots || []).map(slot => ({
        ...slot,
        _timeLabel: slot.time_label,
        _allTaken: slot.items ? slot.items.every(item => item.taken) : false,
        _statusItemClass: this._getStatusClass(slot.status),
        _statusLabel: slot.status_label
      }));

      this.setData({
        slots,
        summary: data.summary || {},
        date: dateUtil.formatDateCN(data.date),
        isLoading: false
      });
    } catch (err) {
      console.error('加载看板数据失败:', err);
      this.setData({ isLoading: false, loadError: true });
    }
  },

  /**
   * 获取状态对应的 CSS class
   */
  _getStatusClass(status) {
    const map = {
      'confirmed': 'status-confirmed',
      'pending': 'status-pending',
      'sent': 'status-pending',
      'missed': 'status-missed',
      'skipped': 'status-skipped'
    };
    return map[status] || 'status-pending';
  },

  /**
   * 查看用药历史
   */
  handleViewHistory() {
    wx.navigateTo({
      url: `/pages/reminder/history/history?elder_user_id=${this.data.elderUserId}&elder_name=${this.data.elderName}`
    });
  },

  /**
   * 管理用药计划
   */
  handleManageSchedule() {
    wx.navigateTo({
      url: `/pages/medicine/schedule/schedule?elder_user_id=${this.data.elderUserId}&family_id=${this.data.familyId}`
    });
  },

  /**
   * 重试
   */
  handleRetry() {
    this.loadData();
  },

  // 家人代确认
  async proxyConfirm(e) {
    var rid = e.currentTarget.dataset.rid
    if (!rid) return
    try {
      await api.post('/reminders/' + rid + '/confirm', { item_ids: [] })
      wx.showToast({ title: '已标记服用', icon: 'success' })
      this.loadData()
    } catch (err) {
      wx.showToast({ title: '操作失败', icon: 'none' })
    }
  },

  viewSchedule() {
    wx.switchTab({ url: '/pages/index/index' })
  }
});
