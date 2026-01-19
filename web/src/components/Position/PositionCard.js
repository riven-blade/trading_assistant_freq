import React from 'react';
import { useSystemConfig } from '../../hooks/useSystemConfig';

/**
 * Grind 状态徽章组件
 * 显示各个 grind 类型的入场/退出状态
 */
const GrindStatusBadges = ({ grindSummary }) => {
  if (!grindSummary) return null;

  const grinds = [
    { key: 'grind_1', label: 'G1', data: grindSummary.grind_1 },
    { key: 'grind_2', label: 'G2', data: grindSummary.grind_2 },
    { key: 'grind_3', label: 'G3', data: grindSummary.grind_3 },
    { key: 'grind_x', label: 'Gx', data: grindSummary.grind_x },
  ];

  // 只显示有入场或有退出的 grind
  const activeGrinds = grinds.filter(g => g.data?.has_entry || g.data?.has_exit);
  
  if (activeGrinds.length === 0) return null;

  return (
    <div style={{ display: 'flex', gap: '3px', marginLeft: '4px' }}>
      {activeGrinds.map(grind => {
        const hasEntry = grind.data?.has_entry;
        const hasExit = grind.data?.has_exit;
        
        // 有入场未退出：绿色（活跃状态）
        // 有退出：灰色（已完成）
        // 只有入场没退出：橙色闪烁（需关注）
        let bgColor = '#52c41a'; // 默认绿色
        let textColor = '#fff';
        let animation = '';
        
        if (hasEntry && !hasExit) {
          bgColor = '#52c41a'; // 绿色 - 有入场未退出
        } else if (hasExit && !hasEntry) {
          bgColor = '#8c8c8c'; // 灰色 - 已退出无新入场
        } else if (hasEntry && hasExit) {
          bgColor = '#faad14'; // 橙色 - 有入场且之前有退出
        }
        
        return (
          <span
            key={grind.key}
            title={`${grind.label}: ${hasEntry ? '有入场' : '无入场'}, ${hasExit ? '有退出' : '无退出'}`}
            style={{
              fontSize: '9px',
              padding: '1px 4px',
              borderRadius: '3px',
              backgroundColor: bgColor,
              color: textColor,
              fontWeight: 600,
              lineHeight: '14px',
              animation: animation,
            }}
          >
            {grind.label}
          </span>
        );
      })}
    </div>
  );
};

/**
 * 持仓卡片组件
 * @param {Object} position - 持仓信息
 * @param {number} currentPrice - 实时标记价格
 * @param {Function} onAction - 操作回调 (position, action)
 * @param {Function} onViewDetails - 查看详情回调
 * @param {Function} onGrindDetail - 查看 Grind 详情回调
 */
