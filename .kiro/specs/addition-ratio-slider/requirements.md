# Requirements Document

## Introduction

本功能用于调整加仓比例滑块的范围和默认值。当前加仓比例滑块的最大值为100%，默认值为50%。根据业务需求，需要将最大值调整为30%，默认值调整为15%，以更好地控制加仓风险。

## Glossary

- **Addition_Ratio_Slider**: 加仓比例滑块组件，用于在持仓页面设置加仓时的仓位比例
- **QuantitySlider**: 通用数量滑块组件，根据操作类型（加仓/止盈）显示不同的范围和刻度
- **Positions_Page**: 持仓管理页面，包含加仓操作抽屉

## Requirements

### Requirement 1: 加仓比例滑块最大值调整

**User Story:** As a trader, I want the addition ratio slider to have a maximum value of 30%, so that I can better control my position risk and avoid over-leveraging.

#### Acceptance Criteria

1. WHEN a user opens the addition drawer for a position, THE Addition_Ratio_Slider SHALL display a maximum value of 30%
2. WHEN the addition ratio slider is rendered, THE Addition_Ratio_Slider SHALL show marks at 0%, 10%, 20%, and 30%
3. WHEN a user attempts to set a ratio above 30%, THE Addition_Ratio_Slider SHALL prevent the value from exceeding 30%

### Requirement 2: 加仓比例滑块默认值调整

**User Story:** As a trader, I want the addition ratio slider to default to 15%, so that I start with a conservative position size that aligns with my risk management strategy.

#### Acceptance Criteria

1. WHEN a user opens the addition drawer for a position, THE Addition_Ratio_Slider SHALL initialize with a default value of 15%
2. WHEN the drawer is closed and reopened, THE Addition_Ratio_Slider SHALL reset to the default value of 15%

### Requirement 3: 止盈比例滑块保持不变

**User Story:** As a trader, I want the take profit ratio slider to remain unchanged at 100% maximum, so that I can still close my entire position when taking profits.

#### Acceptance Criteria

1. WHEN a user opens the take profit drawer for a position, THE QuantitySlider SHALL display a maximum value of 100%
2. WHEN the take profit ratio slider is rendered, THE QuantitySlider SHALL show marks at 0%, 25%, 50%, 75%, and 100%
