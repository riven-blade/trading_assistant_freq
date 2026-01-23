/**
 * 智能格式化价格，根据价格大小自动调整小数位数
 * @param {number} price - 价格值
 * @param {number} minDecimals - 最小小数位数（默认2）
 * @param {number} maxDecimals - 最大小数位数（默认8）
 * @returns {string} 格式化后的价格字符串
 */
export const formatPrice = (price, minDecimals = 2, maxDecimals = 8) => {
  if (price === null || price === undefined || isNaN(price)) {
    return '0.00';
  }

  const absPrice = Math.abs(price);
  
  // 根据价格大小决定小数位数
  let decimals;
  
  if (absPrice >= 1000) {
    decimals = 2; // 大于1000: 2位小数
  } else if (absPrice >= 1) {
    decimals = 2; // 1-1000: 2位小数
  } else if (absPrice >= 0.01) {
    decimals = 4; // 0.01-1: 4位小数
  } else if (absPrice >= 0.0001) {
    decimals = 6; // 0.0001-0.01: 6位小数
  } else {
    decimals = 8; // 小于0.0001: 8位小数
  }
  
  // 确保在最小和最大范围内
  decimals = Math.max(minDecimals, Math.min(maxDecimals, decimals));
  
  return price.toFixed(decimals);
};

/**
 * 格式化价格变化百分比
 * @param {number} percentage - 百分比值
 * @returns {string} 格式化后的百分比字符串
 */
export const formatPercentage = (percentage) => {
  if (percentage === null || percentage === undefined || isNaN(percentage)) {
    return '0.00';
  }
  
  return percentage.toFixed(2);
};
