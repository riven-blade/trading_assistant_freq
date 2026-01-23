import React, { useState, useEffect, useCallback, useRef } from 'react';
import { 
  message, 
  Spin,
  Modal,
  Select,
  InputNumber,
  Form
} from 'antd';
import { useLocation, useSearchParams } from 'react-router-dom';
import { createChart, ColorType, CrosshairMode, CandlestickSeries, HistogramSeries } from 'lightweight-charts';

import ChartToolbar from '../components/Chart/ChartToolbar';
import api from '../services/api';
import useEstimates from '../hooks/useEstimates';
import useAccountData from '../hooks/useAccountData';

const { Option } = Select;

const ChartPage = () => {
  const location = useLocation();
  const [searchParams, setSearchParams] = useSearchParams();
  
  // Helpers
  const parseUrlParams = (search) => {
    const searchParams = new URLSearchParams(search);
    return {
      symbol: searchParams.get('symbol'), // Now optional, can be derived from ID
      id: searchParams.get('id'),
      interval: searchParams.get('interval') || '15m'
    };
  };

  const [selectedCoin, setSelectedCoin] = useState(null);
  const [selectedInterval, setSelectedInterval] = useState('15m');
  const [loading, setLoading] = useState(false);
  const [analysisId, setAnalysisId] = useState(null);
  const [analysisData, setAnalysisData] = useState(null);
  const [currentPrice, setCurrentPrice] = useState(null);
  const [priceChange, setPriceChange] = useState(null);
  const [isMobile, setIsMobile] = useState(false);
  
  // 价格监听创建对话框
  const [createEstimateModalVisible, setCreateEstimateModalVisible] = useState(false);
  const [selectedPriceType, setSelectedPriceType] = useState(null); // 'support' or 'resistance'
  const [form] = Form.useForm();

  // 使用 useEstimates hook 获取监听价格
  const { getEstimatesBySymbol, symbolEstimates } = useEstimates();
  
  // 使用 useAccountData hook 获取持仓信息
  const { hasPosition } = useAccountData();

  // 处理价格线点击 - 必须在 useEffect 之前定义
  const handlePriceLevelClick = useCallback((price, type) => {
    setSelectedPriceType(type);
    
    // 根据类型设置默认值
    const defaultSide = type === 'resistance' ? 'short' : 'long';
    
    form.setFieldsValue({
      target_price: price,
      side: defaultSide,
      leverage: 3, // 默认杠杆3倍
      enabled: true // 默认启用
    });
    
    setCreateEstimateModalVisible(true);
  }, [form]);

  // Chart References
  const chartContainerRef = useRef(null);
  const chartRef = useRef(null);
  const candlestickSeriesRef = useRef(null);
  const volumeSeriesRef = useRef(null);
  const estimatePriceLinesRef = useRef([]); // 存储监听价格线的引用
  const supportLevelsRef = useRef([]); // 存储支撑位价格
  const resistanceLevelsRef = useRef([]); // 存储压力位价格
  
  // Data Refs
  const isMountedRef = useRef(true);

  // 1. Initial Parse and Setup
  useEffect(() => {
    const params = parseUrlParams(location.search);
    setAnalysisId(params.id);
    setSelectedInterval(params.interval);
    // If no symbol and no ID, default to BTCUSDT
    if (params.symbol) {
        setSelectedCoin(params.symbol);
    } else if (!params.id) {
        setSelectedCoin('BTCUSDT');
    }
  }, [location.search]);

  // 2. Fetch Analysis Data if ID exists
  useEffect(() => {
    if (!analysisId) return;

    const fetchAnalysis = async () => {
      setLoading(true);
      try {
        const response = await api.get(`/analysis/${analysisId}`);
        const data = response.data.data;
        if (data) {
          setAnalysisData(data);
          setSelectedCoin(data.symbol); // Set symbol from analysis
          // We might want to use the timeframe from analysis as default, 
          // but user might want to change it, so we keep selectedInterval controlled by UI state/URL
        }
      } catch (error) {
        console.error('Failed to fetch analysis details:', error);
        message.error('加载分析详情失败');
      } finally {
        setLoading(false);
      }
    };

    fetchAnalysis();
  }, [analysisId]);

  // 3. Initialize Chart
  useEffect(() => {
    if (!chartContainerRef.current) return;

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

    candlestickSeriesRef.current = chart.addSeries(CandlestickSeries, {
      upColor: '#26a69a',
      downColor: '#ef5350',
      borderVisible: false,
      wickUpColor: '#26a69a',
      wickDownColor: '#ef5350',
    });

    volumeSeriesRef.current = chart.addSeries(HistogramSeries, {
      priceFormat: {
        type: 'volume',
      },
      priceScaleId: 'volume', // 使用独立的价格刻度
    });
    
    // 设置交易量独立刻度，显示在底部
    chart.priceScale('volume').applyOptions({
      scaleMargins: {
        top: 0.85, // 交易量占底部15%
        bottom: 0,
      },
    });

    // Resize handler
    const handleResize = () => {
      if (chartContainerRef.current) {
        chart.applyOptions({ 
          width: chartContainerRef.current.clientWidth,
          height: chartContainerRef.current.clientHeight
        });
      }
      setIsMobile(window.innerWidth < 768);
    };

    window.addEventListener('resize', handleResize);
    handleResize(); // Init check

    // 检查鼠标是否在价格线附近
    const isNearPriceLine = (price) => {
      if (!price || isNaN(price)) return null;
      
      const priceThreshold = Math.abs(price) * 0.005; // 0.5% 的误差范围
      
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
      
      const price = candlestickSeriesRef.current.coordinateToPrice(param.point.y);
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
      
      // 获取点击位置的价格
      const clickedPrice = candlestickSeriesRef.current.coordinateToPrice(param.point.y);
      
      if (!clickedPrice || isNaN(clickedPrice)) return;
      
      console.log('Clicked price:', clickedPrice);
      console.log('Support levels:', supportLevelsRef.current);
      console.log('Resistance levels:', resistanceLevelsRef.current);
      
      // 检查是否点击了支撑位或压力位
      const nearPriceLine = isNearPriceLine(clickedPrice);
      
      if (nearPriceLine) {
        console.log(`Clicked ${nearPriceLine.type} level:`, nearPriceLine.price);
        handlePriceLevelClick(nearPriceLine.price, nearPriceLine.type);
      }
    };
    
    chart.subscribeCrosshairMove(handleCrosshairMove);
    chart.subscribeClick(handleChartClick);

    return () => {
      window.removeEventListener('resize', handleResize);
      chart.unsubscribeCrosshairMove(handleCrosshairMove);
      chart.unsubscribeClick(handleChartClick);
      chart.remove();
    };
  }, [handlePriceLevelClick]);

  // 4. Fetch and Render K-line Data
  const fetchAndRenderKlines = useCallback(async () => {
    if (!selectedCoin || !selectedInterval || !chartRef.current) return;

    setLoading(true);
    try {
      const response = await api.get('/klines', {
        params: {
          symbol: selectedCoin,
          interval: selectedInterval,
          limit: 1000
        }
      });
      
      const rawData = response.data.data || [];
      
      // Calculate price change if we have data
      if (rawData.length > 0) {
        const latestCandle = rawData[rawData.length - 1];
        const firstCandle = rawData[0];
        const latest = parseFloat(latestCandle.close);
        const first = parseFloat(firstCandle.open);
        
        setCurrentPrice(latest);
        setPriceChange(((latest - first) / first) * 100);
      }
      
      // Transform data for lightweight-charts
      // Input: { time, open, high, low, close, volume } (Assuming API returns this structure or similar)
      const candleData = rawData.map(item => ({
        time: item.timestamp / 1000, // Unix timestamp in seconds
        open: parseFloat(item.open),
        high: parseFloat(item.high),
        low: parseFloat(item.low),
        close: parseFloat(item.close),
      })).sort((a, b) => a.time - b.time); // Ensure sorted

      const volumeData = rawData.map(item => ({
        time: item.timestamp / 1000,
        value: parseFloat(item.volume),
        color: parseFloat(item.close) >= parseFloat(item.open) ? '#26a69a' : '#ef5350',
      })).sort((a, b) => a.time - b.time);

      if (candlestickSeriesRef.current) candlestickSeriesRef.current.setData(candleData);
      if (volumeSeriesRef.current) volumeSeriesRef.current.setData(volumeData);

    } catch (error) {
      console.error('Fetch klines failed:', error);
      message.error('获取K线数据失败');
    } finally {
      if (isMountedRef.current) setLoading(false);
    }
  }, [selectedCoin, selectedInterval]);

  useEffect(() => {
    fetchAndRenderKlines();
  }, [fetchAndRenderKlines]);

  // 5. Render Support/Resistance Lines from Analysis Data
  useEffect(() => {
    if (!analysisData || !candlestickSeriesRef.current) return;

    // 清空之前的支撑/压力位数据
    supportLevelsRef.current = [];
    resistanceLevelsRef.current = [];
    
    // Draw Support (Green)
    if (analysisData.support_levels) {
      analysisData.support_levels.forEach(level => {
        const price = parseFloat(level);
        supportLevelsRef.current.push(price);
        
        candlestickSeriesRef.current.createPriceLine({
          price: price,
          color: '#26a69a',
          lineWidth: 2,
          lineStyle: 1, // Dotted
          axisLabelVisible: true,
          title: '支撑',
        });
      });
    }

    // Draw Resistance (Red)
    if (analysisData.resistance_levels) {
      analysisData.resistance_levels.forEach(level => {
        const price = parseFloat(level);
        resistanceLevelsRef.current.push(price);
        
        candlestickSeriesRef.current.createPriceLine({
          price: price,
          color: '#ef5350',
          lineWidth: 2,
          lineStyle: 1, // Dotted
          axisLabelVisible: true,
          title: '压力',
        });
      });
    }
    
    console.log('Support levels set:', supportLevelsRef.current);
    console.log('Resistance levels set:', resistanceLevelsRef.current);

  }, [analysisData]);

  // 6. Render Estimate Price Lines (监听价格线)
  useEffect(() => {
    if (!selectedCoin || !candlestickSeriesRef.current) return;

    // 清除之前的价格线
    estimatePriceLinesRef.current.forEach(line => {
      if (line && typeof line.remove === 'function') {
        try {
          candlestickSeriesRef.current.removePriceLine(line);
        } catch (e) {
          console.warn('Failed to remove price line:', e);
        }
      }
    });
    estimatePriceLinesRef.current = [];

    // 获取当前币种的监听价格
    const estimates = getEstimatesBySymbol(selectedCoin, 'listening');
    
    console.log(`[ChartPage] Rendering price lines for ${selectedCoin}:`, estimates);
    
    if (estimates && estimates.length > 0) {
      estimates.forEach(estimate => {
        console.log(`[ChartPage] Creating line for estimate ID ${estimate.id}:`, {
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
          
          // 添加方向标识
          const sideText = estimate.side === 'long' ? '多' : '空';
          lineTitle = `${lineTitle}(${sideText})`;
          
          // 创建价格线
          const priceLine = candlestickSeriesRef.current.createPriceLine({
            price: parseFloat(estimate.target_price),
            color: lineColor,
            lineWidth: 2,
            lineStyle: estimate.enabled ? 0 : 1, // 启用：实线，禁用：虚线
            axisLabelVisible: true,
            title: lineTitle,
          });
          
          estimatePriceLinesRef.current.push(priceLine);
        } catch (e) {
          console.error('Failed to create price line:', e);
        }
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedCoin, symbolEstimates]); // 使用 symbolEstimates 而不是 getEstimatesBySymbol

  // 创建价格监听
  const handleCreateEstimate = async () => {
    try {
      const values = await form.validateFields();
      
      // 数据库中的 symbol 已经是 BTCUSDT 格式（无斜杠），无需处理
      const symbol = selectedCoin || '';
      
      // 根据持仓情况自动判断操作类型
      // 如果该币种该方向有持仓，则为加仓；否则为开仓
      const hasExistingPosition = hasPosition(symbol, values.side);
      const action_type = hasExistingPosition ? 'addition' : 'open';
      
      // 根据操作类型自动设置标签
      // 开仓: manual, 加仓: grind_x_entry
      const tag = action_type === 'addition' ? 'grind_x_entry' : 'manual';
      
      const estimateData = {
        symbol: symbol,
        target_price: values.target_price,
        side: values.side,
        action_type: action_type, // 自动判断的操作类型
        leverage: values.leverage,
        enabled: values.enabled !== undefined ? values.enabled : true,
        tag: tag, // 自动生成的标签
        percentage: 100, // 默认 100%
        trigger_type: 'condition', // 默认条件触发
        margin_mode: 'ISOLATED', // 默认逐仓
        order_type: 'limit' // 默认限价单
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

  return (
    <div style={{ height: '100%', display: 'flex', flexDirection: 'column', background: '#fafbfc' }}>
      <ChartToolbar
        selectedCoin={selectedCoin}
        currentPrice={currentPrice}
        priceChange={priceChange}
        selectedInterval={selectedInterval}
        onIntervalChange={(i) => {
            setSelectedInterval(i);
            const newParams = new URLSearchParams(searchParams);
            newParams.set('interval', i);
            setSearchParams(newParams);
        }}
        loading={loading}
        isMobile={isMobile}
      />
      
      <div style={{ flex: 1, position: 'relative', margin: '4px' }}>
        {loading && (
            <div style={{
              position: 'absolute', top: 0, left: 0, right: 0, bottom: 0,
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              background: 'rgba(255,255,255,0.6)', zIndex: 10
            }}>
                <Spin size="large" />
            </div>
        )}
        <div ref={chartContainerRef} style={{ width: '100%', height: '100%' }} />
      </div>
      
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
              precision={2}
              min={0}
            />
          </Form.Item>
          
          <Form.Item
            label="方向"
            name="side"
            rules={[{ required: true, message: '请选择方向' }]}
            help={
              form.getFieldValue('side') && selectedCoin ? (
                hasPosition(selectedCoin, form.getFieldValue('side')) ? 
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
    </div>
  );
};

export default ChartPage;
