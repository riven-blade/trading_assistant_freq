import React, { useState, useEffect } from 'react';
import { Drawer, Progress, Empty, InputNumber, Button, message, Modal } from 'antd';

/**
 * Grind 详情抽屉组件
 * 显示持仓的 grind 分布详情，并支持退出操作
 */
const GrindDetailDrawer = ({ visible, position, onClose, onTakeProfit }) => {
  const [exitAmounts, setExitAmounts] = useState({});
  const [loading, setLoading] = useState({});
  
  // 当 position 变化时，清空输入框状态
  useEffect(() => {
    setExitAmounts({});
    setLoading({});
  }, [position?.symbol, position?.side]);

  if (!position) return null;

  const grindSummary = position.grind_summary;
  const grinds = grindSummary ? [
    { key: 'grind_x', label: 'Grind X', data: grindSummary.grind_x, color: '#1890ff' },
    { key: 'grind_1', label: 'Grind 1', data: grindSummary.grind_1, color: '#52c41a' },
    { key: 'grind_2', label: 'Grind 2', data: grindSummary.grind_2, color: '#faad14' },
    { key: 'grind_3', label: 'Grind 3', data: grindSummary.grind_3, color: '#722ed1' },
  ] : [];

  // 过滤出有数据的 grind
  const activeGrinds = grinds.filter(g => 
    g.data?.has_entry || g.data?.has_exit || g.data?.total_amount > 0
  );

  const formatNumber = (num, decimals = 4) => {
    if (num === undefined || num === null || isNaN(num)) return '0';
    return Number(num).toFixed(decimals);
  };

  const formatPercent = (num) => {
    if (num === undefined || num === null || isNaN(num)) return '0';
    return Number(num).toFixed(1);
  };

  const getStatusText = (grind) => {
    const hasEntry = grind.data?.has_entry;
    const hasExit = grind.data?.has_exit;
    
    if (hasEntry && !hasExit) {
      return { text: '持仓中', color: '#52c41a', bg: '#f6ffed', canExit: true };
    } else if (hasExit && !hasEntry) {
      return { text: '已退出', color: '#8c8c8c', bg: '#fafafa', canExit: false };
    } else if (hasEntry && hasExit) {
      return { text: '活跃中', color: '#faad14', bg: '#fffbe6', canExit: true };
    }
    return { text: '无', color: '#d9d9d9', bg: '#fafafa', canExit: false };
  };

  const handleExitAmountChange = (grindKey, value) => {
    setExitAmounts(prev => ({
      ...prev,
      [grindKey]: value
    }));
  };

  const handleSetFullAmount = (grindKey, totalAmount) => {
    setExitAmounts(prev => ({
      ...prev,
      [grindKey]: totalAmount
    }));
  };

  const handleTakeProfit = async (grind) => {
    const exitQuantity = exitAmounts[grind.key];  // 用户输入的是数量
    const totalQuantity = grind.data?.total_amount || 0;
    
    if (!exitQuantity || exitQuantity <= 0) {
      message.warning('请输入退出数量');
      return;
    }
    if (exitQuantity > totalQuantity) {
      message.warning('退出数量不能超过持仓数量');
      return;
    }

    Modal.confirm({
      title: '确认退出',
      content: (
        <div>
          <p>确定要退出 <strong>{grind.label}</strong> 吗？</p>
          <p>退出数量: <strong>{formatNumber(exitQuantity, 6)}</strong></p>
        </div>
      ),
      okText: '确认退出',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: async () => {
        setLoading(prev => ({ ...prev, [grind.key]: true }));
        try {
          if (onTakeProfit) {
            // 发送数量给后端
            await onTakeProfit(position, grind.key, exitQuantity);
          }
          message.success(`${grind.label} 退出请求已提交`);
          // 清空该 grind 的退出数量
          setExitAmounts(prev => ({ ...prev, [grind.key]: undefined }));
        } catch (error) {
          message.error(`退出失败: ${error.message}`);
        } finally {
          setLoading(prev => ({ ...prev, [grind.key]: false }));
        }
      }
    });
  };

  return (
    <Drawer
      title={
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <span style={{ fontSize: '16px', fontWeight: 600 }}>Grind 详情</span>
          <span style={{ fontSize: '18px', fontWeight: 700, color: '#1890ff' }}>
            {position.symbol}
          </span>
          <span style={{
            background: position.side === 'LONG' ? '#f6ffed' : '#fff2f0',
            color: position.side === 'LONG' ? '#52c41a' : '#ff4d4f',
            padding: '4px 8px',
            borderRadius: '6px',
            fontSize: '12px',
            fontWeight: 600,
          }}>
            {position.side === 'LONG' ? '多头' : '空头'}
          </span>
        </div>
      }
      placement="right"
      onClose={onClose}
      open={visible}
      width={window.innerWidth < 768 ? '100%' : 420}
    >
      {activeGrinds.length === 0 ? (
        <Empty description="暂无 Grind 数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          {activeGrinds.map(grind => {
            const status = getStatusText(grind);
            const percentage = grind.data?.percentage || 0;
            const totalAmount = grind.data?.total_amount || 0;
            const currentExitAmount = exitAmounts[grind.key];
            
            return (
              <div 
                key={grind.key}
                style={{
                  background: '#fafafa',
                  borderRadius: 12,
                  padding: 16,
                  border: status.canExit ? `2px solid ${grind.color}` : '1px solid #f0f0f0',
                }}
              >
                {/* 头部 */}
                <div style={{ 
                  display: 'flex', 
                  justifyContent: 'space-between', 
                  alignItems: 'center',
                  marginBottom: 12,
                }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span style={{
                      fontSize: '14px',
                      fontWeight: 600,
                      color: grind.color,
                    }}>
                      {grind.label}
                    </span>
                    <span style={{
                      fontSize: '11px',
                      padding: '2px 8px',
                      borderRadius: '4px',
                      background: status.bg,
                      color: status.color,
                      fontWeight: 500,
                    }}>
                      {status.text}
                    </span>
                  </div>
                  <span style={{
                    fontSize: '12px',
                    color: '#8c8c8c',
                  }}>
                    {grind.data?.entry_count || 0} 笔入场
                  </span>
                </div>

                {/* 进度条 */}
                <Progress 
                  percent={percentage} 
                  strokeColor={grind.color}
                  trailColor="#e8e8e8"
                  size="small"
                  format={() => `${formatPercent(percentage)}%`}
                />

                {/* 详细数据 */}
                <div style={{ 
                  display: 'grid', 
                  gridTemplateColumns: '1fr 1fr', 
                  gap: 12,
                  marginTop: 12,
                }}>
                  <div>
                    <div style={{ fontSize: '11px', color: '#8c8c8c' }}>数量</div>
                    <div style={{ fontSize: '13px', fontWeight: 500 }}>
                      {formatNumber(totalAmount, 6)}
                    </div>
                  </div>
                  <div style={{ flex: 1 }}>
                    <div style={{ fontSize: '11px', color: '#8c8c8c' }}>保证金</div>
                    <div style={{ fontSize: '12px', fontWeight: 500 }}>
                      ${formatNumber(grind.data.stake_amount, 2)}
                    </div>
                  </div>
                  <div>
                    <div style={{ fontSize: '11px', color: '#8c8c8c' }}>均价</div>
                    <div style={{ fontSize: '13px', fontWeight: 500 }}>
                      ${formatNumber(grind.data?.open_rate, 4)}
                    </div>
                  </div>
                  <div>
                    <div style={{ fontSize: '11px', color: '#8c8c8c' }}>占比</div>
                    <div style={{ fontSize: '13px', fontWeight: 500, color: grind.color }}>
                      {formatPercent(percentage)}%
                    </div>
                  </div>
                </div>

                {/* 退出操作区 - 仅当有入场未退出时显示 */}
                {status.canExit && totalAmount > 0 && (
                  <div style={{
                    marginTop: 16,
                    padding: 12,
                    background: '#fff',
                    borderRadius: 8,
                    border: '1px dashed #d9d9d9',
                  }}>
                    <div style={{ 
                      display: 'flex', 
                      alignItems: 'center', 
                      gap: 8,
                    }}>
                      <span style={{ fontSize: '12px', color: '#666', whiteSpace: 'nowrap' }}>退出数量:</span>
                      <InputNumber
                        size="small"
                        min={0}
                        max={totalAmount}
                        step={totalAmount / 10}
                        value={currentExitAmount}
                        onChange={(value) => handleExitAmountChange(grind.key, value)}
                        style={{ flex: 1, minWidth: 80 }}
                        placeholder="数量"
                        precision={6}
                      />
                      <Button 
                        size="small"
                        type="text"
                        onClick={() => handleSetFullAmount(grind.key, totalAmount)}
                        style={{ 
                          padding: '2px 8px',
                          fontSize: '11px',
                          color: '#1890ff',
                          background: '#f0f9ff',
                          border: '1px solid #bae6fd',
                          borderRadius: '4px',
                        }}
                      >
                        全部
                      </Button>

                      <Button
                        size="small"
                        loading={loading[grind.key]}
                        disabled={!currentExitAmount || currentExitAmount <= 0}
                        onClick={() => handleTakeProfit(grind)}
                        style={{
                          padding: '2px 12px',
                          fontSize: '11px',
                          fontWeight: 500,
                          color: '#1890ff',
                          background: '#f0f9ff',
                          border: '1px solid #bae6fd',
                          borderRadius: '4px',
                        }}
                      >
                        止盈
                      </Button>
                    </div>
                  </div>
                )}
              </div>
            );
          })}

          {/* 汇总信息 */}
          <div style={{
            background: '#f0f5ff',
            borderRadius: 12,
            padding: 16,
            border: '1px solid #d6e4ff',
            marginTop: 8,
          }}>
            <div style={{ fontSize: '13px', fontWeight: 600, color: '#1890ff', marginBottom: 8 }}>
              总计
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
              <div>
                <div style={{ fontSize: '11px', color: '#8c8c8c' }}>总仓位</div>
                <div style={{ fontSize: '14px', fontWeight: 600 }}>
                  {formatNumber(position.size, 6)}
                </div>
              </div>
              <div>
                <div style={{ fontSize: '11px', color: '#8c8c8c' }}>保证金</div>
                <div style={{ fontSize: '14px', fontWeight: 600 }}>
                  ${formatNumber(position.notional, 2)}
                </div>
              </div>
            </div>
          </div>
        </div>
      )}
    </Drawer>
  );
};

export default GrindDetailDrawer;
