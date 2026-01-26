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

  const isPercentMode = action === 'addition' || action === 'take_profit';
  const isAddition = action === 'addition';

  // 加仓比例：0-200%
  const additionMarks = {
    0: '0%',
    50: '50%',
    100: '100%',
    150: '150%',
    200: '200%'
  };

  // 止盈比例：0-100%
  const takeProfitMarks = {
    0: '0%',
    25: '25%',
    50: '50%',
    75: '75%',
    100: '100%'
  };

  // 数量模式
  const quantityMarks = {
    0: '0%',
    [maxQuantity * 0.25]: '25%',
    [maxQuantity * 0.5]: '50%',
    [maxQuantity * 0.75]: '75%',
    [maxQuantity]: '100%'
  };

  // 根据操作类型选择marks和max
  const getMarksAndMax = () => {
    if (isAddition) {
      return { marks: additionMarks, max: 200 };
    } else if (isPercentMode) {
      return { marks: takeProfitMarks, max: 100 };
    }
    return { marks: quantityMarks, max: maxQuantity };
  };

  const { marks, max } = getMarksAndMax();

  return (
    <TradingSlider
      title={config.quantityLabel || `${baseAsset}数量`}
      value={quantity}
      min={isPercentMode ? 0 : 0.001}
      max={max}
      step={isPercentMode ? 1 : 0.001}
      onChange={onQuantityChange}
      marks={marks}
      tooltipFormatter={(value) => {
        if (isPercentMode) {
          return `${value}%`;
        }
        return `${value?.toFixed(6)} ${baseAsset}`;
      }}
      action={action}
    />
  );
};

export default QuantitySlider;
