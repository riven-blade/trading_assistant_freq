import { useState, useEffect, useCallback, createContext, useContext } from 'react';
import { getSystemConfig } from '../services/api';

// 创建 Context
const SystemConfigContext = createContext(null);

// 默认配置
const DEFAULT_CONFIG = {
  market_type: 'future',
  exchange_type: 'binance'
};

/**
 * 系统配置 Provider
 */
export const SystemConfigProvider = ({ children }) => {
  const [config, setConfig] = useState(DEFAULT_CONFIG);
  const [loading, setLoading] = useState(true);

  // 获取配置
  const fetchConfig = useCallback(async () => {
    try {
      console.log('[SystemConfig] 开始获取系统配置...');
      const data = await getSystemConfig();
      console.log('[SystemConfig] 获取成功:', data);
      setConfig(data);
    } catch (error) {
      console.error('[SystemConfig] 获取系统配置失败:', error);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchConfig();
  }, [fetchConfig]);

  // 判断是否为现货模式
  const isSpotMode = config.market_type === 'spot';
  
  // 判断是否为期货模式
  const isFutureMode = config.market_type === 'future';

  const value = {
    config,
    loading,
    isSpotMode,
    isFutureMode,
    marketType: config.market_type,
    exchangeType: config.exchange_type,
    refetch: fetchConfig
  };

  return (
    <SystemConfigContext.Provider value={value}>
      {children}
    </SystemConfigContext.Provider>
  );
};

/**
 * 使用系统配置的 Hook
 */
export const useSystemConfig = () => {
  const context = useContext(SystemConfigContext);
  if (!context) {
    throw new Error('useSystemConfig must be used within a SystemConfigProvider');
  }
  return context;
};

export default useSystemConfig;

