import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Drawer, Select, Space, message, Modal, Form, InputNumber } from 'antd';
import { createChart, ColorType, CrosshairMode, CandlestickSeries, HistogramSeries } from 'lightweight-charts';
import api from '../../services/api';
import useAccountData from '../../hooks/useAccountData';
import useEstimates from '../../hooks/useEstimates';

const { Option } = Select;

/**
 * K线图抽屉组件
 * @param {boolean} visible - 抽屉是否可见
 * @param {function} onClose - 关闭抽屉的回调
 * @param {string} symbol - 交易对符号 (如 BTCUSDT)
 * @param {object} analysisData - 分析数据（可选），包含 support_levels 和 resistance_levels
 * @param {string} defaultInterval - 默认时间周期（可选，默认 1h）
 */
const KlineDrawer = ({ visible, onClose, symbol, analysisData, defaultInterval = '1h' }) => {
  const [selectedInterval, setSelectedInterval] = useState(defaultInterval);
  const [loading, setLoading] = useState(false);
  const chartContainerRef = useRef(null);
  const chartRef = useRef(null);
  const candlestickSeriesRef = useRef(null);
  const volumeSeriesRef = useRef(null);
  const supportLevelsRef = useRef([]); // 存储支撑位价格
  const resistanceLevelsRef = useRef([]); // 存储压力位价格
  const estimatePriceLinesRef = useRef([]); // 存储监听价格线的引用 [{line, estimate}]
  const positionPriceLinesRef = useRef([]); // 存储仓位价格线的引用

  // 价格监听创建对话框
  const [createEstimateModalVisible, setCreateEstimateModalVisible] = useState(false);
  const [selectedPriceType, setSelectedPriceType] = useState(null);
  const [form] = Form.useForm();

  // 价格监听删除对话框
  const [deleteEstimateModalVisible, setDeleteEstimateModalVisible] = useState(false);
  const [selectedEstimate, setSelectedEstimate] = useState(null);

  // 使用 hooks
  const { hasPosition, positions } = useAccountData();
  const { getEstimatesBySymbol, deleteEstimate } = useEstimates();

  // 当 symbol 或 visible 变化时，重置时间周期为默认值
  useEffect(() => {
    if (visible) {
      setSelectedInterval(defaultInterval);
    }
  }, [visible, symbol, defaultInterval]);

  // 处理价格线点击
  const handlePriceLevelClick = useCallback((price, type) => {
    setSelectedPriceType(type);
    
    // 根据类型设置默认值
    const defaultSide = type === 'resistance' ? 'short' : 'long';
    
    form.setFieldsValue({
      target_price: price,
      side: defaultSide,
      leverage: 3,
      enabled: true
    });
    
    setCreateEstimateModalVisible(true);
  }, [form]);

  // 初始化图表
  useEffect(() => {
    if (!visible || !chartContainerRef.current || chartRef.current) return;

    const chartOptions = {
      layout: {
        background: { type: ColorType.Solid, color: '#ffffff' },
        textColor: '#333',
      },
      grid: {
        vertLines: { color: '#f0f0f0' },
        horzLines: { color: '#f0f0f0' },
      },
      crosshair: {
        mode: CrosshairMode.Normal,
      },
      timeScale: {
        borderColor: '#f0f0f0',
        timeVisible: true,
      },
      rightPriceScale: {
        borderColor: '#f0f0f0',
      },
      height: chartContainerRef.current.clientHeight,
      width: chartContainerRef.current.clientWidth,
    };

    const chart = createChart(chartContainerRef.current, chartOptions);
    chartRef.current = chart;

    // 添加K线系列 - 使用正确的 API
    const candlestickSeries = chart.addSeries(CandlestickSeries, {
      upColor: '#26a69a',
      downColor: '#ef5350',
      borderVisible: false,
      wickUpColor: '#26a69a',
      wickDownColor: '#ef5350',
      priceFormat: {
        type: 'price',
        precision: 4, // 默认4位小数，会在数据加载后动态调整
        minMove: 0.0001,
      },
    });
    candlestickSeriesRef.current = candlestickSeries;

    // 添加成交量系列
    const volumeSeries = chart.addSeries(HistogramSeries, {
      priceFormat: {
        type: 'volume',
      },
      priceScaleId: 'volume',
    });
    volumeSeriesRef.current = volumeSeries;

    // 设置交易量独立刻度，显示在底部
    chart.priceScale('volume').applyOptions({
      scaleMargins: {
        top: 0.85, // 交易量占底部15%
        bottom: 0,
      },
    });

    // 设置交易量独立刻度
    chart.priceScale('volume').applyOptions({
      scaleMargins: {
        top: 0.85,
        bottom: 0,
      },
    });

    // 窗口大小调整
    const handleResize = () => {
      if (chartContainerRef.current && chartRef.current) {
        chartRef.current.applyOptions({
          width: chartContainerRef.current.clientWidth,
          height: chartContainerRef.current.clientHeight,
        });
      }
    };

    window.addEventListener('resize', handleResize);

    // 检查鼠标是否在价格线附近
    const isNearPriceLine = (price) => {
      if (!price || isNaN(price)) return null;
      
      const priceThreshold = Math.abs(price) * 0.005; // 0.5% 的误差范围
      
      // 检查价格监听线
      for (const item of estimatePriceLinesRef.current) {
        if (item && item.estimate) {
          const estimatePrice = parseFloat(item.estimate.target_price);
          if (Math.abs(price - estimatePrice) < priceThreshold) {
            return { price: estimatePrice, type: 'estimate', estimate: item.estimate };
          }
        }
      }
      
      // 检查支撑位
      for (const supportPrice of supportLevelsRef.current) {
        if (Math.abs(price - supportPrice) < priceThreshold) {
          return { price: supportPrice, type: 'support' };
        }
      }
      
      // 检查压力位
      for (const resistancePrice of resistanceLevelsRef.current) {
        if (Math.abs(price - resistancePrice) < priceThreshold) {
          return { price: resistancePrice, type: 'resistance' };
        }
      }
      
      return null;
    };
    
    // 添加鼠标移动事件监听，改变光标样式
    const handleCrosshairMove = (param) => {
      if (!param.point || !chartContainerRef.current) return;
      
      const price = candlestickSeries.coordinateToPrice(param.point.y);
      const nearPriceLine = isNearPriceLine(price);
      
      if (nearPriceLine) {
        chartContainerRef.current.style.cursor = 'pointer';
      } else {
        chartContainerRef.current.style.cursor = 'default';
      }
    };
    
    // 添加图表点击事件监听
    const handleChartClick = (param) => {
      if (!param.point) return;
      
      const clickedPrice = candlestickSeries.coordinateToPrice(param.point.y);
      
      if (!clickedPrice || isNaN(clickedPrice)) return;
      
      // 检查是否点击了价格线
      const nearPriceLine = isNearPriceLine(clickedPrice);
      
      if (nearPriceLine) {
        if (nearPriceLine.type === 'estimate') {
          // 点击了价格监听线，显示删除确认对话框
          setSelectedEstimate(nearPriceLine.estimate);
          setDeleteEstimateModalVisible(true);
        } else {
          // 点击了支撑位或压力位，显示创建监听对话框
          handlePriceLevelClick(nearPriceLine.price, nearPriceLine.type);
        }
      }
    };
    
    chart.subscribeCrosshairMove(handleCrosshairMove);
    chart.subscribeClick(handleChartClick);

    return () => {
      window.removeEventListener('resize', handleResize);
      chart.unsubscribeCrosshairMove(handleCrosshairMove);
      chart.unsubscribeClick(handleChartClick);
    };
  }, [visible, handlePriceLevelClick]);

  // 获取并渲染K线数据
  useEffect(() => {
    if (!visible || !symbol || !candlestickSeriesRef.current) return;

    const fetchKlines = async () => {
      setLoading(true);
      try {
        const response = await api.get('/klines', {
          params: {
            symbol: symbol,
            interval: selectedInterval,
            limit: 1000,
          },
        });

        const rawData = response.data.data || [];

        // 计算价格范围以确定精度
        let maxPrice = 0;
        let minPrice = Infinity;
        rawData.forEach(item => {
          const high = parseFloat(item.high);
          const low = parseFloat(item.low);
          if (high > maxPrice) maxPrice = high;
          if (low < minPrice) minPrice = low;
        });

        // 根据价格范围动态计算精度
        // 目标：保持2位有效数字
        let precision = 2; // 默认2位小数
        if (maxPrice > 0) {
          const magnitude = Math.floor(Math.log10(maxPrice));
          if (magnitude < 0) {
            // 价格小于1，需要更多小数位
            precision = Math.abs(magnitude) + 1; // 2位有效数字
          } else if (magnitude >= 2) {
            // 价格 >= 100，使用2位小数
            precision = 2;
          } else {
            // 价格在1-99之间，使用3位小数
            precision = 3;
          }
        }

        // 更新价格格式
        candlestickSeriesRef.current.applyOptions({
          priceFormat: {
            type: 'price',
            precision: precision,
            minMove: Math.pow(10, -precision),
          },
        });

        // K线数据
        const candleData = rawData
          .map((item) => ({
            time: item.timestamp / 1000,
            open: parseFloat(item.open),
            high: parseFloat(item.high),
            low: parseFloat(item.low),
            close: parseFloat(item.close),
          }))
          .sort((a, b) => a.time - b.time);

        // 成交量数据
        const volumeData = rawData
          .map((item) => ({
            time: item.timestamp / 1000,
            value: parseFloat(item.volume),
            color: parseFloat(item.close) >= parseFloat(item.open) ? '#26a69a' : '#ef5350',
          }))
          .sort((a, b) => a.time - b.time);

        if (candlestickSeriesRef.current) {
          candlestickSeriesRef.current.setData(candleData);
        }

        if (volumeSeriesRef.current) {
          volumeSeriesRef.current.setData(volumeData);
        }
      } catch (error) {
        console.error('Failed to fetch klines:', error);
        message.error('获取K线数据失败');
      } finally {
        setLoading(false);
      }
    };

    fetchKlines();
  }, [visible, symbol, selectedInterval]);

  // 渲染支撑位和压力位
  useEffect(() => {
    if (!visible || !analysisData || !candlestickSeriesRef.current) return;

    // 清空之前的支撑/压力位数据
    supportLevelsRef.current = [];
    resistanceLevelsRef.current = [];

    // 绘制支撑位
    if (analysisData.support_levels && Array.isArray(analysisData.support_levels)) {
      analysisData.support_levels.forEach((level) => {
        const price = parseFloat(level);
        supportLevelsRef.current.push(price);
        
        candlestickSeriesRef.current.createPriceLine({
          price: price,
          color: '#00d4aa',  // 鲜明的青绿色
          lineWidth: 2,
          lineStyle: 2, // 点状虚线
          axisLabelVisible: true,
          title: '支撑',
        });
      });
    }

    // 绘制压力位
    if (analysisData.resistance_levels && Array.isArray(analysisData.resistance_levels)) {
      analysisData.resistance_levels.forEach((level) => {
        const price = parseFloat(level);
        resistanceLevelsRef.current.push(price);
        
        candlestickSeriesRef.current.createPriceLine({
          price: price,
          color: '#ff1744',  // 鲜红色
          lineWidth: 2,
          lineStyle: 2, // 点状虚线
          axisLabelVisible: true,
          title: '压力',
        });
      });
    }
  }, [visible, analysisData]);

  // 渲染价格监听线（开仓、加仓、止盈）
  useEffect(() => {
    if (!visible || !symbol || !candlestickSeriesRef.current) return;

    // 清除之前的价格线
    estimatePriceLinesRef.current.forEach(item => {
      if (item && item.line && typeof item.line.remove === 'function') {
        try {
          candlestickSeriesRef.current.removePriceLine(item.line);
        } catch (e) {
          console.warn('Failed to remove price line:', e);
        }
      }
    });
    estimatePriceLinesRef.current = [];

    // 获取当前币种的监听价格
    const estimates = getEstimatesBySymbol(symbol, 'listening');
    
    console.log(`[KlineDrawer] Rendering price lines for ${symbol}:`, estimates);
    
    if (estimates && estimates.length > 0) {
      estimates.forEach(estimate => {
        console.log(`[KlineDrawer] Creating line for estimate ID ${estimate.id}:`, {
          price: estimate.target_price,
          action_type: estimate.action_type,
          side: estimate.side,
          enabled: estimate.enabled
        });
        
        try {
          // 根据操作类型和方向确定颜色
          let lineColor = '#1890ff'; // 默认蓝色
          let lineTitle = '';
          
          // 根据 action_type 设置颜色和标题
          if (estimate.action_type === 'take_profit') {
            lineColor = '#52c41a'; // 止盈：绿色
            lineTitle = '止盈';
          } else if (estimate.action_type === 'open') {
            lineColor = '#fa8c16'; // 开仓：橙色
            lineTitle = '开仓';
          } else if (estimate.action_type === 'addition') {
            lineColor = '#faad14'; // 加仓：黄色
            lineTitle = '加仓';
          }
          
          // 添加方向标识和价格
          const sideText = estimate.side === 'long' ? '多' : '空';
          const priceText = `$${parseFloat(estimate.target_price).toFixed(4)}`;
          lineTitle = `${lineTitle}(${sideText}) ${priceText}`;
          
          // 创建价格线
          const priceLine = candlestickSeriesRef.current.createPriceLine({
            price: parseFloat(estimate.target_price),
            color: lineColor,
            lineWidth: 2,
            lineStyle: estimate.enabled ? 0 : 1, // 启用：实线，禁用：虚线
            axisLabelVisible: true,
            title: lineTitle,
          });
          
          // 存储价格线和关联的estimate数据
          estimatePriceLinesRef.current.push({ line: priceLine, estimate: estimate });
        } catch (e) {
          console.error('Failed to create price line:', e);
        }
      });
    }
  }, [visible, symbol, getEstimatesBySymbol]);

  // 渲染当前仓位开仓价
  useEffect(() => {
    if (!visible || !symbol || !candlestickSeriesRef.current) return;

    // 清除之前的仓位价格线
    positionPriceLinesRef.current.forEach(line => {
      if (line && typeof line.remove === 'function') {
        try {
          candlestickSeriesRef.current.removePriceLine(line);
        } catch (e) {
          console.warn('Failed to remove position price line:', e);
        }
      }
    });
    positionPriceLinesRef.current = [];

    // 获取当前币种的所有仓位
    const allPositions = positions.filter(pos => pos.symbol === symbol && Math.abs(pos.size) > 0);
    
    console.log(`[KlineDrawer] Checking positions for ${symbol}:`, allPositions);

    allPositions.forEach(position => {
      const isLong = position.side.toUpperCase() === 'LONG';
      
      try {
        const priceLine = candlestickSeriesRef.current.createPriceLine({
          price: parseFloat(position.entry_price),
          color: isLong ? '#2196f3' : '#9c27b0',  // 蓝色(多) / 紫色(空)
          lineWidth: 2,  // 2px，与其他线条一致
          lineStyle: 0,  // 实线
          axisLabelVisible: true,
          title: `${isLong ? '多头' : '空头'}持仓`,
        });
        positionPriceLinesRef.current.push(priceLine);
        console.log(`[KlineDrawer] Added ${isLong ? 'long' : 'short'} position line at ${position.entry_price}`);
      } catch (e) {
        console.error(`Failed to create ${isLong ? 'long' : 'short'} position price line:`, e);
      }
    });
  }, [visible, symbol, positions]);

  // 创建价格监听
  const handleCreateEstimate = async () => {
    try {
      const values = await form.validateFields();
      
      // 根据持仓情况自动判断操作类型
      const hasExistingPosition = hasPosition(symbol, values.side);
      const action_type = hasExistingPosition ? 'addition' : 'open';
      
      // 根据操作类型自动设置标签
      const tag = action_type === 'addition' ? 'grind_x_entry' : 'manual';
      
      const estimateData = {
        symbol: symbol,
        target_price: values.target_price,
        side: values.side,
        action_type: action_type,
        leverage: values.leverage,
        enabled: values.enabled !== undefined ? values.enabled : true,
        tag: tag,
        percentage: 100,
        trigger_type: 'condition',
        margin_mode: 'ISOLATED',
        order_type: 'limit'
      };
      
      await api.post('/estimates', estimateData);
      message.success(`价格监听创建成功 (${action_type === 'open' ? '开仓' : '加仓'} - ${tag})`);
      setCreateEstimateModalVisible(false);
      form.resetFields();
    } catch (error) {
      console.error('创建价格监听失败:', error);
      message.error(error.response?.data?.error || '创建价格监听失败');
    }
  };

  // 删除价格监听
  const handleDeleteEstimate = async () => {
    if (!selectedEstimate || !candlestickSeriesRef.current) return;
    
    try {
      // 先从图表中移除对应的价格线
      const estimateId = selectedEstimate.id;
      const itemIndex = estimatePriceLinesRef.current.findIndex(
        item => item && item.estimate && item.estimate.id === estimateId
      );
      
      if (itemIndex !== -1) {
        const item = estimatePriceLinesRef.current[itemIndex];
        if (item.line && typeof item.line.remove === 'function') {
          try {
            candlestickSeriesRef.current.removePriceLine(item.line);
          } catch (e) {
            console.warn('Failed to remove price line from chart:', e);
          }
        }
        // 从数组中移除该项
        estimatePriceLinesRef.current.splice(itemIndex, 1);
      }
      
      // 然后调用API删除
      await deleteEstimate(estimateId);
      message.success('价格监听删除成功');
      setDeleteEstimateModalVisible(false);
      setSelectedEstimate(null);
    } catch (error) {
      console.error('删除价格监听失败:', error);
      message.error(error.response?.data?.error || '删除价格监听失败');
    }
  };

  // 关闭抽屉时清理图表
  const handleClose = () => {
    if (chartRef.current) {
      chartRef.current.remove();
      chartRef.current = null;
      candlestickSeriesRef.current = null;
      volumeSeriesRef.current = null;
    }
    onClose();
  };

  return (
    <Drawer
      title={
        <Space>
          <span>{symbol} K线图</span>
          <Select
            value={selectedInterval}
            onChange={setSelectedInterval}
            style={{ width: 100 }}
            size="small"
            loading={loading}
          >
            <Option value="1m">1分钟</Option>
            <Option value="5m">5分钟</Option>
            <Option value="15m">15分钟</Option>
            <Option value="1h">1小时</Option>
            <Option value="4h">4小时</Option>
            <Option value="1d">1天</Option>
            <Option value="1w">1周</Option>
          </Select>
        </Space>
      }
      placement="right"
      width="90%"
      onClose={handleClose}
      open={visible}
      destroyOnClose
    >
      <div
        ref={chartContainerRef}
        style={{
          width: '100%',
          height: 'calc(100vh - 120px)',
        }}
      />

      {/* 创建价格监听对话框 */}
      <Modal
        title={`创建价格监听 - ${selectedPriceType === 'support' ? '支撑位' : '压力位'}`}
        open={createEstimateModalVisible}
        onOk={handleCreateEstimate}
        onCancel={() => {
          setCreateEstimateModalVisible(false);
          form.resetFields();
        }}
        okText="创建"
        cancelText="取消"
      >
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            leverage: 3,
            enabled: true
          }}
        >
          <Form.Item
            label="目标价格"
            name="target_price"
            rules={[{ required: true, message: '请输入目标价格' }]}
          >
            <InputNumber
              style={{ width: '100%' }}
              precision={6}
              min={0}
            />
          </Form.Item>
          
          <Form.Item
            label="方向"
            name="side"
            rules={[{ required: true, message: '请选择方向' }]}
            help={
              form.getFieldValue('side') && symbol ? (
                hasPosition(symbol, form.getFieldValue('side')) ? 
                  <span style={{ color: '#faad14' }}>检测到现有持仓，将创建加仓监听</span> : 
                  <span style={{ color: '#52c41a' }}>无现有持仓，将创建开仓监听</span>
              ) : null
            }
          >
            <Select onChange={() => form.validateFields(['side'])}>
              <Option value="long">做多</Option>
              <Option value="short">做空</Option>
            </Select>
          </Form.Item>
          
          <Form.Item
            label="杠杆"
            name="leverage"
            rules={[{ required: true, message: '请输入杠杆' }]}
          >
            <InputNumber
              style={{ width: '100%' }}
              min={1}
              max={20}
              precision={0}
              placeholder="请输入杠杆倍数 (1-20)"
            />
          </Form.Item>
          
          <Form.Item
            label="启用状态"
            name="enabled"
          >
            <Select>
              <Option value={true}>启用</Option>
              <Option value={false}>禁用</Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      {/* 删除价格监听确认对话框 */}
      <Modal
        title="删除价格监听"
        open={deleteEstimateModalVisible}
        onOk={handleDeleteEstimate}
        onCancel={() => {
          setDeleteEstimateModalVisible(false);
          setSelectedEstimate(null);
        }}
        okText="确认删除"
        cancelText="取消"
      >
        {selectedEstimate && (
          <div>
            <p>确认要删除以下价格监听吗？</p>
            <div style={{ marginTop: 16, padding: 12, background: '#f5f5f5', borderRadius: 4 }}>
              <p><strong>目标价格:</strong> ${parseFloat(selectedEstimate.target_price).toFixed(6)}</p>
              <p><strong>方向:</strong> {selectedEstimate.side === 'long' ? '做多' : '做空'}</p>
              <p><strong>操作类型:</strong> {selectedEstimate.action_type === 'open' ? '开仓' : selectedEstimate.action_type === 'addition' ? '加仓' : '止盈'}</p>
              <p><strong>杠杆:</strong> {selectedEstimate.leverage}x</p>
            </div>
          </div>
        )}
      </Modal>
    </Drawer>
  );
};

export default KlineDrawer;
