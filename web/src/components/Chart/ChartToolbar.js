import React from 'react';
import { Typography } from 'antd';
import { formatPrice, formatPercentage } from '../../utils/priceFormatter';

const { Text } = Typography;

// 时间周期配置
const TIME_INTERVALS = [
  { value: '1m', label: '1m' },
  { value: '5m', label: '5m' },
  { value: '15m', label: '15m' },
  { value: '1h', label: '1h' },
  { value: '4h', label: '4h' },
  { value: '1d', label: '1d' },
  { value: '1w', label: '1w' }
];

const ChartToolbar = ({
  selectedCoin,
  currentPrice,
  priceChange,
  selectedInterval,
  onIntervalChange,
  loading = false,
  isMobile = false
}) => {
  
  // 判断价格涨跌
  const isPriceUp = priceChange && priceChange > 0;
  const isPriceDown = priceChange && priceChange < 0;
  const priceColor = isPriceUp ? '#26a69a' : isPriceDown ? '#ef5350' : '#666';
  
  return (
    <div style={{
      display: 'flex',
      flexDirection: isMobile ? 'column' : 'row',
      alignItems: isMobile ? 'stretch' : 'center',
      justifyContent: 'space-between',
      padding: isMobile ? '8px 12px' : '12px 20px',
      background: '#ffffff',
      borderBottom: '1px solid #f0f0f0',
      gap: isMobile ? '8px' : '0'
    }}>
      {/* 左侧：币种信息和当前价格 */}
      <div style={{ 
        display: 'flex', 
        alignItems: 'center', 
        gap: isMobile ? '12px' : '20px',
        flexWrap: 'nowrap'
      }}>
        {/* 币种名称 */}
        <div style={{
          display: 'flex',
          alignItems: 'center',
          gap: '8px'
        }}>
          <Text strong style={{ 
            fontSize: isMobile ? '16px' : '18px',
            color: '#262626'
          }}>
            {selectedCoin ? selectedCoin.replace('/', '') : 'BTCUSDT'}
          </Text>
        </div>

        {/* 当前价格 */}
        {currentPrice && (
          <div style={{
            display: 'flex',
            alignItems: 'baseline',
            gap: '8px'
          }}>
            <Text strong style={{ 
              fontSize: isMobile ? '18px' : '20px',
              color: priceColor,
              fontFamily: 'monospace'
            }}>
              ${formatPrice(currentPrice)}
            </Text>
            {priceChange !== undefined && priceChange !== null && (
              <Text style={{ 
                fontSize: isMobile ? '12px' : '14px',
                color: priceColor,
                fontWeight: '500'
              }}>
                {isPriceUp ? '+' : ''}{formatPercentage(priceChange)}%
              </Text>
            )}
          </div>
        )}
      </div>

      {/* 右侧：时间周期选择 */}
      <div style={{ 
        display: 'flex', 
        gap: isMobile ? '2px' : '4px',
        alignItems: 'center'
      }}>
        {TIME_INTERVALS.map(interval => (
          <button
            key={interval.value}
            onClick={() => onIntervalChange(interval.value)}
            style={{
              padding: isMobile ? '6px 10px' : '6px 12px',
              fontSize: isMobile ? '13px' : '13px',
              border: 'none',
              borderRadius: '4px',
              backgroundColor: selectedInterval === interval.value ? '#1890ff' : 'transparent',
              color: selectedInterval === interval.value ? '#ffffff' : '#666',
              cursor: 'pointer',
              transition: 'all 0.15s',
              fontWeight: selectedInterval === interval.value ? '500' : '400',
              minWidth: isMobile ? '36px' : '40px',
              height: isMobile ? '32px' : '32px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center'
            }}
            onMouseEnter={(e) => {
              if (selectedInterval !== interval.value) {
                e.target.style.backgroundColor = 'rgba(0,0,0,0.04)';
              }
            }}
            onMouseLeave={(e) => {
              if (selectedInterval !== interval.value) {
                e.target.style.backgroundColor = 'transparent';
              }
            }}
          >
            {interval.label}
          </button>
        ))}
      </div>
    </div>
  );
};

export default ChartToolbar;