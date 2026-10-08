// 药品管理 — 家庭药品 / 个人药品
const api = require('../../../utils/api')
const app = getApp()

Page({
  data: {
    medTab: 'family',
    medicines: [],
    filteredMeds: [],
    isLoading: false,
    isElder: false,
    saving: false,

    // 表单
    showForm: false,
    formName: '',
    formSpec: '',
    formMfr: '',
    formNote: '',
    formStock: '30',
    formAlert: '7',
    formExpiry: '',
    addError: '',

    // 搜索
    keyword: '',

    // 药品详情
    detailMed: null,

    // 库存编辑
    editingStock: null,
    editStockVal: '0',
    savingStock: false
  },

  onLoad() {
    const role = wx.getStorageSync('userRole') || ''
    this.setData({ isElder: role === 'elder' })
  },

  onShow() {
    if (this.data.isElder) return
    this.setData({ showForm: false })
    this.loadMeds()
  },

  /* ===== Tab ===== */
  switchTab(e) {
    this.setData({ medTab: e.currentTarget.dataset.tab }, () => { this.loadMeds() })
  },

  /* ===== 表单 ===== */
  toggleForm() { this.setData({ showForm: !this.data.showForm }) },

  onNameInput(e)  { this.setData({ formName: e.detail.value, addError: '' }) },
  onSpecInput(e)  { this.setData({ formSpec: e.detail.value }) },
  onMfrInput(e)   { this.setData({ formMfr: e.detail.value }) },
  onNoteInput(e)  { this.setData({ formNote: e.detail.value }) },
  onStockInput(e) { this.setData({ formStock: e.detail.value }) },
  onAlertInput(e) { this.setData({ formAlert: e.detail.value }) },
  onExpiryChange(e) { this.setData({ formExpiry: e.detail.value }) },
  onKeywordInput(e) {
    this.setData({ keyword: e.detail.value })
    this._filterMeds()
  },

  /* ===== 数据加载 ===== */
  async loadMeds() {
    this.setData({ isLoading: true })
    var isPersonal = this.data.medTab === 'personal'
    var fid = app.globalData.currentFamilyId
    if (!isPersonal && !fid) {
      this.setData({ medicines: [], filteredMeds: [], isLoading: false })
      return
    }
    var url = isPersonal ? '/personal/medicines' : '/families/' + fid + '/medicines'
    try {
      var data = await api.get(url, {}, { showLoading: false, showError: false })
      var meds = (data.list || []).map(function(m) {
        return {
          ...m,
          _lowStock: m.stock_remaining != null && m.stock_remaining <= (m.stock_alert_threshold || 7) && m.stock_remaining > 0,
          _expired: m.formExpiry && m.formExpiry < new Date().toISOString().slice(0, 10)
        }
      })
      this.setData({ medicines: meds, isLoading: false })
      this._filterMeds()
    } catch (err) {
      this.setData({ isLoading: false, medicines: [], filteredMeds: [] })
    }
  },

  _filterMeds() {
    var kw = (this.data.keyword || '').toLowerCase()
    var list = this.data.medicines.filter(function(m) {
      return !kw || (m.name || '').toLowerCase().includes(kw)
    })
    this.setData({ filteredMeds: list })
  },

  /* ===== 添加药品 ===== */
  async saveMed() {
    var name = this.data.formName.trim()
    if (!name) { this.setData({ addError: '请输入药品名称' }); return }
    var isPersonal = this.data.medTab === 'personal'
    var fid = app.globalData.currentFamilyId
    if (!isPersonal && !fid) { this.setData({ addError: '请先在首页创建家庭组' }); return }

    this.setData({ saving: true, addError: '' })
    var url = isPersonal ? '/personal/medicines' : '/families/' + fid + '/medicines'
    try {
      await api.post(url, {
        name: name,
        specification: this.data.formSpec,
        manufacturer: this.data.formMfr,
        notes: this.data.formNote,
        expiry: this.data.formExpiry,
        initial_stock: { total_count: parseInt(this.data.formStock) || 30, alert_threshold: parseInt(this.data.formAlert) || 7 }
      }, { showLoading: false, showError: false })
      wx.showToast({ title: '已添加', icon: 'success' })
      this.setData({ formName: '', formSpec: '', formMfr: '', formNote: '', formStock: '30', formAlert: '7', formExpiry: '', saving: false, addError: '' })
      this.loadMeds()
    } catch (err) {
      this.setData({ addError: err.message || '添加失败', saving: false })
    }
  },

  /* ===== 药品详情 ===== */
  showDetail(e) {
    var id = e.currentTarget.dataset.id
    var med = this.data.medicines.find(function(m){ return m.id == id })
    if (!med) return
    this.setData({ detailMed: med })
  },
  closeDetail() { this.setData({ detailMed: null }) },
  detailToEdit() {
    var med = this.data.detailMed
    if (!med) return
    this.setData({ detailMed: null })
    this.editStock({ currentTarget: { dataset: { id: med.id } } })
  },

  /* ===== 三点菜单 ===== */
  onMedMore(e) {
    var that = this
    var item = e.currentTarget.dataset.item
    if (!item) return
    wx.showActionSheet({
      itemList: ['编辑信息', '编辑库存', '查看详情', '删除药品'],
      success(res) {
        if (res.tapIndex === 0) {
          that.editInfo({ currentTarget: { dataset: { id: item.id } } })
        } else if (res.tapIndex === 1) {
          that.editStock({ currentTarget: { dataset: { id: item.id } } })
        } else if (res.tapIndex === 2) {
          that.showDetail({ currentTarget: { dataset: { id: item.id } } })
        } else if (res.tapIndex === 3) {
          that.deleteMed({ currentTarget: { dataset: { id: item.id } } })
        }
      }
    })
  },

  /* ===== 编辑药品信息 ===== */
  editInfo(e) {
    var id = e.currentTarget.dataset.id
    var isPersonal = this.data.medTab === 'personal'
    var fid = app.globalData.currentFamilyId
    var url = isPersonal
      ? '/pages/medicine/add/add?id=' + id + '&is_personal=1'
      : '/pages/medicine/add/add?family_id=' + fid + '&id=' + id
    wx.navigateTo({ url: url })
  },

  /* ===== 删除药品 ===== */
  async deleteMed(e) {
    var id = e.currentTarget.dataset.id
    var isPersonal = this.data.medTab === 'personal'
    var fid = app.globalData.currentFamilyId
    var url = isPersonal ? ('/personal/medicines/' + id) : ('/families/' + fid + '/medicines/' + id)
    var res = await new Promise(function(r) { wx.showModal({ title: '删除', content: '确定删除？', success: r }) })
    if (!res.confirm) return
    try {
      await api.del(url, {}, { showLoading: false, showError: false })
      wx.showToast({ title: '已删除', icon: 'success' })
      this.loadMeds()
    } catch (err) {
      wx.showToast({ title: '删除失败', icon: 'none' })
    }
  },

  /* ===== 库存编辑 ===== */
  editStock(e) {
    var id = e.currentTarget.dataset.id
    var item = this.data.medicines.find(function(m){ return m.id == id })
    if (!item) return
    this.setData({ editingStock: item, editStockVal: '0' })
  },
  closeStockEdit() { this.setData({ editingStock: null }) },
  onStockValInput(e) { this.setData({ editStockVal: e.detail.value }) },

  async saveStock() {
    var item = this.data.editingStock
    if (!item) return
    this.setData({ savingStock: true })
    var isPersonal = this.data.medTab === 'personal'
    var fid = app.globalData.currentFamilyId
    var url = isPersonal
      ? '/personal/medicines/' + item.id + '/stock'
      : '/families/' + fid + '/stocks/' + item.id
    var addCount = parseInt(this.data.editStockVal) || 0
    if (addCount <= 0) { wx.showToast({ title: '请输入添加数量', icon: 'none' }); this.setData({ savingStock: false }); return }
    try {
      await api.put(url, {
        remaining_count: (item.stock_remaining || 0) + addCount,
        total_count: (item.stock_total || item.stock_remaining || 0) + addCount
      })
      wx.showToast({ title: '库存已更新', icon: 'success' })
      this.setData({ editingStock: null, savingStock: false })
      await this.loadMeds()
    } catch (err) {
      wx.showToast({ title: '更新失败', icon: 'none' })
      this.setData({ savingStock: false })
    }
  }
})
