// 自定义 TabBar — 老人2项 / 子女3项
Component({
  data: {
    selected: 0,
    list: []
  },

  lifetimes: {
    attached() { this._update() }
  },

  pageLifetimes: {
    show() { this._update() }
  },

  methods: {
    _update() {
      const app = getApp()
      const role = (app && app.globalData && app.globalData.userRole) || ''
      console.log('tabbar _update role:', role)

      let list = []
      if (role === 'elder') {
        list = [
          { pagePath: '/pages/index/index', text: '首页', icon: '🏠' },
          { pagePath: '/pages/settings/profile/profile', text: '我的', icon: '👤' }
        ]
      } else if (role === 'child' || role === 'admin') {
        list = [
          { pagePath: '/pages/index/index', text: '首页', icon: '🏠' },
          { pagePath: '/pages/medicine/list/list', text: '药品', icon: '💊' },
          { pagePath: '/pages/settings/profile/profile', text: '我的', icon: '👤' }
        ]
      }
      // role 为空 → list 空 → TabBar 隐藏
      this.setData({ list })
      this._updateSelected()
    },

    _updateSelected() {
      const pages = getCurrentPages()
      if (pages.length === 0) return
      const route = '/' + pages[pages.length - 1].route
      const list = this.data.list
      for (let i = 0; i < list.length; i++) {
        if (list[i].pagePath === route) {
          this.setData({ selected: i })
          return
        }
      }
    },

    switchTab(e) {
      const index = e.currentTarget.dataset.index
      const target = this.data.list[index]
      if (this.data.selected === index) return
      wx.switchTab({ url: target.pagePath })
    }
  }
})
