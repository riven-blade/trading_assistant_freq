import React from 'react';
import TradingSlider from './TradingSlider';

/**
 * 通用数量滑块组件
 * @param {string} action - 操作类型
 * @param {number} quantity - 当前数量
 * @param {number} maxQuantity - 最大数量
 * @param {Function} onQuantityChange - 数量变化回调
 * @param {string} symbol - 交易对符号
 * @param {Object} config - 操作配置
 */
const QuantitySlider = ({ 
  action, 
  quantity, 
  maxQuantity, 
  onQuantityChange, 
  symbol,
  config 
}) => {
  const baseAsset = symbol.replace('USDT', '');
  
  // 统一使用数量模式
  const optimizedMarks = {
    0: '0%',
    [maxQuantity * 0.25]: '25%',
    [maxQuantity * 0.5]: '50%',
    [maxQuantity * 0.75]: '75%',
    [maxQuantity]: '100%'
  };

  return (
    <TradingSlider
      title={config.quantityLabel || `${baseAsset}数量`}
      value={quantity}
      min={0.001}
      max={maxQuantity}
      step={0.001}
      onChange={onQuantityChange}
      marks={optimizedMarks}
      displayLabel="当前数量:"
      displayValue={`${quantity.toFixed(6)} ${baseAsset}`}
      tooltipFormatter={(value) => `${value?.toFixed(6)} ${baseAsset}`}
      action={action}
    />
  );
};

export default QuantitySlider;
