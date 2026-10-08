const api = require('../../../utils/api')
const app = getApp()

Page({
  data: {
    familyId: 0, schTab: 'family', elderUserId: 0,
    schedules: [], medicines: [], elders: [],
    elderNames: [], selElderId: 0, selElderName: '',
    isLoading: true,
    selMedId: 0, selMedName: '',
    formQty: '30', formDosage: '1', formUnit: '粒',
    units: ['粒','片','毫升','袋'],
    formRel: '', rels: ['不限','饭前','饭后','饭后1小时','饭后2小时','空腹','睡前'],
    formTimes: ['08:00'],
    formStart: '', formEnd: '', addErr: ''
  },

  onLoad(options) {
    if (!app.checkLogin()) return
    var tab = options.tab || 'family'
    this.setData({
      familyId: parseInt(options.fid) || app.globalData.currentFamilyId || 0,
      elderUserId: tab === 'personal' ? ((app.globalData.userInfo||{}).id || 0) : 0,
      schTab: tab
    })
  },

  onShow() {
    if (app.globalData.isLoggedIn) this.loadData()
  },

  async loadData() {
    this.setData({ isLoading: true })
    var isPersonal = this.data.schTab === 'personal'
    var fid = this.data.familyId
    var now = new Date()
    var sd = now.getFullYear()+'-'+(now.getMonth()+1).toString().padStart(2,'0')+'-'+now.getDate().toString().padStart(2,'0')
    var ed = new Date(now.getTime()+7*86400000)
    var es = ed.getFullYear()+'-'+(ed.getMonth()+1).toString().padStart(2,'0')+'-'+ed.getDate().toString().padStart(2,'0')
    this.setData({ formStart: sd, formEnd: es })

    try {
      var sRes = await api.get(isPersonal?'/personal/schedules':('/families/'+fid+'/schedules'), {}, { showLoading:false, showError:false })
      var mRes = await api.get(isPersonal?'/personal/medicines':('/families/'+fid+'/medicines'), {page:1,page_size:100}, { showLoading:false, showError:false })

      // 家庭模式加载成员（选老人）
      if (!isPersonal && fid > 0) {
        var fRes = await api.get('/families/'+fid, {}, { showLoading:false, showError:false })
        var members = (fRes.members||[]).filter(function(m){ return m.user_id !== ((app.globalData.userInfo||{}).id) })
        this.setData({
          elders: members,
          elderNames: members.map(function(m){ return (m.nickname||'用户'+m.user_id)+' ['+m.role+']' }),
          selElderId: 0, selElderName: ''
        })
      }

      this.setData({
        schedules: (sRes.list||[]).map(function(p){
          return { ...p, times: (p.times||[]).join(' / '), start_date: (p.start_date||'').substring(0,10), end_date: (p.end_date||'').substring(0,10) }
        }),
        medicines: mRes.list || [],
        isLoading: false
      })
    } catch(e) {
      this.setData({ isLoading: false })
    }
  },

  onElderChange(e) {
    var m = this.data.elders[parseInt(e.detail.value)]
    if (m) this.setData({ selElderId: m.user_id, selElderName: m.nickname||'用户'+m.user_id })
  },

  pickMed(e) {
    var id = parseInt(e.currentTarget.dataset.id)
    var med = this.data.medicines.find(function(m){ return m.id === id })
    var stock = 30
    if (med) {
      if (med.stock && med.stock.remaining_count > 0) { stock = med.stock.remaining_count }
      else if (med.stock_remaining > 0) { stock = med.stock_remaining }
    }
    this.setData({ selMedId: id, selMedName: e.currentTarget.dataset.name, formQty: String(stock), addErr: '' })
  },

  onQtyInput(e) { this.setData({ formQty: e.detail.value }) },
  onDosageInput(e) { this.setData({ formDosage: e.detail.value }) },
  onUnitChange(e) { this.setData({ formUnit: this.data.units[parseInt(e.detail.value)] }) },
  onRelChange(e) { this.setData({ formRel: this.data.rels[parseInt(e.detail.value)] }) },

  onStartChange(e) { this.setData({ formStart: e.detail.value }) },
  onEndChange(e) { this.setData({ formEnd: e.detail.value }) },

  onTimeItemChange(e) {
    var idx = parseInt(e.currentTarget.dataset.idx)
    var times = this.data.formTimes.slice()
    times[idx] = e.detail.value
    this.setData({ formTimes: times })
  },

  addTime() {
    this.setData({ formTimes: this.data.formTimes.concat(['08:00']) })
  },

  delTime(e) {
    var idx = parseInt(e.currentTarget.dataset.idx)
    var times = this.data.formTimes.slice()
    times.splice(idx, 1)
    this.setData({ formTimes: times.length>0 ? times : ['08:00'] })
  },

  async handleSubmit() {
    var d = this.data
    if (!d.selMedId) { this.setData({ addErr: '请选择药品' }); return }
    if (d.schTab !== 'personal' && !d.selElderId) { this.setData({ addErr: '请选择服药老人' }); return }
    if (!d.formStart || !d.formEnd) { this.setData({ addErr: '请选择起止日期' }); return }

    var isPersonal = d.schTab === 'personal'
    var body = {
      elder_user_id: isPersonal ? ((app.globalData.userInfo||{}).id) : d.selElderId,
      medicine_id: d.selMedId,
      total_quantity: parseInt(d.formQty) || 30,
      dosage: parseFloat(d.formDosage) || 1,
      unit: d.formUnit,
      relationship: d.formRel,
      start_date: d.formStart,
      end_date: d.formEnd,
      times: d.formTimes.filter(function(t){ return t })
    }

    var url = isPersonal ? '/personal/schedules' : ('/families/' + d.familyId + '/schedules')
    try {
      await api.post(url, body, { showLoading: true, loadingText: '保存中...', showError: false })
      wx.showToast({ title: '已添加', icon: 'success' })
      this.setData({ selMedId:0, selMedName:'', selElderId:0, selElderName:'', formQty:'30', formDosage:'1', formUnit:'粒', formRel:'', formTimes:['08:00'], addErr:'' })
      this.loadData()
    } catch(e) {
      this.setData({ addErr: e.message || '保存失败' })
    }
  },

  showDetail(e) {
    var id = parseInt(e.currentTarget.dataset.id)
    var s = this.data.schedules.find(function(x){ return x.id === id })
    if (!s) return
    var info = '药品: '+s.medicine_name+' '+s.dosage+s.unit
             + '\n时间: '+(s.times||'')
             + '\n总量: '+s.total_quantity
             + '\n起止: '+s.start_date+' ~ '+s.end_date
    wx.showModal({ title: '计划详情', content: info, showCancel: false, confirmText: '关闭' })
  },

  async handleDelete(e) {
    var id = e.currentTarget.dataset.id
    var isPersonal = this.data.schTab === 'personal'
    var url = isPersonal?('/personal/schedules/'+id):('/families/'+this.data.familyId+'/schedules/'+id)
    var res = await new Promise(function(r){ wx.showModal({title:'删除',content:'确定删除？',success:r}) })
    if (!res.confirm) return
    try {
      await api.del(url, {}, { showLoading:false, showError:false })
      wx.showToast({ title:'已删除', icon:'success' })
      this.loadData()
    } catch(e) { wx.showToast({ title:'删除失败', icon:'none' }) }
  }
})
