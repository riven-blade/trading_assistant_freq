#!/usr/bin/env python3
"""
SQLite to MySQL 迁移脚本
将 Freqtrade 的 tradesv3.sqlite 数据迁移到 MySQL
"""

import sqlite3
import pymysql
from datetime import datetime

# MySQL 连接配置f
MYSQL_CONFIG = {
    'host': '168.93.214.185',
    'port': 3308,
    'user': 'root',
    'password': 'dou.190824',
    'database': 'binance',
    'charset': 'utf8mb4'
}

# SQLite 文件路径
SQLITE_PATH = 'tradesv3.sqlite'


def create_mysql_tables(cursor):
    """创建 MySQL 表结构"""
    
    # KeyValueStore 表
    cursor.execute("""
        CREATE TABLE IF NOT EXISTS KeyValueStore (
            id INT AUTO_INCREMENT PRIMARY KEY,
            `key` VARCHAR(25) NOT NULL,
            value_type VARCHAR(20) NOT NULL,
            string_value VARCHAR(255),
            datetime_value DATETIME,
            float_value FLOAT,
            int_value INT,
            INDEX idx_key (`key`)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
    """)
    
    # trades 表
    cursor.execute("""
        CREATE TABLE IF NOT EXISTS trades (
            id INT AUTO_INCREMENT PRIMARY KEY,
            exchange VARCHAR(25) NOT NULL,
            pair VARCHAR(25) NOT NULL,
            base_currency VARCHAR(25),
            stake_currency VARCHAR(25),
            is_open BOOLEAN NOT NULL DEFAULT FALSE,
            fee_open FLOAT NOT NULL DEFAULT 0,
            fee_open_cost FLOAT,
            fee_open_currency VARCHAR(25),
            fee_close FLOAT NOT NULL DEFAULT 0,
            fee_close_cost FLOAT,
            fee_close_currency VARCHAR(25),
            open_rate FLOAT NOT NULL,
            open_rate_requested FLOAT,
            open_trade_value FLOAT,
            close_rate FLOAT,
            close_rate_requested FLOAT,
            realized_profit FLOAT,
            close_profit FLOAT,
            close_profit_abs FLOAT,
            stake_amount FLOAT NOT NULL,
            max_stake_amount FLOAT,
            amount FLOAT NOT NULL,
            amount_requested FLOAT,
            open_date DATETIME NOT NULL,
            close_date DATETIME,
            stop_loss FLOAT,
            stop_loss_pct FLOAT,
            initial_stop_loss FLOAT,
            initial_stop_loss_pct FLOAT,
            is_stop_loss_trailing BOOLEAN NOT NULL DEFAULT FALSE,
            max_rate FLOAT,
            min_rate FLOAT,
            exit_reason VARCHAR(255),
            exit_order_status VARCHAR(100),
            strategy VARCHAR(100),
            enter_tag VARCHAR(255),
            timeframe INT,
            trading_mode VARCHAR(7),
            amount_precision FLOAT,
            price_precision FLOAT,
            precision_mode INT,
            precision_mode_price INT,
            contract_size FLOAT,
            leverage FLOAT,
            is_short BOOLEAN NOT NULL DEFAULT FALSE,
            liquidation_price FLOAT,
            interest_rate FLOAT NOT NULL DEFAULT 0,
            funding_fees FLOAT,
            funding_fee_running FLOAT,
            record_version INT NOT NULL DEFAULT 0,
            INDEX idx_is_open (is_open),
            INDEX idx_pair (pair)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
    """)
    
    # pairlocks 表
    cursor.execute("""
        CREATE TABLE IF NOT EXISTS pairlocks (
            id INT AUTO_INCREMENT PRIMARY KEY,
            pair VARCHAR(25) NOT NULL,
            side VARCHAR(25) NOT NULL,
            reason VARCHAR(255),
            lock_time DATETIME NOT NULL,
            lock_end_time DATETIME NOT NULL,
            active BOOLEAN NOT NULL DEFAULT FALSE,
            INDEX idx_pair (pair),
            INDEX idx_active (active),
            INDEX idx_lock_end_time (lock_end_time)
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
    """)
    
    # trade_custom_data 表
    cursor.execute("""
        CREATE TABLE IF NOT EXISTS trade_custom_data (
            id INT AUTO_INCREMENT PRIMARY KEY,
            ft_trade_id INT,
            cd_key VARCHAR(255) NOT NULL,
            cd_type VARCHAR(25) NOT NULL,
            cd_value TEXT NOT NULL,
            created_at DATETIME NOT NULL,
            updated_at DATETIME,
            UNIQUE KEY uk_trade_id_cd_key (ft_trade_id, cd_key),
            INDEX idx_ft_trade_id (ft_trade_id),
            FOREIGN KEY (ft_trade_id) REFERENCES trades(id) ON DELETE CASCADE
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
    """)
    
    # orders 表
    cursor.execute("""
        CREATE TABLE IF NOT EXISTS orders (
            id INT AUTO_INCREMENT PRIMARY KEY,
            ft_trade_id INT NOT NULL,
            ft_order_side VARCHAR(25) NOT NULL,
            ft_pair VARCHAR(25) NOT NULL,
            ft_is_open BOOLEAN NOT NULL DEFAULT FALSE,
            ft_amount FLOAT NOT NULL,
            ft_price FLOAT NOT NULL,
            ft_cancel_reason VARCHAR(255),
            order_id VARCHAR(255) NOT NULL,
            status VARCHAR(255),
            symbol VARCHAR(25),
            order_type VARCHAR(50),
            side VARCHAR(25),
            price FLOAT,
            average FLOAT,
            amount FLOAT,
            filled FLOAT,
            remaining FLOAT,
            cost FLOAT,
            stop_price FLOAT,
            order_date DATETIME,
            order_filled_date DATETIME,
            order_update_date DATETIME,
            funding_fee FLOAT,
            ft_fee_base FLOAT,
            ft_order_tag VARCHAR(255),
            UNIQUE KEY uk_pair_order_id (ft_pair, order_id),
            INDEX idx_order_id (order_id),
            INDEX idx_ft_trade_id (ft_trade_id),
            INDEX idx_ft_is_open (ft_is_open),
            FOREIGN KEY (ft_trade_id) REFERENCES trades(id) ON DELETE CASCADE
        ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4
    """)
    
    print("✅ MySQL 表结构创建完成")


