/**
 * 添加药品页（子女端）
 * 手动录入药品信息：名称、规格、厂家、备注、照片
 */
const api = require('../../../utils/api');
const app = getApp();

Page({
  data: {
    familyId: null,
    isPersonal: false,
    isEdit: false,
    medicineId: null,

    // 表单数据
    form: {
      name: '',
      specification: '',
      manufacturer: '',
      notes: '',
      initial_stock: {
        total_count: '',
        alert_threshold: ''
      }
    },

    // 照片
    photoPath: '',
    photoUrl: '',

    // 表单状态
    isSubmitting: false,
    errors: {}
  },

  onLoad(options) {
    if (!app.checkLogin()) return;

    const { family_id, id, is_personal } = options;
    this.setData({
      familyId: family_id || app.globalData.currentFamilyId,
      isPersonal: is_personal === '1' || is_personal === 'true'
    });

    // 如果是编辑模式
    if (id) {
      this.setData({ isEdit: true, medicineId: id });
      wx.setNavigationBarTitle({ title: '编辑药品' });
      this._loadMedicineDetail(id);
    }
  },

  /**
   * 加载药品详情（编辑模式）
   */
  async _loadMedicineDetail(id) {
    try {
      const url = this.data.isPersonal
        ? `/personal/medicines/${id}`
        : `/families/${this.data.familyId}/medicines/${id}`;
      const data = await api.get(url, {}, {
        showLoading: true,
        showError: false
      });

      this.setData({
        form: {
          name: data.name || '',
          specification: data.specification || '',
          manufacturer: data.manufacturer || '',
          notes: data.notes || '',
          initial_stock: {
            total_count: data.stock ? data.stock.total_count : '',
            alert_threshold: data.stock ? data.stock.alert_threshold : ''
          }
        },
        photoUrl: data.image_url || ''
      });
    } catch (err) {
      wx.showToast({ title: '加载药品信息失败', icon: 'none' });
    }
  },

  /**
   * 输入字段
   */
  handleFieldInput(e) {
    const field = e.currentTarget.dataset.field;
    const value = e.detail.value;
    this.setData({
      [`form.${field}`]: value,
      [`errors.${field}`]: ''
    });
  },

  /**
   * 输入库存字段
   */
  handleStockInput(e) {
    const field = e.currentTarget.dataset.field;
    const value = parseInt(e.detail.value, 10) || '';
    this.setData({
      [`form.initial_stock.${field}`]: value
    });
  },

  /**
   * 选择照片
   */
  handleChoosePhoto() {
    wx.chooseImage({
      count: 1,
      sizeType: ['compressed'],
      sourceType: ['album', 'camera'],
      success: (res) => {
        this.setData({ photoPath: res.tempFilePaths[0] });
      }
    });
  },

  /**
   * 删除照片
   */
  handleRemovePhoto() {
    this.setData({ photoPath: '', photoUrl: '' });
  },

  /**
   * 表单校验
   */
  _validate() {
    const errors = {};
    const { form } = this.data;

    if (!form.name.trim()) {
      errors.name = '请输入药品名称';
    }
    if (form.name.trim().length > 50) {
      errors.name = '药品名称不超过50个字符';
    }
    if (form.initial_stock.total_count && isNaN(form.initial_stock.total_count)) {
      errors.total_count = '请输入有效数字';
    }
    if (form.initial_stock.alert_threshold && isNaN(form.initial_stock.alert_threshold)) {
      errors.alert_threshold = '请输入有效数字';
    }

    this.setData({ errors });
    return Object.keys(errors).length === 0;
  },

  /**
   * 提交表单
   */
  async handleSubmit() {
    if (!this._validate() || this.data.isSubmitting) return;

    const { form, isEdit, medicineId, familyId } = this.data;

    this.setData({ isSubmitting: true });

    try {
      const payload = {
        name: form.name.trim(),
        specification: form.specification.trim(),
        manufacturer: form.manufacturer.trim(),
        notes: form.notes.trim(),
        initial_stock: {
          total_count: parseInt(form.initial_stock.total_count, 10) || 0,
          alert_threshold: parseInt(form.initial_stock.alert_threshold, 10) || 0
        }
      };

      if (isEdit) {
        const url = this.data.isPersonal
          ? `/personal/medicines/${medicineId}`
          : `/families/${familyId}/medicines/${medicineId}`;
        await api.put(url, payload, {
          showLoading: true,
          loadingText: '保存中...',
          showError: false
        });
        wx.showToast({ title: '保存成功', icon: 'success' });
      } else {
        const url = this.data.isPersonal
          ? `/personal/medicines`
          : `/families/${familyId}/medicines`;
        await api.post(url, payload, {
          showLoading: true,
          loadingText: '添加中...',
          showError: false
        });
        wx.showToast({ title: '添加成功', icon: 'success' });
      }

      // 如果有照片，上传照片
      if (this.data.photoPath) {
        // TODO: 上传药品图片（P1 OCR 阶段会完善）
      }

      this.setData({ isSubmitting: false });

      // 返回列表
      setTimeout(() => {
        wx.navigateBack();
      }, 1500);

    } catch (err) {
      this.setData({ isSubmitting: false });
      wx.showToast({ title: err.message || '操作失败', icon: 'none' });
    }
  }
});
