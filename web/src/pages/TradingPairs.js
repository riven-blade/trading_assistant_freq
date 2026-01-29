import React, { useState, useEffect, useCallback } from 'react';
import {
  Drawer,
  Input,
  message,
  Spin,
  Empty,
  Tag,
  Select,
  Table,
  Row,
  Col,
  Modal,
  Switch,
} from 'antd';

import {
  PlusOutlined,
  ReloadOutlined
} from '@ant-design/icons';
import api, { toggleEstimateEnabled, updateCoinTier } from '../services/api';

// 通用组件和Hooks
import PageHeader from '../components/Common/PageHeader';
import TradeDrawer from '../components/Trading/TradeDrawer';
import TradingPairCard from '../components/Trading/TradingPairCard';
import MonitoringCard from '../components/Monitoring/MonitoringCard';
import KlineDrawer from '../components/Chart/KlineDrawer';
import useAccountData from '../hooks/useAccountData';
import useEstimates from '../hooks/useEstimates';
import usePriceData from '../hooks/usePriceData';
import { useSystemConfig } from '../hooks/useSystemConfig';
import { DEFAULT_CONFIG, COIN_TIER_OPTIONS } from '../utils/constants';

const { Option } = Select;

const TradingPairs = () => {
  // 获取系统配置
  const { isSpotMode } = useSystemConfig();

  // 移动端检测
  const [isMobile, setIsMobile] = useState(window.innerWidth <= 768);

  // 基础状态
  const [selectedPairs, setSelectedPairs] = useState([]);
  const [allPairs, setAllPairs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false); // 刷新状态
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [coinFilter, setCoinFilter] = useState([]); // 币种筛选
  const [showUnselectedOnly, setShowUnselectedOnly] = useState(true); // 只显示未选中的交易对，默认开启
  
  // 监听窗口大小变化
  useEffect(() => {
    const handleResize = () => {
      setIsMobile(window.innerWidth <= 768);
    };
    
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);
  const [syncing, setSyncing] = useState(false);
  const [addingAll, setAddingAll] = useState(false); // 批量添加状态
  const [searchValue, setSearchValue] = useState('');
  const [filteredPairs, setFilteredPairs] = useState([]);
  
  // 排序相关状态
  const [sortBy, setSortBy] = useState('quote_volume'); // 排序字段
  const [sortOrder, setSortOrder] = useState('desc'); // 排序顺序

  // 交易相关状态
  const [tradeModalVisible, setTradeModalVisible] = useState(false);
  const [selectedTradeSymbol, setSelectedTradeSymbol] = useState('');
  const [tradeSide, setTradeSide] = useState('long');
  const [selectedLeverage, setSelectedLeverage] = useState(DEFAULT_CONFIG.leverage);
  const [targetPrice, setTargetPrice] = useState(0);
  const [orderType, setOrderType] = useState(DEFAULT_CONFIG.orderType);
  const [entryTag, setEntryTag] = useState('manual'); // 入场标签
  
  // 从 sessionStorage 读取开仓金额，为开多和开空分别保存
  const getStoredStakeAmount = (side) => {
    const key = `stakeAmount_${side}`;
    const stored = sessionStorage.getItem(key);
    return stored ? parseFloat(stored) : 0;
  };
  
  const [stakeAmount, setStakeAmount] = useState(0); // 开仓金额，默认为0

  // 监控抽屉相关状态
  const [monitorDrawerVisible, setMonitorDrawerVisible] = useState(false);
  const [selectedMonitorSymbol, setSelectedMonitorSymbol] = useState('');
  const [symbolEstimatesData, setSymbolEstimatesData] = useState([]);
  const [estimatesLoading, setEstimatesLoading] = useState(false);

  // 等级编辑相关状态
  const [tierModalVisible, setTierModalVisible] = useState(false);
  const [editingTierSymbol, setEditingTierSymbol] = useState('');
  const [selectedTier, setSelectedTier] = useState('');

  // K线抽屉相关状态
  const [klineDrawerVisible, setKlineDrawerVisible] = useState(false);
  const [selectedKlineSymbol, setSelectedKlineSymbol] = useState('');
  const [klineAnalysisData, setKlineAnalysisData] = useState(null);

  // 使用自定义Hooks
  const { 
    hasPosition, 
    hasAnyPosition,
    positions
  } = useAccountData();

  const { 
    symbolEstimates, 
    hasAnyEstimate,
    deleteEstimate,
    getEstimatesBySymbol
  } = useEstimates();

  // 使用全局价格数据管理
  const { 
    priceData, 
    getPriceBySymbol
  } = usePriceData();

  // 使用系统配置
  const { exchangeType, marketType } = useSystemConfig();

  // 全局排名映射
  const [globalVolumeRanks, setGlobalVolumeRanks] = useState({});

  // 初始化
  useEffect(() => {
    fetchSelectedPairs();
    fetchAllPairs();
  }, []);

  // 计算全局成交额排名
  useEffect(() => {
    if (allPairs.length > 0) {
      // 按成交额降序排序所有币种
      const sortedPairs = [...allPairs].sort((a, b) => {
        const aVolume = parseFloat(a.quote_volume || '0');
        const bVolume = parseFloat(b.quote_volume || '0');
        return bVolume - aVolume;
      });

      // 创建排名映射
      const ranks = {};
      sortedPairs.forEach((pair, index) => {
        ranks[pair.symbol] = index + 1;
      });

      setGlobalVolumeRanks(ranks);
    }
  }, [allPairs]);

  // 搜索过滤和排序
  useEffect(() => {
    let filtered = allPairs;
    
    // 先进行搜索过滤
    if (searchValue) {
      filtered = allPairs.filter(pair => 
        pair.symbol.toLowerCase().includes(searchValue.toLowerCase()) ||
        pair.base_asset.toLowerCase().includes(searchValue.toLowerCase()) ||
        pair.quote_asset.toLowerCase().includes(searchValue.toLowerCase())
      );
    }

    // 过滤已选中的交易对
    if (showUnselectedOnly) {
      filtered = filtered.filter(pair => 
        !selectedPairs.some(selected => selected.symbol === pair.symbol)
      );
    }
    
    // 再进行排序
    const sorted = [...filtered].sort((a, b) => {
      let aValue, bValue;
      
      switch (sortBy) {
        case 'symbol':
          aValue = a.symbol;
          bValue = b.symbol;
          break;
        case 'price_change_percent':
          aValue = parseFloat(a.price_change_percent || '0');
          bValue = parseFloat(b.price_change_percent || '0');
          break;
        case 'volume':
          aValue = parseFloat(a.volume || '0');
          bValue = parseFloat(b.volume || '0');
          break;
        case 'quote_volume':
          aValue = parseFloat(a.quote_volume || '0');
          bValue = parseFloat(b.quote_volume || '0');
          break;
        case 'price':
          aValue = parseFloat(a.price || '0');
          bValue = parseFloat(b.price || '0');
          break;
        case 'onboard_date':
          aValue = a.onboard_date || 0;
          bValue = b.onboard_date || 0;
          break;
        default:
          aValue = a.symbol;
          bValue = b.symbol;
      }
      
      // 字符串比较
      if (typeof aValue === 'string' && typeof bValue === 'string') {
        return sortOrder === 'asc' ? aValue.localeCompare(bValue) : bValue.localeCompare(aValue);
      }
      
      // 数字比较
      if (sortOrder === 'asc') {
        return aValue - bValue;
      } else {
        return bValue - aValue;
      }
    });
    
    setFilteredPairs(sorted);
  }, [searchValue, allPairs, sortBy, sortOrder, showUnselectedOnly, selectedPairs]);



  // 获取选中的交易对
  const fetchSelectedPairs = async () => {
    try {
      const response = await api.get('/coins/selected');
      const pairs = response.data.data || [];
      setSelectedPairs(pairs);
    } catch (error) {
      message.error('获取选中交易对失败');
    } finally {
      setLoading(false);
    }
  };

  const fetchAllPairs = async () => {
    try {
      const response = await api.get('/coins');
      setAllPairs(response.data.data || []);
    } catch (error) {
      message.error('获取交易对列表失败');
    }
  };



  // 同步交易对
  const syncPairs = async () => {
    setSyncing(true);
    try {
      await api.post('/coins/sync');
      message.success('交易对同步成功');
      fetchAllPairs();
    } catch (error) {
      message.error('同步失败');
    } finally {
      setSyncing(false);
    }
  };

  // 添加交易对
  const addPair = async (symbol) => {
    try {
      await api.post('/coins/select', {
        symbol,
        is_selected: true
      });
      message.success(`${symbol && symbol.length > 8 ? symbol.substring(0, 8) + '...' : symbol} 添加成功`);
      await fetchSelectedPairs();
      // 价格数据会自动更新，无需手动刷新
      setDrawerVisible(false);
    } catch (error) {
      message.error(`${symbol && symbol.length > 8 ? symbol.substring(0, 8) + '...' : symbol} 添加失败`);
    }
  };


  const isSelected = (symbol) => {
    return selectedPairs.some(pair => pair.symbol === symbol);
  };

  // 检查是否存在同方向的开仓监听
  const hasOpenEstimate = (symbol, side) => {
    const estimates = getEstimatesBySymbol(symbol, 'listening');
    return estimates.some(estimate => 
      estimate.action_type === 'open' && estimate.side === side
    );
  };

  // 打开交易模态框
  const openTradeModal = (symbol, side) => {
    // 检查是否已有同方向仓位
    if (hasPosition(symbol, side)) {
      const positionText = isSpotMode ? '持仓' : (side === 'long' ? '多头' : '空头');
      const actionText = isSpotMode ? '买入' : '开仓';
      message.warning(`${symbol && symbol.length > 8 ? symbol.substring(0, 8) + '...' : symbol} 已有${positionText}仓位 | 无法重复${actionText}`);
      return;
    }

    // 检查是否已有同方向开仓监听
    if (hasOpenEstimate(symbol, side)) {
      const positionText = isSpotMode ? '买入' : (side === 'long' ? '多头' : '空头');
      const actionText = isSpotMode ? '买入' : '开仓';
      message.warning(`${symbol && symbol.length > 8 ? symbol.substring(0, 8) + '...' : symbol} 已有${positionText}${actionText}监听 | 无法重复${actionText}`);
      return;
    }

    setSelectedTradeSymbol(symbol);
    setTradeSide(side);
    setSelectedLeverage(DEFAULT_CONFIG.leverage);
    setOrderType(DEFAULT_CONFIG.orderType);
    
    // 设置初始目标价格
    const price = priceData[symbol];
    let initialPrice = 0;
    if (price && price.markPrice > 0) {
      initialPrice = price.markPrice;
    }
    setTargetPrice(initialPrice);
    
    
    // 重置状态到默认值
    setSelectedLeverage(DEFAULT_CONFIG.leverage);
    setOrderType(DEFAULT_CONFIG.orderType);
    setEntryTag('manual');
    
    // 从 sessionStorage 读取对应方向的开仓金额
    const storedAmount = getStoredStakeAmount(side);
    setStakeAmount(storedAmount);
    
    setTradeModalVisible(true);
  };


  const getCurrentPrice = () => {
    const price = getPriceBySymbol(selectedTradeSymbol);
    return price?.markPrice || 0;
  };

  // 删除/取消选中交易对
  const removePair = async (symbol) => {
    // 检查是否有仓位
    if (hasAnyPosition(symbol)) {
      message.error(`${symbol && symbol.length > 8 ? symbol.substring(0, 8) + '...' : symbol} 存在仓位，无法删除`);
      return;
    }
    
    // 检查是否有监听
    if (hasAnyEstimate(symbol)) {
      message.error(`${symbol && symbol.length > 8 ? symbol.substring(0, 8) + '...' : symbol} 存在监听，无法删除`);
      return;
    }
    
    try {
      await api.post('/coins/select', {
        symbol,
        is_selected: false
      });
      message.success(`${symbol && symbol.length > 8 ? symbol.substring(0, 8) + '...' : symbol} 已取消选中`);
      await fetchSelectedPairs();
    } catch (error) {
      message.error(`${symbol && symbol.length > 8 ? symbol.substring(0, 8) + '...' : symbol} 删除失败`);
    }
  };

  // 批量添加交易对
  const handleAddAll = () => {
    // 从当前过滤列表中筛选出未选中的交易对
    const unselectedPairs = filteredPairs.filter(pair => !isSelected(pair.symbol));

    if (unselectedPairs.length === 0) {
      message.info('当前列表中没有未添加的交易对');
      return;
    }

    Modal.confirm({
      title: '批量添加交易对',
      content: `确定要添加当前列表中的 ${unselectedPairs.length} 个交易对吗？`,
      okText: '确定添加',
      cancelText: '取消',
      onOk: async () => {
        setAddingAll(true);
        try {
          // 串行添加，避免并发请求过多
          let successCount = 0;
          for (const pair of unselectedPairs) {
            try {
              await api.post('/coins/select', {
                symbol: pair.symbol,
                is_selected: true
              });
              successCount++;
            } catch (err) {
              console.error(`Failed to add ${pair.symbol}`, err);
            }
          }
          message.success(`成功添加 ${successCount} 个交易对`);
          await fetchSelectedPairs();
        } catch (error) {
          message.error('批量添加失败');
        } finally {
          setAddingAll(false);
        }
      }
    });
  };

  // 统一的卡片操作处理
  const handleCardAction = async (symbol, action) => {
    switch (action) {
      case 'long':
        openTradeModal(symbol, 'long');
        break;
      case 'short':
        openTradeModal(symbol, 'short');
        break;
      case 'kline':
        // 打开K线抽屉并加载分析数据
        setSelectedKlineSymbol(symbol);
        setKlineDrawerVisible(true);
        await fetchAnalysisData(symbol);
        break;
      case 'monitor':
        openMonitorDrawer(symbol);
        break;
      case 'remove':
        removePair(symbol);
        break;
      default:
        break;
    }
  };

  // 简化的处理函数
  const handlePriceChange = (value) => {
    setTargetPrice(value || 0);
  };

  const handleLeverageChange = (value) => {
    setSelectedLeverage(value);
  };

  const handleOrderTypeChange = (value) => {
    setOrderType(value);
  };

  const handleEntryTagChange = (value) => {
    setEntryTag(value);
  };

  const handleStakeAmountChange = (value) => {
    const amount = value || 0;
    setStakeAmount(amount);
    // 保存到 sessionStorage，区分开多和开空
    const key = `stakeAmount_${tradeSide}`;
    sessionStorage.setItem(key, amount.toString());
  };

  // 创建交易
  const createTrade = async (selectedEntryTag) => {
    // 使用传入的entryTag参数，如果没有则使用状态中的值
    const tagToUse = selectedEntryTag || entryTag;
    try {
      const markPrice = getCurrentPrice();
      if (!markPrice || markPrice <= 0) {
        message.error('价格数据不可用，请稍后再试');
        return;
      }


      // 检查入场标签
      if (!tagToUse) {
        message.error('请选择入场标签');
        return;
      }
      
      // 根据订单类型确定价格
      let orderPrice = markPrice;
      if (orderType === 'limit') {
        orderPrice = targetPrice || markPrice;
        if (!orderPrice || orderPrice <= 0) {
          message.error('请设置有效的限价价格');
          return;
        }
      }

      const orderData = {
        symbol: selectedTradeSymbol,
        side: tradeSide,
        action_type: 'open',
        target_price: orderPrice,
        percentage: 100, // 开仓时默认100%
        leverage: selectedLeverage,
        tag: tagToUse,
        margin_mode: 'ISOLATED',
        order_type: orderType,
        trigger_type: 'condition',
        stake_amount: stakeAmount
      };

      await api.post('/estimates', orderData);

      const actionText = isSpotMode ? '买入' : (tradeSide === 'long' ? '开多' : '开空');
      const orderTypeText = orderType === 'market' ? '市价' : '限价';

      const baseAsset = selectedTradeSymbol.replace('USDT', '');
      // 现货模式不显示杠杆和保证金模式
      const leverageText = isSpotMode ? '' : ` ${selectedLeverage}x杠杆 逐仓`;
      message.success(`${actionText}预估价已创建 | ${orderTypeText}单${leverageText} 标签${tagToUse} ${baseAsset} (${stakeAmount} USDT)`);
      setTradeModalVisible(false);
    } catch (error) {
      message.error('创建订单失败: ' + (error.response?.data?.error || error.message));
    }
  };

  // 监控相关功能
  const openMonitorDrawer = (symbol) => {
    setSelectedMonitorSymbol(symbol);
    setMonitorDrawerVisible(true);
    fetchSymbolEstimates(symbol);
  };

  const closeMonitorDrawer = () => {
    setMonitorDrawerVisible(false);
    setSelectedMonitorSymbol('');
    setSymbolEstimatesData([]);
  };

  const fetchSymbolEstimates = useCallback((symbol) => {
    setEstimatesLoading(true);
    try {
      // 从全局estimates数据中过滤出该币种相关的监听
      const filteredEstimates = getEstimatesBySymbol(symbol, 'listening');
      
      // 获取当前价格并计算差异
      const currentPrice = getPriceBySymbol(symbol)?.markPrice || 0;
      
      const estimatesWithPrice = filteredEstimates.map(estimate => {
        const priceDiff = currentPrice > 0 ? 
          ((estimate.target_price - currentPrice) / currentPrice * 100) : 0;
        
        return {
          ...estimate,
          current_price: currentPrice,
          price_difference: priceDiff,
          is_close_to_trigger: Math.abs(priceDiff) <= 2
        };
      });
      
      setSymbolEstimatesData(estimatesWithPrice);
    } catch (error) {
      console.error('获取币种监控数据失败:', error);
      setSymbolEstimatesData([]);
    } finally {
      setEstimatesLoading(false);
    }
  }, [getEstimatesBySymbol, getPriceBySymbol]);

  // 获取币种对应的仓位信息
  const getPositionForSymbol = useCallback((symbol, side) => {
    if (!positions || positions.length === 0) {
      return null;
    }
    
    return positions.find(position => 
      position.symbol === symbol && position.side === side.toUpperCase()
    ) || null;
  }, [positions]);

  // 监听estimates数据变化，实时更新监控抽屉中的数据
  useEffect(() => {
    if (selectedMonitorSymbol && monitorDrawerVisible) {
      fetchSymbolEstimates(selectedMonitorSymbol);
    }
  }, [symbolEstimates, selectedMonitorSymbol, monitorDrawerVisible, fetchSymbolEstimates]);

  const handleDeleteEstimate = async (estimateId) => {
    await deleteEstimate(estimateId);
    // 重新获取当前币种的监控数据
    if (selectedMonitorSymbol) {
      fetchSymbolEstimates(selectedMonitorSymbol);
    }
  };

  const handleToggleEstimate = async (estimateId, enabled) => {
    try {
      await toggleEstimateEnabled(estimateId, enabled);
      message.success(`监听已${enabled ? '开启' : '关闭'}`);

      // 重新获取当前币种的监控数据
      if (selectedMonitorSymbol) {
        fetchSymbolEstimates(selectedMonitorSymbol);
      }
    } catch (error) {
      message.error('切换监听状态失败');
    }
  };

  // 等级编辑相关函数
  const openTierModal = (symbol) => {
    const pair = selectedPairs.find(p => p.symbol === symbol);
    setEditingTierSymbol(symbol);
    setSelectedTier(pair?.tier || '');
    setTierModalVisible(true);
  };

  const handleTierChange = async (tier) => {
    try {
      await updateCoinTier(editingTierSymbol, tier);
      message.success(`${editingTierSymbol} 等级已更新为 ${tier || '无'}`);
      setTierModalVisible(false);
      // 刷新选中的币种列表
      await fetchSelectedPairs();
    } catch (error) {
      message.error('更新等级失败');
    }
  };

  // 获取分析数据
  const fetchAnalysisData = async (symbol) => {
    try {
      // 查询该交易对的最新分析数据
      const response = await api.get('/analysis/results', {
        params: {
          symbol: symbol,
          exchange: exchangeType,
          market_type: marketType,
          page: 1,
          pageSize: 1
        }
      });

      const results = response.data.data || [];
      if (results.length > 0) {
        // 使用最新的分析结果
        setKlineAnalysisData(results[0]);
      } else {
        // 没有分析数据
        setKlineAnalysisData(null);
      }
    } catch (error) {
      console.error('获取分析数据失败:', error);
      setKlineAnalysisData(null);
    }
  };

  // 页面操作配置
  const headerActions = [
    <Select
      key="filter"
      mode="multiple"
      placeholder="筛选币种"
      value={coinFilter}
      onChange={setCoinFilter}
      style={{ width: isMobile ? 150 : 300, minWidth: 150 }}
      allowClear
      maxTagCount={1}
      maxTagPlaceholder={(omittedValues) => `+${omittedValues.length}个币种`}
      showSearch
      filterOption={(input, option) =>
        (option?.label ?? '').toLowerCase().includes(input.toLowerCase())
      }
      virtual={false}
      listHeight={256}
      getPopupContainer={(trigger) => trigger.parentElement}
      options={selectedPairs.map(pair => ({
        label: pair.symbol,
        value: pair.symbol
      }))}
    />,
    <button 
      key="refresh"
      className="control-btn primary-btn trading-pairs-header-btn"
      onClick={async () => {
        setRefreshing(true);
        try {
          await fetchSelectedPairs();
        } catch (error) {
          console.error('刷新失败:', error);
        } finally {
          setRefreshing(false);
        }
      }}
      disabled={refreshing}
    >
      <ReloadOutlined style={{ marginRight: 4 }} />
      刷新
    </button>,
    <button 
      key="add"
      className="control-btn success-btn trading-pairs-header-btn"
      onClick={() => setDrawerVisible(true)}
    >
      <PlusOutlined style={{ marginRight: 4 }} />
      添加交易对
    </button>
  ];

  if (loading) {
    return <Spin size="large" style={{ display: 'block', textAlign: 'center', padding: '50px' }} />;
  }

  return (
    <div>
      <PageHeader 
        title="币种" 
        actions={headerActions}
      />

      {selectedPairs.length === 0 ? (
        <Empty description="暂无选中的交易对" />
      ) : (
        <Row gutter={[16, 16]}>
          {selectedPairs
            .filter(pair => {
              // 如果没有选择筛选条件，显示全部
              if (coinFilter.length === 0) return true;
              // 否则只显示选中的币种
              return coinFilter.includes(pair.symbol);
            })
            .sort((a, b) => {
              // 按交易额降序排序
              const aVolume = parseFloat(a.quote_volume || '0');
              const bVolume = parseFloat(b.quote_volume || '0');
              return bVolume - aVolume;
            })
            .map((pair) => (
            <Col 
              xs={24} 
              sm={12} 
              md={8} 
              lg={6} 
              xl={4} 
              key={pair.symbol}
            >
              <TradingPairCard
                pair={pair}
                priceInfo={getPriceBySymbol(pair.symbol)}
                onAction={handleCardAction}
                hasPosition={hasPosition}
                hasAnyPosition={hasAnyPosition}
                hasAnyEstimate={hasAnyEstimate}
                hasOpenEstimate={hasOpenEstimate}
                symbolEstimates={symbolEstimates}
                isMobile={isMobile}
                volumeRank={globalVolumeRanks[pair.symbol]}
                onEditTags={openTierModal}
              />
            </Col>
          ))}
        </Row>
      )}

      {/* 添加交易对抽屉 */}
      <Drawer
        title="添加交易对"
        open={drawerVisible}
        onClose={() => setDrawerVisible(false)}
        width={isMobile ? '100%' : 800}
        placement="right"
        styles={{
          body: { 
            padding: isMobile ? '12px' : '16px',
            height: '100%',
            overflow: 'hidden',
            display: 'flex',
            flexDirection: 'column'
          }
        }}
      >
        {/* 固定顶部控制区 */}
        <div style={{ 
          flexShrink: 0,
          marginBottom: '12px',
          paddingBottom: '12px',
          borderBottom: '1px solid #f0f0f0'
        }}>
          {/* 按钮组：同步和一键添加 */}
          <div style={{ display: 'flex', gap: '8px', marginBottom: '8px' }}>
            <button 
              className="control-btn primary-btn trading-pairs-header-btn"
              onClick={syncPairs}
              disabled={syncing || addingAll}
              style={{ 
                flex: 1,
                height: '32px',
                fontSize: isMobile ? '12px' : '13px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                whiteSpace: 'nowrap'
              }}
            >
              <ReloadOutlined style={{ marginRight: 4 }} spin={syncing} />
              同步交易对
            </button>
            
            <button 
              className="control-btn success-btn trading-pairs-header-btn"
              onClick={handleAddAll}
              disabled={syncing || addingAll}
              style={{ 
                flex: 1,
                height: '32px',
                fontSize: isMobile ? '12px' : '13px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                whiteSpace: 'nowrap'
              }}
            >
              <PlusOutlined style={{ marginRight: 4 }} />
              一键添加
            </button>
          </div>
          
          {/* 搜索框 - 统一样式 */}
          <Input.Search
            placeholder="搜索交易对"
            value={searchValue}
            onChange={(e) => setSearchValue(e.target.value)}
            style={{ 
              width: '100%',
              marginBottom: '8px'
            }}
            size="default"
            allowClear
            bordered={true}
          />
          
          {/* 排序控制 - 强制水平布局 */}
          {/* 排序控制 */}
          <div 
            style={{ 
              display: 'flex', 
              alignItems: 'center',
              gap: '8px',
              width: '100%',
              marginTop: '4px',
              flexWrap: 'nowrap'
            }}
          >
            <div style={{ 
              fontSize: '12px', 
              color: '#888',
              flexShrink: 0,
            }}>
              排序
            </div>
            <div style={{ flex: 1, minWidth: 0 }}>
              <Select
                value={sortBy}
                onChange={setSortBy}
                style={{ width: '100%' }}
                size="middle"
                placeholder="排序方式"
              >
                <Option value="price_change_percent">涨跌幅</Option>
                <Option value="volume">成交量</Option>
                <Option value="quote_volume">成交额</Option>
                <Option value="price">价格</Option>
                <Option value="onboard_date">上市天数</Option>
                <Option value="symbol">名称</Option>
              </Select>
            </div>
            <div style={{ width: '75px', flexShrink: 0 }}>
              <Select
                value={sortOrder}
                onChange={setSortOrder}
                style={{ width: '100%' }}
                size="middle"
              >
                <Option value="desc">降序</Option>
                <Option value="asc">升序</Option>
              </Select>
            </div>
            
            <div style={{ 
              display: 'flex', 
              alignItems: 'center', 
              flexShrink: 0,
              paddingLeft: '8px',
              borderLeft: '1px solid #f0f0f0',
              marginLeft: '4px'
            }}>
              <Switch 
                checked={showUnselectedOnly}
                onChange={setShowUnselectedOnly}
                size="middle"
              />
              <span style={{ fontSize: '12px', color: '#666', marginLeft: '4px', whiteSpace: 'nowrap' }}>只看未选</span>
            </div>
          </div>
        </div>

        {/* 可滚动内容区 */}
        <div style={{ 
          flex: 1,
          overflow: 'auto',
          minHeight: 0
        }}>
          <Table
            dataSource={filteredPairs.slice(0, 500)}
            pagination={{ 
              pageSize: isMobile ? 15 : 20, 
              showSizeChanger: false,
              showQuickJumper: !isMobile,
              size: 'small',
              showTotal: (total, range) => `共 ${total} 条，显示 ${range[0]}-${range[1]} 条`
            }}
            size="small"
            scroll={{ y: 'calc(100vh - 400px)' }}
            columns={[
              {
                title: '交易对',
                dataIndex: 'symbol',
                key: 'symbol',
                width: isMobile ? 80 : 120,
                render: (text, record) => (
                  <div>
                    <div style={{ 
                      fontWeight: 'bold', 
                      fontSize: isMobile ? '12px' : '14px',
                      lineHeight: 1.2
                    }}>
                      {text && text.length > 8 ? text.substring(0, 8) + '...' : text}
                    </div>
                    <div style={{ 
                      fontSize: isMobile ? '10px' : '12px', 
                      color: '#666',
                      marginTop: '2px'
                    }}>
                      {record.base_asset}/{record.quote_asset}
                    </div>
                  </div>
                ),
              },
              {
                title: '价格',
                dataIndex: 'price',
                key: 'price',
                width: isMobile ? 70 : 100,
                align: 'right',
                render: (price) => (
                  <span style={{ fontSize: isMobile ? '11px' : '13px' }}>
                    {price ? parseFloat(price).toFixed(4) : '-'}
                  </span>
                ),
              },
              {
                title: '涨跌幅',
                dataIndex: 'price_change_percent',
                key: 'price_change_percent',
                width: isMobile ? 60 : 90,
                align: 'right',
                render: (percent) => {
                  if (!percent) return '-';
                  const value = parseFloat(percent);
                  return (
                    <span style={{ 
                      color: value >= 0 ? '#3f8600' : '#cf1322',
                      fontWeight: 'bold',
                      fontSize: isMobile ? '10px' : '12px'
                    }}>
                      {value >= 0 ? '+' : ''}{value.toFixed(2)}%
                    </span>
                  );
                },
              },
              ...(isMobile ? [] : [
                {
                  title: '成交量',
                  dataIndex: 'volume',
                  key: 'volume',
                  width: 100,
                  align: 'right',
                  render: (volume) => volume ? `${(parseFloat(volume) / 1000).toFixed(0)}K` : '-',
                },
                {
                  title: '成交额',
                  dataIndex: 'quote_volume', 
                  key: 'quote_volume',
                  width: 100,
                  align: 'right',
                  render: (quoteVolume) => quoteVolume ? `${(parseFloat(quoteVolume) / 1000000).toFixed(1)}M` : '-',
                },
                {
                  title: '上市',
                  dataIndex: 'onboard_date',
                  key: 'onboard_date',
                  width: 80,
                  align: 'right',
                  render: (onboardDate) => {
                    if (!onboardDate || onboardDate <= 0) return '-';
                    const days = Math.floor((Date.now() - onboardDate) / (1000 * 60 * 60 * 24));
                    const isNew = days <= 30;
                    return (
                      <span style={{ 
                        color: isNew ? '#ea580c' : '#666',
                        fontWeight: isNew ? '600' : '400'
                      }}>
                        {days}天
                      </span>
                    );
                  },
                }
              ]),
              {
                title: '操作',
                key: 'action',
                width: isMobile ? 50 : 80,
                align: 'center',
                render: (_, record) => 
                  isSelected(record.symbol) ? (
                    <Tag 
                      color="success" 
                      style={{ 
                        fontSize: isMobile ? '10px' : '12px',
                        margin: 0
                      }}
                    >
                      已选
                    </Tag>
                  ) : (
                    <button 
                      className="control-btn success-btn trading-pairs-action-btn"
                      onClick={() => addPair(record.symbol)}
                      style={{ 
                        fontSize: isMobile ? '10px' : '12px',
                        padding: isMobile ? '0 4px' : '0 8px'
                      }}
                    >
                      添加
                    </button>
                  )
              }
            ]}
          />
          
          {filteredPairs.length === 0 && searchValue && (
            <div style={{ padding: '40px 20px', textAlign: 'center' }}>
              <Empty 
                description="未找到匹配的交易对" 
                imageStyle={{ height: 60 }}
              />
            </div>
          )}
        </div>
      </Drawer>

      {/* 交易抽屉 */}
      <TradeDrawer
        visible={tradeModalVisible}
        onClose={() => {
          setTradeModalVisible(false);
          // 价格数据会自动更新，无需手动刷新
        }}
        symbol={selectedTradeSymbol}
        side={tradeSide}
        onSubmit={createTrade}
        priceData={priceData}
        targetPrice={targetPrice}
        onPriceChange={handlePriceChange}
        orderType={orderType}
        onOrderTypeChange={handleOrderTypeChange}
        selectedLeverage={selectedLeverage}
        onLeverageChange={handleLeverageChange}
        entryTag={entryTag}
        onEntryTagChange={handleEntryTagChange}
        stakeAmount={stakeAmount}
        onStakeAmountChange={handleStakeAmountChange}
        pricePrecision={selectedPairs.find(p => p.symbol === selectedTradeSymbol)?.price_precision}
      />

      {/* 监控抽屉 */}
      <Drawer
        title={
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <span style={{ fontSize: '16px', fontWeight: 600, color: '#1f2937' }}>
              {selectedMonitorSymbol}
            </span>
            <span style={{ 
              background: '#f9fafb',
              color: '#6b7280',
              padding: '2px 6px',
              borderRadius: '4px',
              fontSize: '10px',
              fontWeight: 500
            }}>
              {symbolEstimatesData.length}个监听
            </span>
          </div>
        }
        placement="right"
        onClose={closeMonitorDrawer}
        open={monitorDrawerVisible}
        width={isMobile ? '100%' : 500}
        styles={{
          body: {
            paddingTop: 0,
            height: '100%'
          }
        }}
      >
        <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
          <div className="monitoring-list-container" style={{ paddingTop: '16px' }}>
            {estimatesLoading ? (
              <Spin style={{ display: 'block', textAlign: 'center', margin: '20px 0' }} />
            ) : symbolEstimatesData.length === 0 ? (
              <Empty 
                description="暂无价格监听" 
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                style={{ margin: '20px 0' }}
              />
            ) : (
              <div className="monitoring-list-content">
                {symbolEstimatesData.map((estimate) => {
                  // 获取对应的仓位信息
                  const currentPosition = getPositionForSymbol(estimate.symbol, estimate.side);
                  
                  return (
                    <MonitoringCard
                      key={estimate.id}
                      estimate={estimate}
                      currentPosition={currentPosition}
                      onDelete={handleDeleteEstimate}
                      onToggle={handleToggleEstimate}
                    />
                  );
                })}
              </div>
            )}
          </div>
        </div>
      </Drawer>

      {/* 等级选择弹窗 */}
      <Modal
        title={`设置 ${editingTierSymbol} 等级`}
        open={tierModalVisible}
        onCancel={() => setTierModalVisible(false)}
        footer={null}
        width={320}
        centered
      >
        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', padding: '12px 0' }}>
          {COIN_TIER_OPTIONS.map((tier) => (
            <button
              key={tier.key}
              onClick={() => handleTierChange(tier.key)}
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                padding: '12px 16px',
                border: selectedTier === tier.key ? `2px solid ${tier.borderColor}` : '1px solid #e5e7eb',
                borderRadius: '8px',
                background: selectedTier === tier.key ? tier.bgColor : '#fff',
                color: selectedTier === tier.key ? tier.color : '#374151',
                cursor: 'pointer',
                transition: 'all 0.2s ease',
                fontSize: '14px',
                fontWeight: selectedTier === tier.key ? '600' : '500'
              }}
            >
              <span style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                <span style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  width: '28px',
                  height: '28px',
                  borderRadius: '6px',
                  background: tier.bgColor,
                  color: tier.color,
                  fontWeight: '700',
                  fontSize: '14px'
                }}>
                  {tier.label}
                </span>
                <span>{tier.description}</span>
              </span>
            </button>
          ))}
          {/* 清除等级按钮 */}
          <button
            onClick={() => handleTierChange('')}
            style={{
              padding: '12px 16px',
              border: '1px solid #e5e7eb',
              borderRadius: '8px',
              background: '#f9fafb',
              color: '#6b7280',
              cursor: 'pointer',
              transition: 'all 0.2s ease',
              fontSize: '14px'
            }}
          >
            清除等级
          </button>
        </div>
      </Modal>

      {/* K线抽屉 */}
      <KlineDrawer
        visible={klineDrawerVisible}
        onClose={() => {
          setKlineDrawerVisible(false);
          setKlineAnalysisData(null);
        }}
        symbol={selectedKlineSymbol}
        analysisData={klineAnalysisData}
        defaultInterval="1h"
      />

    </div>
  );
};

export default TradingPairs;
