import { useState, useCallback, useEffect } from 'react';
import api from '../services/api';

/**
 * 持仓数据管理Hook
 * 通过HTTP API获取持仓数据
 */
const useAccountData = () => {
  
  // 持仓相关状态
  const [positions, setPositions] = useState([]);
  const [totalPnl, setTotalPnl] = useState(0);
  const [positionCount, setPositionCount] = useState(0);
  const [positionsLoading, setPositionsLoading] = useState(true);
  
  // 通用状态
  const [lastUpdate, setLastUpdate] = useState(null);
  const [error, setError] = useState(null);

  // 从API获取持仓数据
  const fetchPositions = useCallback(async () => {
    try {
      setPositionsLoading(true);
      const response = await api.get('/positions');
      
      if (response.data.success && response.data.data) {
        const positionsData = response.data.data.positions || [];
        const totalPnl = response.data.data.total_pnl || 0;
        
        // 转换freqtrade数据格式为前端需要的格式
        const formattedPositions = positionsData.map(position => {
          // 计算真实的未实现盈亏（包含资金费用）
          const calculateUnrealizedPnl = () => {
            const openRate = Number(position.open_rate) || 0;
            const currentRate = Number(position.current_rate) || 0;
            const amount = Number(position.amount) || 0;
            const fundingFees = Number(position.funding_fees) || 0;
            
            if (openRate === 0 || currentRate === 0 || amount === 0) return 0;
            
            // 多头：(当前价 - 开仓价) × 数量
            // 空头：(开仓价 - 当前价) × 数量
            const isLong = !position.is_short;
            const priceDiff = isLong 
              ? (currentRate - openRate)
              : (openRate - currentRate);
            
            const priceProfit = priceDiff * amount;
            
            // 加入资金费用（正值表示支付，负值表示收入）
            return priceProfit + fundingFees;
          };

          const formatSymbol = (pair) => {
            if (!pair) return '';
            // 移除 :USDT 后缀，然后将 / 替换为空字符串
            return pair.replace(/:\w+$/, '').replace('/', '');
          };

          return {
            symbol: formatSymbol(position.pair),
            side: position.is_short ? 'SHORT' : 'LONG', 
            size: position.amount,
            entry_price: position.open_rate,
            mark_price: position.current_rate,
            unrealized_pnl: calculateUnrealizedPnl(), // 使用计算的盈亏
            leverage: position.leverage || 1,
            margin_mode: 'ISOLATED',
            notional: position.stake_amount,
            updated_at: new Date(position.open_timestamp),
            // 添加一些额外的有用信息
            trade_id: position.trade_id,
            liquidation_price: position.liquidation_price,
            funding_fees: position.funding_fees,
            realized_profit: position.realized_profit,
            original_pair: position.pair, // 保留原始格式以备后用
            orders: position.orders || [] // 包含订单数据用于加仓计算
          };
        });
        
        // 重新计算总盈亏（基于我们计算的真实盈亏）
        const recalculatedTotalPnl = formattedPositions.reduce((sum, pos) => sum + (pos.unrealized_pnl || 0), 0);
        
        setPositions(formattedPositions);
        setPositionCount(formattedPositions.length);
        setTotalPnl(recalculatedTotalPnl);
        setLastUpdate(new Date());
        setError(null);
      }
    } catch (err) {
      console.error('获取持仓数据失败:', err);
      setError(err.message || '获取持仓数据失败');
    } finally {
      setPositionsLoading(false);
    }
  }, []);

  
  // 获取指定仓位
  const getPositionBySymbol = useCallback((symbol) => {
    return positions.find(pos => pos.symbol === symbol) || null;
  }, [positions]);
  
  // 检查是否有仓位
  const hasPosition = useCallback((symbol, side) => {
    const position = positions.find(pos => 
      pos.symbol === symbol && 
      pos.side.toLowerCase() === side.toLowerCase() && 
      Math.abs(pos.size) > 0
    );
    return !!position;
  }, [positions]);

  // 检查是否有任何方向的仓位
  const hasAnyPosition = useCallback((symbol) => {
    const symbolPositions = positions.filter(pos => 
      pos.symbol === symbol && Math.abs(pos.size) > 0
    );
    return symbolPositions.length > 0;
  }, [positions]);
  
  // 组件初始化时自动获取持仓数据
  useEffect(() => {
    fetchPositions();
  }, [fetchPositions]);

  
  
  return {
    // 持仓数据
    positions,           // 持仓列表
    totalPnl,            // 总盈亏
    positionCount,       // 持仓数量
    
    // 通用状态
    positionsLoading,    // 持仓数据加载状态
    lastUpdate,          // 最后更新时间
    error,               // 错误信息
    
    // 方法
    refreshPositions: fetchPositions, // 手动刷新持仓数据
    
    // 持仓查询方法
    getPositionBySymbol, // 获取指定仓位
    hasPosition,         // 检查是否有仓位
    hasAnyPosition,      // 检查是否有任何方向的仓位
  };
};

export default useAccountData;