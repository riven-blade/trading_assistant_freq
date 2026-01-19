import logging
import numpy as np
import pandas as pd
import pandas_ta as pta
from freqtrade.strategy.interface import IStrategy
from freqtrade.strategy import merge_informative_pair
from pandas import DataFrame, Series
from functools import reduce
from freqtrade.persistence import Trade, Order
from datetime import datetime, timedelta
import time
from typing import Optional
import warnings

log = logging.getLogger(__name__)
# log.setLevel(logging.DEBUG)
warnings.simplefilter(action="ignore", category=pd.errors.PerformanceWarning)


class NostalgiaForInfinityX6(IStrategy):
    INTERFACE_VERSION = 3

    def version(self) -> str:
        return "v1.0.168"

    stoploss = -0.99

    # Trailing stoploss
    trailing_stop = False
    trailing_only_offset_is_reached = True
    trailing_stop_positive = 0.01
    trailing_stop_positive_offset = 0.03

    use_custom_stoploss = False

    # Optimal timeframe for the strategy.
    timeframe = "5m"
    info_timeframes = ["15m", "1h", "4h", "1d"]

    # Run "populate_indicators()" only for new candle.
    process_only_new_candles = True

    # These values can be overridden in the "ask_strategy" section in the config.
    use_exit_signal = True
    exit_profit_only = False
    ignore_roi_if_entry_signal = True

    # Number of candles the strategy requires before producing valid signals
    startup_candle_count: int = 800

    # Long grind mode name
    long_grind_mode_name = "long_grind"

    # Shorting
    short_grind_mode_name = "short_grind"

    is_futures_mode = False
    futures_mode_leverage = 3.0

    # user specified fees to be used for profit calculations
    custom_fee_open_rate = None
    custom_fee_close_rate = None

    # Position adjust feature
    position_adjustment_enable = True

    # Grinding v2
    grind_v2_stake_multiplier_spot = 5
    grind_v2_stake_multiplier_futures = 5

    grind_v2_stake_multiplier_first = 0.15

    grind_v2_profit_exit_threshold = 0.25

    grinding_v2_max_stake = 1.0

    grinding_v2_grind_1_enable = True
    grinding_v2_grind_1_stakes_spot = [0.05, 0.10]
    grinding_v2_grind_1_thresholds_spot = [-0.06, -0.12]
    grinding_v2_grind_1_stakes_futures = [0.05, 0.10]
    grinding_v2_grind_1_thresholds_futures = [-0.06, -0.12]
    grinding_v2_grind_1_profit_threshold_spot = 0.05
    grinding_v2_grind_1_profit_threshold_futures = 0.05
    grinding_v2_grind_1_use_derisk = True
    grinding_v2_grind_1_derisk_spot = -0.20
    grinding_v2_grind_1_derisk_futures = -0.20

    grinding_v2_grind_2_enable = True
    grinding_v2_grind_2_stakes_spot = [0.05, 0.10]
    grinding_v2_grind_2_thresholds_spot = [-0.06, -0.20]
    grinding_v2_grind_2_stakes_futures = [0.05, 0.10]
    grinding_v2_grind_2_thresholds_futures = [-0.06, -0.20]
    grinding_v2_grind_2_profit_threshold_spot = 0.10
    grinding_v2_grind_2_profit_threshold_futures = 0.10
    grinding_v2_grind_2_use_derisk = True
    grinding_v2_grind_2_derisk_spot = -0.28
    grinding_v2_grind_2_derisk_futures = -0.28

    grinding_v2_grind_3_enable = True
    grinding_v2_grind_3_stakes_spot = [0.10, 0.20]
    grinding_v2_grind_3_thresholds_spot = [-0.06, -0.24]
    grinding_v2_grind_3_stakes_futures = [0.10, 0.20]
    grinding_v2_grind_3_thresholds_futures = [-0.06, -0.24]
    grinding_v2_grind_3_profit_threshold_spot = 0.15
    grinding_v2_grind_3_profit_threshold_futures = 0.15
    grinding_v2_grind_3_use_derisk = True
    grinding_v2_grind_3_derisk_spot = -0.32
    grinding_v2_grind_3_derisk_futures = -0.32

    grinding_v2_grind_x_enable = True
    grinding_v2_grind_x_stakes_spot = [0.15, 0.30]
    grinding_v2_grind_x_thresholds_spot = [-0.06, -0.24]
    grinding_v2_grind_x_stakes_futures = [0.15, 0.30]
    grinding_v2_grind_x_thresholds_futures = [-0.06, -0.24]
    grinding_v2_grind_x_profit_threshold_spot = 0.20
    grinding_v2_grind_x_profit_threshold_futures = 0.20
    grinding_v2_grind_x_use_derisk = True
    grinding_v2_grind_x_derisk_spot = -0.32
    grinding_v2_grind_x_derisk_futures = -0.32

    def __init__(self, config: dict) -> None:
        super().__init__(config)
        self.is_futures_mode = self.config.get("trading_mode", "spot") == "futures"
        if self.is_futures_mode:
            self.futures_mode_leverage = self.config.get("leverage", 3.0)
            self.can_short = True
        self.exchange_name = self.config.get("exchange", {}).get("name", "").lower()
        self.stake_currency = self.config.get("stake_currency", "USDT")
        self.exit_price_side = self.config.get("exit_pricing", {}).get("price_side", "same")

    # Plot configuration for FreqUI
    # ---------------------------------------------------------------------------------------------
    @property
    def plot_config(self):
        plot_config = {
            "main_plot": {
                "EMA_12": {"color": "LightGreen"},
                "EMA_26": {"color": "Yellow"},
            }
        }

        return plot_config

    # Calc Total Profit
    # ---------------------------------------------------------------------------------------------
    def calc_total_profit(
            self, trade: "Trade", filled_entries: "Orders", filled_exits: "Orders", exit_rate: float
    ) -> tuple:
        """
        Calculates the absolute profit for open trades.

        :param trade: trade object.
        :param filled_entries: Filled entries list.
        :param filled_exits: Filled exits list.
        :param exit_rate: The exit rate.
        :return tuple: The total profit in stake, ratio, ratio based on current stake, and ratio based on the first entry stake.
        """
        fee_open_rate = trade.fee_open if self.custom_fee_open_rate is None else self.custom_fee_open_rate
        fee_close_rate = trade.fee_close if self.custom_fee_close_rate is None else self.custom_fee_close_rate

        total_amount = 0.0
        total_stake = 0.0
        total_profit = 0.0
        current_stake = 0.0
        for entry_order in filled_entries:
            if trade.is_short:
                entry_stake = entry_order.safe_filled * \
                              entry_order.safe_price * (1 - fee_open_rate)
                total_amount += entry_order.safe_filled
                total_stake += entry_stake
                total_profit += entry_stake
            else:
                entry_stake = entry_order.safe_filled * \
                              entry_order.safe_price * (1 + fee_open_rate)
                total_amount += entry_order.safe_filled
                total_stake += entry_stake
                total_profit -= entry_stake
        for exit_order in filled_exits:
            if trade.is_short:
                exit_stake = exit_order.safe_filled * \
                             exit_order.safe_price * (1 + fee_close_rate)
                total_amount -= exit_order.safe_filled
                total_profit -= exit_stake
            else:
                exit_stake = exit_order.safe_filled * \
                             exit_order.safe_price * (1 - fee_close_rate)
                total_amount -= exit_order.safe_filled
                total_profit += exit_stake
        if trade.is_short:
            current_stake = total_amount * exit_rate * (1 + fee_close_rate)
            total_profit -= current_stake
        else:
            current_stake = total_amount * exit_rate * (1 - fee_close_rate)
            total_profit += current_stake
        if self.is_futures_mode:
            total_profit += trade.funding_fees

        total_profit_ratio = total_profit / total_stake
        current_profit_ratio = total_profit / current_stake
        stake_cost = trade.get_custom_data(key="stake_cost")
        init_profit_ratio = total_profit / (stake_cost if stake_cost else filled_entries[0].cost)
        return total_profit, total_profit_ratio, current_profit_ratio, init_profit_ratio

    # Custom Should Exit
    # ---------------------------------------------------------------------------------------------
    def should_exit(
            self,
            trade: Trade,
            rate: float,
            current_time: datetime,
            *,
            enter: bool,
            exit_: bool,
            low: float | None = None,
            high: float | None = None,
            force_stoploss: float = 0,
    ) -> list:
        exits = super().should_exit(
            trade,
            rate,
            current_time,
            enter=enter,
            exit_=exit_,
            low=low,
            high=high,
            force_stoploss=force_stoploss,
        )
        if not exits:
            return []
        if all(exit_.exit_reason == "liquidation" for exit_ in exits):
            return []
        exits_without_liquidation = [exit_ for exit_ in exits if exit_.exit_reason != "liquidation"]
        return exits_without_liquidation

    # Custom Exit
    # ---------------------------------------------------------------------------------------------
    def custom_exit(
            self, pair: str, trade: "Trade", current_time: "datetime", current_rate: float, current_profit: float,
            **kwargs
    ):
        df, _ = self.dp.get_analyzed_dataframe(pair, self.timeframe)
        if len(df) < 2:
            return None
        last_candle = df.iloc[-1].squeeze()
        previous_candle_1 = df.iloc[-2].squeeze()

        enter_tag = "empty"
        if hasattr(trade, "enter_tag") and trade.enter_tag is not None:
            enter_tag = trade.enter_tag

        filled_entries = trade.select_filled_orders(trade.entry_side)
        filled_exits = trade.select_filled_orders(trade.exit_side)

        profit_stake, profit_ratio, profit_current_stake_ratio, profit_init_ratio = self.calc_total_profit(
            trade, filled_entries, filled_exits, current_rate
        )

        if not trade.is_short:
            sell, signal_name = self.long_exit_grind(
                profit_init_ratio,
                last_candle,
                previous_candle_1,
            )
            if sell and (signal_name is not None):
                return f"{signal_name} ( {enter_tag})"
        else:
            # Short trades
            sell, signal_name = self.short_exit_grind(
                profit_init_ratio,
                last_candle,
                previous_candle_1,
            )
            if sell and (signal_name is not None):
                return f"{signal_name} ( {enter_tag})"

        return None

    # Custom Stake Amount
    # ---------------------------------------------------------------------------------------------
    def custom_stake_amount(
            self,
            pair: str,
            current_time: datetime,
            current_rate: float,
            proposed_stake: float,
            min_stake: Optional[float],
            max_stake: float,
            leverage: float,
            entry_tag: Optional[str],
            side: str,
            **kwargs,
    ) -> float:
        stake_multiplier = (
            self.grind_v2_stake_multiplier_futures if self.is_futures_mode else self.grind_v2_stake_multiplier_spot
        )
        stake = proposed_stake * stake_multiplier * self.grind_v2_stake_multiplier_first
        if stake > min_stake:
            return stake

        return min_stake

    # Order filled Callback
    # ---------------------------------------------------------------------------------------------
    def order_filled(self, pair: str, trade: Trade, order: Order, current_time: datetime, **kwargs) -> None:
        # Check if it's the first entry and stake_cost doesn't exist
        if trade.nr_of_successful_entries == 1 and trade.nr_of_successful_exits == 0:
            if trade.get_custom_data(key="stake_cost") is None:
                trade.set_custom_data(key="stake_cost", value=order.cost / self.grind_v2_stake_multiplier_first)
        stake_cost = trade.get_custom_data(key="stake_cost")
        log.info(f"[{trade.pair}] of trade stake_cost: {stake_cost}")
        return None

    # Adjust Trade Position
    # ---------------------------------------------------------------------------------------------
    def adjust_trade_position(
            self,
            trade: Trade,
            current_time: datetime,
            current_rate: float,
            current_profit: float,
            min_stake: Optional[float],
            max_stake: float,
            current_entry_rate: float,
            current_exit_rate: float,
            current_entry_profit: float,
            current_exit_profit: float,
            **kwargs,
    ):
        if not self.position_adjustment_enable:
            return None

        enter_tag = "empty"
        if hasattr(trade, "enter_tag") and trade.enter_tag is not None:
            enter_tag = trade.enter_tag
        enter_tags = enter_tag.split()

        if not trade.is_short:
            return self.long_grind_adjust_trade_position_v2(
                trade,
                enter_tags,
                current_time,
                current_rate,
                current_profit,
                min_stake,
                max_stake,
                current_entry_rate,
                current_exit_rate,
                current_entry_profit,
                current_exit_profit,
            )
        else:
            return self.short_grind_adjust_trade_position_v2(
                trade,
                enter_tags,
                current_time,
                current_rate,
                current_profit,
                min_stake,
                max_stake,
                current_entry_rate,
                current_exit_rate,
                current_entry_profit,
                current_exit_profit,
            )

    def notification_msg(
            self,
            msg_type: str,
            tag: str,
            pair: str,
            rate: float,
            stake_amount: float,
            profit_stake: float,
            profit_ratio: float,
            grind_profit_stake: float = None,
            grind_profit_pct: float = None,
            stake_currency: str = None,
            coin_amount: float = None,
    ) -> str:
        # Headers for different message types
        headers = {
            "grinding-entry": f"✅ ​**Grinding entry:** `({tag})`\n",
            "grinding-exit": f"❎​ ​**Grinding exit:** `({tag})`\n",
            "grinding-derisk": f"❌​​ ​**Grinding de-risk:** `({tag})`\n",
            "grinding-stop": f"❌ ​**Grinding stop exit:** `({tag})`\n",
            "re-entry": f"✅ ​**Re-entry:** `({tag})`\n",
            "de-risk": f"❌​​ ​**De-risk:** `({tag})`\n",
        }

        # Start with the header
        msg = headers.get(msg_type, None)

        # Add exchange information
        msg += f"🏦 **Exchange:** `{self.exchange_name.capitalize()}`\n"

        # Common fields
        msg += (
            f"🪙 **Pair:** `{pair}`\n"
            f"〽️ **Rate:** `{rate}`\n"
            f"💰 **Stake amount:** `{stake_amount:.2f}{'' if stake_currency is None else ' ' + stake_currency}`\n"
        )

        # Add coin amount if available (exit/stop cases)
        if coin_amount is not None:
            msg += f"🪙 **Coin amount:** `{coin_amount}`\n"

        # Profit section
        profit_pct = profit_ratio * 100
        msg += (
            f"💵 **Profit (stake):** `{profit_stake:.2f}{'' if stake_currency is None else ' ' + stake_currency}`\n"
            f"💸 **Profit (percent):** `{profit_pct:.2f}%`\n"
        )

        # Grind profit calculation
        if grind_profit_stake is not None and grind_profit_stake != 0:
            msg += f"💶 **Grind profit (stake):** `{grind_profit_stake:.2f}{'' if stake_currency is None else ' ' + stake_currency}`\n"
        if grind_profit_pct is not None and grind_profit_pct != 0:
            msg += f"💸 **Grind profit (percent):** `{(grind_profit_pct * 100.0):.2f}%`"

        return msg

    # Informative Pairs
    # ---------------------------------------------------------------------------------------------
    def informative_pairs(self):
        # get access to all pairs available in whitelist.
        pairs = self.dp.current_whitelist()
        # Assign tf to each pair so they can be downloaded and cached for strategy.
        informative_pairs = []
        for info_timeframe in self.info_timeframes:
            informative_pairs.extend(
                [(pair, info_timeframe) for pair in pairs])

        return informative_pairs

    # Informative 1d Timeframe Indicators
    # ---------------------------------------------------------------------------------------------
    def informative_1d_indicators(self, metadata: dict, info_timeframe) -> DataFrame:
        tik = time.perf_counter()
        assert self.dp, "DataProvider is required for multiple timeframes."
        # Get the informative pair
        informative_1d = self.dp.get_pair_dataframe(
            pair=metadata["pair"], timeframe=info_timeframe)

        # Indicators
        # RSI
        informative_1d["RSI_3"] = pta.rsi(informative_1d["close"], length=3)
        # ROC
        informative_1d["ROC_2"] = pta.roc(informative_1d["close"], length=2)
        informative_1d["ROC_9"] = pta.roc(informative_1d["close"], length=9)

        # Performance logging
        # -----------------------------------------------------------------------------------------
        tok = time.perf_counter()
        log.debug(
            f"[{metadata['pair']}] informative_1d_indicators took: {tok - tik:0.4f} seconds.")

        return informative_1d

    # Informative 4h Timeframe Indicators
    # ---------------------------------------------------------------------------------------------
    def informative_4h_indicators(self, metadata: dict, info_timeframe) -> DataFrame:
        tik = time.perf_counter()
        assert self.dp, "DataProvider is required for multiple timeframes."
        # Get the informative pair
        informative_4h = self.dp.get_pair_dataframe(
            pair=metadata["pair"], timeframe=info_timeframe)

        # Indicators
        # RSI
        informative_4h["RSI_3"] = pta.rsi(informative_4h["close"], length=3)
        informative_4h["RSI_14"] = pta.rsi(informative_4h["close"], length=14)
        # ROC
        informative_4h["ROC_2"] = pta.roc(informative_4h["close"], length=2)
        informative_4h["ROC_9"] = pta.roc(informative_4h["close"], length=9)
        # Max highs
        informative_4h["high_max_12"] = informative_4h["high"].rolling(
            12).max()
        informative_4h["high_max_24"] = informative_4h["high"].rolling(
            24).max()
        # Min lows
        informative_4h["low_min_12"] = informative_4h["low"].rolling(12).min()
        informative_4h["low_min_24"] = informative_4h["low"].rolling(24).min()

        # Performance logging
        # -----------------------------------------------------------------------------------------
        tok = time.perf_counter()
        log.debug(
            f"[{metadata['pair']}] informative_4h_indicators took: {tok - tik:0.4f} seconds.")

        return informative_4h

    # Informative 1h Timeframe Indicators
    # ---------------------------------------------------------------------------------------------
    def informative_1h_indicators(self, metadata: dict, info_timeframe) -> DataFrame:
        tik = time.perf_counter()
        assert self.dp, "DataProvider is required for multiple timeframes."
        # Get the informative pair
        informative_1h = self.dp.get_pair_dataframe(
            pair=metadata["pair"], timeframe=info_timeframe)

        # Indicators
        # -----------------------------------------------------------------------------------------
        # RSI
        informative_1h["RSI_3"] = pta.rsi(informative_1h["close"], length=3)
        informative_1h["RSI_14"] = pta.rsi(informative_1h["close"], length=14)
        # BB 20 - STD2
        bbands_20_2 = pta.bbands(informative_1h["close"], length=20)
        informative_1h["BBB_20_2.0"] = bbands_20_2["BBB_20_2.0"] if isinstance(
            bbands_20_2, pd.DataFrame) else np.nan
        # Williams %R
        informative_1h["WILLR_84"] = pta.willr(
            informative_1h["high"], informative_1h["low"], informative_1h["close"], length=84
        )
        # ROC
        informative_1h["ROC_2"] = pta.roc(informative_1h["close"], length=2)
        informative_1h["ROC_9"] = pta.roc(informative_1h["close"], length=9)
        # Max highs
        informative_1h["high_max_6"] = informative_1h["high"].rolling(6).max()
        informative_1h["high_max_12"] = informative_1h["high"].rolling(
            12).max()
        # Min lows
        informative_1h["low_min_6"] = informative_1h["low"].rolling(6).min()
        informative_1h["low_min_12"] = informative_1h["low"].rolling(12).min()

        # Performance logging
        # -----------------------------------------------------------------------------------------
        tok = time.perf_counter()
        log.debug(
            f"[{metadata['pair']}] informative_1h_indicators took: {tok - tik:0.4f} seconds.")

        return informative_1h

    # Informative 15m Timeframe Indicators
    # ---------------------------------------------------------------------------------------------
    def informative_15m_indicators(self, metadata: dict, info_timeframe) -> DataFrame:
        tik = time.perf_counter()
        assert self.dp, "DataProvider is required for multiple timeframes."

        # Get the informative pair
        informative_15m = self.dp.get_pair_dataframe(
            pair=metadata["pair"], timeframe=info_timeframe)

        # Indicators
        # RSI
        informative_15m["RSI_3"] = pta.rsi(informative_15m["close"], length=3)
        # AROON
        aroon_14 = pta.aroon(
            informative_15m["high"], informative_15m["low"], length=14)
        informative_15m["AROONU_14"] = aroon_14["AROONU_14"] if isinstance(
            aroon_14, pd.DataFrame) else np.nan
        informative_15m["AROOND_14"] = aroon_14["AROOND_14"] if isinstance(
            aroon_14, pd.DataFrame) else np.nan
        # Stochastic RSI
        stochrsi = pta.stochrsi(informative_15m["close"])
        informative_15m["STOCHRSIk_14_14_3_3"] = (
            stochrsi["STOCHRSIk_14_14_3_3"] if isinstance(
                stochrsi, pd.DataFrame) else np.nan
        )

        # Performance logging
        # -----------------------------------------------------------------------------------------
        tok = time.perf_counter()
        log.debug(
            f"[{metadata['pair']}] informative_15m_indicators took: {tok - tik:0.4f} seconds.")

        return informative_15m

    # Coin Pair Base Timeframe Indicators
    # ---------------------------------------------------------------------------------------------
    def base_tf_5m_indicators(self, metadata: dict, df: DataFrame) -> DataFrame:
        tik = time.perf_counter()
        # RSI
        df["RSI_3"] = pta.rsi(df["close"], length=3)
        df["RSI_14"] = pta.rsi(df["close"], length=14)
        df["RSI_20"] = pta.rsi(df["close"], length=20)
        # EMA
        df["EMA_9"] = pta.ema(df["close"], length=9)
        df["EMA_12"] = pta.ema(df["close"], length=12)
        df["EMA_16"] = pta.ema(df["close"], length=16)
        df["EMA_20"] = pta.ema(df["close"], length=20)
        df["EMA_26"] = pta.ema(df["close"], length=26)
        df["EMA_100"] = pta.ema(df["close"], length=100, fillna=0.0)
        # SMA
        df["SMA_9"] = pta.sma(df["close"], length=9)
        df["SMA_16"] = pta.sma(df["close"], length=16)
        df["SMA_21"] = pta.sma(df["close"], length=21)
        df["SMA_30"] = pta.sma(df["close"], length=30)
        # BB 20 - STD2
        bbands_20_2 = pta.bbands(df["close"], length=20)
        df["BBL_20_2.0"] = bbands_20_2["BBL_20_2.0"] if isinstance(
            bbands_20_2, pd.DataFrame) else np.nan
        df["BBU_20_2.0"] = bbands_20_2["BBU_20_2.0"] if isinstance(
            bbands_20_2, pd.DataFrame) else np.nan
        # Williams %R
        df["WILLR_14"] = pta.willr(
            df["high"], df["low"], df["close"], length=14)
        # AROON
        aroon_14 = pta.aroon(df["high"], df["low"], length=14)
        df["AROONU_14"] = aroon_14["AROONU_14"] if isinstance(
            aroon_14, pd.DataFrame) else np.nan
        df["AROOND_14"] = aroon_14["AROOND_14"] if isinstance(
            aroon_14, pd.DataFrame) else np.nan
        # Stochastic RSI
        stochrsi = pta.stochrsi(df["close"])
        df["STOCHRSIk_14_14_3_3"] = stochrsi["STOCHRSIk_14_14_3_3"] if isinstance(
            stochrsi, pd.DataFrame) else np.nan
        # Close max
        df["close_max_12"] = df["close"].rolling(12).max()
        df["close_max_48"] = df["close"].rolling(48).max()
        # Close min
        df["close_min_12"] = df["close"].rolling(12).min()
        df["close_min_48"] = df["close"].rolling(48).min()

        # Performance logging
        # -----------------------------------------------------------------------------------------
        tok = time.perf_counter()
        log.debug(
            f"[{metadata['pair']}] base_tf_5m_indicators took: {tok - tik:0.4f} seconds.")

        return df

    # Coin Pair Indicator Switch Case
    # ---------------------------------------------------------------------------------------------
    def info_switcher(self, metadata: dict, info_timeframe) -> DataFrame:
        if info_timeframe == "1d":
            return self.informative_1d_indicators(metadata, info_timeframe)
        elif info_timeframe == "4h":
            return self.informative_4h_indicators(metadata, info_timeframe)
        elif info_timeframe == "1h":
            return self.informative_1h_indicators(metadata, info_timeframe)
        elif info_timeframe == "15m":
            return self.informative_15m_indicators(metadata, info_timeframe)
        else:
            raise RuntimeError(
                f"{info_timeframe} not supported as informative timeframe for BTC pair.")

    # Populate Indicators
    # ---------------------------------------------------------------------------------------------
    def populate_indicators(self, df: DataFrame, metadata: dict) -> DataFrame:
        tik = time.perf_counter()
        """
        --> Indicators on informative timeframes
        ___________________________________________________________________________________________
        """
        for info_timeframe in self.info_timeframes:
            info_indicators = self.info_switcher(metadata, info_timeframe)
            df = merge_informative_pair(
                df, info_indicators, self.timeframe, info_timeframe, ffill=True)
            # Customize what we drop - in case we need to maintain some informative timeframe ohlcv data
            # Default drop all except base timeframe ohlcv data
            drop_columns = {
                "1d": [f"{s}_{info_timeframe}" for s in ["date", "open", "high", "low", "close", "volume"]],
                "4h": [f"{s}_{info_timeframe}" for s in ["date", "open", "high", "low", "close", "volume"]],
                "1h": [f"{s}_{info_timeframe}" for s in ["date", "open", "high", "low", "close", "volume"]],
                "15m": [f"{s}_{info_timeframe}" for s in ["date", "high", "low", "volume"]],
            }.get(info_timeframe, [f"{s}_{info_timeframe}" for s in ["date", "open", "high", "low", "close", "volume"]])
            df.drop(columns=df.columns.intersection(
                drop_columns), inplace=True)

        """
        --> The indicators for the base timeframe  (5m)
        ___________________________________________________________________________________________
        """
        df = self.base_tf_5m_indicators(metadata, df)
        df["RSI_14_1h"] = df["RSI_14_1h"].astype(np.float64).replace(
            to_replace=[np.nan, None], value=(50.0))

        tok = time.perf_counter()
        log.debug(
            f"[{metadata['pair']}] Populate indicators took a total of: {tok - tik:0.4f} seconds.")

        return df

    # Confirm Trade Exit
    # ---------------------------------------------------------------------------------------------
    def confirm_trade_exit(
            self,
            pair: str,
            trade: Trade,
            order_type: str,
            amount: float,
            rate: float,
            time_in_force: str,
            exit_reason: str,
            current_time: datetime,
            **kwargs,
    ) -> bool:
        # Allow force exits
        if exit_reason != "force_exit":
            if exit_reason in ["stop_loss", "trailing_stop_loss"]:
                return False

        return True

    # Bot Loop Start
    # ---------------------------------------------------------------------------------------------
    def bot_loop_start(self, current_time: datetime, **kwargs) -> None:
        if self.config["runmode"].value not in ("live", "dry_run"):
            return super().bot_loop_start(datetime, **kwargs)

        # Check and set stake_cost for existing trades that don't have it
        trades = Trade.get_trades_proxy(is_open=True)
        for trade in trades:
            stake_cost = trade.get_custom_data(key="stake_cost")
            if stake_cost is None:
                filled_entries = trade.select_filled_orders(trade.entry_side)
                if filled_entries:
                    # Use the first entry order's cost as stake_cost
                    first_entry = filled_entries[0]
                    trade.set_custom_data(key="stake_cost", value=first_entry.cost / self.grind_v2_stake_multiplier_first)
                    log.info(f"[{trade.pair}] Set stake_cost to {first_entry.cost} for existing trade")

        return super().bot_loop_start(current_time, **kwargs)

    # Leverage
    # ---------------------------------------------------------------------------------------------
    def leverage(
            self,
            pair: str,
            current_time: datetime,
            current_rate: float,
            proposed_leverage: float,
            max_leverage: float,
            entry_tag: Optional[str],
            side: str,
            **kwargs,
    ) -> float:
        return self.futures_mode_leverage

    # Correct Min Stake
    # ---------------------------------------------------------------------------------------------
    def correct_min_stake(self, min_stake: float) -> float:
        if self.exchange_name == "bybit":
            if self.is_futures_mode:
                if min_stake < 5.0 / self.futures_mode_leverage:
                    min_stake = 5.0 / self.futures_mode_leverage
        return min_stake

    def is_backtest_mode(self) -> bool:
        """Check if the current run mode is backtest or hyperopt"""
        return self.dp.runmode.value in ["backtest", "hyperopt"]

    # Populate Exit Trend
    # ---------------------------------------------------------------------------------------------
    def populate_exit_trend(self, df: DataFrame, metadata: dict) -> DataFrame:
        df.loc[:, "exit_long"] = 0
        df.loc[:, "exit_short"] = 0

        return df

    # Populate Entry Trend
    # ---------------------------------------------------------------------------------------------
    def populate_entry_trend(self, df: DataFrame, metadata: dict) -> DataFrame:
        long_entry_conditions = []
        short_entry_conditions = []

        df.loc[:, "enter_tag"] = ""
        df.loc[:, "enter_long"] = 0
        df.loc[:, "enter_short"] = 0

        long_entry_conditions.append(df["STOCHRSIk_14_14_3_3"] < 20.0)
        long_entry_conditions.append(df["WILLR_14"] < -80.0)
        long_entry_conditions.append(df["AROONU_14"] < 25.0)
        long_entry_conditions.append(df["close"] < (df["EMA_20"] * 0.978))
        long_entry_conditions.append(df["volume"] > 0)

        short_entry_conditions.append(df["EMA_12"] > df["EMA_26"])
        short_entry_conditions.append((df["EMA_12"] - df["EMA_26"]) > (df["open"] * 0.010))
        short_entry_conditions.append((df["EMA_12"].shift() - df["EMA_26"].shift()) > (df["open"] / 100.0))
        short_entry_conditions.append(df["close"] > (df["BBU_20_2.0"] * 1.002))

        # if long_entry_conditions:
        #     df.loc[:, "enter_long"] = reduce(
        #         lambda x, y: x & y, long_entry_conditions)

        # if short_entry_conditions:
        #     df.loc[:, "enter_short"] = reduce(
        #         lambda x, y: x & y, short_entry_conditions)

        return df

    ###############################################################################################

    # Long Exit Grind
    # ---------------------------------------------------------------------------------------------
    def long_exit_grind(
            self,
            profit_init_ratio: float,
            last_candle,
            previous_candle_1,
    ) -> tuple:
        if profit_init_ratio > self.grind_v2_profit_exit_threshold:
            return True, f"exit_{self.long_grind_mode_name}_g"

        #  Here ends exit signal conditions for long_exit_grind
        return False, None

    # Long Grinding Adjust Trade Position v2
    # ---------------------------------------------------------------------------------------------
    def long_grind_adjust_trade_position_v2(
            self,
            trade: Trade,
            enter_tags,
            current_time: datetime,
            current_rate: float,
            current_profit: float,
            min_stake: Optional[float],
            max_stake: float,
            current_entry_rate: float,
            current_exit_rate: float,
            current_entry_profit: float,
            current_exit_profit: float,
            **kwargs,
    ):
        is_backtest = self.is_backtest_mode()
        min_stake = self.correct_min_stake(min_stake)
        df, _ = self.dp.get_analyzed_dataframe(trade.pair, self.timeframe)
        if len(df) < 2:
            return None
        last_candle = df.iloc[-1].squeeze()
        previous_candle = df.iloc[-2].squeeze()

        # we already waiting for an order to get filled
        if trade.has_open_orders:
            return None

        filled_orders = trade.select_filled_orders()
        filled_entries = trade.select_filled_orders(trade.entry_side)
        filled_exits = trade.select_filled_orders(trade.exit_side)

        exit_rate = current_rate
        if self.dp.runmode.value in ("live", "dry_run"):
            ticker = self.dp.ticker(trade.pair)
            if ("bid" in ticker) and ("ask" in ticker):
                if trade.is_short:
                    if self.exit_price_side in ["ask", "other"]:
                        if ticker["ask"] is not None:
                            exit_rate = ticker["ask"]
                else:
                    if self.exit_price_side in ["bid", "other"]:
                        if ticker["bid"] is not None:
                            exit_rate = ticker["bid"]

        profit_stake, profit_ratio, profit_current_stake_ratio, profit_init_ratio = self.calc_total_profit(
            trade, filled_entries, filled_exits, exit_rate
        )

        current_stake_amount = trade.amount * exit_rate
        stake_cost = trade.get_custom_data(key="stake_cost")
        slice_amount = stake_cost if stake_cost else filled_entries[0].cost
        slice_profit = (
                               exit_rate - filled_orders[-1].safe_price) / filled_orders[-1].safe_price

        has_order_tags = False
        if hasattr(filled_orders[0], "ft_order_tag"):
            has_order_tags = True

        fee_open_rate = trade.fee_open if self.custom_fee_open_rate is None else self.custom_fee_open_rate
        fee_close_rate = trade.fee_close if self.custom_fee_close_rate is None else self.custom_fee_close_rate

        grind_1_max_sub_grinds = 0
        grind_1_stakes = (
            self.grinding_v2_grind_1_stakes_futures.copy()
            if self.is_futures_mode
            else self.grinding_v2_grind_1_stakes_spot.copy()
        )
        grind_1_sub_thresholds = (
            self.grinding_v2_grind_1_thresholds_futures
            if self.is_futures_mode
            else self.grinding_v2_grind_1_thresholds_spot
        )
        if (slice_amount * grind_1_stakes[0] / (trade.leverage if self.is_futures_mode else 1.0)) < min_stake:
            multi = min_stake / slice_amount / \
                    grind_1_stakes[0] * trade.leverage
            for i, _ in enumerate(grind_1_stakes):
                grind_1_stakes[i] *= multi
        grind_1_max_sub_grinds = len(grind_1_stakes)
        grind_1_derisk_grinds = (
            self.grinding_v2_grind_1_derisk_futures if self.is_futures_mode else self.grinding_v2_grind_1_derisk_spot
        )
        grind_1_profit_threshold = (
            self.grinding_v2_grind_1_profit_threshold_futures
            if self.is_futures_mode
            else self.grinding_v2_grind_1_profit_threshold_spot
        )

        grind_2_max_sub_grinds = 0
        grind_2_stakes = (
            self.grinding_v2_grind_2_stakes_futures.copy()
            if self.is_futures_mode
            else self.grinding_v2_grind_2_stakes_spot.copy()
        )
        grind_2_sub_thresholds = (
            self.grinding_v2_grind_2_thresholds_futures
            if self.is_futures_mode
            else self.grinding_v2_grind_2_thresholds_spot
        )
        if (slice_amount * grind_2_stakes[0] / (trade.leverage if self.is_futures_mode else 1.0)) < min_stake:
            multi = min_stake / slice_amount / \
                    grind_2_stakes[0] * trade.leverage
            for i, _ in enumerate(grind_2_stakes):
                grind_2_stakes[i] *= multi
        grind_2_max_sub_grinds = len(grind_2_stakes)
        grind_2_derisk_grinds = (
            self.grinding_v2_grind_2_derisk_futures if self.is_futures_mode else self.grinding_v2_grind_2_derisk_spot
        )
        grind_2_profit_threshold = (
            self.grinding_v2_grind_2_profit_threshold_futures
            if self.is_futures_mode
            else self.grinding_v2_grind_2_profit_threshold_spot
        )

        grind_3_max_sub_grinds = 0
        grind_3_stakes = (
            self.grinding_v2_grind_3_stakes_futures.copy()
            if self.is_futures_mode
            else self.grinding_v2_grind_3_stakes_spot.copy()
        )
        grind_3_sub_thresholds = (
            self.grinding_v2_grind_3_thresholds_futures
            if self.is_futures_mode
            else self.grinding_v2_grind_3_thresholds_spot
        )
        if (slice_amount * grind_3_stakes[0] / (trade.leverage if self.is_futures_mode else 1.0)) < min_stake:
            multi = min_stake / slice_amount / \
                    grind_3_stakes[0] * trade.leverage
            for i, _ in enumerate(grind_3_stakes):
                grind_3_stakes[i] *= multi
        grind_3_max_sub_grinds = len(grind_3_stakes)
        grind_3_derisk_grinds = (
            self.grinding_v2_grind_3_derisk_futures if self.is_futures_mode else self.grinding_v2_grind_3_derisk_spot
        )
        grind_3_profit_threshold = (
            self.grinding_v2_grind_3_profit_threshold_futures
            if self.is_futures_mode
            else self.grinding_v2_grind_3_profit_threshold_spot
        )

        grind_x_max_sub_grinds = 0
        grind_x_stakes = (
            self.grinding_v2_grind_x_stakes_futures.copy()
            if self.is_futures_mode
            else self.grinding_v2_grind_x_stakes_spot.copy()
        )
        grind_x_sub_thresholds = (
            self.grinding_v2_grind_x_thresholds_futures
            if self.is_futures_mode
            else self.grinding_v2_grind_x_thresholds_spot
        )
        if (slice_amount * grind_x_stakes[0] / (trade.leverage if self.is_futures_mode else 1.0)) < min_stake:
            multi = min_stake / slice_amount / \
                    grind_x_stakes[0] * trade.leverage
            for i, _ in enumerate(grind_x_stakes):
                grind_x_stakes[i] *= multi
        grind_x_max_sub_grinds = len(grind_x_stakes)
        grind_x_derisk_grinds = (
            self.grinding_v2_grind_x_derisk_futures if self.is_futures_mode else self.grinding_v2_grind_x_derisk_spot
        )
        grind_x_profit_threshold = (
            self.grinding_v2_grind_x_profit_threshold_futures
            if self.is_futures_mode
            else self.grinding_v2_grind_x_profit_threshold_spot
        )

        grind_1_sub_grind_count = 0
        grind_1_total_amount = 0.0
        grind_1_total_cost = 0.0
        grind_1_current_open_rate = 0.0
        grind_1_current_grind_stake = 0.0
        grind_1_current_grind_stake_profit = 0.0
        grind_1_is_exit_found = False
        grind_1_found = False
        grind_1_buy_orders = []
        grind_1_orders = []
        grind_1_distance_ratio = 0.0
        grind_2_sub_grind_count = 0
        grind_2_total_amount = 0.0
        grind_2_total_cost = 0.0
        grind_2_current_open_rate = 0.0
        grind_2_current_grind_stake = 0.0
        grind_2_current_grind_stake_profit = 0.0
        grind_2_is_exit_found = False
        grind_2_found = False
        grind_2_buy_orders = []
        grind_2_orders = []
        grind_2_distance_ratio = 0.0
        grind_3_sub_grind_count = 0
        grind_3_total_amount = 0.0
        grind_3_total_cost = 0.0
        grind_3_current_open_rate = 0.0
        grind_3_current_grind_stake = 0.0
        grind_3_current_grind_stake_profit = 0.0
        grind_3_is_exit_found = False
        grind_3_found = False
        grind_3_buy_orders = []
        grind_3_orders = []
        grind_3_distance_ratio = 0.0
        grind_x_sub_grind_count = 0
        grind_x_total_amount = 0.0
        grind_x_total_cost = 0.0
        grind_x_current_open_rate = 0.0
        grind_x_current_grind_stake = 0.0
        grind_x_current_grind_stake_profit = 0.0
        grind_x_is_exit_found = False
        grind_x_found = False
        grind_x_buy_orders = []
        grind_x_orders = []
        grind_x_distance_ratio = 0.0
        for order in reversed(filled_orders):
            if order.ft_order_side == "buy":
                order_tag = ""
                if has_order_tags:
                    if order.ft_order_tag is not None:
                        order_tag = order.ft_order_tag
                if not grind_1_is_exit_found and order_tag == "grind_1_entry":
                    grind_1_sub_grind_count += 1
                    grind_1_total_amount += order.safe_filled
                    grind_1_total_cost += order.safe_filled * order.safe_price
                    grind_1_buy_orders.append(order.id)
                    grind_1_orders.append(order)
                    if not grind_1_found:
                        grind_1_distance_ratio = (
                                                         exit_rate - order.safe_price) / order.safe_price
                        grind_1_found = True
                elif not grind_2_is_exit_found and order_tag == "grind_2_entry":
                    grind_2_sub_grind_count += 1
                    grind_2_total_amount += order.safe_filled
                    grind_2_total_cost += order.safe_filled * order.safe_price
                    grind_2_buy_orders.append(order.id)
                    grind_2_orders.append(order)
                    if not grind_2_found:
                        grind_2_distance_ratio = (
                                                         exit_rate - order.safe_price) / order.safe_price
                        grind_2_found = True
                elif not grind_3_is_exit_found and order_tag == "grind_3_entry":
                    grind_3_sub_grind_count += 1
                    grind_3_total_amount += order.safe_filled
                    grind_3_total_cost += order.safe_filled * order.safe_price
                    grind_3_buy_orders.append(order.id)
                    grind_3_orders.append(order)
                    if not grind_3_found:
                        grind_3_distance_ratio = (
                                                         exit_rate - order.safe_price) / order.safe_price
                        grind_3_found = True
                elif not grind_x_is_exit_found and order_tag not in ["grind_1_entry", "grind_2_entry", "grind_3_entry"]:
                    grind_x_sub_grind_count += 1
                    grind_x_total_amount += order.safe_filled
                    grind_x_total_cost += order.safe_filled * order.safe_price
                    grind_x_buy_orders.append(order.id)
                    grind_x_orders.append(order)
                    if not grind_x_found:
                        grind_x_distance_ratio = (
                                                         exit_rate - order.safe_price) / order.safe_price
                        grind_x_found = True
            elif order.ft_order_side == "sell":
                order_tag = ""
                if has_order_tags:
                    if order.ft_order_tag is not None:
                        sell_order_tag = order.ft_order_tag
                        order_mode = sell_order_tag.split(" ", 1)
                        if len(order_mode) > 0:
                            order_tag = order_mode[0]
                if not grind_1_is_exit_found and order_tag in ["grind_1_exit", "grind_1_derisk"]:
                    grind_1_is_exit_found = True
                elif not grind_2_is_exit_found and order_tag in ["grind_2_exit", "grind_2_derisk"]:
                    grind_2_is_exit_found = True
                elif not grind_3_is_exit_found and order_tag in ["grind_3_exit", "grind_3_derisk"]:
                    grind_3_is_exit_found = True
                elif not grind_x_is_exit_found and order_tag in ["grind_x_exit", "grind_x_derisk", "derisk_v2", "grind_v2_exit"]:
                    grind_x_is_exit_found = True

        if grind_1_sub_grind_count > 0:
            grind_1_current_open_rate = grind_1_total_cost / grind_1_total_amount
            grind_1_current_grind_stake = grind_1_total_amount * \
                                          exit_rate * (1 - trade.fee_close)
            grind_1_current_grind_stake_profit = grind_1_current_grind_stake - grind_1_total_cost
        if grind_2_sub_grind_count > 0:
            grind_2_current_open_rate = grind_2_total_cost / grind_2_total_amount
            grind_2_current_grind_stake = grind_2_total_amount * \
                                          exit_rate * (1 - trade.fee_close)
            grind_2_current_grind_stake_profit = grind_2_current_grind_stake - grind_2_total_cost
        if grind_3_sub_grind_count > 0:
            grind_3_current_open_rate = grind_3_total_cost / grind_3_total_amount
            grind_3_current_grind_stake = grind_3_total_amount * \
                                          exit_rate * (1 - trade.fee_close)
            grind_3_current_grind_stake_profit = grind_3_current_grind_stake - grind_3_total_cost
        if grind_x_sub_grind_count > 0:
            grind_x_current_open_rate = grind_x_total_cost / grind_x_total_amount
            grind_x_current_grind_stake = grind_x_total_amount * \
                                          exit_rate * (1 - trade.fee_close)
            grind_x_current_grind_stake_profit = grind_x_current_grind_stake - grind_x_total_cost

        # all grinds
        num_open_grinds = grind_1_sub_grind_count + grind_2_sub_grind_count + grind_3_sub_grind_count + grind_x_sub_grind_count

        # not reached the max allowed stake for all grinds
        is_not_trade_max_stake = current_stake_amount < (slice_amount * self.grinding_v2_max_stake)

        is_long_extra_checks_entry = (
                (current_time - timedelta(minutes=5) > filled_entries[-1].order_filled_utc)
                and ((current_time - timedelta(hours=2) > filled_orders[-1].order_filled_utc) or (slice_profit < -0.06))
        )
        is_long_grind_entry = (
                self.long_grind_entry_v2(
                    last_candle, previous_candle, slice_profit)
                or (
                        (num_open_grinds == 0)
                        and (
                                (last_candle["RSI_3"] > 10.0)
                                and (last_candle["RSI_3_15m"] > 20.0)
                                and (last_candle["RSI_3_1h"] > 20.0)
                                and (last_candle["RSI_3_1h"] > 20.0)
                                and (last_candle["AROONU_14"] < 50.0)
                                and (last_candle["AROONU_14_15m"] < 50.0)
                        )
                )
                or (
                        self.is_futures_mode
                        and trade.liquidation_price is not None
                        and (
                                (trade.is_short and current_rate >
                                 trade.liquidation_price * 0.90)
                                or (not trade.is_short and current_rate < trade.liquidation_price * 1.10)
                        )
                        and (slice_profit < -0.03)
                        and (last_candle["RSI_3"] > 10.0)
                        and (last_candle["RSI_3_15m"] > 20.0)
                        and (last_candle["AROONU_14"] < 50.0)
                        and (last_candle["AROONU_14_15m"] < 50.0)
                )
        )

        # Grinding 1

        if (
                self.grinding_v2_grind_1_enable
                and is_long_grind_entry
                and is_long_extra_checks_entry
                and (grind_1_sub_grind_count < grind_1_max_sub_grinds)
                and (
                (grind_1_sub_grind_count == 0)
                or (grind_1_distance_ratio < grind_1_sub_thresholds[grind_1_sub_grind_count])
        )
                and is_not_trade_max_stake
        ):
            buy_amount = slice_amount * \
                         grind_1_stakes[grind_1_sub_grind_count] / trade.leverage
            if buy_amount < (min_stake * 1.5):
                buy_amount = min_stake * 1.5
            if buy_amount > max_stake:
                return None
            self.dp.send_msg(
                self.notification_msg(
                    "grinding-entry",
                    tag="grind_1_entry",
                    pair=trade.pair,
                    rate=current_rate,
                    stake_amount=buy_amount,
                    profit_stake=profit_stake,
                    profit_ratio=profit_ratio,
                    stake_currency=self.stake_currency,
                )
            )
            log.info(
                f"Grinding entry (grind_1_entry) [{current_time}] [{trade.pair}] | Rate: {current_rate} | Stake amount: {buy_amount} | Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}%"
            )
            order_tag = "grind_1_entry"
            if has_order_tags:
                return buy_amount, order_tag
            else:
                return buy_amount

        if grind_1_sub_grind_count > 0:
            grind_profit = (exit_rate - grind_1_current_open_rate) / \
                           grind_1_current_open_rate
            if (grind_profit > (grind_1_profit_threshold + fee_open_rate + fee_close_rate)) and self.long_grind_exit_v2(
                    last_candle, previous_candle, slice_profit
            ):
                sell_amount = grind_1_total_amount * exit_rate / trade.leverage
                if ((current_stake_amount / trade.leverage) - sell_amount) < (min_stake * 1.55):
                    sell_amount = (trade.amount * exit_rate /
                                   trade.leverage) - (min_stake * 1.55)
                ft_sell_amount = sell_amount * trade.leverage * \
                                 (trade.stake_amount / trade.amount) / exit_rate
                if sell_amount > min_stake and ft_sell_amount > min_stake:
                    self.dp.send_msg(
                        self.notification_msg(
                            "grinding-exit",
                            tag="grind_1_exit",
                            pair=trade.pair,
                            rate=exit_rate,
                            stake_amount=sell_amount,
                            profit_stake=profit_stake,
                            profit_ratio=profit_ratio,
                            stake_currency=self.stake_currency,
                            grind_profit_stake=grind_profit * sell_amount * trade.leverage,
                            grind_profit_pct=grind_profit,
                            coin_amount=grind_1_total_amount,
                        )
                    )
                    log.info(
                        f"Grinding exit (grind_1_exit) [{current_time}] [{trade.pair}] | Rate: {exit_rate} | Stake amount: {sell_amount} | Coin amount: {grind_1_total_amount} | Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}% | Grind profit: {(grind_profit * 100.0):.2f}% ({grind_profit * sell_amount * trade.leverage} {self.stake_currency})"
                    )
                    order_tag = "grind_1_exit"
                    for grind_entry_id in grind_1_buy_orders:
                        order_tag += " " + str(grind_entry_id)
                    if has_order_tags:
                        return -ft_sell_amount, order_tag
                    else:
                        return -ft_sell_amount

        if (
                self.grinding_v2_grind_1_use_derisk
                and (grind_1_sub_grind_count > 0)
                and (((exit_rate - grind_1_current_open_rate) / grind_1_current_open_rate) < grind_1_derisk_grinds)
                and (grind_1_orders[-1].order_date_utc.replace(tzinfo=None) >= datetime(2025, 8, 3) or is_backtest)
        ):
            sell_amount = grind_1_total_amount * exit_rate / trade.leverage
            if ((current_stake_amount / trade.leverage) - sell_amount) < (min_stake * 1.55):
                sell_amount = (trade.amount * exit_rate /
                               trade.leverage) - (min_stake * 1.55)
            ft_sell_amount = sell_amount * trade.leverage * \
                             (trade.stake_amount / trade.amount) / exit_rate
            if sell_amount > min_stake and ft_sell_amount > min_stake:
                grind_profit = 0.0
                if grind_1_current_open_rate > 0.0:
                    grind_profit = (
                        ((exit_rate - grind_1_current_open_rate) /
                         grind_1_current_open_rate)
                        if grind_1_is_exit_found
                        else profit_ratio
                    )
                self.dp.send_msg(
                    self.notification_msg(
                        "grinding-derisk",
                        tag="grind_1_derisk",
                        pair=trade.pair,
                        rate=exit_rate,
                        stake_amount=sell_amount,
                        profit_stake=profit_stake,
                        profit_ratio=profit_ratio,
                        stake_currency=self.stake_currency,
                        grind_profit_stake=grind_1_current_grind_stake_profit,
                        grind_profit_pct=grind_profit,
                        coin_amount=grind_1_total_amount,
                    )
                )
                log.info(
                    f"Grinding de-risk (grind_1_derisk) [{current_time}] [{trade.pair}] | Rate: {exit_rate} | Stake amount: {sell_amount} | Coin amount: {grind_1_total_amount} | Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}% | Grind profit: {(grind_profit * 100.0):.2f}% ({grind_1_current_grind_stake_profit} {self.stake_currency})"
                )
                order_tag = "grind_1_derisk"
                for grind_entry_id in grind_1_buy_orders:
                    order_tag += " " + str(grind_entry_id)
                if has_order_tags:
                    return -ft_sell_amount, order_tag
                else:
                    return -ft_sell_amount

        # Grinding 2

        if (
                self.grinding_v2_grind_2_enable
                and is_long_grind_entry
                and is_long_extra_checks_entry
                and (grind_2_sub_grind_count < grind_2_max_sub_grinds)
                and (
                (grind_2_sub_grind_count == 0) and (grind_1_sub_grind_count > 1)
                or (grind_2_distance_ratio < grind_2_sub_thresholds[grind_2_sub_grind_count])
        )
                and is_not_trade_max_stake
        ):
            buy_amount = slice_amount * \
                         grind_2_stakes[grind_2_sub_grind_count] / trade.leverage
            if buy_amount < (min_stake * 1.5):
                buy_amount = min_stake * 1.5
            if buy_amount > max_stake:
                return None
            self.dp.send_msg(
                self.notification_msg(
                    "grinding-entry",
                    tag="grind_2_entry",
                    pair=trade.pair,
                    rate=current_rate,
                    stake_amount=buy_amount,
                    profit_stake=profit_stake,
                    profit_ratio=profit_ratio,
                    stake_currency=self.stake_currency,
                )
            )
            log.info(
                f"Grinding entry (grind_2_entry) [{current_time}] [{trade.pair}] | Rate: {current_rate} | Stake amount: {buy_amount} | Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}%"
            )
            order_tag = "grind_2_entry"
            if has_order_tags:
                return buy_amount, order_tag
            else:
                return buy_amount

        if grind_2_sub_grind_count > 0:
            grind_profit = (exit_rate - grind_2_current_open_rate) / \
                           grind_2_current_open_rate
            if (grind_profit > (grind_2_profit_threshold + fee_open_rate + fee_close_rate)) and self.long_grind_exit_v2(
                    last_candle, previous_candle, slice_profit
            ):
                sell_amount = grind_2_total_amount * exit_rate / trade.leverage
                if ((current_stake_amount / trade.leverage) - sell_amount) < (min_stake * 1.55):
                    sell_amount = (trade.amount * exit_rate /
                                   trade.leverage) - (min_stake * 1.55)
                ft_sell_amount = sell_amount * trade.leverage * \
                                 (trade.stake_amount / trade.amount) / exit_rate
                if sell_amount > min_stake and ft_sell_amount > min_stake:
                    self.dp.send_msg(
                        self.notification_msg(
                            "grinding-exit",
                            tag="grind_2_exit",
                            pair=trade.pair,
                            rate=exit_rate,
                            stake_amount=sell_amount,
                            profit_stake=profit_stake,
                            profit_ratio=profit_ratio,
                            stake_currency=self.stake_currency,
                            grind_profit_stake=grind_profit * sell_amount * trade.leverage,
                            grind_profit_pct=grind_profit,
                            coin_amount=grind_2_total_amount,
                        )
                    )
                    log.info(
                        f"Grinding exit (grind_2_exit) [{current_time}] [{trade.pair}] | Rate: {exit_rate} | Stake amount: {sell_amount} | Coin amount: {grind_2_total_amount} | Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}% | Grind profit: {(grind_profit * 100.0):.2f}% ({grind_profit * sell_amount * trade.leverage} {self.stake_currency})"
                    )
                    order_tag = "grind_2_exit"
                    for grind_entry_id in grind_2_buy_orders:
                        order_tag += " " + str(grind_entry_id)
                    if has_order_tags:
                        return -ft_sell_amount, order_tag
                    else:
                        return -ft_sell_amount

        if (
                self.grinding_v2_grind_2_use_derisk
                and (grind_2_sub_grind_count > 0)
                and (((exit_rate - grind_2_current_open_rate) / grind_2_current_open_rate) < grind_2_derisk_grinds)
                and (grind_2_orders[-1].order_date_utc.replace(tzinfo=None) >= datetime(2025, 8, 3) or is_backtest)
        ):
            sell_amount = grind_2_total_amount * exit_rate / trade.leverage
            if ((current_stake_amount / trade.leverage) - sell_amount) < (min_stake * 1.55):
                sell_amount = (trade.amount * exit_rate /
                               trade.leverage) - (min_stake * 1.55)
            ft_sell_amount = sell_amount * trade.leverage * \
                             (trade.stake_amount / trade.amount) / exit_rate
            if sell_amount > min_stake and ft_sell_amount > min_stake:
                grind_profit = 0.0
                if grind_2_current_open_rate > 0.0:
                    grind_profit = (
                        ((exit_rate - grind_2_current_open_rate) /
                         grind_2_current_open_rate)
                        if grind_2_is_exit_found
                        else profit_ratio
                    )
                self.dp.send_msg(
                    self.notification_msg(
                        "grinding-derisk",
                        tag="grind_2_derisk",
                        pair=trade.pair,
                        rate=exit_rate,
                        stake_amount=sell_amount,
                        profit_stake=profit_stake,
                        profit_ratio=profit_ratio,
                        stake_currency=self.stake_currency,
                        grind_profit_stake=grind_2_current_grind_stake_profit,
                        grind_profit_pct=grind_profit,
                        coin_amount=grind_2_total_amount,
                    )
                )
                log.info(
                    f"Grinding de-risk (grind_2_derisk) [{current_time}] [{trade.pair}] | Rate: {exit_rate} | Stake amount: {sell_amount} | Coin amount: {grind_2_total_amount} | Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}% | Grind profit: {(grind_profit * 100.0):.2f}% ({grind_2_current_grind_stake_profit} {self.stake_currency})"
                )
                order_tag = "grind_2_derisk"
                for grind_entry_id in grind_2_buy_orders:
                    order_tag += " " + str(grind_entry_id)
                if has_order_tags:
                    return -ft_sell_amount, order_tag
                else:
                    return -ft_sell_amount

        # Grinding 3 Entry
        if (
                self.grinding_v2_grind_3_enable
                and is_long_grind_entry
                and is_long_extra_checks_entry
                and (grind_3_sub_grind_count < grind_3_max_sub_grinds)
                and (
                (grind_3_sub_grind_count == 0) and (grind_2_sub_grind_count > 1)
                or (grind_3_distance_ratio < grind_3_sub_thresholds[grind_3_sub_grind_count])
        )
                and is_not_trade_max_stake
        ):
            buy_amount = slice_amount * grind_3_stakes[grind_3_sub_grind_count] / trade.leverage
            if buy_amount < (min_stake * 1.5):
                buy_amount = min_stake * 1.5
            if buy_amount > max_stake:
                return None
            self.dp.send_msg(
                self.notification_msg(
                    "grinding-entry",
                    tag="grind_3_entry",
                    pair=trade.pair,
                    rate=current_rate,
                    stake_amount=buy_amount,
                    profit_stake=profit_stake,
                    profit_ratio=profit_ratio,
                    stake_currency=self.stake_currency,
                )
            )
            log.info(
                f"Grinding entry (grind_3_entry) [{current_time}] [{trade.pair}] | Rate: {current_rate} | Stake amount: {buy_amount} | Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}%"
            )
            order_tag = "grind_3_entry"
            if has_order_tags:
                return buy_amount, order_tag
            else:
                return buy_amount

        # Grinding 3 Exit
        if grind_3_sub_grind_count > 0:
            grind_profit = (exit_rate - grind_3_current_open_rate) / grind_3_current_open_rate
            if (grind_profit > (grind_3_profit_threshold + fee_open_rate + fee_close_rate)) and self.long_grind_exit_v2(
                    last_candle, previous_candle, slice_profit
            ):
                sell_amount = grind_3_total_amount * exit_rate / trade.leverage
                if ((current_stake_amount / trade.leverage) - sell_amount) < (min_stake * 1.55):
                    sell_amount = (trade.amount * exit_rate /
                                   trade.leverage) - (min_stake * 1.55)
                ft_sell_amount = sell_amount * trade.leverage * \
                                 (trade.stake_amount / trade.amount) / exit_rate
                if sell_amount > min_stake and ft_sell_amount > min_stake:
                    self.dp.send_msg(
                        self.notification_msg(
                            "grinding-exit",
                            tag="grind_3_exit",
                            pair=trade.pair,
                            rate=exit_rate,
                            stake_amount=sell_amount,
                            profit_stake=profit_stake,
                            profit_ratio=profit_ratio,
                            stake_currency=self.stake_currency,
                            grind_profit_stake=grind_profit * sell_amount * trade.leverage,
                            grind_profit_pct=grind_profit,
                            coin_amount=grind_3_total_amount,
                        )
                    )
                    log.info(
                        f"Grinding exit (grind_3_exit) [{current_time}] [{trade.pair}] | Rate: {exit_rate} | Stake amount: {sell_amount} | Coin amount: {grind_3_total_amount} | Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}% | Grind profit: {(grind_profit * 100.0):.2f}% ({grind_profit * sell_amount * trade.leverage} {self.stake_currency})"
                    )
                    order_tag = "grind_3_exit"
                    for grind_entry_id in grind_3_buy_orders:
                        order_tag += " " + str(grind_entry_id)
                    if has_order_tags:
                        return -ft_sell_amount, order_tag
                    else:
                        return -ft_sell_amount

        # Grinding 3 De-risk
        if (
                self.grinding_v2_grind_3_use_derisk
                and (grind_3_sub_grind_count > 0)
                and (((exit_rate - grind_3_current_open_rate) / grind_3_current_open_rate) < grind_3_derisk_grinds)
                and (grind_3_orders[-1].order_date_utc.replace(tzinfo=None) >= datetime(2025, 8, 3) or is_backtest)
        ):
            sell_amount = grind_3_total_amount * exit_rate / trade.leverage
            if ((current_stake_amount / trade.leverage) - sell_amount) < (min_stake * 1.55):
                sell_amount = (trade.amount * exit_rate /
                               trade.leverage) - (min_stake * 1.55)
            ft_sell_amount = sell_amount * trade.leverage * \
                             (trade.stake_amount / trade.amount) / exit_rate
            if sell_amount > min_stake and ft_sell_amount > min_stake:
                grind_profit = 0.0
                if grind_3_current_open_rate > 0.0:
                    grind_profit = (
                        ((exit_rate - grind_3_current_open_rate) /
                         grind_3_current_open_rate)
                        if grind_3_is_exit_found
                        else profit_ratio
                    )
                self.dp.send_msg(
                    self.notification_msg(
                        "grinding-derisk",
                        tag="grind_3_derisk",
                        pair=trade.pair,
                        rate=exit_rate,
                        stake_amount=sell_amount,
                        profit_stake=profit_stake,
                        profit_ratio=profit_ratio,
                        stake_currency=self.stake_currency,
                        grind_profit_stake=grind_3_current_grind_stake_profit,
                        grind_profit_pct=grind_profit,
                        coin_amount=grind_3_total_amount,
                    )
                )
                log.info(
                    f"Grinding de-risk (grind_3_derisk) [{current_time}] [{trade.pair}] | Rate: {exit_rate} | Stake amount: {sell_amount} | Coin amount: {grind_3_total_amount} | Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}% | Grind profit: {(grind_profit * 100.0):.2f}% ({grind_3_current_grind_stake_profit} {self.stake_currency})"
                )
                order_tag = "grind_3_derisk"
                for grind_entry_id in grind_3_buy_orders:
                    order_tag += " " + str(grind_entry_id)
                if has_order_tags:
                    return -ft_sell_amount, order_tag
                else:
                    return -ft_sell_amount

        # Grinding X
        if (
                self.grinding_v2_grind_x_enable
                and is_long_grind_entry
                and is_long_extra_checks_entry
                and (grind_x_sub_grind_count < grind_x_max_sub_grinds)
                and grind_x_sub_grind_count != 0
                and (grind_x_distance_ratio < grind_x_sub_thresholds[grind_x_sub_grind_count])
                and is_not_trade_max_stake
        ):
            buy_amount = slice_amount * \
                         grind_x_stakes[grind_x_sub_grind_count] / trade.leverage
            if buy_amount < (min_stake * 1.5):
                buy_amount = min_stake * 1.5
            if buy_amount > max_stake:
                return None
            self.dp.send_msg(
                self.notification_msg(
                    "grinding-entry",
                    tag="grind_x_entry",
                    pair=trade.pair,
                    rate=current_rate,
                    stake_amount=buy_amount,
                    profit_stake=profit_stake,
                    profit_ratio=profit_ratio,
                    stake_currency=self.stake_currency,
                )
            )
            log.info(
                f"Grinding entry (grind_x_entry) [{current_time}] [{trade.pair}] | Rate: {current_rate} | Stake amount: {buy_amount} | Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}%"
            )
            order_tag = "grind_x_entry"
            if has_order_tags:
                return buy_amount, order_tag
            else:
                return buy_amount

        # Grinding X - Exit
        if grind_x_sub_grind_count > 0:
            grind_profit = (exit_rate - grind_x_current_open_rate) / \
                           grind_x_current_open_rate
            if (grind_profit > (grind_x_profit_threshold + fee_open_rate + fee_close_rate)) and self.long_grind_exit_v2(
                    last_candle, previous_candle, slice_profit
            ):
                sell_amount = grind_x_total_amount * exit_rate / trade.leverage
                if ((current_stake_amount / trade.leverage) - sell_amount) < (min_stake * 1.55):
                    sell_amount = (trade.amount * exit_rate /
                                   trade.leverage) - (min_stake * 1.55)
                ft_sell_amount = sell_amount * trade.leverage * \
                                 (trade.stake_amount / trade.amount) / exit_rate
                if sell_amount > min_stake and ft_sell_amount > min_stake:
                    self.dp.send_msg(
                        self.notification_msg(
                            "grinding-exit",
                            tag="grind_x_exit",
                            pair=trade.pair,
                            rate=exit_rate,
                            stake_amount=sell_amount,
                            profit_stake=profit_stake,
                            profit_ratio=profit_ratio,
                            stake_currency=self.stake_currency,
                            grind_profit_stake=grind_profit * sell_amount * trade.leverage,
                            grind_profit_pct=grind_profit,
                            coin_amount=grind_x_total_amount,
                        )
                    )
                    log.info(
                        f"Grinding exit (grind_x_exit) [{current_time}] [{trade.pair}] | Rate: {exit_rate} | Stake amount: {sell_amount} | Coin amount: {grind_x_total_amount} | Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}% | Grind profit: {(grind_profit * 100.0):.2f}% ({grind_profit * sell_amount * trade.leverage} {self.stake_currency})"
                    )
                    order_tag = "grind_x_exit"
                    for grind_entry_id in grind_x_buy_orders:
                        order_tag += " " + str(grind_entry_id)
                    if has_order_tags:
                        return -ft_sell_amount, order_tag
                    else:
                        return -ft_sell_amount

        # Grinding X De-risk
        if (
                self.grinding_v2_grind_x_use_derisk
                and (grind_x_sub_grind_count > 0)
                and (((exit_rate - grind_x_current_open_rate) / grind_x_current_open_rate) < grind_x_derisk_grinds)
                and (grind_x_orders[-1].order_date_utc.replace(tzinfo=None) >= datetime(2025, 8, 3) or is_backtest)
        ):
            sell_amount = grind_x_total_amount * exit_rate / trade.leverage
            if ((current_stake_amount / trade.leverage) - sell_amount) < (min_stake * 1.55):
                sell_amount = (trade.amount * exit_rate /
                               trade.leverage) - (min_stake * 1.55)
            ft_sell_amount = sell_amount * trade.leverage * \
                             (trade.stake_amount / trade.amount) / exit_rate
            if sell_amount > min_stake and ft_sell_amount > min_stake:
                grind_profit = 0.0
                if grind_x_current_open_rate > 0.0:
                    grind_profit = (
                        ((exit_rate - grind_x_current_open_rate) /
                         grind_x_current_open_rate)
                        if grind_x_is_exit_found
                        else profit_ratio
                    )
                self.dp.send_msg(
                    self.notification_msg(
                        "grinding-derisk",
                        tag="grind_x_derisk",
                        pair=trade.pair,
                        rate=exit_rate,
                        stake_amount=sell_amount,
                        profit_stake=profit_stake,
                        profit_ratio=profit_ratio,
                        stake_currency=self.stake_currency,
                        grind_profit_stake=grind_x_current_grind_stake_profit,
                        grind_profit_pct=grind_profit,
                        coin_amount=grind_x_total_amount,
                    )
                )
                log.info(
                    f"Grinding de-risk (grind_x_derisk) [{current_time}] [{trade.pair}] | Rate: {exit_rate} | Stake amount: {sell_amount} | Coin amount: {grind_x_total_amount} | Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}% | Grind profit: {(grind_profit * 100.0):.2f}% ({grind_x_current_grind_stake_profit} {self.stake_currency})"
                )
                order_tag = "grind_x_derisk"
                for grind_entry_id in grind_x_buy_orders:
                    order_tag += " " + str(grind_entry_id)
                if has_order_tags:
                    return -ft_sell_amount, order_tag
                else:
                    return -ft_sell_amount

        return None

    def long_grind_entry_v2(self, last_candle: Series, previous_candle: Series, slice_profit: float) -> float:
        if (
                (last_candle["enter_long"] == True)
                or (
                (last_candle["RSI_14"] < 46.0)
                and (last_candle["RSI_3"] > 10.0)
                and (last_candle["RSI_3_15m"] > 15.0)
                and (last_candle["RSI_3_1h"] > 15.0)
                and (last_candle["RSI_3_4h"] > 15.0)
                and (last_candle["ROC_2_1h"] > -10.0)
                and (last_candle["ROC_2_4h"] > -10.0)
                and (last_candle["ROC_2_1d"] > -10.0)
                and (last_candle["ROC_9_1h"] > -25.0)
                and (last_candle["ROC_9_4h"] > -25.0)
                and (last_candle["ROC_9_1d"] > -25.0)
                and (last_candle["AROONU_14"] < 25.0)
                and (last_candle["close"] > (last_candle["close_max_48"] * 0.90))
                and (last_candle["close"] > (last_candle["high_max_6_1h"] * 0.85))
                and (last_candle["close"] > (last_candle["high_max_12_1h"] * 0.80))
                and (last_candle["close"] < (last_candle["low_min_24_4h"] * 1.20))
                and (last_candle["close"] < (last_candle["EMA_16"] * 0.968))
        )
                or (
                (last_candle["RSI_14"] < 36.0)
                and (last_candle["RSI_3"] > 5.0)
                and (last_candle["RSI_3_15m"] > 15.0)
                and (last_candle["RSI_3_1h"] > 15.0)
                and (last_candle["RSI_3_4h"] > 15.0)
                and (last_candle["ROC_2_1h"] > -10.0)
                and (last_candle["ROC_2_4h"] > -10.0)
                and (last_candle["ROC_9_1h"] > -10.0)
                and (last_candle["ROC_9_4h"] > -10.0)
                and (last_candle["ROC_9_1d"] > -30.0)
                and (last_candle["STOCHRSIk_14_14_3_3"] < 50.0)
                and (last_candle["EMA_26"] > last_candle["EMA_12"])
                and ((last_candle["EMA_26"] - last_candle["EMA_12"]) > (last_candle["open"] * 0.020))
                and ((previous_candle["EMA_26"] - previous_candle["EMA_12"]) > (last_candle["open"] / 100.0))
        )
                or (
                (last_candle["RSI_14"] < 36.0)
                and (last_candle["RSI_3"] > 10.0)
                and (last_candle["RSI_3_15m"] > 10.0)
                and (last_candle["RSI_3_1h"] > 10.0)
                and (last_candle["RSI_3_4h"] > 10.0)
                and (last_candle["RSI_3_1d"] > 10.0)
                and (last_candle["ROC_2_1h"] > -5.0)
                and (last_candle["ROC_2_4h"] > -5.0)
                and (last_candle["ROC_2_1d"] > -5.0)
                and (last_candle["ROC_9_1h"] > -10.0)
                and (last_candle["ROC_9_4h"] > -10.0)
                and (last_candle["ROC_9_1d"] > -10.0)
                and (last_candle["AROONU_14_15m"] < 25.0)
                and (last_candle["close"] > (last_candle["close_max_48"] * 0.90))
                and (last_candle["close"] > (last_candle["high_max_6_1h"] * 0.85))
                and (last_candle["close"] > (last_candle["high_max_12_1h"] * 0.80))
                and (last_candle["close"] < (last_candle["EMA_12"] * 0.980))
        )
                or (
                (last_candle["RSI_14"] < 36.0)
                and (last_candle["RSI_3"] > 10.0)
                and (last_candle["RSI_3_15m"] > 10.0)
                and (last_candle["RSI_3_1h"] > 10.0)
                and (last_candle["RSI_3_4h"] > 10.0)
                and (last_candle["RSI_3_1d"] > 10.0)
                and (last_candle["ROC_2_1h"] > -10.0)
                and (last_candle["ROC_2_4h"] > -10.0)
                and (last_candle["ROC_2_1d"] > -10.0)
                and (last_candle["AROONU_14"] < 25.0)
                and (last_candle["close"] > (last_candle["close_max_48"] * 0.90))
                and (last_candle["close"] > (last_candle["high_max_6_1h"] * 0.85))
                and (last_candle["close"] > (last_candle["high_max_12_1h"] * 0.80))
                and (last_candle["close"] < (last_candle["EMA_26"] * 0.962))
                and (last_candle["close"] < (last_candle["BBL_20_2.0"] * 0.999))
        )
                or (
                (last_candle["RSI_14"] < 35.0)
                and (last_candle["RSI_3"] > 10.0)
                and (last_candle["RSI_3_15m"] > 10.0)
                and (last_candle["RSI_3_1h"] > 10.0)
                and (last_candle["RSI_3_4h"] > 10.0)
                and (last_candle["ROC_2_1h"] > -10.0)
                and (last_candle["ROC_2_4h"] > -10.0)
                and (last_candle["ROC_2_1d"] > -10.0)
                and (last_candle["ROC_9_1h"] > -10.0)
                and (last_candle["ROC_9_4h"] > -10.0)
                and (last_candle["AROONU_14"] < 25.0)
                and (last_candle["close"] < (last_candle["low_min_12_4h"] * 1.25))
                and (last_candle["close"] < (last_candle["EMA_9"] * 0.968))
                and (last_candle["close"] < (last_candle["EMA_20"] * 0.980))
        )
                or (
                (last_candle["RSI_14"] > 35.0)
                and (last_candle["RSI_3"] > 10.0)
                and (last_candle["RSI_3"] < 40.0)
                and (last_candle["RSI_3_15m"] > 15.0)
                and (last_candle["ROC_2_1h"] > -5.0)
                and (last_candle["ROC_2_4h"] > -5.0)
                and (last_candle["ROC_9_1h"] > -10.0)
                and (last_candle["ROC_9_4h"] > -10.0)
                and (last_candle["AROONU_14"] < 25.0)
                and (last_candle["RSI_20"] < previous_candle["RSI_20"])
                and (last_candle["close"] < (last_candle["SMA_16"] * 0.955))
        )
                or (
                (last_candle["RSI_3"] > 5.0)
                and (last_candle["RSI_3_15m"] > 10.0)
                and (last_candle["RSI_3_1h"] > 10.0)
                and (last_candle["RSI_3_4h"] > 10.0)
                and (last_candle["ROC_2_1h"] > -5.0)
                and (last_candle["ROC_2_4h"] > -5.0)
                and (last_candle["ROC_9_1h"] > -5.0)
                and (last_candle["ROC_9_4h"] > -5.0)
                and (last_candle["WILLR_14"] < -50.0)
                and (last_candle["STOCHRSIk_14_14_3_3"] < 20.0)
                and (last_candle["WILLR_84_1h"] < -70.0)
                and (last_candle["close"] < (last_candle["low_min_24_4h"] * 1.30))
                and (last_candle["BBB_20_2.0_1h"] > 12.0)
                and (last_candle["close_max_48"] >= (last_candle["close"] * 1.10))
        )
                or (
                (last_candle["RSI_3"] < 30.0)
                and (last_candle["RSI_3"] > 5.0)
                and (last_candle["RSI_3_15m"] > 5.0)
                and (last_candle["RSI_3_1h"] > 10.0)
                and (last_candle["RSI_3_4h"] > 10.0)
                and (last_candle["ROC_9_1d"] > -30.0)
                and (last_candle["EMA_26"] > last_candle["EMA_12"])
                and ((last_candle["EMA_26"] - last_candle["EMA_12"]) > (last_candle["open"] * 0.034))
                and ((previous_candle["EMA_26"] - previous_candle["EMA_12"]) > (last_candle["open"] / 100.0))
        )
                or (
                (last_candle["RSI_3"] > 5.0)
                and (last_candle["RSI_3_15m"] > 25.0)
                and (last_candle["RSI_3_1h"] > 30.0)
                and (last_candle["close"] < (last_candle["high_max_24_4h"] * 0.90))
                and (last_candle["close"] < (last_candle["close_max_48"] * 0.90))
                and (last_candle["close"] > (last_candle["close_min_12"] * 1.08))
        )
                or (
                (last_candle["RSI_3"] > 5.0)
                and (last_candle["RSI_3_15m"] > 5.0)
                and (last_candle["STOCHRSIk_14_14_3_3"] < 20.0)
                and (last_candle["RSI_14"] < (last_candle["RSI_14_1h"] - 45.0))
        )
                or (
                (last_candle["RSI_3"] > 10.0)
                and (last_candle["RSI_3_15m"] > 10.0)
                and (last_candle["RSI_3_1h"] > 10.0)
                and (last_candle["RSI_3_4h"] > 10.0)
                and (last_candle["RSI_3_1d"] > 10.0)
                and (last_candle["STOCHRSIk_14_14_3_3"] < 20.0)
                and (last_candle["close"] < (last_candle["SMA_30"] * 0.978))
                and (last_candle["close"] < (last_candle["BBL_20_2.0"] * 0.999))
        )
                or (
                (last_candle["RSI_14"] < 36.0)
                and (last_candle["RSI_3"] > 5.0)
                and (last_candle["RSI_3_15m"] > 10.0)
                and (last_candle["RSI_3_1h"] > 10.0)
                and (last_candle["RSI_3_4h"] > 10.0)
                and (last_candle["RSI_3_1d"] > 10.0)
                and (last_candle["STOCHRSIk_14_14_3_3"] < 30.0)
                and (last_candle["close"] > (last_candle["close_max_48"] * 0.85))
                and (last_candle["close"] > (last_candle["high_max_6_1h"] * 0.80))
                and (last_candle["close"] > (last_candle["high_max_12_1h"] * 0.75))
                and (last_candle["close"] < (last_candle["low_min_12_4h"] * 1.25))
                and (last_candle["EMA_26"] > last_candle["EMA_12"])
                and ((last_candle["EMA_26"] - last_candle["EMA_12"]) > (last_candle["open"] * 0.018))
                and ((previous_candle["EMA_26"] - previous_candle["EMA_12"]) > (last_candle["open"] / 100.0))
        )
                or (
                (last_candle["RSI_3"] > 5.0)
                and (previous_candle["SMA_9"] < previous_candle["SMA_21"])
                and (last_candle["SMA_9"] > last_candle["SMA_21"])
                and (last_candle["close"] < (last_candle["EMA_100"] * 0.984))
                and (last_candle["RSI_3_1h"] > 20.0)
                and (last_candle["RSI_3_4h"] > 20.0)
        )
                or (
                (slice_profit < -0.12)
                and (last_candle["RSI_3"] > 5.0)
                and (last_candle["RSI_3_15m"] > 10.0)
                and (last_candle["RSI_14"] < 40.0)
                and (last_candle["AROONU_14"] < 25.0)
                and (last_candle["AROONU_14_15m"] < 30.0)
                and (last_candle["STOCHRSIk_14_14_3_3"] < 20.0)
                and (last_candle["STOCHRSIk_14_14_3_3_15m"] < 30.0)
                and (last_candle["RSI_14_1h"] < 50.0)
                and (last_candle["RSI_14_4h"] < 50.0)
        )
                or (
                (last_candle["RSI_14"] < 36.0)
                and (last_candle["RSI_3"] > 5.0)
                and (last_candle["RSI_3_1h"] > 10.0)
                and (last_candle["RSI_3_4h"] > 10.0)
                and (last_candle["close"] < (last_candle["EMA_12"] * 0.999))
                and (last_candle["close"] < (last_candle["BBL_20_2.0"] * 0.996))
        )
        ):
            return True

        return False

    def long_grind_exit_v2(self, last_candle: Series, previous_candle: Series, slice_profit: float) -> float:
        if (
                (last_candle["RSI_3"] > 99.0)
                or (last_candle["RSI_14"] > 70.0)
                or (last_candle["WILLR_14"] > -0.1)
                or (last_candle["STOCHRSIk_14_14_3_3"] > 95.0)
                or (last_candle["close"] > (last_candle["BBU_20_2.0"] * 1.01))
                or ((last_candle["RSI_3"] > 90.0) and (last_candle["RSI_14"] < 50.0))
        ):
            return True

        return False

    ###############################################################################################

    # SHORT EXIT FUNCTIONS STARTS HERE

    ###############################################################################################
    # Short Exit Grind
    # ---------------------------------------------------------------------------------------------
    def short_exit_grind(
            self,
            profit_init_ratio: float,
            last_candle,
            previous_candle_1,
    ) -> tuple:
        if profit_init_ratio > self.grind_v2_profit_exit_threshold:
            return True, f"exit_{self.short_grind_mode_name}_g"

        #  Here ends exit signal conditions for short_exit_grind
        return False, None

    ###############################################################################################

    # SHORT GRIND FUNCTIONS STARTS HERE

    ###############################################################################################

    # Short Grinding Adjust Trade Position v2
    # ---------------------------------------------------------------------------------------------
    def short_grind_adjust_trade_position_v2(
            self,
            trade: Trade,
            enter_tags,
            current_time: datetime,
            current_rate: float,
            current_profit: float,
            min_stake: Optional[float],
            max_stake: float,
            current_entry_rate: float,
            current_exit_rate: float,
            current_entry_profit: float,
            current_exit_profit: float,
            **kwargs,
    ):
        is_backtest = self.is_backtest_mode()
        min_stake = self.correct_min_stake(min_stake)
        df, _ = self.dp.get_analyzed_dataframe(trade.pair, self.timeframe)
        if len(df) < 2:
            return None
        last_candle = df.iloc[-1].squeeze()
        previous_candle = df.iloc[-2].squeeze()

        # we already waiting for an order to get filled
        if trade.has_open_orders:
            return None

        filled_orders = trade.select_filled_orders()
        filled_entries = trade.select_filled_orders(trade.entry_side)
        filled_exits = trade.select_filled_orders(trade.exit_side)

        exit_rate = current_rate
        if self.dp.runmode.value in ("live", "dry_run"):
            ticker = self.dp.ticker(trade.pair)
            if ("bid" in ticker) and ("ask" in ticker):
                if trade.is_short:
                    if self.exit_price_side in ["ask", "other"]:
                        if ticker["ask"] is not None:
                            exit_rate = ticker["ask"]
                else:
                    if self.exit_price_side in ["bid", "other"]:
                        if ticker["bid"] is not None:
                            exit_rate = ticker["bid"]

        profit_stake, profit_ratio, profit_current_stake_ratio, profit_init_ratio = self.calc_total_profit(
            trade, filled_entries, filled_exits, exit_rate
        )

        current_stake_amount = trade.amount * exit_rate
        stake_cost = trade.get_custom_data(key="stake_cost")
        slice_amount = stake_cost if stake_cost else filled_entries[0].cost
        slice_profit = (
                               exit_rate - filled_orders[-1].safe_price) / filled_orders[-1].safe_price

        has_order_tags = False
        if hasattr(filled_orders[0], "ft_order_tag"):
            has_order_tags = True

        fee_open_rate = trade.fee_open if self.custom_fee_open_rate is None else self.custom_fee_open_rate
        fee_close_rate = trade.fee_close if self.custom_fee_close_rate is None else self.custom_fee_close_rate

        grind_1_max_sub_grinds = 0
        grind_1_stakes = (
            self.grinding_v2_grind_1_stakes_futures.copy()
            if self.is_futures_mode
            else self.grinding_v2_grind_1_stakes_spot.copy()
        )
        grind_1_sub_thresholds = (
            self.grinding_v2_grind_1_thresholds_futures
            if self.is_futures_mode
            else self.grinding_v2_grind_1_thresholds_spot
        )
        if (slice_amount * grind_1_stakes[0] / (trade.leverage if self.is_futures_mode else 1.0)) < min_stake:
            multi = min_stake / slice_amount / \
                    grind_1_stakes[0] * trade.leverage
            for i, _ in enumerate(grind_1_stakes):
                grind_1_stakes[i] *= multi
        grind_1_max_sub_grinds = len(grind_1_stakes)
        grind_1_derisk_grinds = (
            self.grinding_v2_grind_1_derisk_futures if self.is_futures_mode else self.grinding_v2_grind_1_derisk_spot
        )
        grind_1_profit_threshold = (
            self.grinding_v2_grind_1_profit_threshold_futures
            if self.is_futures_mode
            else self.grinding_v2_grind_1_profit_threshold_spot
        )

        grind_2_max_sub_grinds = 0
        grind_2_stakes = (
            self.grinding_v2_grind_2_stakes_futures.copy()
            if self.is_futures_mode
            else self.grinding_v2_grind_2_stakes_spot.copy()
        )
        grind_2_sub_thresholds = (
            self.grinding_v2_grind_2_thresholds_futures
            if self.is_futures_mode
            else self.grinding_v2_grind_2_thresholds_spot
        )
        if (slice_amount * grind_2_stakes[0] / (trade.leverage if self.is_futures_mode else 1.0)) < min_stake:
            multi = min_stake / slice_amount / \
                    grind_2_stakes[0] * trade.leverage
            for i, _ in enumerate(grind_2_stakes):
                grind_2_stakes[i] *= multi
        grind_2_max_sub_grinds = len(grind_2_stakes)
        grind_2_derisk_grinds = (
            self.grinding_v2_grind_2_derisk_futures if self.is_futures_mode else self.grinding_v2_grind_2_derisk_spot
        )
        grind_2_profit_threshold = (
            self.grinding_v2_grind_2_profit_threshold_futures
            if self.is_futures_mode
            else self.grinding_v2_grind_2_profit_threshold_spot
        )

        grind_3_max_sub_grinds = 0
        grind_3_stakes = (
            self.grinding_v2_grind_3_stakes_futures.copy()
            if self.is_futures_mode
            else self.grinding_v2_grind_3_stakes_spot.copy()
        )
        grind_3_sub_thresholds = (
            self.grinding_v2_grind_3_thresholds_futures
            if self.is_futures_mode
            else self.grinding_v2_grind_3_thresholds_spot
        )
        if (slice_amount * grind_3_stakes[0] / (trade.leverage if self.is_futures_mode else 1.0)) < min_stake:
            multi = min_stake / slice_amount / \
                    grind_3_stakes[0] * trade.leverage
            for i, _ in enumerate(grind_3_stakes):
                grind_3_stakes[i] *= multi
        grind_3_max_sub_grinds = len(grind_3_stakes)
        grind_3_derisk_grinds = (
            self.grinding_v2_grind_3_derisk_futures if self.is_futures_mode else self.grinding_v2_grind_3_derisk_spot
        )
        grind_3_profit_threshold = (
            self.grinding_v2_grind_3_profit_threshold_futures
            if self.is_futures_mode
            else self.grinding_v2_grind_3_profit_threshold_spot
        )

        grind_x_max_sub_grinds = 0
        grind_x_stakes = (
            self.grinding_v2_grind_x_stakes_futures.copy()
            if self.is_futures_mode
            else self.grinding_v2_grind_x_stakes_spot.copy()
        )
        grind_x_sub_thresholds = (
            self.grinding_v2_grind_x_thresholds_futures
            if self.is_futures_mode
            else self.grinding_v2_grind_x_thresholds_spot
        )
        if (slice_amount * grind_x_stakes[0] / (trade.leverage if self.is_futures_mode else 1.0)) < min_stake:
            multi = min_stake / slice_amount / \
                    grind_x_stakes[0] * trade.leverage
            for i, _ in enumerate(grind_x_stakes):
                grind_x_stakes[i] *= multi
        grind_x_max_sub_grinds = len(grind_x_stakes)
        grind_x_derisk_grinds = (
            self.grinding_v2_grind_x_derisk_futures if self.is_futures_mode else self.grinding_v2_grind_x_derisk_spot
        )
        grind_x_profit_threshold = (
            self.grinding_v2_grind_x_profit_threshold_futures
            if self.is_futures_mode
            else self.grinding_v2_grind_x_profit_threshold_spot
        )

        grind_1_sub_grind_count = 0
        grind_1_total_amount = 0.0
        grind_1_total_cost = 0.0
        grind_1_current_open_rate = 0.0
        grind_1_current_grind_stake = 0.0
        grind_1_current_grind_stake_profit = 0.0
        grind_1_is_exit_found = False
        grind_1_found = False
        grind_1_buy_orders = []
        grind_1_orders = []
        grind_1_distance_ratio = 0.0
        grind_2_sub_grind_count = 0
        grind_2_total_amount = 0.0
        grind_2_total_cost = 0.0
        grind_2_current_open_rate = 0.0
        grind_2_current_grind_stake = 0.0
        grind_2_current_grind_stake_profit = 0.0
        grind_2_is_exit_found = False
        grind_2_found = False
        grind_2_buy_orders = []
        grind_2_orders = []
        grind_2_distance_ratio = 0.0
        grind_3_sub_grind_count = 0
        grind_3_total_amount = 0.0
        grind_3_total_cost = 0.0
        grind_3_current_open_rate = 0.0
        grind_3_current_grind_stake = 0.0
        grind_3_current_grind_stake_profit = 0.0
        grind_3_is_exit_found = False
        grind_3_found = False
        grind_3_buy_orders = []
        grind_3_orders = []
        grind_3_distance_ratio = 0.0
        grind_x_sub_grind_count = 0
        grind_x_total_amount = 0.0
        grind_x_total_cost = 0.0
        grind_x_current_open_rate = 0.0
        grind_x_current_grind_stake = 0.0
        grind_x_current_grind_stake_profit = 0.0
        grind_x_is_exit_found = False
        grind_x_found = False
        grind_x_buy_orders = []
        grind_x_orders = []
        grind_x_distance_ratio = 0.0
        for order in reversed(filled_orders):
            if order.ft_order_side == "sell":
                order_tag = ""
                if has_order_tags:
                    if order.ft_order_tag is not None:
                        order_tag = order.ft_order_tag
                if not grind_1_is_exit_found and order_tag == "grind_1_entry":
                    grind_1_sub_grind_count += 1
                    grind_1_total_amount += order.safe_filled
                    grind_1_total_cost += order.safe_filled * order.safe_price
                    grind_1_buy_orders.append(order.id)
                    grind_1_orders.append(order)
                    if not grind_1_found:
                        grind_1_distance_ratio = (
                                                         exit_rate - order.safe_price) / order.safe_price
                        grind_1_found = True
                elif not grind_2_is_exit_found and order_tag == "grind_2_entry":
                    grind_2_sub_grind_count += 1
                    grind_2_total_amount += order.safe_filled
                    grind_2_total_cost += order.safe_filled * order.safe_price
                    grind_2_buy_orders.append(order.id)
                    grind_2_orders.append(order)
                    if not grind_2_found:
                        grind_2_distance_ratio = (
                                                         exit_rate - order.safe_price) / order.safe_price
                        grind_2_found = True
                elif not grind_3_is_exit_found and order_tag == "grind_3_entry":
                    grind_3_sub_grind_count += 1
                    grind_3_total_amount += order.safe_filled
                    grind_3_total_cost += order.safe_filled * order.safe_price
                    grind_3_buy_orders.append(order.id)
                    grind_3_orders.append(order)
                    if not grind_3_found:
                        grind_3_distance_ratio = (
                                                         exit_rate - order.safe_price) / order.safe_price
                        grind_3_found = True
                elif not grind_x_is_exit_found and order_tag not in ["grind_1_entry", "grind_2_entry", "grind_3_entry"]:
                    grind_x_sub_grind_count += 1
                    grind_x_total_amount += order.safe_filled
                    grind_x_total_cost += order.safe_filled * order.safe_price
                    grind_x_buy_orders.append(order.id)
                    grind_x_orders.append(order)
                    if not grind_x_found:
                        grind_x_distance_ratio = (
                                                         exit_rate - order.safe_price) / order.safe_price
                        grind_x_found = True
            elif order.ft_order_side == "buy":
                order_tag = ""
                if has_order_tags:
                    if order.ft_order_tag is not None:
                        sell_order_tag = order.ft_order_tag
                        order_mode = sell_order_tag.split(" ", 1)
                        if len(order_mode) > 0:
                            order_tag = order_mode[0]
                if not grind_1_is_exit_found and order_tag in ["grind_1_exit", "grind_1_derisk"]:
                    grind_1_is_exit_found = True
                elif not grind_2_is_exit_found and order_tag in ["grind_2_exit", "grind_2_derisk"]:
                    grind_2_is_exit_found = True
                elif not grind_3_is_exit_found and order_tag in ["grind_3_exit", "grind_3_derisk"]:
                    grind_3_is_exit_found = True
                elif not grind_x_is_exit_found and order_tag in ["grind_x_exit", "grind_x_derisk", "derisk_v2", "grind_v2_exit"]:
                    grind_x_is_exit_found = True

        if grind_1_sub_grind_count > 0:
            grind_1_current_open_rate = grind_1_total_cost / grind_1_total_amount
            grind_1_current_grind_stake = grind_1_total_amount * \
                                          exit_rate * (1 - trade.fee_close)
            grind_1_current_grind_stake_profit = grind_1_current_grind_stake - grind_1_total_cost
        if grind_2_sub_grind_count > 0:
            grind_2_current_open_rate = grind_2_total_cost / grind_2_total_amount
            grind_2_current_grind_stake = grind_2_total_amount * \
                                          exit_rate * (1 - trade.fee_close)
            grind_2_current_grind_stake_profit = grind_2_current_grind_stake - grind_2_total_cost
        if grind_3_sub_grind_count > 0:
            grind_3_current_open_rate = grind_3_total_cost / grind_3_total_amount
            grind_3_current_grind_stake = grind_3_total_amount * \
                                          exit_rate * (1 - trade.fee_close)
            grind_3_current_grind_stake_profit = grind_3_current_grind_stake - grind_3_total_cost
        if grind_x_sub_grind_count > 0:
            grind_x_current_open_rate = grind_x_total_cost / grind_x_total_amount
            grind_x_current_grind_stake = grind_x_total_amount * \
                                          exit_rate * (1 - trade.fee_close)
            grind_x_current_grind_stake_profit = grind_x_current_grind_stake - grind_x_total_cost

        # all grinds
        num_open_grinds = grind_1_sub_grind_count + grind_2_sub_grind_count + grind_3_sub_grind_count + grind_x_sub_grind_count

        # not reached the max allowed stake for all grinds
        is_not_trade_max_stake = current_stake_amount < (
                slice_amount * self.grinding_v2_max_stake)

        is_short_extra_checks_entry = (
                (current_time - timedelta(minutes=5) > filled_entries[-1].order_filled_utc)
                and ((current_time - timedelta(hours=2) > filled_orders[-1].order_filled_utc) or (slice_profit > 0.06))
        )
        is_short_grind_entry = (
                self.short_grind_entry_v2(
                    last_candle, previous_candle, slice_profit)
                or (
                        (num_open_grinds == 0)
                        and (
                                (last_candle["RSI_3"] < 90.0)
                                and (last_candle["RSI_3_15m"] < 80.0)
                                and (last_candle["RSI_3_1h"] < 80.0)
                                and (last_candle["AROOND_14"] < 50.0)
                                and (last_candle["AROOND_14_15m"] < 50.0)
                        )
                )
                or (
                        self.is_futures_mode
                        and trade.liquidation_price is not None
                        and (
                                (trade.is_short and current_rate >
                                 trade.liquidation_price * 0.90)
                                or (not trade.is_short and current_rate < trade.liquidation_price * 1.10)
                        )
                        and (slice_profit > 0.03)
                        and (last_candle["RSI_3"] < 90.0)
                        and (last_candle["RSI_3_15m"] < 80.0)
                        and (last_candle["AROOND_14"] < 50.0)
                        and (last_candle["AROOND_14_15m"] < 50.0)
                )
        )

        # Grinding 1

        if (
                self.grinding_v2_grind_1_enable
                and is_short_grind_entry
                and is_short_extra_checks_entry
                and (grind_1_sub_grind_count < grind_1_max_sub_grinds)
                and (
                (grind_1_sub_grind_count == 0)
                or (-grind_1_distance_ratio < grind_1_sub_thresholds[grind_1_sub_grind_count])
        )
                and is_not_trade_max_stake
        ):
            buy_amount = slice_amount * \
                         grind_1_stakes[grind_1_sub_grind_count] / trade.leverage
            if buy_amount < (min_stake * 1.5):
                buy_amount = min_stake * 1.5
            if buy_amount > max_stake:
                return None
            self.dp.send_msg(
                self.notification_msg(
                    "grinding-entry",
                    tag="grind_1_entry",
                    pair=trade.pair,
                    rate=current_rate,
                    stake_amount=buy_amount,
                    profit_stake=profit_stake,
                    profit_ratio=profit_ratio,
                    stake_currency=self.stake_currency,
                )
            )
            log.info(
                f"Grinding entry (grind_1_entry) [{current_time}] [{trade.pair}] | Rate: {current_rate} | Stake amount: {buy_amount} | Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}%"
            )
            order_tag = "grind_1_entry"
            if has_order_tags:
                return buy_amount, order_tag
            else:
                return buy_amount

        if grind_1_sub_grind_count > 0:
            grind_profit = - \
                               (exit_rate - grind_1_current_open_rate) / \
                           grind_1_current_open_rate
            if (
                    grind_profit > (grind_1_profit_threshold +
                                    fee_open_rate + fee_close_rate)
            ) and self.short_grind_exit_v2(last_candle, previous_candle, slice_profit):
                sell_amount = grind_1_total_amount * exit_rate / trade.leverage
                if ((current_stake_amount / trade.leverage) - sell_amount) < (min_stake * 1.55):
                    sell_amount = (trade.amount * exit_rate /
                                   trade.leverage) - (min_stake * 1.55)
                ft_sell_amount = sell_amount * trade.leverage * \
                                 (trade.stake_amount / trade.amount) / exit_rate
                if sell_amount > min_stake and ft_sell_amount > min_stake:
                    self.dp.send_msg(
                        self.notification_msg(
                            "grinding-exit",
                            tag="grind_1_exit",
                            pair=trade.pair,
                            rate=exit_rate,
                            stake_amount=sell_amount,
                            profit_stake=profit_stake,
                            profit_ratio=profit_ratio,
                            stake_currency=self.stake_currency,
                            grind_profit_stake=grind_profit * sell_amount * trade.leverage,
                            grind_profit_pct=grind_profit,
                            coin_amount=grind_1_total_amount,
                        )
                    )
                    log.info(
                        f"Grinding exit (grind_1_exit) [{current_time}] [{trade.pair}] | Rate: {exit_rate} |"
                        f" Stake amount: {sell_amount} | Coin amount: {grind_1_total_amount} | "
                        f"Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}% | "
                        f"Grind profit: {(grind_profit * 100.0):.2f}% ({grind_profit * sell_amount * trade.leverage} {self.stake_currency})"
                    )
                    order_tag = "grind_1_exit"
                    for grind_entry_id in grind_1_buy_orders:
                        order_tag += " " + str(grind_entry_id)
                    if has_order_tags:
                        return -ft_sell_amount, order_tag
                    else:
                        return -ft_sell_amount

        if (
                self.grinding_v2_grind_1_use_derisk
                and (grind_1_sub_grind_count > 0)
                and ((-(exit_rate - grind_1_current_open_rate) / grind_1_current_open_rate) < grind_1_derisk_grinds)
                and (grind_1_orders[-1].order_date_utc.replace(tzinfo=None) >= datetime(2025, 8, 3) or is_backtest)
        ):
            sell_amount = grind_1_total_amount * exit_rate / trade.leverage
            if ((current_stake_amount / trade.leverage) - sell_amount) < (min_stake * 1.55):
                sell_amount = (trade.amount * exit_rate /
                               trade.leverage) - (min_stake * 1.55)
            ft_sell_amount = sell_amount * trade.leverage * \
                             (trade.stake_amount / trade.amount) / exit_rate
            if sell_amount > min_stake and ft_sell_amount > min_stake:
                grind_profit = 0.0
                if grind_1_current_open_rate > 0.0:
                    grind_profit = (
                        -((exit_rate - grind_1_current_open_rate) /
                          grind_1_current_open_rate)
                        if grind_1_is_exit_found
                        else profit_ratio
                    )
                self.dp.send_msg(
                    self.notification_msg(
                        "grinding-derisk",
                        tag="grind_1_derisk",
                        pair=trade.pair,
                        rate=exit_rate,
                        stake_amount=sell_amount,
                        profit_stake=profit_stake,
                        profit_ratio=profit_ratio,
                        stake_currency=self.stake_currency,
                        grind_profit_stake=grind_profit * sell_amount * trade.leverage,
                        grind_profit_pct=grind_profit,
                        coin_amount=grind_1_total_amount,
                    )
                )
                log.info(
                    f"Grinding de-risk (grind_1_derisk) [{current_time}] [{trade.pair}] | Rate: {exit_rate} |"
                    f" Stake amount: {sell_amount} | Coin amount: {grind_1_total_amount} | "
                    f"Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}% | Grind profit: {(grind_profit * 100.0):.2f}%"
                )
                order_tag = "grind_1_derisk"
                for grind_entry_id in grind_1_buy_orders:
                    order_tag += " " + str(grind_entry_id)
                if has_order_tags:
                    return -ft_sell_amount, order_tag
                else:
                    return -ft_sell_amount

        # Grinding 2

        if (
                self.grinding_v2_grind_2_enable
                and is_short_grind_entry
                and is_short_extra_checks_entry
                and (grind_2_sub_grind_count < grind_2_max_sub_grinds)
                and (
                (grind_2_sub_grind_count == 0) and (grind_1_sub_grind_count > 1)
                or (-grind_2_distance_ratio < grind_2_sub_thresholds[grind_2_sub_grind_count])
        )
                and is_not_trade_max_stake
        ):
            buy_amount = slice_amount * \
                         grind_2_stakes[grind_2_sub_grind_count] / trade.leverage
            if buy_amount < (min_stake * 1.5):
                buy_amount = min_stake * 1.5
            if buy_amount > max_stake:
                return None
            self.dp.send_msg(
                self.notification_msg(
                    "grinding-entry",
                    tag="grind_2_entry",
                    pair=trade.pair,
                    rate=current_rate,
                    stake_amount=buy_amount,
                    profit_stake=profit_stake,
                    profit_ratio=profit_ratio,
                    stake_currency=self.stake_currency,
                )
            )
            log.info(
                f"Grinding entry (grind_2_entry) [{current_time}] [{trade.pair}] | Rate: {current_rate} | Stake amount: {buy_amount} | Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}%"
            )
            order_tag = "grind_2_entry"
            if has_order_tags:
                return buy_amount, order_tag
            else:
                return buy_amount

        if grind_2_sub_grind_count > 0:
            grind_profit = - \
                               (exit_rate - grind_2_current_open_rate) / \
                           grind_2_current_open_rate
            if (
                    grind_profit > (grind_2_profit_threshold +
                                    fee_open_rate + fee_close_rate)
            ) and self.short_grind_exit_v2(last_candle, previous_candle, slice_profit):
                sell_amount = grind_2_total_amount * exit_rate / trade.leverage
                if ((current_stake_amount / trade.leverage) - sell_amount) < (min_stake * 1.55):
                    sell_amount = (trade.amount * exit_rate /
                                   trade.leverage) - (min_stake * 1.55)
                ft_sell_amount = sell_amount * trade.leverage * \
                                 (trade.stake_amount / trade.amount) / exit_rate
                if sell_amount > min_stake and ft_sell_amount > min_stake:
                    self.dp.send_msg(
                        self.notification_msg(
                            "grinding-exit",
                            tag="grind_2_exit",
                            pair=trade.pair,
                            rate=exit_rate,
                            stake_amount=sell_amount,
                            profit_stake=profit_stake,
                            profit_ratio=profit_ratio,
                            stake_currency=self.stake_currency,
                            grind_profit_stake=grind_profit * sell_amount * trade.leverage,
                            grind_profit_pct=grind_profit,
                            coin_amount=grind_2_total_amount,
                        )
                    )
                    log.info(
                        f"Grinding exit (grind_2_exit) [{current_time}] [{trade.pair}] | Rate: {exit_rate} |"
                        f" Stake amount: {sell_amount} | Coin amount: {grind_2_total_amount} |"
                        f" Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}% |"
                        f" Grind profit: {(grind_profit * 100.0):.2f}% ({grind_profit * sell_amount * trade.leverage} {self.stake_currency})"
                    )
                    order_tag = "grind_2_exit"
                    for grind_entry_id in grind_2_buy_orders:
                        order_tag += " " + str(grind_entry_id)
                    if has_order_tags:
                        return -ft_sell_amount, order_tag
                    else:
                        return -ft_sell_amount

        if (
                self.grinding_v2_grind_2_use_derisk
                and (grind_2_sub_grind_count > 0)
                and ((-(exit_rate - grind_2_current_open_rate) / grind_2_current_open_rate) < grind_2_derisk_grinds)
                and (grind_2_orders[-1].order_date_utc.replace(tzinfo=None) >= datetime(2025, 8, 3) or is_backtest)
        ):
            sell_amount = grind_2_total_amount * exit_rate / trade.leverage
            if ((current_stake_amount / trade.leverage) - sell_amount) < (min_stake * 1.55):
                sell_amount = (trade.amount * exit_rate /
                               trade.leverage) - (min_stake * 1.55)
            ft_sell_amount = sell_amount * trade.leverage * \
                             (trade.stake_amount / trade.amount) / exit_rate
            if sell_amount > min_stake and ft_sell_amount > min_stake:
                grind_profit = 0.0
                if grind_2_current_open_rate > 0.0:
                    grind_profit = (
                        -((exit_rate - grind_2_current_open_rate) /
                          grind_2_current_open_rate)
                        if grind_2_is_exit_found
                        else profit_ratio
                    )
                self.dp.send_msg(
                    self.notification_msg(
                        "grinding-derisk",
                        tag="grind_2_derisk",
                        pair=trade.pair,
                        rate=exit_rate,
                        stake_amount=sell_amount,
                        profit_stake=profit_stake,
                        profit_ratio=profit_ratio,
                        stake_currency=self.stake_currency,
                        grind_profit_stake=grind_profit * sell_amount * trade.leverage,
                        grind_profit_pct=grind_profit,
                        coin_amount=grind_2_total_amount,
                    )
                )
                log.info(
                    f"Grinding de-risk (grind_2_derisk) [{current_time}] [{trade.pair}] | Rate: {exit_rate} | Stake amount: {sell_amount} | Coin amount: {grind_2_total_amount} | Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}% | Grind profit: {(grind_profit * 100.0):.2f}%"
                )
                order_tag = "grind_2_derisk"
                for grind_entry_id in grind_2_buy_orders:
                    order_tag += " " + str(grind_entry_id)
                if has_order_tags:
                    return -ft_sell_amount, order_tag
                else:
                    return -ft_sell_amount

        # Grinding 3 Entry
        if (
                self.grinding_v2_grind_3_enable
                and is_short_grind_entry
                and is_short_extra_checks_entry
                and (grind_3_sub_grind_count < grind_3_max_sub_grinds)
                and (
                (grind_3_sub_grind_count == 0) and (grind_2_sub_grind_count > 1)
                or (-grind_3_distance_ratio < grind_3_sub_thresholds[grind_3_sub_grind_count])
        )
                and is_not_trade_max_stake
        ):
            buy_amount = slice_amount * grind_3_stakes[grind_3_sub_grind_count] / trade.leverage
            if buy_amount < (min_stake * 1.5):
                buy_amount = min_stake * 1.5
            if buy_amount > max_stake:
                return None
            self.dp.send_msg(
                self.notification_msg(
                    "grinding-entry",
                    tag="grind_3_entry",
                    pair=trade.pair,
                    rate=current_rate,
                    stake_amount=buy_amount,
                    profit_stake=profit_stake,
                    profit_ratio=profit_ratio,
                    stake_currency=self.stake_currency,
                )
            )
            log.info(
                f"Grinding entry (grind_3_entry) [{current_time}] [{trade.pair}] | Rate: {current_rate} | Stake amount: {buy_amount} | Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}%"
            )
            order_tag = "grind_3_entry"
            if has_order_tags:
                return buy_amount, order_tag
            else:
                return buy_amount

        # Grinding 3 Exit
        if grind_3_sub_grind_count > 0:
            grind_profit = - (exit_rate - grind_3_current_open_rate) / grind_3_current_open_rate
            if (
                    grind_profit > (grind_3_profit_threshold +
                                    fee_open_rate + fee_close_rate)
            ) and self.short_grind_exit_v2(last_candle, previous_candle, slice_profit):
                sell_amount = grind_3_total_amount * exit_rate / trade.leverage
                if ((current_stake_amount / trade.leverage) - sell_amount) < (min_stake * 1.55):
                    sell_amount = (trade.amount * exit_rate /
                                   trade.leverage) - (min_stake * 1.55)
                ft_sell_amount = sell_amount * trade.leverage * \
                                 (trade.stake_amount / trade.amount) / exit_rate
                if sell_amount > min_stake and ft_sell_amount > min_stake:
                    self.dp.send_msg(
                        self.notification_msg(
                            "grinding-exit",
                            tag="grind_3_exit",
                            pair=trade.pair,
                            rate=exit_rate,
                            stake_amount=sell_amount,
                            profit_stake=profit_stake,
                            profit_ratio=profit_ratio,
                            stake_currency=self.stake_currency,
                            grind_profit_stake=grind_profit * sell_amount * trade.leverage,
                            grind_profit_pct=grind_profit,
                            coin_amount=grind_3_total_amount,
                        )
                    )
                    log.info(
                        f"Grinding exit (grind_3_exit) [{current_time}] [{trade.pair}] | Rate: {exit_rate} |"
                        f" Stake amount: {sell_amount} | Coin amount: {grind_3_total_amount} |"
                        f" Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}% |"
                        f" Grind profit: {(grind_profit * 100.0):.2f}% ({grind_profit * sell_amount * trade.leverage} {self.stake_currency})"
                    )
                    order_tag = "grind_3_exit"
                    for grind_entry_id in grind_3_buy_orders:
                        order_tag += " " + str(grind_entry_id)
                    if has_order_tags:
                        return -ft_sell_amount, order_tag
                    else:
                        return -ft_sell_amount

        # Grinding 3 De-risk
        if (
                self.grinding_v2_grind_3_use_derisk
                and (grind_3_sub_grind_count > 0)
                and ((-(exit_rate - grind_3_current_open_rate) / grind_3_current_open_rate) < grind_3_derisk_grinds)
                and (grind_3_orders[-1].order_date_utc.replace(tzinfo=None) >= datetime(2025, 8, 3) or is_backtest)
        ):
            sell_amount = grind_3_total_amount * exit_rate / trade.leverage
            if ((current_stake_amount / trade.leverage) - sell_amount) < (min_stake * 1.55):
                sell_amount = (trade.amount * exit_rate /
                               trade.leverage) - (min_stake * 1.55)
            ft_sell_amount = sell_amount * trade.leverage * \
                             (trade.stake_amount / trade.amount) / exit_rate
            if sell_amount > min_stake and ft_sell_amount > min_stake:
                grind_profit = 0.0
                if grind_3_current_open_rate > 0.0:
                    grind_profit = (
                        -((exit_rate - grind_3_current_open_rate) /
                          grind_3_current_open_rate)
                        if grind_3_is_exit_found
                        else profit_ratio
                    )
                self.dp.send_msg(
                    self.notification_msg(
                        "grinding-derisk",
                        tag="grind_3_derisk",
                        pair=trade.pair,
                        rate=exit_rate,
                        stake_amount=sell_amount,
                        profit_stake=profit_stake,
                        profit_ratio=profit_ratio,
                        stake_currency=self.stake_currency,
                        grind_profit_stake=grind_profit * sell_amount * trade.leverage,
                        grind_profit_pct=grind_profit,
                        coin_amount=grind_3_total_amount,
                    )
                )
                log.info(
                    f"Grinding de-risk (grind_3_derisk) [{current_time}] [{trade.pair}] | Rate: {exit_rate} | Stake amount: {sell_amount} | Coin amount: {grind_3_total_amount} | Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}% | Grind profit: {(grind_profit * 100.0):.2f}%"
                )
                order_tag = "grind_3_derisk"
                for grind_entry_id in grind_3_buy_orders:
                    order_tag += " " + str(grind_entry_id)
                if has_order_tags:
                    return -ft_sell_amount, order_tag
                else:
                    return -ft_sell_amount

        # Grinding X - Entry (for entries not matching grind_1, 2, 3)
        if (
                self.grinding_v2_grind_x_enable
                and is_short_grind_entry
                and is_short_extra_checks_entry
                and (grind_x_sub_grind_count < grind_x_max_sub_grinds)
                and grind_x_sub_grind_count != 0
                and (-grind_x_distance_ratio < grind_x_sub_thresholds[grind_x_sub_grind_count])
                and is_not_trade_max_stake
        ):
            buy_amount = slice_amount * \
                         grind_x_stakes[grind_x_sub_grind_count] / trade.leverage
            if buy_amount < (min_stake * 1.5):
                buy_amount = min_stake * 1.5
            if buy_amount > max_stake:
                return None
            self.dp.send_msg(
                self.notification_msg(
                    "grinding-entry",
                    tag="grind_x_entry",
                    pair=trade.pair,
                    rate=current_rate,
                    stake_amount=buy_amount,
                    profit_stake=profit_stake,
                    profit_ratio=profit_ratio,
                    stake_currency=self.stake_currency,
                )
            )
            log.info(
                f"Grinding entry (grind_x_entry) [{current_time}] [{trade.pair}] | Rate: {current_rate} | Stake amount: {buy_amount} | Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}%"
            )
            order_tag = "grind_x_entry"
            if has_order_tags:
                return buy_amount, order_tag
            else:
                return buy_amount

        # Grinding X - Exit
        if grind_x_sub_grind_count > 0:
            grind_profit = - \
                               (exit_rate - grind_x_current_open_rate) / \
                           grind_x_current_open_rate
            if (
                    grind_profit > (grind_x_profit_threshold +
                                    fee_open_rate + fee_close_rate)
            ) and self.short_grind_exit_v2(last_candle, previous_candle, slice_profit):
                sell_amount = grind_x_total_amount * exit_rate / trade.leverage
                if ((current_stake_amount / trade.leverage) - sell_amount) < (min_stake * 1.55):
                    sell_amount = (trade.amount * exit_rate /
                                   trade.leverage) - (min_stake * 1.55)
                ft_sell_amount = sell_amount * trade.leverage * \
                                 (trade.stake_amount / trade.amount) / exit_rate
                if sell_amount > min_stake and ft_sell_amount > min_stake:
                    self.dp.send_msg(
                        self.notification_msg(
                            "grinding-exit",
                            tag="grind_x_exit",
                            pair=trade.pair,
                            rate=exit_rate,
                            stake_amount=sell_amount,
                            profit_stake=profit_stake,
                            profit_ratio=profit_ratio,
                            stake_currency=self.stake_currency,
                            grind_profit_stake=grind_profit * sell_amount * trade.leverage,
                            grind_profit_pct=grind_profit,
                            coin_amount=grind_x_total_amount,
                        )
                    )
                    log.info(
                        f"Grinding exit (grind_x_exit) [{current_time}] [{trade.pair}] | Rate: {exit_rate} |"
                        f" Stake amount: {sell_amount} | Coin amount: {grind_x_total_amount} | "
                        f"Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}% | "
                        f"Grind profit: {(grind_profit * 100.0):.2f}% ({grind_profit * sell_amount * trade.leverage} {self.stake_currency})"
                    )
                    order_tag = "grind_x_exit"
                    for grind_entry_id in grind_x_buy_orders:
                        order_tag += " " + str(grind_entry_id)
                    if has_order_tags:
                        return -ft_sell_amount, order_tag
                    else:
                        return -ft_sell_amount

        # Grinding X De-risk
        if (
                self.grinding_v2_grind_x_use_derisk
                and (grind_x_sub_grind_count > 0)
                and ((-(exit_rate - grind_x_current_open_rate) / grind_x_current_open_rate) < grind_x_derisk_grinds)
                and (grind_x_orders[-1].order_date_utc.replace(tzinfo=None) >= datetime(2025, 8, 3) or is_backtest)
        ):
            sell_amount = grind_x_total_amount * exit_rate / trade.leverage
            if ((current_stake_amount / trade.leverage) - sell_amount) < (min_stake * 1.55):
                sell_amount = (trade.amount * exit_rate /
                               trade.leverage) - (min_stake * 1.55)
            ft_sell_amount = sell_amount * trade.leverage * \
                             (trade.stake_amount / trade.amount) / exit_rate
            if sell_amount > min_stake and ft_sell_amount > min_stake:
                grind_profit = 0.0
                if grind_x_current_open_rate > 0.0:
                    grind_profit = (
                        -((exit_rate - grind_x_current_open_rate) /
                          grind_x_current_open_rate)
                        if grind_x_is_exit_found
                        else profit_ratio
                    )
                self.dp.send_msg(
                    self.notification_msg(
                        "grinding-derisk",
                        tag="grind_x_derisk",
                        pair=trade.pair,
                        rate=exit_rate,
                        stake_amount=sell_amount,
                        profit_stake=profit_stake,
                        profit_ratio=profit_ratio,
                        stake_currency=self.stake_currency,
                        grind_profit_stake=grind_profit * sell_amount * trade.leverage,
                        grind_profit_pct=grind_profit,
                        coin_amount=grind_x_total_amount,
                    )
                )
                log.info(
                    f"Grinding de-risk (grind_x_derisk) [{current_time}] [{trade.pair}] | Rate: {exit_rate} | Stake amount: {sell_amount} | Coin amount: {grind_x_total_amount} | Profit (stake): {profit_stake} | Profit: {(profit_ratio * 100.0):.2f}% | Grind profit: {(grind_profit * 100.0):.2f}%"
                )
                order_tag = "grind_x_derisk"
                for grind_entry_id in grind_x_buy_orders:
                    order_tag += " " + str(grind_entry_id)
                if has_order_tags:
                    return -ft_sell_amount, order_tag
                else:
                    return -ft_sell_amount

        return None

    def short_grind_entry_v2(self, last_candle: Series, previous_candle: Series, slice_profit: float) -> float:
        if (
                (last_candle["enter_short"] == True)
                or (
                (last_candle["RSI_14"] > 54.0)
                and (last_candle["RSI_3"] < 90.0)
                and (last_candle["RSI_3_15m"] < 85.0)
                and (last_candle["RSI_3_1h"] < 85.0)
                and (last_candle["RSI_3_4h"] < 85.0)
                and (last_candle["ROC_2_1h"] < 10.0)
                and (last_candle["ROC_2_4h"] < 10.0)
                and (last_candle["ROC_2_1d"] < 10.0)
                and (last_candle["ROC_9_1h"] < 25.0)
                and (last_candle["ROC_9_4h"] < 25.0)
                and (last_candle["ROC_9_1d"] < 25.0)
                and (last_candle["AROOND_14"] < 25.0)
                and (last_candle["close"] < (last_candle["close_min_48"] * 1.10))
                and (last_candle["close"] < (last_candle["low_min_6_1h"] * 1.18))
                and (last_candle["close"] < (last_candle["low_min_12_1h"] * 1.25))
                and (last_candle["close"] > (last_candle["high_max_24_4h"] * 0.85))
                and (last_candle["close"] > (last_candle["EMA_16"] * 1.032))
        )
                or (
                (last_candle["RSI_14"] > 64.0)
                and (last_candle["RSI_3"] < 95.0)
                and (last_candle["RSI_3_15m"] < 85.0)
                and (last_candle["RSI_3_1h"] < 85.0)
                and (last_candle["RSI_3_4h"] < 85.0)
                and (last_candle["ROC_2_1h"] < 10.0)
                and (last_candle["ROC_2_4h"] < 10.0)
                and (last_candle["ROC_9_1h"] < 10.0)
                and (last_candle["ROC_9_4h"] < 10.0)
                and (last_candle["ROC_9_1d"] < 30.0)
                and (last_candle["STOCHRSIk_14_14_3_3"] > 50.0)
                and (last_candle["EMA_12"] > last_candle["EMA_26"])
                and ((last_candle["EMA_12"] - last_candle["EMA_26"]) > (last_candle["open"] * 0.020))
                and ((previous_candle["EMA_12"] - previous_candle["EMA_26"]) > (last_candle["open"] / 100.0))
        )
                or (
                (last_candle["RSI_14"] > 64.0)
                and (last_candle["RSI_3"] < 90.0)
                and (last_candle["RSI_3_15m"] < 90.0)
                and (last_candle["RSI_3_1h"] < 90.0)
                and (last_candle["RSI_3_4h"] < 90.0)
                and (last_candle["RSI_3_1d"] < 90.0)
                and (last_candle["ROC_2_1h"] < 5.0)
                and (last_candle["ROC_2_4h"] < 5.0)
                and (last_candle["ROC_2_1d"] < 5.0)
                and (last_candle["ROC_9_1h"] < 10.0)
                and (last_candle["ROC_9_4h"] < 10.0)
                and (last_candle["ROC_9_1d"] < 10.0)
                and (last_candle["AROOND_14_15m"] < 25.0)
                and (last_candle["close"] < (last_candle["close_min_48"] * 1.10))
                and (last_candle["close"] < (last_candle["low_min_6_1h"] * 1.18))
                and (last_candle["close"] < (last_candle["low_min_12_1h"] * 1.25))
                and (last_candle["close"] > (last_candle["EMA_12"] * 1.020))
        )
                or (
                (last_candle["RSI_14"] > 64.0)
                and (last_candle["RSI_3"] < 90.0)
                and (last_candle["RSI_3_15m"] < 90.0)
                and (last_candle["RSI_3_1h"] < 90.0)
                and (last_candle["RSI_3_4h"] < 90.0)
                and (last_candle["RSI_3_1d"] < 90.0)
                and (last_candle["ROC_2_1h"] < 10.0)
                and (last_candle["ROC_2_4h"] < 10.0)
                and (last_candle["ROC_2_1d"] < 10.0)
                and (last_candle["AROOND_14"] < 25.0)
                and (last_candle["close"] < (last_candle["close_min_48"] * 1.10))
                and (last_candle["close"] < (last_candle["low_min_6_1h"] * 1.18))
                and (last_candle["close"] < (last_candle["low_min_12_1h"] * 1.25))
                and (last_candle["close"] > (last_candle["EMA_26"] * 1.038))
                and (last_candle["close"] > (last_candle["BBU_20_2.0"] * 1.0))
        )
                or (
                (last_candle["RSI_14"] > 65.0)
                and (last_candle["RSI_3"] < 90.0)
                and (last_candle["RSI_3_15m"] < 90.0)
                and (last_candle["RSI_3_1h"] < 90.0)
                and (last_candle["RSI_3_4h"] < 90.0)
                and (last_candle["ROC_2_1h"] < 10.0)
                and (last_candle["ROC_2_4h"] < 10.0)
                and (last_candle["ROC_2_1d"] < 10.0)
                and (last_candle["ROC_9_1h"] < 10.0)
                and (last_candle["ROC_9_4h"] < 10.0)
                and (last_candle["AROOND_14"] < 25.0)
                and (last_candle["close"] > (last_candle["high_max_12_4h"] * 0.80))
                and (last_candle["close"] > (last_candle["EMA_9"] * 1.032))
                and (last_candle["close"] > (last_candle["EMA_20"] * 1.020))
        )
                or (
                (last_candle["RSI_14"] > 65.0)
                and (last_candle["RSI_3"] < 90.0)
                and (last_candle["RSI_3"] > 60.0)
                and (last_candle["RSI_3_15m"] < 85.0)
                and (last_candle["ROC_2_1h"] < 5.0)
                and (last_candle["ROC_2_4h"] < 5.0)
                and (last_candle["ROC_9_1h"] < 10.0)
                and (last_candle["ROC_9_4h"] < 10.0)
                and (last_candle["AROOND_14"] < 25.0)
                and (last_candle["RSI_20"] > previous_candle["RSI_20"])
                and (last_candle["close"] > (last_candle["SMA_16"] * 1.045))
        )
                or (
                (last_candle["RSI_3"] < 95.0)
                and (last_candle["RSI_3_15m"] < 90.0)
                and (last_candle["RSI_3_1h"] < 90.0)
                and (last_candle["RSI_3_4h"] < 90.0)
                and (last_candle["ROC_2_1h"] < 5.0)
                and (last_candle["ROC_2_4h"] < 5.0)
                and (last_candle["ROC_9_1h"] < 5.0)
                and (last_candle["ROC_9_4h"] < 5.0)
                and (last_candle["WILLR_14"] > -50.0)
                and (last_candle["STOCHRSIk_14_14_3_3"] > 80.0)
                and (last_candle["WILLR_84_1h"] > -30.0)
                and (last_candle["close"] < (last_candle["high_max_24_4h"] * 0.77))
                and (last_candle["BBB_20_2.0_1h"] > 12.0)
                and (last_candle["close_min_48"] <= (last_candle["close"] * 0.90))
        )
                or (
                (last_candle["RSI_3"] > 70.0)
                and (last_candle["RSI_3"] < 95.0)
                and (last_candle["RSI_3_15m"] < 95.0)
                and (last_candle["RSI_3_1h"] < 90.0)
                and (last_candle["RSI_3_4h"] < 90.0)
                and (last_candle["ROC_9_1d"] < 30.0)
                and (last_candle["EMA_12"] > last_candle["EMA_26"])
                and ((last_candle["EMA_12"] - last_candle["EMA_26"]) > (last_candle["open"] * 0.034))
                and ((previous_candle["EMA_12"] - previous_candle["EMA_26"]) > (last_candle["open"] / 100.0))
        )
                or (
                (last_candle["RSI_3"] < 95.0)
                and (last_candle["RSI_3_15m"] < 75.0)
                and (last_candle["RSI_3_1h"] < 70.0)
                and (last_candle["close"] > (last_candle["low_min_24_4h"] * 1.10))
                and (last_candle["close"] > (last_candle["close_min_48"] * 1.10))
                and (last_candle["close"] < (last_candle["close_max_12"] * 0.92))
        )
                or (
                (last_candle["RSI_3"] < 95.0)
                and (last_candle["RSI_3_15m"] < 95.0)
                and (last_candle["STOCHRSIk_14_14_3_3"] > 80.0)
                and (last_candle["RSI_14"] > (last_candle["RSI_14_1h"] + 45.0))
        )
                or (
                (last_candle["RSI_3"] < 90.0)
                and (last_candle["RSI_3_15m"] < 90.0)
                and (last_candle["RSI_3_1h"] < 90.0)
                and (last_candle["RSI_3_4h"] < 90.0)
                and (last_candle["RSI_3_1d"] < 90.0)
                and (last_candle["STOCHRSIk_14_14_3_3"] > 80.0)
                and (last_candle["close"] > (last_candle["SMA_30"] * 1.022))
                and (last_candle["close"] > (last_candle["BBU_20_2.0"] * 1.0))
        )
                or (
                (last_candle["RSI_14"] > 64.0)
                and (last_candle["RSI_3"] < 95.0)
                and (last_candle["RSI_3_15m"] < 90.0)
                and (last_candle["RSI_3_1h"] < 90.0)
                and (last_candle["RSI_3_4h"] < 90.0)
                and (last_candle["RSI_3_1d"] < 90.0)
                and (last_candle["STOCHRSIk_14_14_3_3"] > 70.0)
                and (last_candle["close"] < (last_candle["close_min_48"] * 1.15))
                and (last_candle["close"] < (last_candle["low_min_6_1h"] * 1.20))
                and (last_candle["close"] < (last_candle["low_min_12_1h"] * 1.33))
                and (last_candle["close"] > (last_candle["high_max_12_4h"] * 0.75))
                and (last_candle["EMA_12"] > last_candle["EMA_26"])
                and ((last_candle["EMA_12"] - last_candle["EMA_26"]) > (last_candle["open"] * 0.018))
                and ((previous_candle["EMA_12"] - previous_candle["EMA_26"]) > (last_candle["open"] / 100.0))
        )
                or (
                (last_candle["RSI_3"] < 95.0)
                and (previous_candle["SMA_9"] > previous_candle["SMA_21"])
                and (last_candle["SMA_9"] < last_candle["SMA_21"])
                and (last_candle["close"] > (last_candle["EMA_100"] * 1.016))
                and (last_candle["RSI_3_1h"] < 80.0)
                and (last_candle["RSI_3_4h"] < 80.0)
        )
                or (
                (slice_profit > 0.12)
                and (last_candle["RSI_3"] < 95.0)
                and (last_candle["RSI_3_15m"] < 90.0)
                and (last_candle["RSI_14"] > 60.0)
                and (last_candle["AROOND_14"] < 25.0)
                and (last_candle["AROOND_14_15m"] < 30.0)
                and (last_candle["STOCHRSIk_14_14_3_3"] > 80.0)
                and (last_candle["STOCHRSIk_14_14_3_3_15m"] > 70.0)
                and (last_candle["RSI_14_1h"] > 50.0)
                and (last_candle["RSI_14_4h"] > 50.0)
        )
                or (
                (last_candle["RSI_14"] > 64.0)
                and (last_candle["RSI_3"] < 95.0)
                and (last_candle["RSI_3_1h"] < 90.0)
                and (last_candle["RSI_3_4h"] < 90.0)
                and (last_candle["close"] > (last_candle["EMA_12"] * 1.001))
                and (last_candle["close"] > (last_candle["BBL_20_2.0"] * 1.004))
        )
        ):
            return True

        return False

    def short_grind_exit_v2(self, last_candle: Series, previous_candle: Series, slice_profit: float) -> float:
        if (
                (last_candle["RSI_3"] < 1.0)
                or (last_candle["RSI_14"] < 30.0)
                or (last_candle["WILLR_14"] < -99.9)
                or (last_candle["STOCHRSIk_14_14_3_3"] < 5.0)
                or (last_candle["close"] < (last_candle["BBL_20_2.0"] * 0.99))
                or ((last_candle["RSI_3"] < 10.0) and (last_candle["RSI_14"] > 50.0))
        ):
            return True

        return False
