/**
 * 加载态组件
 * 骨架屏 / spinner 两种模式
 */
Component({
  properties: {
    // 模式：skeleton（骨架屏）/ spinner（转圈）
    mode: {
      type: String,
      value: 'spinner'
    },
    // 加载文案
    text: {
      type: String,
      value: '加载中...'
    },
    // 骨架屏行数
    skeletonRows: {
      type: Number,
      value: 3
    },
    // 是否全屏显示
    fullscreen: {
      type: Boolean,
      value: false
    },
    // 是否显示（spinner 模式）
    visible: {
      type: Boolean,
      value: true
    }
  },

  data: {
    skeletonArray: []
  },

  lifetimes: {
    attached() {
      this.setData({
        skeletonArray: new Array(this.properties.skeletonRows).fill(0)
      });
    }
  }
});
