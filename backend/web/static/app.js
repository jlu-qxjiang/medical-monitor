// 按时吃药 — 前端公共方法
const API_BASE = '';
let isRefreshing = false;
let refreshQueue = [];

async function api(path, opts = {}) {
  const headers = { 'Content-Type': 'application/json' };
  const token = sessionStorage.getItem('token');
  if (token) headers['Authorization'] = 'Bearer ' + token;
  try {
    const r = await fetch(API_BASE + path, { ...opts, headers: { ...headers, ...(opts.headers || {}) } });
    if (r.status === 401 && token) {
      // Token 过期，尝试刷新
      return refreshAndRetry(path, opts);
    }
    return await r.json();
  } catch (e) {
    return { code: -1, message: '网络错误' };
  }
}

async function refreshAndRetry(path, opts) {
  if (!isRefreshing) {
    isRefreshing = true;
    const refreshToken = sessionStorage.getItem('refreshToken');
    if (!refreshToken) { logout(); return { code: 1001, message: '请重新登录' }; }
    const res = await fetch(API_BASE + '/api/v1/auth/refresh', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken })
    });
    if (res.status === 200) {
      const data = await res.json();
      if (data.code === 0) {
        sessionStorage.setItem('token', data.data.access_token);
        sessionStorage.setItem('refreshToken', data.data.refresh_token);
        isRefreshing = false;
        // 重试队列中的请求
        refreshQueue.forEach(cb => cb());
        refreshQueue = [];
        // 重试当前请求
        return api(path, opts);
      }
    }
    isRefreshing = false;
    refreshQueue = [];
    logout();
    return { code: 1001, message: '请重新登录' };
  } else {
    // 正在刷新中，排队等待
    return new Promise(resolve => {
      refreshQueue.push(() => resolve(api(path, opts)));
    });
  }
}

function toast(msg, ok = true) {
  const t = document.createElement('div');
  t.className = 'toast ' + (ok ? 'toast-ok' : 'toast-err');
  t.textContent = msg;
  document.body.appendChild(t);
  setTimeout(() => t.remove(), 2200);
}

function checkAuth() {
  const token = sessionStorage.getItem('token');
  if (!token) { location.href = '/static/login.html'; return null; }
  return {
    token,
    role: sessionStorage.getItem('role'),
    userId: +sessionStorage.getItem('userId'),
    nickname: sessionStorage.getItem('nickname'),
  };
}

function logout() {
  sessionStorage.clear();
  location.href = '/static/login.html';
}

// 弹窗工具
function closeModal(id) { document.getElementById(id).classList.remove('show'); }
function openModal(id) { document.getElementById(id).classList.add('show'); }

// 时间格式化
function fmtTime(t) { if (!t) return ''; return new Date(t).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }); }
function fmtDate(t) { if (!t) return ''; return new Date(t).toLocaleDateString('zh-CN'); }