def migrate_table(sqlite_cursor, mysql_cursor, table_name, columns):
    """迁移单个表的数据"""
    sqlite_cursor.execute(f"SELECT * FROM {table_name}")
    rows = sqlite_cursor.fetchall()
    
    if not rows:
        print(f"⚠️  表 {table_name} 没有数据")
        return 0
    
    placeholders = ', '.join(['%s'] * len(columns))
    column_names = ', '.join([f'`{col}`' for col in columns])
    
    # 使用 REPLACE INTO 避免重复键冲突
    insert_sql = f"REPLACE INTO {table_name} ({column_names}) VALUES ({placeholders})"
    
    count = 0
    for row in rows:
        try:
            mysql_cursor.execute(insert_sql, row)
            count += 1
        except Exception as e:
            print(f"⚠️  插入 {table_name} 行失败: {e}")
    
    print(f"✅ 表 {table_name}: 迁移了 {count}/{len(rows)} 条记录")
    return count


def main():
    print("=" * 50)
    print("SQLite to MySQL 迁移工具")
    print("=" * 50)
    
    # 连接 SQLite
    print("\n📂 连接 SQLite 数据库...")
    sqlite_conn = sqlite3.connect(SQLITE_PATH)
    sqlite_cursor = sqlite_conn.cursor()
    
    # 连接 MySQL
    print("🔌 连接 MySQL 数据库...")
    mysql_conn = pymysql.connect(**MYSQL_CONFIG)
    mysql_cursor = mysql_conn.cursor()
    
    try:
        # 创建表结构
        print("\n📝 创建 MySQL 表结构...")
        create_mysql_tables(mysql_cursor)
        mysql_conn.commit()
        
        # 迁移数据
        print("\n📦 开始迁移数据...")
        
        # 定义各表的列（按 SQLite schema 顺序）
        tables = {
            'KeyValueStore': ['id', 'key', 'value_type', 'string_value', 'datetime_value', 'float_value', 'int_value'],
            'trades': ['id', 'exchange', 'pair', 'base_currency', 'stake_currency', 'is_open', 
                      'fee_open', 'fee_open_cost', 'fee_open_currency', 'fee_close', 'fee_close_cost', 
                      'fee_close_currency', 'open_rate', 'open_rate_requested', 'open_trade_value',
                      'close_rate', 'close_rate_requested', 'realized_profit', 'close_profit', 
                      'close_profit_abs', 'stake_amount', 'max_stake_amount', 'amount', 'amount_requested',
                      'open_date', 'close_date', 'stop_loss', 'stop_loss_pct', 'initial_stop_loss',
                      'initial_stop_loss_pct', 'is_stop_loss_trailing', 'max_rate', 'min_rate',
                      'exit_reason', 'exit_order_status', 'strategy', 'enter_tag', 'timeframe',
                      'trading_mode', 'amount_precision', 'price_precision', 'precision_mode',
                      'precision_mode_price', 'contract_size', 'leverage', 'is_short', 'liquidation_price',
                      'interest_rate', 'funding_fees', 'funding_fee_running', 'record_version'],
            'pairlocks': ['id', 'pair', 'side', 'reason', 'lock_time', 'lock_end_time', 'active'],
            'trade_custom_data': ['id', 'ft_trade_id', 'cd_key', 'cd_type', 'cd_value', 'created_at', 'updated_at'],
            'orders': ['id', 'ft_trade_id', 'ft_order_side', 'ft_pair', 'ft_is_open', 'ft_amount',
                      'ft_price', 'ft_cancel_reason', 'order_id', 'status', 'symbol', 'order_type',
                      'side', 'price', 'average', 'amount', 'filled', 'remaining', 'cost', 'stop_price',
                      'order_date', 'order_filled_date', 'order_update_date', 'funding_fee', 
                      'ft_fee_base', 'ft_order_tag']
        }
        
        total = 0
        # 按顺序迁移（先迁移没有外键依赖的表）
        for table_name in ['KeyValueStore', 'trades', 'pairlocks', 'trade_custom_data', 'orders']:
            total += migrate_table(sqlite_cursor, mysql_cursor, table_name, tables[table_name])
        
        mysql_conn.commit()
        
        print("\n" + "=" * 50)
        print(f"🎉 迁移完成！共迁移 {total} 条记录")
        print("=" * 50)
        
    except Exception as e:
        print(f"\n❌ 迁移失败: {e}")
        mysql_conn.rollback()
        raise
    finally:
        sqlite_cursor.close()
        sqlite_conn.close()
        mysql_cursor.close()
        mysql_conn.close()


if __name__ == '__main__':
    main()
