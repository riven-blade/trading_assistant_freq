// 计算工具函数

/**
 * 计算所需USDT保证金
 * @param {number} quantity - 数量
 * @param {number} price - 价格
 * @param {number} leverage - 杠杆倍数
 * @returns {number} 所需USDT金额
 */
export const calculateUsdtAmount = (quantity, price, leverage = 1) => {
  if (!quantity || !price) return 0;
  return (quantity * price) / leverage;
};

/**
 * 计算盈亏用于显示
 * @param {number} quantity - 数量
 * @param {number} entryPrice - 开仓价格
 * @param {number} targetPrice - 目标价格
 * @param {string} side - 持仓方向 ('LONG' | 'SHORT')
 * @returns {number} 盈亏金额
 */
export const calculatePnl = (quantity, entryPrice, targetPrice, side) => {
  if (!quantity || !entryPrice || !targetPrice) return 0;
  
  const isLong = side === 'LONG';
  let profitPerCoin;
  
  if (isLong) {
    profitPerCoin = targetPrice - entryPrice;
  } else {
    profitPerCoin = entryPrice - targetPrice;
  }
  
  return quantity * profitPerCoin;
};

/**
 * 格式化数字为指定的有效数字位数
 * @param {number} value - 要格式化的数字
 * @param {number} significantDigits - 有效数字位数，默认6位
 * @returns {string} 格式化后的字符串
 */
export const formatSignificantDigits = (value, significantDigits = 6) => {
  if (!value || isNaN(value)) return '0';
  
  // 使用 toPrecision 获取指定有效数字
  const formatted = Number(value).toPrecision(significantDigits);
  
  // 转换为数字再转回字符串，去除不必要的科学计数法和尾随零
  const num = parseFloat(formatted);
  
  // 如果数字太小或太大，使用科学计数法
  if (Math.abs(num) < 0.000001 || Math.abs(num) >= 1000000) {
    return num.toExponential(significantDigits - 1);
  }
  
  // 否则返回普通格式
  return num.toString();
};

/**
 * 根据价格精度格式化价格
 * @param {number} value - 要格式化的价格
 * @param {number} pricePrecision - 价格精度（小数位数，从后端获取）
 * @returns {string} 格式化后的价格字符串
 */
export const formatPrice = (value, pricePrecision) => {
  if (!value || isNaN(value)) return '0';
  
  // 如果有明确的价格精度，使用 toFixed
  if (pricePrecision !== undefined && pricePrecision !== null && pricePrecision >= 0) {
    return Number(value).toFixed(pricePrecision);
  }
  
  // 否则使用6个有效数字作为后备方案
  return formatSignificantDigits(value, 6);
};