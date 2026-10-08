<template>
  <view class="page">
    <!-- 弹窗卡片 -->
    <view class="card">

      <!-- 顶部图标 + 标题 -->
      <view class="card-header">
        <!-- 圆形SVG：老人头像+红心 -->
        <view class="avatar-wrap">
          <svg viewBox="0 0 120 120" class="avatar-svg">
            <!-- 圆形背景 -->
            <circle cx="60" cy="60" r="60" fill="#ECF5FF" />
            <!-- 简化老人头像：圆头+身体 -->
            <circle cx="60" cy="42" r="20" fill="#C0C0C0" />
            <ellipse cx="60" cy="82" rx="28" ry="22" fill="#C0C0C0" />
            <!-- 红心 -->
            <path d="M88 35 C88 28 82 24 76 24 C71 24 67 27 65 31 C63 27 59 24 54 24 C48 24 42 28 42 35 C42 48 65 60 65 60 C65 60 88 48 88 35Z" fill="#FF4757" />
          </svg>
        </view>
        <text class="title">请选择您的角色</text>
      </view>

      <!-- 三个角色选项 -->
      <view class="roles">
        <!-- 老人模式 -->
        <view class="role-item role-elder" hover-class="role-hover" @tap="goLogin('elder')">
          <view class="role-bar bar-elder"></view>
          <view class="role-icon">👴</view>
          <view class="role-text">
            <text class="role-name">老人模式</text>
            <text class="role-desc">查看用药提醒 · 确认服药</text>
          </view>
          <text class="role-arrow">›</text>
        </view>

        <!-- 子女模式 -->
        <view class="role-item role-child" hover-class="role-hover" @tap="goLogin('child')">
          <view class="role-bar bar-child"></view>
          <view class="role-icon">👨‍👧</view>
          <view class="role-text">
            <text class="role-name">子女模式</text>
            <text class="role-desc">管理家人用药 · 远程监护</text>
          </view>
          <text class="role-arrow">›</text>
        </view>

        <!-- 管理员模式 -->
        <view class="role-item role-admin" hover-class="role-hover" @tap="goLogin('admin')">
          <view class="role-bar bar-admin"></view>
          <view class="role-icon">⚙️</view>
          <view class="role-text">
            <text class="role-name">管理员模式</text>
            <text class="role-desc">后台数据管理 · 系统统计</text>
          </view>
          <text class="role-arrow">›</text>
        </view>
      </view>

      <!-- 底部版本号 -->
      <text class="version">开发版 v2.0</text>
    </view>
  </view>
</template>

<script>
export default {
  methods: {
    goLogin(role) {
      uni.navigateTo({
        url: '/pages/login/login?role=' + role
      })
    }
  }
}
</script>

<style scoped>
/* ===== 页面背景：线性渐变 ===== */
.page {
  min-height: 100vh;
  background: linear-gradient(160deg, #4096ff 0%, #b07bff 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40rpx;
}

/* ===== 白色圆角弹窗 ===== */
.card {
  width: 100%;
  max-width: 640rpx;
  background: #FFFFFF;
  border-radius: 40rpx;
  padding: 56rpx 40rpx 36rpx;
  box-shadow:
    0 8rpx 32rpx rgba(0, 0, 0, 0.08),
    0 24rpx 64rpx rgba(0, 0, 0, 0.12);
}

/* ===== 顶部图标区 ===== */
.card-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 40rpx;
}

.avatar-wrap {
  width: 140rpx;
  height: 140rpx;
  margin-bottom: 20rpx;
}

.avatar-svg {
  width: 100%;
  height: 100%;
}

.title {
  font-size: 36rpx;
  font-weight: 700;
  color: #1A1A1A;
  letter-spacing: 2rpx;
}

/* ===== 角色选项 ===== */
.roles {
  display: flex;
  flex-direction: column;
  gap: 20rpx;
  margin-bottom: 32rpx;
}

.role-item {
  display: flex;
  align-items: center;
  background: #F8F9FA;
  border-radius: 20rpx;
  padding: 24rpx 20rpx;
  transition: all 0.2s;
  position: relative;
  overflow: hidden;
}

.role-hover {
  transform: scale(0.97);
  box-shadow: 0 4rpx 16rpx rgba(0, 0, 0, 0.1);
}

/* 左侧彩色竖条 */
.role-bar {
  width: 8rpx;
  height: 56rpx;
  border-radius: 4rpx;
  margin-right: 20rpx;
  flex-shrink: 0;
}

.bar-elder  { background: #FF9500; }
.bar-child  { background: #4096FF; }
.bar-admin  { background: #34C759; }

/* 角色图标 */
.role-icon {
  font-size: 44rpx;
  margin-right: 20rpx;
  flex-shrink: 0;
}

/* 角色文字 */
.role-text {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
}

.role-name {
  font-size: 32rpx;
  font-weight: 700;
  color: #1A1A1A;
  line-height: 1.3;
}

.role-desc {
  font-size: 24rpx;
  color: #999999;
  line-height: 1.3;
}

/* 右箭头 */
.role-arrow {
  font-size: 40rpx;
  color: #CCCCCC;
  font-weight: 300;
  flex-shrink: 0;
}

/* ===== 底部版本号 ===== */
.version {
  text-align: center;
  font-size: 22rpx;
  color: #BBBBBB;
  display: block;
}
</style>
