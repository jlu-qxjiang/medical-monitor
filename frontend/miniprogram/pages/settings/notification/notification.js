const api = require('../../../utils/api')

Page({
  data: {
    masterOn: true,
    reminderOn: true,
    stockOn: true
  },

  onLoad() {
    this.setData({
      masterOn: wx.getStorageSync('notify_master') !== false,
      reminderOn: wx.getStorageSync('notify_reminder') !== false,
      stockOn: wx.getStorageSync('notify_stock') !== false
    })
  },

  onMasterSwitch(e) {
    var v = e.detail.value
    this.setData({ masterOn: v })
    wx.setStorageSync('notify_master', v)
    if (!v) {
      // 总开关关闭 → 同时关闭子开关 + 调接口
      this.setData({ reminderOn: false, stockOn: false })
      wx.setStorageSync('notify_reminder', false)
      wx.setStorageSync('notify_stock', false)
      api.put('/user/settings', { notify_reminder: false, notify_stock: false })
    }
  },

  onReminderSwitch(e) {
    var v = e.detail.value
    this.setData({ reminderOn: v })
    wx.setStorageSync('notify_reminder', v)
    api.put('/user/settings', { notify_reminder: v })
  },

  onStockSwitch(e) {
    var v = e.detail.value
    this.setData({ stockOn: v })
    wx.setStorageSync('notify_stock', v)
    api.put('/user/settings', { notify_stock: v })
  }
})
