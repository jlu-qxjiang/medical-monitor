/**
 * 确认吃药弹窗组件
 * 大按钮二次确认：「确认已服用」/「跳过」
 * 老人端使用，防误触设计
 */
Component({
  properties: {
    // 是否显示弹窗
    visible: {
      type: Boolean,
      value: false
    },
    // 弹窗标题
    title: {
      type: String,
      value: '确认用药'
    },
    // 药品列表文案
    medicineList: {
      type: Array,
      value: []
    },
    // 确认按钮文字
    confirmText: {
      type: String,
      value: '确认已服用'
    },
    // 次要按钮文字
    cancelText: {
      type: String,
      value: '跳过'
    },
    // 是否显示次要按钮
    showCancel: {
      type: Boolean,
      value: true
    },
    // 是否在加载中
    loading: {
      type: Boolean,
      value: false
    },
    // 是否为老人端（控制字号大小）
    isElder: {
      type: Boolean,
      value: true
    }
  },

  methods: {
    /**
     * 确认按钮点击
     */
    onConfirm() {
      if (this.properties.loading) return;
      this.triggerEvent('confirm');
    },

    /**
     * 跳过/取消按钮点击
     */
    onCancel() {
      if (this.properties.loading) return;
      this.triggerEvent('cancel');
    },

    /**
     * 点击蒙层关闭
     */
    onOverlayTap() {
      // 老人端不允许点击蒙层关闭，防止误触
      if (this.properties.isElder) return;
      this.triggerEvent('close');
    }
  }
});
