const app = getApp()
const api = require('../../../utils/api')

Page({
  data: {
    nickname: '用户',
    role: '',
    familyName: '暂无',
    familyId: 0,
    isPersonal: false,
    inviteCode: '',
    myRole: '',
    members: [],
    roleOptions: ['老人', '子女'],
    isLoading: false
  },

  onShow() {
    var role = wx.getStorageSync('userRole') || ''
    var info = app.globalData.userInfo || {}
    this.setData({ nickname: info.nickname || '用户', role: role })
    this.loadFamily()
  },

  async loadFamily() {
    var fid = app.globalData.currentFamilyId
    if (!fid) {
      var famData = null
      try {
        var listData = await api.get('/families', {}, { showLoading: false, showError: false })
        var list = listData.list || []
        var shared = list.filter(function(f){ return !f.is_personal })
        if (shared.length > 0) {
          famData = shared[0]
          app.globalData.currentFamilyId = famData.id
        }
      } catch(e) {}
      if (!famData) {
        this.setData({ familyName: '暂无家庭组', familyId: 0, members: [], myRole: '' })
        return
      }
      fid = famData.id
    }
    this.setData({ isLoading: true })
    try {
      var data = await api.get('/families/' + fid, {}, { showLoading: false, showError: false })
      var myRole = ''
      var members = data.members || []
      var uid = (app.globalData.userInfo || {}).id
      var roleIdx = {elder:0, member:1}
      var roleText = {elder:'老人', member:'子女'}
      members.forEach(function(m){
        if(m.user_id===uid) myRole=m.role
        m._roleIdx = roleIdx[m.role] != null ? roleIdx[m.role] : 1
        m._roleText = roleText[m.role] || '子女'
      })
      this.setData({
        familyName: data.name || '家庭',
        familyId: data.id,
        isPersonal: data.is_personal || false,
        inviteCode: data.invite_code || '',
        myRole: myRole,
        members: members,
        isLoading: false
      })
    } catch (err) {
      this.setData({ isLoading: false, familyName: '加载失败' })
    }
  },

  createFamily() {
    wx.navigateTo({ url: '/pages/family/create/create' })
  },

  deleteFamily() {
    var that = this
    wx.showModal({
      title: '解散家庭组',
      content: '确定解散「'+that.data.familyName+'」？此操作不可恢复。',
      confirmColor: '#FA5151',
      success: async function(res) {
        if (!res.confirm) return
        try {
          // 先移除所有成员再删除
          for (var i=0; i<that.data.members.length; i++) {
            var m = that.data.members[i]
            if (m.user_id !== (app.globalData.userInfo||{}).id) {
              await api.del('/families/'+that.data.familyId+'/members/'+m.user_id, {}, { showLoading: false, showError: false })
            }
          }
          await api.del('/families/'+that.data.familyId+'/leave', {}, { showLoading: false, showError: false })
          wx.showToast({ title: '已解散', icon: 'success' })
          app.globalData.currentFamilyId = null
          that.loadFamily()
        } catch(e) {
          wx.showToast({ title: '解散失败', icon: 'none' })
        }
      }
    })
  },

  leaveFamily() {
    var that = this
    wx.showModal({
      title: '退出家庭组',
      content: '确定退出「'+that.data.familyName+'」？',
      confirmColor: '#FA5151',
      success: async function(res) {
        if (!res.confirm) return
        try {
          await api.del('/families/'+that.data.familyId+'/leave', {}, { showLoading: false, showError: false })
          wx.showToast({ title: '已退出', icon: 'success' })
          app.globalData.currentFamilyId = null
          that.loadFamily()
        } catch(e) {
          wx.showToast({ title: '退出失败', icon: 'none' })
        }
      }
    })
  },

  async changeRole(e) {
    var uid = e.currentTarget.dataset.uid
    var idx = parseInt(e.detail.value)
    var roles = ['elder', 'member']
    var newRole = roles[idx]
    try {
      await api.put('/families/'+this.data.familyId+'/members/'+uid+'/role', { role: newRole }, { showLoading: false, showError: false })
      wx.showToast({ title: '已更新', icon: 'success' })
      this.loadFamily()
    } catch(err) {
      wx.showToast({ title: '修改失败', icon: 'none' })
    }
  },

  async refreshInvite() {
    try {
      var data = await api.post('/families/'+this.data.familyId+'/refresh-invite', {}, { showLoading:false })
      this.setData({ inviteCode: data.invite_code })
      wx.showToast({ title: '已刷新', icon: 'success' })
    } catch(e) { wx.showToast({ title: '刷新失败', icon: 'none' }) }
  },

  // 编辑资料
  openEditProfile() {
    this.setData({ showEditProfile: true, editNickname: this.data.nickname })
  },
  closeEditProfile() { this.setData({ showEditProfile: false }) },
  onEditNickname(e) { this.setData({ editNickname: e.detail }) },

  async saveProfile() {
    var name = (this.data.editNickname || '').trim()
    if (!name) { wx.showToast({ title: '请输入昵称', icon: 'none' }); return }
    this.setData({ savingProfile: true })
    try {
      await api.put('/user/profile', { nickname: name }, { showLoading: false, showError: false })
      wx.showToast({ title: '已更新', icon: 'success' })
      this.setData({ nickname: name, showEditProfile: false, savingProfile: false })
      if (app.globalData.userInfo) app.globalData.userInfo.nickname = name
    } catch (err) {
      wx.showToast({ title: '修改失败', icon: 'none' })
      this.setData({ savingProfile: false })
    }
  },

  goNotification() {
    wx.navigateTo({ url: '/pages/settings/notification/notification' })
  },

  handleLogout() {
    wx.showModal({
      title: '退出登录',
      content: '确定退出？',
      success: function(res) {
        if (res.confirm) {
          wx.clearStorageSync()
          app.globalData.isLoggedIn = false
          wx.reLaunch({ url: '/pages/index/index' })
        }
      }
    })
  }
})
