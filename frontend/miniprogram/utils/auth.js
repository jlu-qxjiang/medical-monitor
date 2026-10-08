/**
 * 登录鉴权工具
 * wx.login() 封装、token 存储/读取、过期检测、自动刷新
 */

const STORAGE_KEYS = {
  ACCESS_TOKEN: 'auth_access_token',
  REFRESH_TOKEN: 'auth_refresh_token',
  EXPIRES_IN: 'auth_expires_in',
  EXPIRES_AT: 'auth_expires_at',
  USER_INFO: 'auth_user_info',
  CURRENT_FAMILY: 'current_family'
};

/**
 * 调起微信登录，获取 code
 */
function wxLogin() {
  return new Promise((resolve, reject) => {
    wx.login({
      success: (res) => {
        if (res.code) {
          resolve(res.code);
        } else {
          reject(new Error('微信登录失败'));
        }
      },
      fail: (err) => {
        reject(err);
      }
    });
  });
}

/**
 * 存储 token 到本地
 */
function setToken(tokenData) {
  const { accessToken, refreshToken, expiresIn } = tokenData;
  const expiresAt = Date.now() + expiresIn * 1000;

  wx.setStorageSync(STORAGE_KEYS.ACCESS_TOKEN, accessToken);
  wx.setStorageSync(STORAGE_KEYS.REFRESH_TOKEN, refreshToken);
  wx.setStorageSync(STORAGE_KEYS.EXPIRES_IN, expiresIn);
  wx.setStorageSync(STORAGE_KEYS.EXPIRES_AT, expiresAt);
}

/**
 * 获取本地存储的 token
 */
function getToken() {
  const accessToken = wx.getStorageSync(STORAGE_KEYS.ACCESS_TOKEN);
  const refreshToken = wx.getStorageSync(STORAGE_KEYS.REFRESH_TOKEN);
  const expiresIn = wx.getStorageSync(STORAGE_KEYS.EXPIRES_IN);
  const expiresAt = wx.getStorageSync(STORAGE_KEYS.EXPIRES_AT);

  if (accessToken) {
    return { accessToken, refreshToken, expiresIn, expiresAt };
  }
  return null;
}

/**
 * 获取 access token
 */
function getAccessToken() {
  return wx.getStorageSync(STORAGE_KEYS.ACCESS_TOKEN) || '';
}

/**
 * 判断 token 是否已过期
 */
function isTokenExpired(tokenData) {
  if (!tokenData || !tokenData.expiresAt) return true;
  // 提前 5 分钟刷新
  return Date.now() > tokenData.expiresAt - 5 * 60 * 1000;
}

/**
 * 清除本地 token
 */
function clearToken() {
  wx.removeStorageSync(STORAGE_KEYS.ACCESS_TOKEN);
  wx.removeStorageSync(STORAGE_KEYS.REFRESH_TOKEN);
  wx.removeStorageSync(STORAGE_KEYS.EXPIRES_IN);
  wx.removeStorageSync(STORAGE_KEYS.EXPIRES_AT);
}

/**
 * 存储用户信息
 */
function setUserInfo(userInfo) {
  wx.setStorageSync(STORAGE_KEYS.USER_INFO, JSON.stringify(userInfo));
}

/**
 * 获取用户信息
 */
function getUserInfo() {
  const raw = wx.getStorageSync(STORAGE_KEYS.USER_INFO);
  if (raw) {
    try {
      return JSON.parse(raw);
    } catch (e) {
      return null;
    }
  }
  return null;
}

/**
 * 存储当前家庭组
 */
function setCurrentFamily(family) {
  wx.setStorageSync(STORAGE_KEYS.CURRENT_FAMILY, JSON.stringify(family));
}

/**
 * 获取当前家庭组
 */
function getCurrentFamily() {
  const raw = wx.getStorageSync(STORAGE_KEYS.CURRENT_FAMILY);
  if (raw) {
    try {
      return JSON.parse(raw);
    } catch (e) {
      return null;
    }
  }
  return null;
}

/**
 * 检查登录状态
 */
function checkLoginStatus() {
  const tokenData = getToken();
  if (!tokenData || !tokenData.accessToken) {
    return false;
  }
  return !isTokenExpired(tokenData);
}

/**
 * 清除所有本地存储的登录信息
 */
function clearAll() {
  clearToken();
  wx.removeStorageSync(STORAGE_KEYS.USER_INFO);
  wx.removeStorageSync(STORAGE_KEYS.CURRENT_FAMILY);
}

module.exports = {
  wxLogin,
  setToken,
  getToken,
  getAccessToken,
  isTokenExpired,
  clearToken,
  setUserInfo,
  getUserInfo,
  setCurrentFamily,
  getCurrentFamily,
  checkLoginStatus,
  clearAll
};
