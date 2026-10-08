/**
 * 首页 — 角色分流
 * 老人端：今日用药提醒列表
 * 子女端：家庭看板（所有老人用药状态卡片列表）
 */
const api = require('../../utils/api');
const auth = require('../../utils/auth');
const dateUtil = require('../../utils/date');
const subscribe = require('../../utils/subscribe');

const app = getApp();

Page({
  data: {
    // 通用
    isLoggedIn: false,
    userRole: '',
    isElder: false,
    isLoading: true,
    loadError: false,

    // 老人端数据
    hasSharedFamily: false,
    todayReminders: [],
    plans: [],
    elderName: '',
    elderTab: 'today',
    inviteCode: '',
    joining: false,
    joinError: '',

    // 子女端数据
    familyName: '',
    inviteCode: '',
    elders: [],
    currentFamilyId: null,
    schTab: 'family',
    schList: [],
    myReminders: [],
    alerts: [],
    lowStocks: [],
    userRole: '',

    // 计划详情弹窗
    schDetail: null,

    // 日历
    calYear: 2026,
    calMonth: 8,
    calWeekDays: ['日', '一', '二', '三', '四', '五', '六'],
    calDays: [],
    calDateStatus: {},  // { "2026-08-01": "completed", ... }
    calSelDate: '',

    // 计划编辑弹窗
    editSch: null,
    editDosage: '',
    editUnit: '粒',
    editQty: '',
    editRel: '',
    editStart: '',
    editEnd: '',
    editTimes: ['08:00'],
    savingEdit: false,
  },

  onLoad() {
    // 不强制跳登录，未登录显示角色选择
    this.setData({ isLoggedIn: app.globalData.isLoggedIn || false })
    if (app.globalData.isLoggedIn) {
      this._initPage()
    }
  },

  onShow() {
    const loggedIn = app.globalData.isLoggedIn || false
    this.setData({ isLoggedIn: loggedIn })
    if (loggedIn) {
      wx.showTabBar()
      this._initPage()
      this._requestSubscribe()
    } else {
      wx.hideTabBar()
    }
  },

  // 未登录时点击角色 → 跳登录页
  goLogin(e) {
    const role = e.currentTarget.dataset.role
    wx.setStorageSync('userRole', role)
    wx.navigateTo({ url: '/pages/login/login?role=' + role })
  },

  // 根据角色请求订阅消息授权（老人→服药提醒，子女→服药告警）
  _requestSubscribe() {
    const userRole = wx.getStorageSync('userRole') || app.globalData.userRole || '';
    if (userRole === 'elder') {
      subscribe.requestSubscribe([subscribe.TEMPLATE_IDS.medicationReminder]);
    } else if (userRole === 'child') {
      subscribe.requestSubscribe([subscribe.TEMPLATE_IDS.medicationAlert]);
    }
  },

  async onPullDownRefresh() {
    await this._initPage();
    wx.stopPullDownRefresh();
  },

  /**
   * 初始化页面
   */
  async _initPage() {
    const userRole = wx.getStorageSync('userRole') || app.globalData.userRole || '';
    const isElder = userRole === 'elder';
    console.log('_initPage userRole:', userRole);
    this.setData({ userRole, isElder, isLoading: true, loadError: false });
    try {
      if (isElder) {
        await this._loadElderData();
      } else {
        await this._loadChildData();
      }
    } catch (err) {
      console.error('_initPage error:', err);
      this.setData({ loadError: true, isLoading: false });
    }
  },

  /**
   * 加载老人端数据
   */
  async _loadElderData() {
    try {
      // 先查家庭组，判断有无共享家庭
      const famData = await api.get('/families', {}, { showLoading: false, showError: false })
      const list = famData.list || []
      const shared = list.filter(f => !f.is_personal)
      const hasShared = shared.length > 0

      if (!hasShared) {
        this.setData({ hasSharedFamily: false, isLoading: false })
        return
      }

      const familyId = shared[0].id
      app.globalData.currentFamilyId = familyId

      // 加载今日提醒
      const data = await api.get('/reminders/today', { family_id: familyId }, { showLoading: false, showError: false })
      const reminders = (data.reminders || []).map(r => {
        var hour = parseInt((r.time_label||'0').split(':')[0]) || 0
        var icon = hour < 9 ? 'clock-o' : hour < 12 ? 'clock-o' : hour < 17 ? 'clock-o' : 'clock-o'
        return { ...r, _allTaken: r.items ? r.items.every(i => i.taken) : false, _isMissed: r.status==='missed', _isSkipped: r.status==='skipped', _icon: icon }
      })

      // 加载用药计划
      let plans = []
      try {
        const planData = await api.get('/families/' + familyId + '/schedules', {}, { showLoading: false, showError: false })
        var today = new Date().toISOString().substring(0,10)
        plans = (planData.list || []).map(p => ({
          ...p,
          times: (p.times || []).join(' / '),
          start_date: (p.start_date||'').substring(0,10),
          end_date: (p.end_date||'').substring(0,10),
          _isEnded: (p.end_date||'').substring(0,10) < today
        }))
      } catch (e) {}

      this.setData({
        hasSharedFamily: true, todayReminders: reminders, plans,
        elderName: (app.globalData.userInfo || {}).nickname || '',
        isLoading: false
      })
    } catch (err) {
      this.setData({ isLoading: false, loadError: true })
    }
  },

  /**
   * 加载子女端数据
   */
  async _loadChildData() {
    const familyId = app.globalData.currentFamilyId;
    if (!familyId) { this.setData({ isLoading: false }); return; }

    try {
      // 并行加载：看板 + 家庭详情(邀请码) + 计划
      const [dashData, famData] = await Promise.all([
        api.get('/dashboard/overview', { family_id: familyId }, { showLoading: false, showError: false }),
        api.get('/families/' + familyId, {}, { showLoading: false, showError: false })
      ])

      var elders = (dashData.elders || []).map(function(e) {
        var pct = e.total > 0 ? (e.confirmed / e.total * 100) : 0
        e._pct = pct.toFixed(2)
        return e
      })

      this.setData({
        familyName: dashData.family_name || famData.name || '',
        inviteCode: famData.invite_code || '',
        elders: elders,
        currentFamilyId: familyId,
        isLoading: false
      })
      this._loadSchList()
      this._loadCalData()
      this._loadMyReminders()
      this._loadAlerts()
      this._loadLowStocks()
    } catch (err) {
      this.setData({ isLoading: false, loadError: true })
    }
  },
  async _loadLowStocks() {
    // 检查库存提醒开关
    if (wx.getStorageSync('notify_master') === false) { this.setData({ lowStocks: [] }); return }
    if (wx.getStorageSync('notify_stock') === false) { this.setData({ lowStocks: [] }); return }

    var fid = app.globalData.currentFamilyId
    if (!fid) return
    try {
      var data = await api.get('/families/'+fid+'/stocks', {}, { showLoading: false, showError: false })
      var low = (data.list || []).filter(function(s){
        return s.status === 'empty' || s.status === 'low'
      })
      this.setData({ lowStocks: low })
    } catch(e) { this.setData({ lowStocks: [] }) }
  },

  async _loadAlerts() {
    var fid = app.globalData.currentFamilyId
    if (!fid) return
    try {
      var data = await api.get('/alerts', { family_id: fid }, { showLoading: false, showError: false })
      this.setData({ alerts: data.list || [] })
    } catch(e) { this.setData({ alerts: [] }) }
  },

  async _loadMyReminders() {
    try {
      var data = await api.get('/reminders/today', { family_id: 0 }, { showLoading: false, showError: false })
      var reminders = (data.reminders || []).map(function(r){
        return { ...r, _allTaken: r.items ? r.items.every(function(i){ return i.taken }) : false }
      })
      this.setData({ myReminders: reminders })
    } catch(e) {
      this.setData({ myReminders: [] })
    }
  },

  async _loadSchList(filterDate) {
    const fid = app.globalData.currentFamilyId
    if (!fid) return
    const isPersonal = this.data.schTab === 'personal'
    var url = isPersonal ? '/personal/schedules' : ('/families/' + fid + '/schedules')
    var params = {}
    if (filterDate) { params.date = filterDate }
    try {
      const data = await api.get(url, params, { showLoading: false, showError: false })
      var today = new Date().toISOString().substring(0,10)
      this.setData({ schList: (data.list || []).map(function(p){
        var sd = (p.start_date||'').substring(0,10); var ed = (p.end_date||'').substring(0,10)
        return { ...p, times: (p.times || []).join(' / '), start_date: sd, end_date: ed, _isEnded: ed && ed < today }
      }) })
    } catch (e) { this.setData({ schList: [] }) }
  },

  switchSchTab(e) {
    this.setData({ schTab: e.currentTarget.dataset.tab, calSelDate: '' }, () => { this._loadSchList(); this._loadCalData() })
  },

  // Vant tabs 切换事件（detail.index: 0=家人计划, 1=个人计划）
  onSchTabChange(e) {
    this.setData({ schTab: e.detail.index === 0 ? 'family' : 'personal', calSelDate: '' }, () => { this._loadSchList(); this._loadCalData() })
  },

  showSchDetail(e) {
    var id = parseInt(e.currentTarget.dataset.id)
    var s = this.data.schList.find(function(x){ return x.id === id })
    if (!s) return
    this.setData({
      schDetail: {
        medicine_name: s.medicine_name,
        dosage: s.dosage,
        unit: s.unit,
        relationship: s.relationship,
        times: s.times || '',
        total_quantity: s.total_quantity,
        elder_nickname: s.elder_nickname || (s.elder_user_id ? 'ID:'+s.elder_user_id : ''),
        startDate: (s.start_date||'').substring(0,10),
        endDate: (s.end_date||'').substring(0,10),
        is_active: s.is_active,
        _isEnded: s._isEnded || false
      }
    })
  },

  showElderPlanDetail(e) {
    var id = parseInt(e.currentTarget.dataset.id)
    var s = this.data.plans.find(function(x){ return x.id === id })
    if (!s) return
    this.setData({
      schDetail: {
        medicine_name: s.medicine_name,
        dosage: s.dosage, unit: s.unit,
        relationship: s.relationship, times: s.times || '',
        total_quantity: s.total_quantity,
        elder_nickname: s.elder_nickname || '',
        startDate: (s.start_date||'').substring(0,10),
        endDate: (s.end_date||'').substring(0,10),
        is_active: s.is_active,
        _isEnded: s._isEnded || false
      }
    })
  },

  closeSchDetail() { this.setData({ schDetail: null }) },

  goAddSchedule() {
    wx.navigateTo({ url: '/pages/medicine/schedule/schedule?tab=' + this.data.schTab + '&fid=' + (app.globalData.currentFamilyId || 0) })
  },

  async refreshInvite() {
    const fid = app.globalData.currentFamilyId
    if (!fid) return
    try {
      const data = await api.post('/families/' + fid + '/refresh-invite', {}, { showLoading: false })
      this.setData({ inviteCode: data.invite_code })
      wx.showToast({ title: '已刷新', icon: 'success' })
    } catch (e) { wx.showToast({ title: '刷新失败', icon: 'none' }) }
  },

  // ===== 老人端事件 =====

  /**
   * 进入今日用药详情
   */
  handleGoToday() {
    wx.switchTab({ url: '/pages/reminder/today/today' });
  },

  // ===== 子女端事件 =====

  /**
   * 点击老人卡片 → 进入详情
   */
  handleElderCardTap(e) {
    const elder = e.currentTarget.dataset.elder;
    if (!elder) return;

    wx.navigateTo({
      url: `/pages/dashboard/overview/overview?elder_user_id=${elder.user_id}&elder_name=${elder.nickname}&family_id=${this.data.currentFamilyId}`
    });
  },

  /**
   * 创建家庭组
   */
  handleCreateFamily() {
    wx.navigateTo({ url: '/pages/family/create/create' });
  },

  /**
   * 加入家庭组
   */
  handleJoinFamily() {
    wx.navigateTo({ url: '/pages/family/join/join' });
  },

  /**
   * 重试加载
   */
  // ===== 老人端操作 =====
  onInviteInput(e) { this.setData({ inviteCode: e.detail.value, joinError: '' }) },

  switchElderTab(e) { this.setData({ elderTab: e.currentTarget.dataset.tab }) },

  // Vant tabs 切换事件（detail.index: 0=今日服药, 1=用药计划）
  onElderTabChange(e) { this.setData({ elderTab: e.detail.index === 0 ? 'today' : 'plan' }) },

  // Vant field 输入事件
  onInviteField(e) { this.setData({ inviteCode: e.detail, joinError: '' }) },

  async joinFamily() {
    const code = this.data.inviteCode.trim()
    if (!code || code.length !== 6) { this.setData({ joinError: '请输入6位邀请码' }); return }
    this.setData({ joining: true, joinError: '' })
    try {
      await api.post('/families/join', { invite_code: code })
      wx.showToast({ title: '加入成功', icon: 'success' })
      this.setData({ joining: false, inviteCode: '' })
      this._loadElderData()
    } catch (err) {
      this.setData({ joining: false, joinError: err.message || '加入失败' })
    }
  },

  // 单条药品确认
  async confirmReminder(e) {
    const { rid, iid } = e.currentTarget.dataset
    try {
      await api.post('/reminders/' + rid + '/confirm', { item_ids: [parseInt(iid)] })
      wx.showToast({ title: '已确认', icon: 'success' })
      this._loadElderData()
    } catch (err) {
      wx.showToast({ title: '操作失败', icon: 'none' })
    }
  },

  // 整张卡片确认（全部药品）
  async confirmReminderAll(e) {
    var rid = e.currentTarget.dataset.rid
    var r = this.data.todayReminders.find(function(x){ return x.id == rid })
    if (!r) return
    var itemIds = (r.items || []).filter(function(it){ return !it.taken }).map(function(it){ return it.id })
    if (itemIds.length === 0) { wx.showToast({ title: '已全部确认', icon: 'success' }); return }
    try {
      await api.post('/reminders/' + rid + '/confirm', { item_ids: itemIds })
      wx.showToast({ title: '已确认', icon: 'success' })
      this._loadElderData()
    } catch (err) {
      wx.showToast({ title: '操作失败', icon: 'none' })
    }
  },

  async missReminder(e) {
    var rid = parseInt(e.currentTarget.dataset.rid)
    // 检查是否已到用药时间
    var r = this.data.todayReminders.find(function(x){ return x.id === rid })
    if (r && r.remind_time && new Date(r.remind_time) > new Date()) {
      wx.showToast({ title: '未到用药时间', icon: 'none' })
      return
    }
    try {
      await api.post('/reminders/' + rid + '/miss', { reason: '忘记了' })
      wx.showToast({ title: '已标记未服用', icon: 'success' })
      this._loadElderData()
    } catch (err) { wx.showToast({ title: '操作失败', icon: 'none' }) }
  },

  async skipReminder(e) {
    var rid = parseInt(e.currentTarget.dataset.rid)
    try {
      await api.post('/reminders/' + rid + '/skip', { reason: '暂时不需要' })
      wx.showToast({ title: '已跳过', icon: 'success' })
      this._loadElderData()
    } catch (err) {
      wx.showToast({ title: '操作失败', icon: 'none' })
    }
  },

  // 个人用药确认
  async myConfirm(e) {
    var rid = e.currentTarget.dataset.rid
    var iid = e.currentTarget.dataset.iid
    try {
      await api.post('/reminders/'+rid+'/confirm', { item_ids: [iid] })
      wx.showToast({ title: '已确认', icon: 'success' })
      this._loadMyReminders()
    } catch(err) { wx.showToast({ title: '操作失败', icon: 'none' }) }
  },

  async myMiss(e) {
    var rid = e.currentTarget.dataset.rid
    try {
      await api.post('/reminders/'+rid+'/miss', { reason: '忘记了' })
      wx.showToast({ title: '已标记', icon: 'success' })
      this._loadMyReminders()
    } catch(err) { wx.showToast({ title: '操作失败', icon: 'none' }) }
  },

  viewAllAlerts() {
    wx.showToast({ title: '提醒历史功能开发中', icon: 'none' })
  },

  goElderDetail() {
    var elder = this.data.elders[0]
    if (!elder) return
    var fid = this.data.currentFamilyId
    wx.navigateTo({ url: '/pages/dashboard/overview/overview?elder_user_id=' + elder.user_id + '&elder_name=' + (elder.nickname || '') + '&family_id=' + fid })
  },

  goStockDetail() {
    wx.switchTab({ url: '/pages/medicine/list/list' })
  },

  handleRetry() {
    this._initPage();
  },

  /* ===== 日历 ===== */
  buildCalendar() {
    var now = new Date()
    var today = now.getFullYear() + '-' + String(now.getMonth() + 1).padStart(2, '0') + '-' + String(now.getDate()).padStart(2, '0')
    var year = this.data.calYear, month = this.data.calMonth
    var firstDay = new Date(year, month - 1, 1).getDay()
    var daysInMonth = new Date(year, month, 0).getDate()
    var days = []
    for (var i = 0; i < firstDay; i++) {
      days.push({ day: '', date: '', hasPlan: false })
    }
    for (var d = 1; d <= daysInMonth; d++) {
      var dateStr = year + '-' + String(month).padStart(2, '0') + '-' + String(d).padStart(2, '0')
      var status = this.data.calDateStatus[dateStr] || ''
      days.push({
        day: d,
        date: dateStr,
        isToday: dateStr === today,
        hasPlan: !!status,
        status: status  // '' / completed / partial / pending
      })
    }
    this.setData({ calDays: days })
  },

  prevMonth() {
    var m = this.data.calMonth - 1, y = this.data.calYear
    if (m < 1) { m = 12; y-- }
    this.setData({ calYear: y, calMonth: m })
    this._loadCalData()
  },
  nextMonth() {
    var m = this.data.calMonth + 1, y = this.data.calYear
    if (m > 12) { m = 1; y++ }
    this.setData({ calYear: y, calMonth: m })
    this._loadCalData()
  },

  onCalDayTap(e) {
    var date = e.currentTarget.dataset.date
    if (!date) return
    this.setData({ calSelDate: date })
    // 筛选选中日期的计划
    this._loadSchList(date)
  },

  _loadCalData() {
    var that = this
    var fid = this.data.currentFamilyId
    var isPersonal = this.data.schTab === 'personal'
    var url = isPersonal ? '/personal/schedules/calendar' : '/families/' + fid + '/schedules/calendar'
    api.get(url, { year: this.data.calYear, month: this.data.calMonth }, { showLoading: false, showError: false }
    ).then(function(data) {
      that.setData({ calDateStatus: data.dates || {} })
      that.buildCalendar()
    }).catch(function() {
      that.buildCalendar()
    })
  },

  /* ===== 计划编辑/删除 ===== */

  // 打开编辑弹窗
  editSchedule(e) {
    var id = parseInt(e.currentTarget.dataset.id)
    var s = this.data.schList.find(function(x){ return x.id === id })
    if (!s) return
    // 从 times 字符串解析回数组
    var times = []
    if (s.times) {
      times = (typeof s.times === 'string' ? s.times.split(' / ') : s.times) || []
    }
    this.setData({
      editSch: s,
      editDosage: String(s.dosage || ''),
      editUnit: s.unit || '粒',
      editQty: String(s.total_quantity || ''),
      editRel: s.relationship || '',
      editStart: (s.start_date || '').substring(0, 10),
      editEnd: (s.end_date || '').substring(0, 10),
      editTimes: times.length > 0 ? times : ['08:00'],
      savingEdit: false
    })
  },

  closeEditSch() { this.setData({ editSch: null }) },

  onEditDosage(e) { this.setData({ editDosage: e.detail }) },
  onEditUnit(e)    { this.setData({ editUnit: e.detail }) },
  onEditQty(e)     { this.setData({ editQty: e.detail }) },
  onEditRel(e)     { this.setData({ editRel: e.detail }) },
  onEditStart(e)   { this.setData({ editStart: e.detail }) },
  onEditEnd(e)     { this.setData({ editEnd: e.detail }) },
  onEditTimeChange(e) {
    var idx = e.currentTarget.dataset.idx
    var times = [...this.data.editTimes]
    times[idx] = e.detail
    this.setData({ editTimes: times })
  },
  addEditTime() { this.setData({ editTimes: [...this.data.editTimes, '12:00'] }) },
  delEditTime(e) {
    var idx = e.currentTarget.dataset.idx
    var times = [...this.data.editTimes]
    times.splice(idx, 1)
    if (times.length === 0) times = ['08:00']
    this.setData({ editTimes: times })
  },

  // 保存编辑
  async saveEditSch() {
    var s = this.data.editSch
    if (!s) return
    this.setData({ savingEdit: true })

    var isPersonal = this.data.schTab === 'personal'
    var fid = this.data.currentFamilyId
    var url = isPersonal
      ? '/personal/schedules/' + s.id
      : '/families/' + fid + '/schedules/' + s.id

    try {
      await api.put(url, {
        dosage: parseFloat(this.data.editDosage) || 0,
        unit: this.data.editUnit,
        total_quantity: parseInt(this.data.editQty) || 0,
        relationship: this.data.editRel,
        start_date: this.data.editStart,
        end_date: this.data.editEnd,
        times: this.data.editTimes,
        is_active: true  // 编辑后自动激活
      })
      wx.showToast({ title: '已更新', icon: 'success' })
      this.setData({ editSch: null, savingEdit: false })
      this._loadSchList()
    } catch (err) {
      this.setData({ savingEdit: false })
      wx.showToast({ title: '保存失败: ' + (err.message || ''), icon: 'none' })
    }
  },

  // 删除计划
  async deleteSchedule(e) {
    var that = this
    var id = parseInt(e.currentTarget.dataset.id)
    var s = this.data.schList.find(function(x){ return x.id === id })
    if (!s) return

    wx.showModal({
      title: '删除计划',
      content: '确定删除「' + (s.medicine_name || '') + '」的用药计划？',
      confirmColor: '#FA5151',
      success: async function(res) {
        if (!res.confirm) return
        var isPersonal = that.data.schTab === 'personal'
        var fid = that.data.currentFamilyId
        var url = isPersonal
          ? '/personal/schedules/' + id
          : '/families/' + fid + '/schedules/' + id
        try {
          await api.del(url)
          wx.showToast({ title: '已删除', icon: 'success' })
          that._loadSchList()
        } catch (err) {
          wx.showToast({ title: '删除失败', icon: 'none' })
        }
      }
    })
  }
});
