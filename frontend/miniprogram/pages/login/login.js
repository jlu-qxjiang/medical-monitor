const api = require('../../utils/api')

Page({
  data: {
    role: '',
    phone: '',
    password: '',
    password2: '',
    nickname: '',
    isRegister: false,
    showPwd: false,
    loading: false,
    errorMsg: ''
  },

  onLoad(options) {
    wx.setNavigationBarTitle({ title: '账号登录' })
    const role = options.role || ''
    if (!role) {
      const saved = wx.getStorageSync('userRole') || ''
      this.setData({ role: saved })
    } else {
      wx.setStorageSync('userRole', role)
      this.setData({ role })
    }
  },

  onPhoneInput(e) { this.setData({ phone: e.detail.value }) },
  onPwdInput(e)  { this.setData({ password: e.detail.value }) },
  onNickInput(e)  { this.setData({ nickname: e.detail.value }) },
  onPwd2Input(e) { this.setData({ password2: e.detail.value }) },
  togglePwd() { this.setData({ showPwd: !this.data.showPwd }) },

  // 切换登录/注册模式
  handleRegister() {
    this.setData({ isRegister: !this.data.isRegister, password: '', password2: '', errorMsg: '' })
  },

  // 微信一键登录
  async handleWechatLogin() {
    if (this.data.loading) return
    this.setData({ loading: true, errorMsg: '' })
    try {
      const code = await api.getWxLoginCode()
      const roleNum = this.data.role === 'elder' ? 1 : this.data.role === 'child' ? 2 : 3
      const result = await api.post('/auth/login', { code, role: roleNum }, { showLoading: false })

      // 微信登录返回的 user.role 是实际角色，用它更新
      const roleMap = { 1: 'elder', 2: 'child', 3: 'admin' }
      const actualRole = result.user && result.user.role ? roleMap[result.user.role] : this.data.role
      wx.setStorageSync('userRole', actualRole)
      this.setData({ role: actualRole })

      this._afterLogin(result)
    } catch (err) {
      this.setData({ loading: false, errorMsg: err.message || '微信登录失败' })
    }
  },

  // 登录或注册（主按钮）
  async handleLogin() {
    const { phone, password, nickname, isRegister, loading } = this.data
    if (loading) return
    if (!phone || phone.length < 11) {
      this.setData({ errorMsg: '请输入正确的手机号' })
      return
    }
    if (!password || password.length < 6) {
      this.setData({ errorMsg: '密码至少6位' })
      return
    }
    if (isRegister && !nickname) {
      this.setData({ errorMsg: '请填写昵称' })
      return
    }
    if (isRegister && password !== this.data.password2) {
      this.setData({ errorMsg: '两次密码不一致' })
      return
    }

    this.setData({ loading: true, errorMsg: '' })

    try {
      const url = isRegister ? '/auth/register' : '/auth/login-password'
      const roleNum = this.data.role === 'elder' ? 1 : this.data.role === 'child' ? 2 : 3
      const body = isRegister ? { phone, password, nickname, role: roleNum } : { phone, password }
      const result = await api.post(url, body)

      // 校验：选择的模式必须与账号角色匹配
      const roleMap = {1:'elder',2:'child',3:'admin'}
      const selectedRole = this.data.role // 从模式选择页传来的
      const actualRole = result.user && result.user.role ? roleMap[result.user.role] : ''

      if (selectedRole && actualRole && selectedRole !== actualRole) {
        // 注册时：以选择的模式为准，保存到后端
        if (isRegister) {
          wx.setStorageSync('userRole', selectedRole)
        } else {
          // 登录时：角色不匹配，拒绝
          this.setData({ loading: false, errorMsg: '该账号是' + (actualRole==='elder'?'老人':'子女') + '账号，与当前模式不匹配' })
          return
        }
      } else {
        wx.setStorageSync('userRole', actualRole || selectedRole)
        this.setData({ role: actualRole || selectedRole })
      }

      this._afterLogin(result)
    } catch (err) {
      this.setData({ loading: false, errorMsg: err.message || '操作失败' })
    }
  },

  _afterLogin(result) {
    const { access_token, refresh_token, user } = result

    const app = getApp()
    app.updateLoginState({ access_token, refresh_token }, user)
    const role = this.data.role
    console.log('login _afterLogin role:', role)
    app.globalData.userRole = role

    this.setData({ loading: false })

    // 先检查是否有家庭组
    api.get('/families', {}, { skipAuth: false }).then(data => {
      const list = data.list || []
      if (list.length > 0 && !list[0].is_personal) {
        app.switchFamily(list[0])
      }

      // 所有角色都先去首页，首页根据 userRole 自动显示对应视图
      wx.switchTab({ url: '/pages/index/index' })
    }).catch(() => {
      wx.switchTab({ url: '/pages/index/index' })
    })
  }
})
