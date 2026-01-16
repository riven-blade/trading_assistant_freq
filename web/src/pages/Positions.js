import React, { useState, useEffect, useCallback } from 'react';
import { 
  Row, 
  Col, 
  Drawer, 
  Typography, 
  message,
  Spin,
  Empty,
  Select
} from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import api, { toggleEstimateEnabled } from '../services/api';
import { calculatePnl } from '../utils/precision';

// 通用组件和Hooks
import PageHeader from '../components/Common/PageHeader';
import PositionCard from '../components/Position/PositionCard';
import PriceSlider from '../components/Trading/PriceSlider';
import QuantitySlider from '../components/Trading/QuantitySlider';
import MonitoringCard from '../components/Monitoring/MonitoringCard';
import useAccountData from '../hooks/useAccountData';
import useEstimates from '../hooks/useEstimates';
import usePriceData from '../hooks/usePriceData';
import { useSystemConfig } from '../hooks/useSystemConfig';
import { ACTIONS } from '../utils/constants';

const { Text } = Typography;

const Positions = () => {
  // 获取系统配置
  const { isSpotMode } = useSystemConfig();

  // 操作相关状态
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [currentPosition, setCurrentPosition] = useState(null);
  const [actionType, setActionType] = useState('');
  const [markPrice, setMarkPrice] = useState(0);
  const [targetPrice, setTargetPrice] = useState(0);
  // 数量/比例：加仓/止盈时表示百分比0-100，其它情况表示绝对数量
  const [quantity, setQuantity] = useState(0);
  const [pricePercentage, setPricePercentage] = useState(0);
  const [confirmLoading, setConfirmLoading] = useState(false);
  const [entryTag, setEntryTag] = useState('manual');
  const [exitTag, setExitTag] = useState('manual');

  // 详情抽屉相关状态
  const [detailDrawerVisible, setDetailDrawerVisible] = useState(false);
  const [detailPosition, setDetailPosition] = useState(null);
  const [positionEstimates, setPositionEstimates] = useState([]);
  const [estimatesLoading, setEstimatesLoading] = useState(false);
  const [refreshing, setRefreshing] = useState(false);

  // 使用自定义Hooks
  const { 
    positions, 
    positionsLoading,
    refreshPositions
  } = useAccountData();

  // 监听数量状态
  const [positionsWithMonitors, setPositionsWithMonitors] = useState([]);

  const { estimates, deleteEstimate, getEstimatesBySymbol } = useEstimates();

  // 使用全局价格数据
  const { 
    getPriceBySymbol,
    loading: priceDataLoading,
    priceData,
    hasPriceData
  } = usePriceData();



  // 获取所有持仓的监听数量
  const fetchPositionsWithMonitors = useCallback((positionsData) => {
    try {
      const positionsWithCount = positionsData.map((position) => {
        
        // 从全局estimates数据中过滤出该持仓相关的监听
        const filteredEstimates = getEstimatesBySymbol(position.symbol, 'listening').filter(estimate => 
          estimate.side === position.side.toLowerCase()
        );
        
        // 按操作类型统计数量（freqtrade的核心操作）
        const additionCount = filteredEstimates.filter(estimate => estimate.action_type === 'addition').length;
        const takeProfitCount = filteredEstimates.filter(estimate => estimate.action_type === 'take_profit').length;
        
        return {
          ...position,
          monitorCount: filteredEstimates.length,
          additionCount,
          takeProfitCount
        };
      });
      
      setPositionsWithMonitors(positionsWithCount);
    } catch (error) {
      console.error('计算持仓监听数量失败:', error);
      setPositionsWithMonitors(positionsData.map(pos => ({ 
        ...pos, 
        monitorCount: 0, 
        additionCount: 0, 
        takeProfitCount: 0 
      })));
    }
  }, [getEstimatesBySymbol]);

  // 获取当前价格 - 使用全局价格数据
  const getCurrentPrice = useCallback((symbol) => {
    // 如果价格数据还在加载中，返回0
    if (priceDataLoading) {
      return 0;
    }
    
    // 检查是否有该币种的价格数据
    if (!hasPriceData(symbol)) {
      console.warn(`[价格] ${symbol} 的价格数据不可用`);
      return 0;
    }
    
    const priceInfo = getPriceBySymbol(symbol);
    const price = priceInfo?.markPrice || 0;
    
    return price;
  }, [getPriceBySymbol, priceDataLoading, hasPriceData]);

  // 统一的操作处理器
  const handleAction = async (position, action) => {
    // 验证参数
    if (!position) {
      console.error('handleAction: position 参数不能为空');
      return;
    }
    
    if (!action || typeof action !== 'string') {
      console.error('handleAction: action 参数无效:', action);
      return;
    }
    
    // 处理K线图跳转
    if (action === 'kline') {
      const klineUrl = `${window.location.origin}/klines?symbol=${position.symbol}&interval=4h`;
      window.open(klineUrl, '_blank', 'noopener,noreferrer');
      return;
    }
    
    const config = ACTIONS[action];
    if (!config) {
      console.error('handleAction: 未知的操作类型:', action);
      return;
    }
    
    setCurrentPosition(position);
    setActionType(action);
    
    const price = getCurrentPrice(position.symbol);
    setMarkPrice(price);
    
    const entryPrice = position.entry_price || price;
    const isLong = position.side === 'LONG';
    
    // 设置默认价格百分比
    let defaultPercentage = 0;
    if (action === 'take_profit') {
      defaultPercentage = isLong ? 5 : -5;
    }
    
    // 计算目标价格，确保 config 和 config.priceBase 存在
    const priceBase = config.priceBase || 'current';
    const basePrice = priceBase === 'current' ? price : entryPrice;
    let defaultTargetPrice = basePrice * (1 + defaultPercentage / 100);

    // 设置默认比例/数量：加仓默认15%，止盈默认50%
    let defaultQuantity = action === 'addition' ? 15 : 50;
    
    setPricePercentage(defaultPercentage);
    setTargetPrice(defaultTargetPrice);
    setQuantity(defaultQuantity);
    
    // 设置默认入场标签：加仓默认grind_1_entry，其他默认manual
    setEntryTag(action === 'addition' ? 'grind_1_entry' : 'manual');
    
    setDrawerVisible(true);
  };

  // 价格滑块变化处理
  const handlePriceSliderChange = (percentage) => {
    setPricePercentage(percentage);
    
    const config = ACTIONS[actionType];
    if (!config) {
      console.error('handlePriceSliderChange: 未知的操作类型:', actionType);
      return;
    }
    
    const priceBase = config.priceBase || 'current';
    const basePrice = priceBase === 'current' ? markPrice : currentPosition.entry_price;
    let newTargetPrice = basePrice * (1 + percentage / 100);
    
    setTargetPrice(newTargetPrice);
    
    // 加仓/止盈改为百分比模式，与价格滑块解耦，不做数量联动
  };

  // 数量滑块变化处理
  const handleQuantitySliderChange = (newQuantity) => {
    setQuantity(newQuantity);
  };

  // 获取第一个订单的cost作为最大加仓金额
  const getFirstOrderCost = (position) => {
    if (!position?.orders || position.orders.length === 0) {
      console.log(`[加仓] ${position?.symbol}: 没有订单数据，使用默认值1000 USDT`);
      return 1000; // 如果没有订单数据，回退到默认值
    }
    
    const firstOrder = position.orders[0];
    const firstOrderCost = Number(firstOrder.cost) || 0;
    
    console.log(`[加仓] ${position.symbol}: 第一个订单cost=${firstOrderCost} USDT (订单ID: ${firstOrder.order_id})`);
    
    // 如果第一个订单的cost为0或无效，使用默认值
    return firstOrderCost > 0 ? firstOrderCost : 1000;
  };

  // 获取最大数量
  const getMaxQuantity = () => {
    if (!currentPosition) return 1;
    
    const positionSize = Math.abs(currentPosition.size);
    if (actionType === 'addition') {
      // 加仓：基于第一个订单的cost和目标价格计算
      const maxUsdtAmount = getFirstOrderCost(currentPosition);
      const priceToUse = targetPrice > 0 ? targetPrice : markPrice;
      if (priceToUse > 0 && currentPosition.leverage > 0) {
        // 最大数量 = (第一个订单Cost × 杠杆) ÷ 目标价格
        const maxQuantity = (maxUsdtAmount * currentPosition.leverage) / priceToUse;
        return parseFloat(maxQuantity.toFixed(6));
      }
      return positionSize;
    } else {
      // 止盈/止损：最多平掉所有仓位
      return positionSize;
    }
  };

  // 确认操作
  const handleConfirm = async () => {
    if (confirmLoading) return;
    
    setConfirmLoading(true);
    try {
      const orderData = {
        symbol: currentPosition.symbol,
        side: currentPosition.side.toLowerCase(),
        action_type: actionType,
        target_price: targetPrice,
        leverage: currentPosition.leverage,
        margin_mode: 'ISOLATED',
        order_type: 'limit',
        trigger_type: 'condition',
        tag: actionType === 'addition' ? (entryTag || 'manual') : 
             actionType === 'take_profit' ? (exitTag || 'manual') : undefined
      };

      // 加仓/止盈使用百分比（0-100）
      if (actionType === 'addition' || actionType === 'take_profit') {
        orderData.percentage = Math.min(100, Math.max(0, quantity));
      }

      await api.post('/estimates', orderData);
      
      message.success(`${ACTIONS[actionType].title}监听已创建`);
      setDrawerVisible(false);
      setEntryTag('manual');
      setExitTag('manual');
      
      // 数据会通过全局estimates自动更新，无需手动刷新
    } catch (error) {
      console.error('创建订单失败:', error);
      message.error('创建订单失败');
    } finally {
      setConfirmLoading(false);
    }
  };

  // 获取仓位详情（从全局estimates数据中过滤）
  const fetchPositionDetails = useCallback((position) => {
    setEstimatesLoading(true);
    try {
      // 从全局estimates数据中过滤出该持仓相关的监听
      const filteredEstimates = getEstimatesBySymbol(position.symbol, 'listening').filter(estimate => 
        estimate.side === position.side.toLowerCase()
      );
      
      // 获取当前价格并计算差异
      const positionMarkPrice = getCurrentPrice(position.symbol);
      
      const estimatesWithPrice = filteredEstimates.map(estimate => {
        const priceDiff = positionMarkPrice > 0 ? 
          ((estimate.target_price - positionMarkPrice) / positionMarkPrice * 100) : 0;
        
        return {
          ...estimate,
          current_price: positionMarkPrice,
          price_difference: priceDiff,
          is_close_to_trigger: Math.abs(priceDiff) <= 2
        };
      });
      
      setPositionEstimates(estimatesWithPrice);
    } catch (error) {
      console.error('获取仓位详情失败:', error);
      setPositionEstimates([]);
    } finally {
      setEstimatesLoading(false);
    }
  }, [getEstimatesBySymbol, getCurrentPrice]);

  // 监听positions变化，获取监听数量
  useEffect(() => {
    if (positions.length > 0) {
      fetchPositionsWithMonitors(positions);
    } else {
      setPositionsWithMonitors([]);
    }
  }, [positions, fetchPositionsWithMonitors]);

  // 监听estimates数据变化，实时更新持仓监听数量
  useEffect(() => {
    if (positions.length > 0) {
      fetchPositionsWithMonitors(positions);
    }
  }, [estimates, positions, fetchPositionsWithMonitors]);

  // 监听estimates数据变化，实时更新详情抽屉中的数据
  useEffect(() => {
    if (detailPosition && detailDrawerVisible) {
      fetchPositionDetails(detailPosition);
    }
  }, [estimates, detailPosition, detailDrawerVisible, fetchPositionDetails]);

  // 监听价格数据加载完成，更新详情抽屉中的价格
  useEffect(() => {
    if (detailPosition && detailDrawerVisible && !priceDataLoading && Object.keys(priceData).length > 0) {
      fetchPositionDetails(detailPosition);
    }
  }, [priceDataLoading, priceData, detailPosition, detailDrawerVisible, fetchPositionDetails]);

  const openDetailDrawer = (position) => {
    setDetailPosition(position);
    setDetailDrawerVisible(true);
    fetchPositionDetails(position);
  };

  const closeDetailDrawer = () => {
    setDetailDrawerVisible(false);
    setDetailPosition(null);
    setPositionEstimates([]);
  };

  const handleDeleteEstimate = async (estimateId) => {
    await deleteEstimate(estimateId);
    // 数据会通过全局estimates自动更新，无需手动刷新
  };

  const handleToggleEstimate = async (estimateId, enabled) => {
    try {
      await toggleEstimateEnabled(estimateId, enabled);
      message.success(`监听已${enabled ? '开启' : '关闭'}`);
      
      // 数据会通过全局estimates自动更新，无需手动刷新
    } catch (error) {
      message.error('切换监听状态失败');
    }
  };

  // 页面操作配置
  const headerActions = [
    <button 
      key="refresh"
      className="control-btn primary-btn trading-pairs-header-btn"
      onClick={async () => {
        setRefreshing(true);
        try {
          await refreshPositions();
        } catch (error) {
          console.error('刷新持仓失败:', error);
        } finally {
          setRefreshing(false);
        }
      }}
      disabled={refreshing || positionsLoading}
    >
      <ReloadOutlined style={{ marginRight: 4 }} />
      刷新持仓
    </button>
  ];

  if (positionsLoading) {
    return <Spin size="large" style={{ display: 'block', textAlign: 'center', padding: '50px' }} />;
  }

  return (
    <div>
      <PageHeader 
        title="持仓管理" 
        actions={headerActions}
      />

      {positionsWithMonitors.length === 0 ? (
        <Empty description="暂无持仓数据" />
      ) : (
        <Row gutter={[16, 16]}>
          {positionsWithMonitors.map((position) => (
              <Col 
                xs={24} 
                sm={12} 
                md={8} 
                lg={6} 
                xl={4} 
                key={`${position.symbol}_${position.side}`}
              >
                <PositionCard
                  position={position}
                  currentPrice={getCurrentPrice(position.symbol)}
                  onAction={handleAction}
                  onViewDetails={openDetailDrawer}
                />
              </Col>
          ))}
        </Row>
      )}

      {/* 操作抽屉 */}
      <Drawer
        title={
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <span style={{ fontSize: '16px', fontWeight: 600 }}>
              {ACTIONS[actionType]?.title}
            </span>
            {currentPosition && (
              <>
                <span style={{ fontSize: '18px', fontWeight: 700, color: '#1890ff' }}>
                  {currentPosition.symbol}
                </span>
                {/* 现货模式不显示多头/空头标签 */}
                {!isSpotMode && (
                  <span style={{
                    background: currentPosition.side === 'LONG' ? '#f6ffed' : '#fff2f0',
                    color: currentPosition.side === 'LONG' ? '#52c41a' : '#ff4d4f',
                    padding: '4px 8px',
                    borderRadius: '6px',
                    fontSize: '12px',
                    fontWeight: 600,
                    border: `1px solid ${currentPosition.side === 'LONG' ? '#b7eb8f' : '#ffb3b3'}`
                  }}>
                    {currentPosition.side === 'LONG' ? '多头' : '空头'}
                  </span>
                )}
                {/* 现货模式不显示杠杆 */}
                {!isSpotMode && (
                  <span style={{
                    background: '#f0f9ff',
                    color: '#1890ff',
                    padding: '4px 8px',
                    borderRadius: '6px',
                    fontSize: '12px',
                    fontWeight: 600,
                    border: '1px solid #bae6fd'
                  }}>
                    {currentPosition.leverage}×
                  </span>
                )}
              </>
            )}
          </div>
        }
        placement="right"
        onClose={() => {
          setDrawerVisible(false);
          setEntryTag('manual');
          setExitTag('manual');
        }}
        open={drawerVisible}
        width={window.innerWidth < 768 ? '100%' : 480}
        extra={
          <button 
            className={`drawer-confirm-btn ${
              actionType === 'addition' ? 'primary' : 
              actionType === 'take_profit' ? 'success' : 'danger'
            }`}
            onClick={handleConfirm}
            disabled={confirmLoading}
          >
            {confirmLoading ? '处理中...' : `确认${ACTIONS[actionType]?.title}`}
          </button>
        }
      >
        {currentPosition && (
          <div>
            {/* 仓位信息卡片 */}
            <div style={{ 
              background: '#f8fafc', 
              padding: 16, 
              borderRadius: 8,
              marginBottom: 24,
              border: '1px solid #e2e8f0'
            }}>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
                <div>
                  <Text type="secondary" style={{ fontSize: '12px', display: 'block' }}>当前价格</Text>
                  <Text strong style={{ fontSize: '14px', color: '#10b981' }}>
                    ${(markPrice || 0).toFixed(4)}
                  </Text>
                </div>
                <div>
                  <Text type="secondary" style={{ fontSize: '12px', display: 'block' }}>开仓价格</Text>
                  <Text strong style={{ fontSize: '14px', color: '#374151' }}>
                    ${(currentPosition.entry_price || 0).toFixed(4)}
                  </Text>
                </div>
                <div>
                  <Text type="secondary" style={{ fontSize: '12px', display: 'block' }}>持仓数量</Text>
                  <Text strong style={{ fontSize: '14px', color: '#374151' }}>
                    {Math.abs(currentPosition.size || 0).toFixed(4)}
                  </Text>
                </div>
                <div>
                  <Text type="secondary" style={{ fontSize: '12px', display: 'block' }}>未实现盈亏</Text>
                  <Text strong style={{ 
                    fontSize: '14px', 
                    color: currentPosition.unrealized_pnl >= 0 ? '#10b981' : '#ef4444' 
                  }}>
                    {(currentPosition.unrealized_pnl || 0) >= 0 ? '+' : ''}{(currentPosition.unrealized_pnl || 0).toFixed(2)} USDT
                  </Text>
                </div>
              </div>
            </div>

            {/* 价格滑块 */}
            <PriceSlider
              action={actionType}
              currentPrice={markPrice}
              entryPrice={currentPosition.entry_price}
              percentage={pricePercentage}
              onPercentageChange={handlePriceSliderChange}
              targetPrice={targetPrice}
              config={ACTIONS[actionType]}
            />

            {/* 数量滑块 */}
            <QuantitySlider
              action={actionType}
              quantity={quantity}
              maxQuantity={getMaxQuantity()}
              onQuantityChange={handleQuantitySliderChange}
              symbol={currentPosition.symbol}
              config={ACTIONS[actionType]}
            />

            {/* 入场标签选择 - 仅在加仓时显示 */}
            {actionType === 'addition' && (
              <div style={{ marginTop: 16, marginBottom: 16 }}>
                <Typography.Text strong style={{ display: 'block', marginBottom: 8, fontSize: '14px' }}>
                  入场标签
                </Typography.Text>
                <Select
                  value={entryTag || 'manual'}
                  onChange={(value) => setEntryTag(value || 'manual')}
                  placeholder="选择入场标签..."
                  allowClear
                  size="large"
                  style={{ width: '100%' }}
                  listHeight={256}
                  virtual={false}
                  getPopupContainer={(trigger) => trigger.parentElement}
                >
                  {/* 加仓标签 */}
                  <Select.Option value="manual">manual</Select.Option>
                  <Select.Option value="grind_1_entry">grind_1_entry</Select.Option>
                  <Select.Option value="grind_2_entry">grind_2_entry</Select.Option>
                  <Select.Option value="grind_3_entry">grind_3_entry</Select.Option>
                  <Select.Option value="grind_4_entry">grind_4_entry</Select.Option>
                  <Select.Option value="grind_5_entry">grind_5_entry</Select.Option>
                  <Select.Option value="force_entry">force_entry</Select.Option>
                  
                  {/* TODO: 开仓标签（在其他地方使用）
                  多头开仓: 163, 162, 161, 144, 143, 142, 141, 120, 104, 103, 102, 101, 62, 61, 46, 45, 44, 43, 42, 41, 21, 6, 5, 4, 3, 2, 1
                  空头开仓: 542, 502, 501
                  */}
                </Select>
              </div>
            )}

            {/* 退出标签选择 - 仅在止盈时显示 */}
            {actionType === 'take_profit' && (
              <div style={{ marginTop: 16, marginBottom: 16 }}>
                <Typography.Text strong style={{ display: 'block', marginBottom: 8, fontSize: '14px' }}>
                  止盈标签
                </Typography.Text>
                <Select
                  value={exitTag || 'manual'}
                  onChange={(value) => setExitTag(value || 'manual')}
                  placeholder="选择止盈标签..."
                  allowClear
                  size="large"
                  style={{ width: '100%' }}
                  listHeight={256}
                  virtual={false}
                  getPopupContainer={(trigger) => trigger.parentElement}
                >
                  {/* 止盈退出标签 */}
                  <Select.Option value="manual">manual</Select.Option>
                  <Select.Option value="grind_1_exit">grind_1_exit</Select.Option>
                  <Select.Option value="grind_2_exit">grind_2_exit</Select.Option>
                  <Select.Option value="grind_3_exit">grind_3_exit</Select.Option>
                  <Select.Option value="grind_4_exit">grind_4_exit</Select.Option>
                  <Select.Option value="grind_5_exit">grind_5_exit</Select.Option>
                  <Select.Option value="force_entry">force_entry</Select.Option>
                </Select>
              </div>
            )}

            {/* 预计盈亏 - 仅在止盈时显示 */}
            {actionType === 'take_profit' && (
              <div style={{ 
                background: '#f0f9ff', 
                padding: 16, 
                borderRadius: 8,
                marginTop: 24,
                border: '1px solid #bae6fd'
              }}>
                {(() => {
                  // 百分比转换为绝对数量再计算
                  const absoluteQty = getMaxQuantity() * (quantity / 100);
                  const expectedPnl = calculatePnl(absoluteQty, currentPosition.entry_price, targetPrice, currentPosition.side);
                  return (
                    <Text strong>预计盈亏: {(expectedPnl || 0).toFixed(2)} USDT</Text>
                  );
                })()}
              </div>
            )}
          </div>
        )}
      </Drawer>

      {/* 详情抽屉 */}
      <Drawer
        title={
          <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
            <span style={{ fontSize: '16px', fontWeight: 600, color: '#1f2937' }}>
              {detailPosition?.symbol}
            </span>
            <span style={{ 
              background: detailPosition?.side === 'LONG' ? '#f0fdf4' : '#fef2f2',
              color: detailPosition?.side === 'LONG' ? '#166534' : '#991b1b',
              padding: '3px 8px',
              borderRadius: '4px',
              fontSize: '10px',
              fontWeight: 600
            }}>
              {detailPosition?.side === 'LONG' ? '多头' : '空头'}
            </span>
            <span style={{ 
              background: '#f9fafb',
              color: '#6b7280',
              padding: '2px 6px',
              borderRadius: '4px',
              fontSize: '10px',
              fontWeight: 500
            }}>
              {positionEstimates.length}个监听
            </span>
          </div>
        }
        placement="right"
        onClose={closeDetailDrawer}
        open={detailDrawerVisible}
        width={window.innerWidth < 768 ? '100%' : 500}
        styles={{
          body: {
            paddingTop: 0,
            height: '100%'
          }
        }}
      >
        {detailPosition && (
          <div style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
            <div className="monitoring-list-container" style={{ paddingTop: '16px' }}>
              {estimatesLoading ? (
                <Spin style={{ display: 'block', textAlign: 'center', margin: '20px 0' }} />
              ) : positionEstimates.length === 0 ? (
                <Empty 
                  description="暂无价格监听" 
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                  style={{ margin: '20px 0' }}
                />
              ) : (
                <div className="monitoring-list-content">
                  {positionEstimates.map((estimate) => (
                    <MonitoringCard
                      key={estimate.id}
                      estimate={estimate}
                      currentPosition={detailPosition}
                      onDelete={handleDeleteEstimate}
                      onToggle={handleToggleEstimate}
                    />
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
      </Drawer>
    </div>
  );
};

export default Positions;
