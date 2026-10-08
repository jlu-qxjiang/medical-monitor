// 药品记录页 — 日历 + 用药计划
const api = require('../../../utils/api')
const app = getApp()

Page({
  data: {
    calTab: 'family',   // family / personal
    calYear: new Date().getFullYear(),
    calMonth: new Date().getMonth() + 1,
    calWeekDays: ['日', '一', '二', '三', '四', '五', '六'],
    calDays: [],
    calDateStatus: {},
    selDate: '',
    schList: [],
    isLoading: false,
  },

  onLoad() {
    this._init()
  },

  onShow() {
    this._init()
  },

  _init() {
    var now = new Date()
    this.setData({ calYear: now.getFullYear(), calMonth: now.getMonth() + 1 })
    this.loadCalData()
  },

  /* ===== Tab ===== */
  switchCalTab(e) {
    var tab = e.currentTarget.dataset.tab
    this.setData({ calTab: tab, selDate: '', schList: [] })
    this.loadCalData()
  },

  /* ===== 日历构建 ===== */
  buildCalendar() {
    var now = new Date()
    var today = now.getFullYear() + '-' + String(now.getMonth() + 1).padStart(2, '0') + '-' + String(now.getDate()).padStart(2, '0')
    var year = this.data.calYear, month = this.data.calMonth
    var firstDay = new Date(year, month - 1, 1).getDay()
    var daysInMonth = new Date(year, month, 0).getDate()
    var days = []
    for (var i = 0; i < firstDay; i++) {
      days.push({ day: '', date: '', hasPlan: false, status: '' })
    }
    for (var d = 1; d <= daysInMonth; d++) {
      var dateStr = year + '-' + String(month).padStart(2, '0') + '-' + String(d).padStart(2, '0')
      var status = this.data.calDateStatus[dateStr] || ''
      days.push({
        day: d, date: dateStr, isToday: dateStr === today,
        hasPlan: !!status, status: status
      })
    }
    this.setData({ calDays: days })
  },

  prevMonth() {
    var m = this.data.calMonth - 1, y = this.data.calYear
    if (m < 1) { m = 12; y-- }
    this.setData({ calYear: y, calMonth: m, selDate: '', schList: [] })
    this.loadCalData()
  },
  nextMonth() {
    var m = this.data.calMonth + 1, y = this.data.calYear
    if (m > 12) { m = 1; y++ }
    this.setData({ calYear: y, calMonth: m, selDate: '', schList: [] })
    this.loadCalData()
  },

  onCalDayTap(e) {
    var date = e.currentTarget.dataset.date
    if (!date) return
    this.setData({ selDate: date })
    this.loadSchList(date)
  },

  onClosePopup() {
    this.setData({ selDate: '' })
  },

  /* ===== 数据加载 ===== */
  loadCalData() {
    var that = this
    var isPersonal = this.data.calTab === 'personal'
    var fid = app.globalData.currentFamilyId
    var year = this.data.calYear
    var month = this.data.calMonth
    if (!isPersonal && !fid) { this.buildCalendar(); return }
    var url = isPersonal ? '/personal/schedules/calendar' : '/families/' + fid + '/schedules/calendar'
    console.log('calling calendar API:', url, year, month)
    api.get(url, { year: year, month: month }, { showLoading: false, showError: false }
    ).then(function(data) {
      console.log('calendar API response:', JSON.stringify(data))
      that.setData({ calDateStatus: (data && data.dates) || {} })
      that.buildCalendar()
    }).catch(function(err) {
      console.log('calendar API error:', JSON.stringify(err))
      that.setData({ calDateStatus: {} })
      that.buildCalendar()
    })
  },

  async loadSchList(date) {
    this.setData({ isLoading: true })
    var isPersonal = this.data.calTab === 'personal'
    var fid = app.globalData.currentFamilyId
    var url = isPersonal ? '/personal/schedules' : '/families/' + fid + '/schedules'
    try {
      var data = await api.get(url, {}, { showLoading: false, showError: false })
      var today = new Date().toISOString().substring(0, 10)
      var all = (data.list || []).map(function(p) {
        var sd = (p.start_date || '').substring(0, 10)
        var ed = (p.end_date || '').substring(0, 10)
        return { ...p, times: (p.times || []).join(' / '), start_date: sd, end_date: ed, _isEnded: ed && ed < today }
      })
      // 按选中日期过滤：计划覆盖范围包含该日期
      var list = all.filter(function(p) {
        var sd = p.start_date
        var ed = p.end_date
        if (sd && sd > date) return false
        if (ed && ed < date) return false
        return true
      })
      this.setData({ schList: list, isLoading: false })
    } catch (e) {
      this.setData({ schList: [], isLoading: false })
    }
  }
})
