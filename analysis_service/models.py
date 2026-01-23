from sqlalchemy import Column, Integer, String, Float, DateTime, JSON, UniqueConstraint
from sqlalchemy.sql import func
from database import Base

class AnalysisResult(Base):
    __tablename__ = "analysis_results"
    
    # 添加唯一约束：一个交易所+币种+市场类型+时间周期只保留一条记录
    __table_args__ = (
        UniqueConstraint('exchange', 'symbol', 'market_type', 'timeframe', name='uix_analysis_unique'),
    )

    id = Column(Integer, primary_key=True, index=True)
    exchange = Column(String(50), index=True, nullable=False)
    symbol = Column(String(50), index=True, nullable=False)
    market_type = Column(String(20), nullable=False)  # spot, future
    timeframe = Column(String(10), nullable=False)
    input_limit = Column(Integer)
    support_levels = Column(JSON)
    resistance_levels = Column(JSON)
    last_price = Column(Float)
    created_at = Column(DateTime(timezone=True), server_default=func.now())
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())

