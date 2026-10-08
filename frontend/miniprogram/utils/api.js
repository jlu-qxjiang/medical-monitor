/**
 * API 请求封装
 * 统一 base URL、自动附加 Authorization header、处理 loading/error/auth 过期
 */

// ===== 环境配置 =====
const ENV = {
  develop: 'http://124.223.183.233:8080/api/v1',
  trial: 'https://api-dev.anshichiyao.com/api/v1',
  release: 'https://api.anshichiyao.com/api/v1'
};
// 根据小程序环境自动切换 base URL
function getBaseUrl() {
  const accountInfo = wx.getAccountInfoSync ? wx.getAccountInfoSync() : null;
  if (accountInfo && accountInfo.miniProgram) {
    const envVersion = accountInfo.miniProgram.envVersion;
    return ENV[envVersion] || ENV.develop;
  }
  return ENV.develop;
}

const BASE_URL = getBaseUrl();

// ===== 请求队列（用于 token 刷新时的请求暂存） =====
let isRefreshing = false;
let refreshSubscribers = [];

function onTokenRefreshed(newToken) {
  refreshSubscribers.forEach(callback => callback(newToken));
  refreshSubscribers = [];
}

function addRefreshSubscriber(callback) {
  refreshSubscribers.push(callback);
}

// ===== 获取微信登录 code =====
function getWxLoginCode() {
  return new Promise((resolve, reject) => {
    wx.login({
      success: (res) => {
        if (res.code) {
          resolve(res.code);
        } else {
          reject(new Error('wx.login 未返回 code'));
        }
      },
      fail: (err) => {
        reject(err);
      }
    });
  });
}

// ===== 核心请求方法 =====
function request(options) {
  const {
    url,
    method = 'GET',
    data = {},
    header = {},
    showLoading = true,
    loadingText = '加载中...',
    skipAuth = false,
    timeout = 30000
  } = options;

  // loading 控制
  if (showLoading) {
    wx.showLoading({ title: loadingText, mask: true });
  }

  // 构建请求头
  const headers = { ...header };
  if (!skipAuth) {
    const app = getApp();
    if (app && app.globalData && app.globalData.accessToken) {
      headers['Authorization'] = `Bearer ${app.globalData.accessToken}`;
    }
  }

  return new Promise((resolve, reject) => {
    wx.request({
      url: `${BASE_URL}${url}`,
      method,
      data,
      header: headers,
      timeout,
      success: (res) => {
        if (showLoading) {
          wx.hideLoading();
        }

        const statusCode = res.statusCode;
        const body = res.data;

        if (statusCode === 200 && body && body.code === 0) {
          // 成功
          resolve(body.data, body);
        } else if (statusCode === 401 || (body && body.code === 1001)) {
          // Token 过期，尝试刷新
          _handleTokenExpired(options, resolve, reject);
        } else if (statusCode === 403) {
          wx.showToast({ title: '无操作权限', icon: 'none' });
          reject(new Error('无操作权限'));
        } else if (statusCode === 404) {
          wx.showToast({ title: '资源不存在', icon: 'none' });
          reject(new Error('资源不存在'));
        } else if (statusCode === 429) {
          wx.showToast({ title: '操作太频繁，请稍后再试', icon: 'none' });
          reject(new Error('请求频率超限'));
        } else if (statusCode >= 500) {
          wx.showToast({ title: '服务器繁忙，请稍后再试', icon: 'none' });
          reject(new Error('服务器错误'));
        } else {
          // 其他业务错误
          const msg = (body && body.message) || '请求失败';
          if (options.showError !== false) {
            wx.showToast({ title: msg, icon: 'none' });
          }
          reject(new Error(msg));
        }
      },
      fail: (err) => {
        if (showLoading) {
          wx.hideLoading();
        }

        const msg = _getNetworkErrorMsg(err);
        if (options.showError !== false) {
          wx.showToast({ title: msg, icon: 'none' });
        }
        reject(err);
      }
    });
  });
}

// ===== Token 过期处理 =====
function _handleTokenExpired(originalOptions, resolve, reject) {
  const app = getApp();
  const refreshToken = app && app.globalData ? app.globalData.refreshToken : '';

  if (!refreshToken) {
    // 无 refresh token，重新登录
    app.logout();
    reject(new Error('登录已过期，请重新登录'));
    return;
  }

  if (!isRefreshing) {
    isRefreshing = true;

    // 调用刷新接口
    wx.request({
      url: `${BASE_URL}/auth/refresh`,
      method: 'POST',
      data: { refresh_token: refreshToken },
      success: (res) => {
        const body = res.data;
        if (res.statusCode === 200 && body && body.code === 0) {
          const tokenData = body.data;
          app.globalData.accessToken = tokenData.access_token;
          app.globalData.refreshToken = tokenData.refresh_token;
          app.globalData.expiresIn = tokenData.expires_in;

          // 更新本地存储
          const auth = require('./auth');
          auth.setToken({
            accessToken: tokenData.access_token,
            refreshToken: tokenData.refresh_token,
            expiresIn: tokenData.expires_in
          });

          onTokenRefreshed(tokenData.access_token);
        } else {
          onTokenRefreshed(null);
          app.logout();
        }
        isRefreshing = false;
      },
      fail: () => {
        onTokenRefreshed(null);
        isRefreshing = false;
        app.logout();
      }
    });
  }

  // 将当前请求加入等待队列
  addRefreshSubscriber((newToken) => {
    if (newToken) {
      // 重试原请求
      request({ ...originalOptions, showLoading: false })
        .then(resolve)
        .catch(reject);
    } else {
      reject(new Error('登录已过期，请重新登录'));
    }
  });
}

// ===== 网络错误提示 =====
function _getNetworkErrorMsg(err) {
  if (err.errMsg) {
    if (err.errMsg.includes('timeout')) return '网络超时，请重试';
    if (err.errMsg.includes('fail')) return '网络异常，请检查网络连接';
  }
  return '网络异常，请重试';
}

// ===== 便捷方法 =====
function get(url, data = {}, options = {}) {
  return request({ url, method: 'GET', data, ...options });
}

function post(url, data = {}, options = {}) {
  return request({ url, method: 'POST', data, ...options });
}

function put(url, data = {}, options = {}) {
  return request({ url, method: 'PUT', data, ...options });
}

function del(url, data = {}, options = {}) {
  return request({ url, method: 'DELETE', data, ...options });
}

/**
 * 上传文件
 * @param {string} url - 上传接口路径
 * @param {string} filePath - 本地文件路径
 * @param {string} name - 文件对应的 key
 * @param {object} formData - 额外的表单数据
 */
function uploadFile(url, filePath, name = 'file', formData = {}) {
  return new Promise((resolve, reject) => {
    const app = getApp();
    wx.uploadFile({
      url: `${BASE_URL}${url}`,
      filePath,
      name,
      formData,
      header: {
        'Authorization': `Bearer ${app.globalData.accessToken}`
      },
      success: (res) => {
        try {
          const body = JSON.parse(res.data);
          if (body.code === 0) {
            resolve(body.data);
          } else {
            wx.showToast({ title: body.message || '上传失败', icon: 'none' });
            reject(new Error(body.message || '上传失败'));
          }
        } catch (e) {
          reject(new Error('解析响应失败'));
        }
      },
      fail: (err) => {
        wx.showToast({ title: '上传失败，请重试', icon: 'none' });
        reject(err);
      }
    });
  });
}

module.exports = {
  getBaseUrl,
  getWxLoginCode,
  request,
  get,
  post,
  put,
  del,
  uploadFile
};
