/**
 * 今日用药页（老人端核心页面）
 * 展示今日所有用药提醒，支持确认/跳过/撤销操作
 */
const api = require('../../../utils/api');
const dateUtil = require('../../../utils/date');

const app = getApp();

Page({
  data: {
    // 提醒数据
    reminders: [],
    dateLabel: '',

    // 页面状态
    isLoading: true,
    loadError: false,
    isSubmitting: false,

    // 确认弹窗
    showConfirmDialog: false,
    confirmReminder: null,
    confirmItems: [],
    dialogLoading: false,

    // 跳过弹窗
    showSkipDialog: false,
    skipReminder: null,
    skipActions: [
      { name: '医生让停药' },
      { name: '今天外出' },
      { name: '其他' }
    ]
  },

  onLoad() {
    if (!app.checkLogin()) return;
    this.loadData();
  },

  async onPullDownRefresh() {
    await this.loadData();
    wx.stopPullDownRefresh();
  },

  /**
   * 加载今日提醒数据
   */
  async loadData() {
    this.setData({ isLoading: true, loadError: false });

    const familyId = app.globalData.currentFamilyId;

    try {
      const data = await api.get('/reminders/today', {
        family_id: familyId
      }, { showLoading: false, showError: false });

      const reminders = (data.reminders || []).map(reminder => ({
        ...reminder,
        _timeLabel: dateUtil.getTimeOfDayLabel(reminder.remind_time),
        _allTaken: reminder.items ? reminder.items.every(item => item.taken) : false,
        _pendingItems: reminder.items ? reminder.items.filter(item => !item.taken) : [],
        _takenItems: reminder.items ? reminder.items.filter(item => item.taken) : []
      }));

      this.setData({
        reminders,
        dateLabel: dateUtil.formatDateCN(data.date),
        isLoading: false
      });
    } catch (err) {
      console.error('加载今日提醒失败:', err);
      this.setData({ isLoading: false, loadError: true });
    }
  },

  /**
   * 点击「已服用」→ 弹出确认弹窗
   */
  handleConfirm(e) {
    const reminder = e.currentTarget.dataset.reminder;
    if (!reminder || this.data.isSubmitting) return;

    // 收集未服用的 items
    const pendingItems = reminder.items.filter(item => !item.taken);

    if (pendingItems.length === 0) {
      wx.showToast({ title: '已全部服用', icon: 'success' });
      return;
    }

    this.setData({
      showConfirmDialog: true,
      confirmReminder: reminder,
      confirmItems: pendingItems
    });
  },

  /**
   * 确认弹窗 — 确认已服用
   */
  async handleDialogConfirm() {
    const reminder = this.data.confirmReminder;
    if (!reminder || this.data.dialogLoading) return;

    this.setData({ dialogLoading: true });

    const itemIds = this.data.confirmItems.map(item => item.id);

    try {
      const result = await api.post(`/reminders/${reminder.id}/confirm`, {
        item_ids: itemIds
      }, { showLoading: false, showError: false });

      wx.showToast({ title: '已确认服用', icon: 'success', duration: 2000 });

      this.setData({
        showConfirmDialog: false,
        dialogLoading: false,
        confirmReminder: null,
        confirmItems: []
      });

      // 存储撤销信息
      if (result && result.can_revoke_until) {
        reminder._canRevokeUntil = result.can_revoke_until;
      }

      // 刷新数据
      setTimeout(() => this.loadData(), 500);

    } catch (err) {
      this.setData({ dialogLoading: false });
      wx.showToast({ title: '操作失败，请重试', icon: 'none' });
    }
  },

  /**
   * 确认弹窗 — 取消
   */
  handleDialogCancel() {
    this.setData({
      showConfirmDialog: false,
      confirmReminder: null,
      confirmItems: []
    });
  },

  /**
   * 点击「跳过」→ 弹出跳过弹窗
   */
  handleSkip(e) {
    const reminder = e.currentTarget.dataset.reminder;
    if (!reminder || this.data.isSubmitting) return;

    this.setData({
      showSkipDialog: true,
      skipReminder: reminder
    });
  },

  /**
   * 跳过原因选择（van-action-sheet）
   */
  handleSkipReasonSelect(e) {
    const reminder = this.data.skipReminder;
    if (!reminder || this.data.isSubmitting) return;

    const reason = this.data.skipActions[e.detail.index].name;
    this.setData({ isSubmitting: true });

    api.post(`/reminders/${reminder.id}/skip`, {
      reason: reason
    }, { showLoading: false, showError: false }).then(() => {
      wx.showToast({ title: '已跳过本次用药', icon: 'success' });
      this.setData({ showSkipDialog: false, skipReminder: null, isSubmitting: false });
      this.loadData();
    }).catch(() => {
      this.setData({ isSubmitting: false });
      wx.showToast({ title: '操作失败，请重试', icon: 'none' });
    });
  },

  /**
   * 取消跳过
   */
  handleSkipCancel() {
    this.setData({ showSkipDialog: false, skipReminder: null });
  },

  /**
   * 撤销确认（5 分钟内）
   */
  async handleRevoke(e) {
    const reminder = e.currentTarget.dataset.reminder;
    if (!reminder || this.data.isSubmitting) return;

    // 检查是否在可撤销时间内
    if (reminder._canRevokeUntil) {
      const deadline = new Date(reminder._canRevokeUntil).getTime();
      if (Date.now() > deadline) {
        wx.showToast({ title: '已超过撤销时间（5分钟）', icon: 'none' });
        return;
      }
    }

    this.setData({ isSubmitting: true });

    try {
      await api.post(`/reminders/${reminder.id}/revoke`, {}, {
        showLoading: false,
        showError: false
      });

      wx.showToast({ title: '已撤销确认', icon: 'success' });

      this.setData({ isSubmitting: false });
      this.loadData();

    } catch (err) {
      this.setData({ isSubmitting: false });
      wx.showToast({ title: '撤销失败，请重试', icon: 'none' });
    }
  },

  /**
   * 标记未服用
   */
  async handleMiss(e) {
    const reminder = e.currentTarget.dataset.reminder;
    if (!reminder || this.data.isSubmitting) return;

    wx.showActionSheet({
      itemList: ['忘记了', '不舒服', '药没了', '其他'],
      success: async (res) => {
        const reasons = ['忘记了', '不舒服', '药没了', '其他'];
        const reason = reasons[res.tapIndex];

        this.setData({ isSubmitting: true });

        try {
          await api.post(`/reminders/${reminder.id}/miss`, { reason }, {
            showLoading: false,
            showError: false
          });

          wx.showToast({ title: '已标记未服用', icon: 'none' });
          this.setData({ isSubmitting: false });
          this.loadData();

        } catch (err) {
          this.setData({ isSubmitting: false });
          wx.showToast({ title: '操作失败，请重试', icon: 'none' });
        }
      }
    });
  },

  /**
   * 重试加载
   */
  handleRetry() {
    this.loadData();
  }
});
