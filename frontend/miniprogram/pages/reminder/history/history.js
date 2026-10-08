/**
 * 用药历史页
 * 按日期查看历史用药记录，日历选择器切换日期
 */
const api = require('../../../utils/api');
const dateUtil = require('../../../utils/date');

const app = getApp();

Page({
  data: {
    elderUserId: null,
    elderName: '',

    // 选中日期
    selectedDate: '',
    selectedDateStr: '',

    // 历史记录
    records: [],
    summary: {},

    // 页面状态
    isLoading: true,
    loadError: false,
    page: 1,
    hasMore: true
  },

  onLoad(options) {
    if (!app.checkLogin()) return;

    const today = dateUtil.formatDate(new Date());
    this.setData({
      elderUserId: options.elder_user_id || '',
      elderName: options.elder_name || '',
      selectedDate: today,
      selectedDateStr: dateUtil.formatDateCN(today)
    });

    if (this.data.elderName) {
      wx.setNavigationBarTitle({ title: `${this.data.elderName}的用药历史` });
    }
  },

  onShow() {
    if (app.globalData.isLoggedIn) {
      this.loadData();
    }
  },

  /**
   * 加载历史记录
   */
  async loadData() {
    this.setData({ isLoading: true, loadError: false });

    const { selectedDate, elderUserId } = this.data;
    const familyId = app.globalData.currentFamilyId;

    try {
      // 使用 /reminders/today 替代历史接口（因为 API 设计中暂无历史接口）
      // 实际联调时会替换为 GET /reminders/history
      const data = await api.get('/reminders/today', {
        family_id: familyId,
        date: selectedDate
      }, { showLoading: false, showError: false });

      const records = (data.reminders || []).map(reminder => ({
        ...reminder,
        _timeLabel: dateUtil.getTimeOfDayLabel(reminder.remind_time),
        _allTaken: reminder.items ? reminder.items.every(item => item.taken) : false
      }));

      this.setData({
        records,
        summary: {
          total: records.length,
          confirmed: records.filter(r => r._allTaken).length
        },
        isLoading: false
      });
    } catch (err) {
      console.error('加载用药历史失败:', err);
      this.setData({ isLoading: false, loadError: true });
    }
  },

  /**
   * 日期选择
   */
  handleDateChange(e) {
    const date = e.detail.value;
    this.setData({
      selectedDate: date,
      selectedDateStr: dateUtil.formatDateCN(date),
      page: 1
    });
    this.loadData();
  },

  /**
   * 前一天
   */
  handlePrevDay() {
    const prevDate = dateUtil.daysAgo(-1, this.data.selectedDate);
    this.setData({
      selectedDate: prevDate,
      selectedDateStr: dateUtil.formatDateCN(prevDate),
      page: 1
    });
    this.loadData();
  },

  /**
   * 后一天
   */
  handleNextDay() {
    const { selectedDate } = this.data;
    if (dateUtil.isToday(selectedDate)) {
      wx.showToast({ title: '已是今天', icon: 'none' });
      return;
    }

    const nextDate = dateUtil.daysAgo(1, selectedDate);
    this.setData({
      selectedDate: nextDate,
      selectedDateStr: dateUtil.formatDateCN(nextDate),
      page: 1
    });
    this.loadData();
  },

  /**
   * 重试
   */
  handleRetry() {
    this.loadData();
  }
});
