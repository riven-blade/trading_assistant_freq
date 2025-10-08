import React from 'react';
import TradingSlider from './TradingSlider';
import { formatPrice } from '../../utils/precision';

/**
 * 通用价格滑块组件
 * @param {string} action - 操作类型
 * @param {number} currentPrice - 当前价格
 * @param {number} entryPrice - 开仓价格
 * @param {number} percentage - 价格百分比
 * @param {Function} onPercentageChange - 百分比变化回调
 * @param {number} targetPrice - 目标价格
 * @param {Object} config - 操作配置
 * @param {number} pricePrecision - 价格精度（小数位数，从后端获取）
 */
const PriceSlider = ({ 
  action, 
  currentPrice, 
  entryPrice, 
  percentage, 
  onPercentageChange, 
  targetPrice, 
  config,
  pricePrecision
}) => {
  
  // 简化marks，避免文字重叠
  const simplifiedMarks = {
    [config.priceRange.min]: `${config.priceRange.min}%`,
    0: config.priceBase === 'current' ? '当前价格' : '开仓价格',
    [config.priceRange.max]: `+${config.priceRange.max}%`
  };

  // 格式化价格显示：根据后端返回的价格精度动态调整
  const formattedPrice = formatPrice(targetPrice, pricePrecision);

  return (
    <TradingSlider
      title={config.priceLabel}
      value={percentage}
      min={config.priceRange.min}
      max={config.priceRange.max}
      step={0.1}
      onChange={onPercentageChange}
      marks={simplifiedMarks}
      displayValue={`$${formattedPrice}`}
      tooltipFormatter={(value) => `${value > 0 ? '+' : ''}${value}%`}
      action={action}
    />
  );
};

export default PriceSlider;
