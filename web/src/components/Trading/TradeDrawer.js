import React, { useState, useEffect } from 'react';
import { Drawer, Form, Typography, Tag, Select, Button, InputNumber } from 'antd';
import PriceSlider from './PriceSlider';
import { ACTIONS } from '../../utils/constants';
import { formatPrice } from '../../utils/precision';

const { Text } = Typography;

/**
 * 交易抽屉组件 - 以数量为主
 * @param {boolean} visible - 是否显示
 * @param {Function} onClose - 关闭回调
 * @param {string} symbol - 交易对
 * @param {string} side - 交易方向 ('long' | 'short')
 * @param {Function} onSubmit - 提交回调
 * @param {Object} priceData - 价格数据
 * @param {number} targetPrice - 目标价格
 * @param {Function} onPriceChange - 价格变化回调
 * @param {string} orderType - 订单类型
 * @param {Function} onOrderTypeChange - 订单类型变化回调
 * @param {number} selectedLeverage - 杠杆
 * @param {Function} onLeverageChange - 杠杆变化回调
 * @param {string} entryTag - 入场标签
 * @param {Function} onEntryTagChange - 入场标签变化回调
 * @param {number} stakeAmount - 开仓金额
 * @param {Function} onStakeAmountChange - 开仓金额变化回调
 * @param {number} pricePrecision - 价格精度（小数位数）
 */
