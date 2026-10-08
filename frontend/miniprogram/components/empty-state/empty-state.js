/**
 * 空状态组件
 * 引导文案 + 操作按钮
 */
Component({
  properties: {
    // 空状态图标（emoji 或图片路径）
    icon: {
      type: String,
      value: '📋'
    },
    // 主提示文案
    title: {
      type: String,
      value: '暂无数据'
    },
    // 副提示文案
    description: {
      type: String,
      value: ''
    },
    // 操作按钮文案（为空则不显示按钮）
    actionText: {
      type: String,
      value: ''
    },
    // 是否显示按钮
    showAction: {
      type: Boolean,
      value: false
    },
    // 是否显示（loading 时隐藏）
    visible: {
      type: Boolean,
      value: true
    }
  },

  methods: {
    onAction() {
      this.triggerEvent('action');
    }
  }
});
