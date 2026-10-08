/**
 * 日期格式化工具函数
 */

/**
 * 格式化日期为 YYYY-MM-DD
 */
function formatDate(date) {
  const d = _toDate(date);
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

/**
 * 格式化日期时间为 YYYY-MM-DD HH:mm
 */
function formatDateTime(date) {
  const d = _toDate(date);
  const dateStr = formatDate(d);
  const hours = String(d.getHours()).padStart(2, '0');
  const minutes = String(d.getMinutes()).padStart(2, '0');
  return `${dateStr} ${hours}:${minutes}`;
}

/**
 * 格式化时间为 HH:mm
 */
function formatTime(date) {
  const d = _toDate(date);
  const hours = String(d.getHours()).padStart(2, '0');
  const minutes = String(d.getMinutes()).padStart(2, '0');
  return `${hours}:${minutes}`;
}

/**
 * 格式化中文日期（如：2026年7月6日）
 */
function formatDateCN(date) {
  const d = _toDate(date);
  return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`;
}

/**
 * 格式化中文日期时间（如：7月6日 08:00）
 */
function formatShortCN(date) {
  const d = _toDate(date);
  return `${d.getMonth() + 1}月${d.getDate()}日 ${formatTime(d)}`;
}

/**
 * 获取相对时间描述（如：刚刚 / 5分钟前 / 3小时前 / 昨天 / 3天前）
 */
function getRelativeTime(date) {
  const d = _toDate(date);
  const now = new Date();
  const diff = now - d;
  const seconds = Math.floor(diff / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (seconds < 60) return '刚刚';
  if (minutes < 60) return `${minutes}分钟前`;
  if (hours < 24) return `${hours}小时前`;
  if (days === 1) return '昨天';
  if (days < 7) return `${days}天前`;
  return formatDate(d);
}

/**
 * 判断是否是今天
 */
function isToday(date) {
  const d = _toDate(date);
  const today = new Date();
  return formatDate(d) === formatDate(today);
}

/**
 * 判断是否已过期（与当前时间对比）
 */
function isPast(date) {
  const d = _toDate(date);
  return d < new Date();
}

/**
 * 计算两个日期相差的天数
 */
function diffDays(date1, date2) {
  const d1 = new Date(formatDate(date1));
  const d2 = new Date(formatDate(date2));
  return Math.floor((d2 - d1) / (1000 * 60 * 60 * 24));
}

/**
 * 获取某一天的开始时间（00:00:00）
 */
function startOfDay(date) {
  const d = _toDate(date);
  d.setHours(0, 0, 0, 0);
  return d;
}

/**
 * 获取某一天的结束时间（23:59:59）
 */
function endOfDay(date) {
  const d = _toDate(date);
  d.setHours(23, 59, 59, 999);
  return d;
}

/**
 * 获取 N 天前的日期
 */
function daysAgo(n, baseDate) {
  const d = _toDate(baseDate || new Date());
  d.setDate(d.getDate() - n);
  return formatDate(d);
}

/**
 * 获取本周一的日期
 */
function getMonday(date) {
  const d = _toDate(date || new Date());
  const day = d.getDay();
  const diff = d.getDate() - day + (day === 0 ? -6 : 1);
  return formatDate(new Date(d.setDate(diff)));
}

/**
 * 获取本周日的日期
 */
function getSunday(date) {
  const d = _toDate(date || new Date());
  const day = d.getDay();
  const diff = d.getDate() - day + (day === 0 ? 0 : 7);
  return formatDate(new Date(d.setDate(diff)));
}

/**
 * 获取当前时间标签（早上/上午/中午/下午/晚上/深夜）
 */
function getTimeOfDayLabel(time) {
  const d = _toDate(time);
  const hour = d.getHours();
  if (hour < 6) return '凌晨';
  if (hour < 9) return '早上';
  if (hour < 12) return '上午';
  if (hour < 14) return '中午';
  if (hour < 18) return '下午';
  if (hour < 22) return '晚上';
  return '深夜';
}

/**
 * 将 ISO 时间字符串转换为本地时间
 */
function toLocalTime(isoString) {
  if (!isoString) return '';
  const d = new Date(isoString);
  return d;
}

/**
 * 内部：统一转为 Date 对象
 */
function _toDate(date) {
  if (date instanceof Date) return new Date(date);
  if (typeof date === 'string') return new Date(date.replace(/-/g, '/'));
  if (typeof date === 'number') return new Date(date);
  return new Date();
}

module.exports = {
  formatDate,
  formatDateTime,
  formatTime,
  formatDateCN,
  formatShortCN,
  getRelativeTime,
  isToday,
  isPast,
  diffDays,
  startOfDay,
  endOfDay,
  daysAgo,
  getMonday,
  getSunday,
  getTimeOfDayLabel,
  toLocalTime
};