const TradeDrawer = ({
  visible,
  onClose,
  symbol,
  side,
  onSubmit,
  priceData,
  targetPrice,
  onPriceChange,
  orderType,
  onOrderTypeChange,
  selectedLeverage,
  onLeverageChange,
  entryTag,
  onEntryTagChange,
  stakeAmount,
  onStakeAmountChange,
  pricePrecision
}) => {
  // 百分比状态管理
  const [pricePercentage, setPricePercentage] = useState(0);
  
  // 定义入场标签选项
  const longEntryTags = [120];
  const shortEntryTags = [620];
  
  // 获取当前方向对应的标签选项
  const availableTags = side === 'long' ? longEntryTags : shortEntryTags;
  
  // 获取当前价格
  const markPrice = priceData?.[symbol]?.markPrice || 0;
  
  // 价格百分比变化处理
  const handlePricePercentageChange = (percentage) => {
    setPricePercentage(percentage);
    
    // 基于当前价格计算目标价格
    const newTargetPrice = markPrice * (1 + percentage / 100);
    onPriceChange(newTargetPrice);
  };

  // 仅在组件打开时初始化，避免任何重置逻辑
  useEffect(() => {
    if (visible) {
      setPricePercentage(0);
      const currentMarkPrice = priceData?.[symbol]?.markPrice || 0;
      if (currentMarkPrice > 0) {
        onPriceChange(currentMarkPrice);
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visible]); // 只依赖visible，避免价格更新时重置

  return (
    <Drawer
      title={
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span>{side === 'long' ? '开多' : '开空'} {symbol && symbol.length > 8 ? symbol.substring(0, 8) + '...' : symbol}</span>
          <Tag color={side === 'long' ? 'green' : 'red'}>
            {side === 'long' ? 'LONG' : 'SHORT'}
          </Tag>
        </div>
      }
      placement="right"
      width={400}
      onClose={onClose}
      open={visible}
      extra={
        <Button 
          type="primary" 
          onClick={() => onSubmit(entryTag)}
        >
          确认{side === 'long' ? '开多' : '开空'}
        </Button>
      }
    >
      <Form layout="vertical">
        {/* 交易信息概要 */}
        <div style={{ 
          background: side === 'long' ? '#f6ffed' : '#fff2f0', 
          border: `2px solid ${side === 'long' ? '#b7eb8f' : '#ffccc7'}`,
          padding: 16, 
          borderRadius: 8, 
          marginBottom: 16 
        }}>
          {/* 标题行 */}
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
            <Text strong style={{ color: side === 'long' ? '#52c41a' : '#ff4d4f', fontSize: '16px' }}>
              交易概要
            </Text>
            <Text strong style={{ color: side === 'long' ? '#52c41a' : '#ff4d4f' }}>
              {side === 'long' ? '做多 (LONG)' : '做空 (SHORT)'}
            </Text>
          </div>
          
          {/* 价格信息 */}
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
            <Text type="secondary">当前价格:</Text>
            <Text strong style={{ color: '#1890ff', fontSize: '16px' }}>${formatPrice(markPrice, pricePrecision)}</Text>
          </div>
          
          {/* 交易参数 */}
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
            <Text type="secondary">开仓金额:</Text>
            <Text strong style={{ color: '#fa8c16' }}>
              {stakeAmount ? `${stakeAmount} USDT` : '未设置'}
            </Text>
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
            <Text type="secondary">杠杆倍数:</Text>
            <Text strong>{selectedLeverage}x</Text>
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
            <Text type="secondary">入场标签:</Text>
            <Text strong style={{ color: '#1890ff' }}>
              {entryTag || 'manual'}
            </Text>
          </div>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
            <Text type="secondary">保证金模式:</Text>
            <Text strong style={{ color: '#fa8c16' }}>
              逐仓
            </Text>
          </div>
          {orderType === 'limit' && (
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
              <Text type="secondary">限价价格:</Text>
              <Text strong>${formatPrice(targetPrice || markPrice, pricePrecision)}</Text>
            </div>
          )}
        </div>

        {/* 开仓金额设置 */}
        <Form.Item
          label="开仓金额 (USDT)"
        >
          <InputNumber
            value={stakeAmount}
            onChange={onStakeAmountChange}
            placeholder="请输入开仓金额"
            min={0}
            max={100000}
            step={1}
            precision={2}
            style={{ width: '100%' }}
            size="large"
            addonAfter="USDT"
          />
        </Form.Item>

        {/* 订单类型选择 */}
        <Form.Item label="订单类型">
          <Select 
            value={orderType} 
            onChange={onOrderTypeChange}
            style={{ width: '100%' }}
            size="large"
            getPopupContainer={(trigger) => trigger.parentElement}
          >
            <Select.Option value="market">市价单</Select.Option>
            <Select.Option value="limit">限价单</Select.Option>
          </Select>
        </Form.Item>

        {/* 限价单价格设置 */}
        {orderType === 'limit' && (
          <Form.Item>
            <PriceSlider
              action="open"
              currentPrice={markPrice}
              entryPrice={markPrice}
              percentage={pricePercentage}
              onPercentageChange={handlePricePercentageChange}
              targetPrice={targetPrice || markPrice}
              config={ACTIONS.open}
              pricePrecision={pricePrecision}
            />
          </Form.Item>
        )}



        {/* 杠杆设置 */}
        <Form.Item label="杠杆倍数">
          <Select 
            value={selectedLeverage} 
            onChange={onLeverageChange}
            style={{ width: '100%' }}
            size="large"
            getPopupContainer={(trigger) => trigger.parentElement}
          >
            {[1, 2, 3, 4, 5].map(leverage => (
              <Select.Option key={leverage} value={leverage}>
                {leverage}x
              </Select.Option>
            ))}
          </Select>
        </Form.Item>

        {/* 入场标签设置 */}
        <Form.Item label="入场标签">
          <Select 
            value={entryTag} 
            onChange={onEntryTagChange}
            style={{ width: '100%' }}
            placeholder="选择入场标签"
            allowClear
            size="large"
            listHeight={256}
            virtual={false}
            getPopupContainer={(trigger) => trigger.parentElement}
          >
            <Select.Option key="manual" value="manual">
              manual
            </Select.Option>
            <Select.Option key="force_entry" value="force_entry">
              force_entry
            </Select.Option>
            {availableTags.map(tag => (
              <Select.Option key={tag} value={tag}>
                {tag}
              </Select.Option>
            ))}
          </Select>
        </Form.Item>


      </Form>
    </Drawer>
  );
};

export default TradeDrawer;