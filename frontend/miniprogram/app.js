/**
 * 「按时吃药」微信小程序入口
 * 全局状态管理：登录态、当前家庭组、当前用户角色
 */
const auth = require('./utils/auth');
const api = require('./utils/api');

App({
  globalData: {
    // 登录态
    isLoggedIn: false,
    accessToken: '',
    refreshToken: '',
    expiresIn: 0,

    // 当前用户信息
    userInfo: null,
    userId: null,

    // 当前家庭组
    currentFamily: null,
    currentFamilyId: null,

    // 用户角色（在家庭组中的角色）
    userRole: '', // 'owner' | 'admin' | 'member' | 'elder'

    // 全局配置
    baseUrl: api.getBaseUrl()
  },

  onLaunch() {
    // 检查本地 token 是否存在，尝试自动登录
    this._restoreSession();
  },

  onShow(options) {
    // 处理邀请码场景
    if (options.query && options.query.invite_code) {
      this.globalData.pendingInviteCode = options.query.invite_code;
    }
  },

  /**
   * 从本地缓存恢复登录态
   */
  _restoreSession() {
    const tokenData = auth.getToken();
    if (tokenData && tokenData.accessToken) {
      const isExpired = auth.isTokenExpired(tokenData);
      if (!isExpired) {
        this.globalData.accessToken = tokenData.accessToken;
        this.globalData.refreshToken = tokenData.refreshToken;
        this.globalData.expiresIn = tokenData.expiresIn;
        this.globalData.isLoggedIn = true;

        // 恢复用户信息
        const userInfo = auth.getUserInfo();
        if (userInfo) {
          this.globalData.userInfo = userInfo;
          this.globalData.userId = userInfo.id;
        }

        // 恢复当前家庭组
        const familyInfo = auth.getCurrentFamily();
        if (familyInfo) {
          this.globalData.currentFamily = familyInfo;
          this.globalData.currentFamilyId = familyInfo.id;
          this.globalData.userRole = familyInfo.my_role || '';
        }
      }
    }
  },

  /**
   * 更新全局登录态
   */
  updateLoginState(tokenData, userInfo) {
    this.globalData.isLoggedIn = true;
    this.globalData.accessToken = tokenData.access_token;
    this.globalData.refreshToken = tokenData.refresh_token;
    this.globalData.expiresIn = tokenData.expires_in;
    this.globalData.userInfo = userInfo;
    this.globalData.userId = userInfo.id;

    auth.setToken({
      accessToken: tokenData.access_token,
      refreshToken: tokenData.refresh_token,
      expiresIn: tokenData.expires_in
    });
    auth.setUserInfo(userInfo);
  },

  /**
   * 切换当前家庭组
   */
  switchFamily(family) {
    this.globalData.currentFamily = family;
    this.globalData.currentFamilyId = family.id;
    this.globalData.userRole = family.my_role || '';
    auth.setCurrentFamily(family);
  },

  /**
   * 退出登录
   */
  logout() {
    this.globalData.isLoggedIn = false;
    this.globalData.accessToken = '';
    this.globalData.refreshToken = '';
    this.globalData.expiresIn = 0;
    this.globalData.userInfo = null;
    this.globalData.userId = null;
    this.globalData.currentFamily = null;
    this.globalData.currentFamilyId = null;
    this.globalData.userRole = '';

    auth.clearAll();
    wx.reLaunch({ url: '/pages/login/login' });
  },

  /**
   * 检查是否已登录，未登录则跳转登录页
   */
  checkLogin() {
    if (!this.globalData.isLoggedIn) {
      wx.reLaunch({ url: '/pages/login/login' });
      return false;
    }
    return true;
  }
});
