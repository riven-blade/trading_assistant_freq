import React from 'react';
import { Switch } from 'antd';
import { getDetailedActionText } from '../../utils/constants';

/**
 * 监听卡片组件
 * @param {Object} estimate - 监听数据
 * @param {Object} currentPosition - 当前持仓信息
 * @param {Function} onDelete - 删除回调
 * @param {Function} onToggle - 监听开关回调
 */
const MonitoringCard = ({ estimate, currentPosition, onDelete, onToggle }) => {
  // 使用详细的操作类型文本
  const actionText = getDetailedActionText(estimate.action_type, estimate.side);

  // 判断是否为基于仓位的操作
  const isPositionBasedAction = ['take_profit', 'addition'].includes(estimate.action_type);
  
  // 计算实际交易数量(用于预估盈亏计算)
  const calculateActualQuantity = () => {
    if (!currentPosition || !currentPosition.size || currentPosition.size === 0) {
      return 0;
    }
    
    if (!estimate.percentage || isNaN(estimate.percentage)) {
      return 0;
    }
    
    // percentage 是 0-100 的比例，需要除以 100
    return Math.abs(currentPosition.size) * (estimate.percentage / 100);
  };

  // 计算预估盈利/止损
  const calculateEstimatedPnL = () => {
    // 只有止盈操作才计算盈亏
    if (estimate.action_type !== 'take_profit') {
      return 0;
    }
    
    if (!currentPosition || !estimate.target_price || !currentPosition.entry_price) {
      return 0;
    }

    const actualQuantity = calculateActualQuantity();
    if (actualQuantity === 0) {
      return 0;
    }

    const isLong = currentPosition.side === 'LONG';
    const priceDiff = isLong 
      ? (estimate.target_price - currentPosition.entry_price)
      : (currentPosition.entry_price - estimate.target_price);
    
    return priceDiff * actualQuantity;
  };

  // 计算仓位比例
  const calculatePositionRatio = () => {
    // 开仓操作不计算仓位比例
    if (!isPositionBasedAction) {
      return null;
    }
    
    if (!estimate.percentage || isNaN(estimate.percentage)) {
      return null;
    }
    
    // percentage 是 0-100 的比例，直接返回
    return estimate.percentage.toFixed(1);
  };

  const estimatedPnL = calculateEstimatedPnL();
  const isProfit = estimatedPnL >= 0;
  const positionRatio = calculatePositionRatio();


  // 格式化数字显示
  return (
    <div className="monitoring-card-clean">
      {/* 头部信息 */}
      <div className="monitoring-header-clean">
        <div className="monitoring-info-row">
          <span className={`monitoring-action-tag ${estimate.action_type} ${estimate.side}`}>
            {actionText}
          </span>
          <span className="monitoring-order-type">{estimate.order_type}</span>
        </div>
        <div className="monitoring-controls">
          <div className="monitor-toggle-switch">
            <Switch
              size="small"
              checked={estimate.enabled}
              onChange={(checked) => onToggle && onToggle(estimate.id, checked)}
              title={estimate.enabled ? "关闭监听" : "开启监听"}
            />
          </div>
          <button 
            className="monitoring-delete-btn"
            onClick={() => onDelete(estimate.id)}
            title="删除监听"
          >
            ×
          </button>
        </div>
      </div>

      {/* 价格核心信息 */}
      <div className="monitoring-price-section">
        <div className="price-row main-prices">
          <div className="price-item">
            <span className="price-label">当前</span>
            <span className="price-value current">
              {estimate.current_price && estimate.current_price > 0 
                ? `$${estimate.current_price.toFixed(4)}`
                : "加载中..."
              }
            </span>
          </div>
          <div className="price-item">
            <span className="price-label">目标</span>
            <span className="price-value target">
              ${estimate.target_price.toFixed(4)}
            </span>
          </div>
        </div>

        {/* 价格变化百分比 */}
        <div className="price-change-indicator">
          {estimate.current_price && estimate.current_price > 0 ? (
            <>
              <span className={`price-change-value ${
                estimate.price_difference >= 0 ? 'positive' : 'negative'
              }`}>
                {estimate.price_difference >= 0 ? '+' : ''}{estimate.price_difference.toFixed(2)}%
              </span>
              {estimate.is_close_to_trigger && (
                <span className="trigger-warning">接近触发</span>
              )}
            </>
          ) : (
            <span className="price-change-value loading">等待价格数据...</span>
          )}
        </div>
      </div>

      {/* 交易详情 */}
      <div className="monitoring-details">
        {/* 仓位比例 - 只对基于仓位的操作显示 */}
        {isPositionBasedAction && (
          <div className="detail-row">
            <span className="detail-label">仓位比例</span>
            <span className="detail-value">
              {positionRatio ? `${positionRatio}%` : 'N/A'}
            </span>
          </div>
        )}
        
        {/* 预估盈亏 - 只对止盈操作显示 */}
        {estimate.action_type === 'take_profit' && (
          <div className="detail-row">
            <span className="detail-label">预估盈亏</span>
            <span className={`detail-value ${isProfit ? 'profit' : 'loss'}`}>
              {estimatedPnL !== 0 
                ? `${isProfit ? '+' : ''}${estimatedPnL.toFixed(2)} USDT`
                : 'N/A'
              }
            </span>
          </div>
        )}
        <div className="detail-row">
          <span className="detail-label">创建时间</span>
          <span className="detail-value time">
            {new Date(estimate.created_at).toLocaleString()}
          </span>
        </div>
      </div>
    </div>
  );
};

export default MonitoringCard;
