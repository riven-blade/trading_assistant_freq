import React, { useState, useEffect, useCallback } from 'react';
import { Table, Card, Tag, Space, Typography, Input, Button, Tooltip, Select, Grid } from 'antd';
import { SearchOutlined, ReloadOutlined } from '@ant-design/icons';
import api from '../services/api';
import moment from 'moment';
import { useSystemConfig } from '../hooks/useSystemConfig';
import KlineDrawer from '../components/Chart/KlineDrawer';

const { Title } = Typography;
const { Option } = Select;
const { useBreakpoint } = Grid;

const Analysis = () => {
  // 获取全局系统配置
  const { exchangeType, marketType, loading: configLoading } = useSystemConfig();
  
  // 检测屏幕断点
  const screens = useBreakpoint();
  const isMobile = !screens.md; // md 以下为移动端
  
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState([]);
  const [pagination, setPagination] = useState({
    current: 1,
    pageSize: 10,
    total: 0,
  });
  const [filters, setFilters] = useState({
    symbol: undefined,
    exchange: undefined,
    market_type: undefined,
  });

  // K线图抽屉状态
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [selectedRecord, setSelectedRecord] = useState(null);

  const { current, pageSize } = pagination;

  // 当系统配置加载完成后，初始化过滤器默认值
  useEffect(() => {
    if (!configLoading && exchangeType && marketType) {
      setFilters(prev => ({
        ...prev,
        exchange: prev.exchange || exchangeType,
        market_type: prev.market_type || marketType,
      }));
    }
  }, [configLoading, exchangeType, marketType]);

  const fetchAnalysis = useCallback(async (page = 1, pageSize = 10, currentFilters = filters) => {
    setLoading(true);
    try {
      const params = {
        page,
        pageSize,
        symbol: currentFilters.symbol,
        exchange: currentFilters.exchange,
        market_type: currentFilters.market_type,
      };
      
      const response = await api.get('/analysis/results', { params });
      const { data, total } = response.data;
      
      setData(data);
      setPagination(prev => ({
        ...prev,
        current: page,
        pageSize: pageSize,
        total: total,
      }));
    } catch (error) {
      console.error('Failed to fetch analysis results:', error);
    } finally {
      setLoading(false);
    }
  }, [filters]);

  useEffect(() => {
    // 只在 filters 有值时才调用接口
    if (filters.exchange && filters.market_type) {
      fetchAnalysis(current, pageSize);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fetchAnalysis, current, pageSize]);

  const handleTableChange = (newPagination) => {
    setPagination(prev => ({
      ...prev,
      current: newPagination.current,
      pageSize: newPagination.pageSize,
    }));
  };

  const handleSearch = (value) => {
    const newFilters = { ...filters, symbol: value };
    setFilters(newFilters);
    setPagination(prev => ({ ...prev, current: 1 }));
  };

  const handleExchangeChange = (value) => {
    const newFilters = { ...filters, exchange: value };
    setFilters(newFilters);
    setPagination(prev => ({ ...prev, current: 1 }));
  };

  const handleMarketTypeChange = (value) => {
    const newFilters = { ...filters, market_type: value };
    setFilters(newFilters);
    setPagination(prev => ({ ...prev, current: 1 }));
  };

  const handleRefresh = () => {
    fetchAnalysis(current, pageSize);
  };

  // 打开K线图抽屉
  const handleOpenChart = (record) => {
    setSelectedRecord(record);
    setDrawerVisible(true);
  };

  // 关闭K线图抽屉
  const handleCloseDrawer = () => {
    setDrawerVisible(false);
    setSelectedRecord(null);
  };

  // 根据屏幕大小动态生成表格列
  const getColumns = () => {
    const baseColumns = [
      {
        title: '交易对',
        dataIndex: 'symbol',
        key: 'symbol',
        width: isMobile ? 100 : 120,
        fixed: isMobile ? 'left' : false,
        render: (text) => <Tag color="blue">{text}</Tag>,
      },
    ];

    // 移动端只显示核心信息
    if (isMobile) {
      return [
        ...baseColumns,
        {
          title: '价格',
          dataIndex: 'last_price',
          key: 'last_price',
          width: 100,
          render: (price) => price?.toFixed(4),
        },
        {
          title: '操作',
          key: 'action',
          width: 80,
          fixed: 'right',
          render: (_, record) => (
            <Button 
              type="link" 
              size="small"
              onClick={() => handleOpenChart(record)}
            >
              详情
            </Button>
          ),
        },
      ];
    }

    // PC端显示完整信息
    return [
      {
        title: '交易所',
        dataIndex: 'exchange',
        key: 'exchange',
        width: 100,
        render: (text) => (
          <Tag color={text === 'binance' ? 'gold' : 'cyan'}>
            {text === 'binance' ? 'Binance' : text === 'bybit' ? 'Bybit' : text}
          </Tag>
        ),
      },
      ...baseColumns,
      {
        title: '市场',
        dataIndex: 'market_type',
        key: 'market_type',
        width: 80,
        render: (text) => (
          <Tag color={text === 'future' ? 'purple' : 'green'}>
            {text === 'future' ? '期货' : '现货'}
          </Tag>
        ),
      },
      {
        title: '周期',
        dataIndex: 'timeframe',
        key: 'timeframe',
        width: 70,
      },
      {
        title: '最新价格',
        dataIndex: 'last_price',
        key: 'last_price',
        width: 120,
        render: (price) => price?.toFixed(4),
      },
      {
        title: '支撑位',
        dataIndex: 'support_levels',
        key: 'support_levels',
        render: (levels) => (
          <Space wrap>
            {levels && JSON.parse(JSON.stringify(levels)).map((level, index) => (
              <Tag key={index} color="success">{Number(level).toFixed(4)}</Tag>
            ))}
          </Space>
        ),
      },
      {
        title: '阻力位',
        dataIndex: 'resistance_levels',
        key: 'resistance_levels',
        render: (levels) => (
          <Space wrap>
            {levels && JSON.parse(JSON.stringify(levels)).map((level, index) => (
              <Tag key={index} color="error">{Number(level).toFixed(4)}</Tag>
            ))}
          </Space>
        ),
      },
      {
        title: '分析时间',
        dataIndex: 'updated_at',
        key: 'updated_at',
        width: 160,
        render: (text) => moment(text).format('YYYY-MM-DD HH:mm:ss'),
      },
      {
        title: '操作',
        key: 'action',
        width: 100,
        render: (_, record) => (
          <Button 
            type="link" 
            size="small"
            onClick={() => handleOpenChart(record)}
          >
            查看详情
          </Button>
        ),
      },
    ];
  };

  return (
    <div style={{ padding: isMobile ? 12 : 24 }}>
      <Card 
        bordered={false} 
        className="shadow-sm"
        title={<Title level={isMobile ? 5 : 4} style={{ margin: 0 }}>数据分析结果</Title>}
        extra={
          isMobile ? (
            <Tooltip title="刷新数据">
              <Button 
                icon={<ReloadOutlined />} 
                onClick={handleRefresh} 
                loading={loading}
                size="small"
              />
            </Tooltip>
          ) : (
            <Space wrap>
              <Select
                placeholder="选择交易所"
                allowClear
                style={{ width: 150 }}
                onChange={handleExchangeChange}
                value={filters.exchange}
              >
                <Option value="binance">Binance</Option>
                <Option value="bybit">Bybit</Option>
              </Select>
              <Select
                placeholder="选择市场类型"
                allowClear
                style={{ width: 150 }}
                onChange={handleMarketTypeChange}
                value={filters.market_type}
              >
                <Option value="spot">现货</Option>
                <Option value="future">期货</Option>
              </Select>
              <Input.Search
                placeholder="搜索交易对"
                allowClear
                onSearch={handleSearch}
                style={{ width: 200 }}
                enterButton={<Button icon={<SearchOutlined />} />}
              />
              <Tooltip title="刷新数据">
                <Button 
                  icon={<ReloadOutlined />} 
                  onClick={handleRefresh} 
                  loading={loading}
                />
              </Tooltip>
            </Space>
          )
        }
      >
        {/* 移动端筛选器 */}
        {isMobile && (
          <Space direction="vertical" style={{ width: '100%', marginBottom: 16 }} size="middle">
            <Space wrap style={{ width: '100%' }}>
              <Select
                placeholder="交易所"
                allowClear
                style={{ flex: 1, minWidth: 120 }}
                onChange={handleExchangeChange}
                value={filters.exchange}
                size="small"
              >
                <Option value="binance">Binance</Option>
                <Option value="bybit">Bybit</Option>
              </Select>
              <Select
                placeholder="市场类型"
                allowClear
                style={{ flex: 1, minWidth: 120 }}
                onChange={handleMarketTypeChange}
                value={filters.market_type}
                size="small"
              >
                <Option value="spot">现货</Option>
                <Option value="future">期货</Option>
              </Select>
            </Space>
            <Input.Search
              placeholder="搜索交易对"
              allowClear
              onSearch={handleSearch}
              enterButton={<Button icon={<SearchOutlined />} size="small" />}
              size="small"
            />
          </Space>
        )}
        
        <Table
          columns={getColumns()}
          dataSource={data}
          rowKey="id"
          pagination={{
            ...pagination,
            showSizeChanger: !isMobile,
            showTotal: (total) => `共 ${total} 条`,
            simple: isMobile,
            pageSize: isMobile ? 20 : pagination.pageSize,
          }}
          loading={loading}
          onChange={handleTableChange}
          scroll={{ x: isMobile ? 400 : 1000 }}
          size={isMobile ? 'small' : 'middle'}
        />
      </Card>

      {/* K线图抽屉 */}
      <KlineDrawer
        visible={drawerVisible}
        onClose={handleCloseDrawer}
        symbol={selectedRecord?.symbol}
        analysisData={selectedRecord}
        defaultInterval="1h"
      />
    </div>
  );
};

export default Analysis;