const PositionCard = ({ position, currentPrice, onAction, onViewDetails, onGrindDetail }) => {
  // 获取系统配置
  const { isSpotMode } = useSystemConfig();
  // 使用实时价格计算盈亏
  const realTimeMarkPrice = currentPrice || position.mark_price;

  // 计算持仓占比：当前保证金 / (第一个订单cost / 杠杆)
  const calculatePositionRatio = () => {
    // 获取第一个订单的cost
    const firstOrderCost = position.orders?.[0]?.cost;
    if (!firstOrderCost || firstOrderCost <= 0) return null;

    const leverage = position.leverage || 1;
    // notional 是映射后的 stake_amount
    const currentMargin = position.notional || 0;

    // 最大持仓金额 = 第一个订单cost / 杠杆 / 0.15  0.15  是freqtrade 的初始仓位
    const maxPositionAmount = firstOrderCost / leverage / 0.15;

    if (maxPositionAmount <= 0) return null;

    // 持仓占比 = 当前保证金 / 最大持仓金额
    const ratio = (currentMargin / maxPositionAmount) * 100;

    return ratio;
  };

  const positionRatio = calculatePositionRatio();
  const isOverThreshold = positionRatio !== null && positionRatio > 50;
  
  // 计算实时未实现盈亏
  const realTimeUnrealizedPnl = (() => {
    if (!realTimeMarkPrice || !position.entry_price || !position.size) return 0;
    
    // 确保所有值都是有效数字
    const markPrice = Number(realTimeMarkPrice) || 0;
    const entryPrice = Number(position.entry_price) || 0;
    const size = Number(position.size) || 0;
    
    if (markPrice === 0 || entryPrice === 0 || size === 0) return 0;
    
    const isLong = (position.side || 'LONG') === 'LONG';
    const priceDiff = isLong 
      ? (markPrice - entryPrice)
      : (entryPrice - markPrice);
    
    return priceDiff * size;
  })();
  
  const isProfit = (realTimeUnrealizedPnl || 0) >= 0;
  
  // 计算保证金 = 实际占用保证金（stake_amount）
  const margin = position.notional || 0;
  
  // 计算盈亏百分比 = 未实现盈亏 / 保证金 * 100%
  const pnlPercentage = (margin > 0 && !isNaN(realTimeUnrealizedPnl)) 
    ? (realTimeUnrealizedPnl / margin * 100) 
    : 0;

  // 格式化数字显示
  const formatSize = (size) => {
    if (size === undefined || size === null || isNaN(size)) return '0.000';
    const numSize = Number(size);
    const abs = Math.abs(numSize);
    if (abs >= 1000) return (abs / 1000).toFixed(2) + 'K';
    if (abs >= 1) return abs.toFixed(3);
    return abs.toFixed(6);
  };

  const formatPrice = (price) => {
    if (price === undefined || price === null || isNaN(price)) return '0.0000';
    const numPrice = Number(price);
    if (numPrice >= 1000) return numPrice.toFixed(2);
    if (numPrice >= 1) return numPrice.toFixed(4);
    return numPrice.toFixed(6);
  };

  const formatValue = (value) => {
    if (value === undefined || value === null || isNaN(value)) return '0';
    const numValue = Number(value);
    const abs = Math.abs(numValue);
    if (abs >= 1000000) return (abs / 1000000).toFixed(1) + 'M';
    if (abs >= 1000) return (abs / 1000).toFixed(1) + 'K';
    return abs.toFixed(0);
  };

  return (
    <div className="position-card-clean">
      {/* 头部信息条 - 点击打开 Grind 详情 */}
      <div 
        className="position-header-clean"
        onClick={() => onGrindDetail && onGrindDetail(position)}
        style={{ cursor: onGrindDetail ? 'pointer' : 'default' }}
      >
        <div className="position-header-top">
          <div className="position-info-row">
            <span className="position-symbol-clean">
              {(() => {
                const symbol = position.symbol || 'N/A';
                return symbol.length > 8 ? symbol.substring(0, 8) + '...' : symbol;
              })()}
            </span>
            <div className="position-badges">
              {/* 现货模式不显示方向徽章（只有买入） */}
              {!isSpotMode && (
                <span className={`side-badge ${(position.side || 'long').toLowerCase()}`}>
                  {position.side?.toLowerCase() === 'long' ? 'L' : 'S'}
                </span>
              )}
              {/* 现货模式不显示杠杆徽章 */}
              {!isSpotMode && (
                <span className="leverage-badge">{position.leverage || 1}×</span>
              )}
            </div>
          </div>
          <div className={`pnl-display ${isProfit ? 'positive' : 'negative'}`}>
            <div className="pnl-amount">
              {isProfit ? '+' : ''}{(realTimeUnrealizedPnl || 0).toFixed(2)}
            </div>
            <div className="pnl-percentage">
              {isProfit ? '+' : ''}{(pnlPercentage || 0).toFixed(2)}%
            </div>
          </div>
        </div>
        {/* Grind 状态指示器 - 单独一行 */}
        {position.grind_summary && (
          <GrindStatusBadges grindSummary={position.grind_summary} />
        )}
      </div>

      {/* 核心数据 */}
      <div className="position-data-section">
        <div className="data-row">
          <div className="data-group">
            <span className="data-label">仓位</span>
            <span className="data-value">{formatSize(position.size)}</span>
          </div>
          <div className="data-group">
            <span className="data-label">{isSpotMode ? '持仓价值' : '保证金'}</span>
            <span className="data-value">
              ${formatValue(margin)}
              {/* 持仓占比仅期货模式显示 */}
              {!isSpotMode && positionRatio !== null && (
                <span style={{
                  color: isOverThreshold ? '#ff4d4f' : '#52c41a',
                  fontSize: '11px',
                  marginLeft: '4px'
                }}>
                  ({positionRatio.toFixed(0)}%)
                </span>
              )}
            </span>
          </div>
        </div>
        <div className="data-row">
          <div className="data-group">
            <span className="data-label">开仓</span>
            <span className="data-value">${formatPrice(position.entry_price)}</span>
          </div>
          <div className="data-group">
            <span className="data-label">标记</span>
            <span className="data-value">${formatPrice(realTimeMarkPrice)}</span>
          </div>
        </div>
      </div>

      {/* 操作按钮 */}
      <div className="position-controls">
        <div className="control-group" style={{ display: 'flex', gap: '8px' }}>
          <button 
            className="control-btn primary-btn"
            onClick={() => onAction(position, 'addition')}
            style={{ flex: 1, position: 'relative' }}
          >
            加仓
            {position.additionCount > 0 && (
              <span className="action-count-badge">
                {position.additionCount}
              </span>
            )}
          </button>
          <button 
            className="control-btn info-btn"
            onClick={() => onViewDetails(position)}
            style={{ flex: 1, position: 'relative' }}
          >
            监听
            {position.monitorCount > 0 && (
              <span className="action-count-badge">
                {position.monitorCount}
              </span>
            )}
          </button>
          <button
            className="control-btn danger-btn"
            onClick={() => onAction(position, 'kline')}
            style={{ flex: 1 }}
          >
            K线
          </button>
        </div>
      </div>
    </div>
  );
};

export default PositionCard;
