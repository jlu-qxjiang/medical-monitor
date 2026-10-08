/**
 * 药品卡片组件
 * 展示药品名、规格、剂量信息
 */
Component({
  properties: {
    // 药品名称
    name: {
      type: String,
      value: ''
    },
    // 规格（如 30mg×7片）
    specification: {
      type: String,
      value: ''
    },
    // 剂量
    dosage: {
      type: Number,
      value: 0
    },
    // 单位（如 粒/片/支）
    unit: {
      type: String,
      value: '粒'
    },
    // 厂家
    manufacturer: {
      type: String,
      value: ''
    },
    // 服用状态：taken/untaken/null
    status: {
      type: String,
      value: ''
    },
    // 是否显示状态标签
    showStatus: {
      type: Boolean,
      value: true
    },
    // 是否可点击
    clickable: {
      type: Boolean,
      value: false
    },
    // 余量
    remainingCount: {
      type: Number,
      value: -1
    },
    // 预警阈值
    alertThreshold: {
      type: Number,
      value: -1
    }
  },

  data: {
    statusText: '',
    statusClass: '',
    stockWarning: ''
  },

  observers: {
    'status': function(status) {
      let statusText = '';
      let statusClass = '';
      switch (status) {
        case 'taken':
        case 'true':
        case true:
          statusText = '已服用';
          statusClass = 'tag-green';
          break;
        case 'untaken':
        case 'false':
        case false:
          statusText = '待服用';
          statusClass = 'tag-yellow';
          break;
        default:
          statusText = '';
          statusClass = '';
      }
      this.setData({ statusText, statusClass });
    },
    'remainingCount, alertThreshold': function(remaining, threshold) {
      if (remaining < 0 || threshold < 0) {
        this.setData({ stockWarning: '' });
        return;
      }
      if (remaining === 0) {
        this.setData({ stockWarning: '已用完' });
      } else if (remaining <= threshold) {
        this.setData({ stockWarning: '即将用完' });
      } else {
        this.setData({ stockWarning: '' });
      }
    }
  },

  methods: {
    onTap() {
      if (this.properties.clickable) {
        this.triggerEvent('tap');
      }
    }
  }
});
