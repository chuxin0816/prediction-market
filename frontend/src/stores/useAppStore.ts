import { create } from 'zustand';
import type { Market, Order } from '../services/api';
import { marketApi, orderApi } from '../services/api';

interface AppState {
  markets: Market[];
  currentMarket: Market | null;
  userOrders: Order[];
  isLoading: boolean;
  error: string | null;

  fetchMarkets: () => Promise<void>;
  fetchMarket: (id: number) => Promise<void>;
  fetchUserOrders: (address: string) => Promise<void>;
  placeOrder: (
    marketId: number,
    outcome: number,
    side: 'buy' | 'sell',
    price: string,
    quantity: string,
    address: string
  ) => Promise<void>;
  cancelOrder: (orderId: number, address: string) => Promise<void>;
}

export const useAppStore = create<AppState>((set, get) => ({
  markets: [],
  currentMarket: null,
  userOrders: [],
  isLoading: false,
  error: null,

  fetchMarkets: async () => {
    set({ isLoading: true, error: null });
    try {
      const res = await marketApi.list();
      set({ markets: res.data, isLoading: false });
    } catch (err: any) {
      set({ error: err.message, isLoading: false });
    }
  },

  fetchMarket: async (id: number) => {
    set({ isLoading: true, error: null });
    try {
      const res = await marketApi.get(id);
      set({ currentMarket: res.data, isLoading: false });
    } catch (err: any) {
      set({ error: err.message, isLoading: false });
    }
  },

  fetchUserOrders: async (address: string) => {
    try {
      const res = await orderApi.getUserOrders(address);
      set({ userOrders: res.data });
    } catch (err: any) {
      console.error('Failed to fetch orders:', err);
    }
  },

  placeOrder: async (marketId, outcome, side, price, quantity, address) => {
    set({ isLoading: true, error: null });
    try {
      await orderApi.place({ market_id: marketId, outcome, side, price, quantity }, address);
      await get().fetchUserOrders(address);
      set({ isLoading: false });
    } catch (err: any) {
      set({ error: err.message, isLoading: false });
      throw err;
    }
  },

  cancelOrder: async (orderId: number, address: string) => {
    try {
      await orderApi.cancel(orderId, address);
      await get().fetchUserOrders(address);
    } catch (err: any) {
      set({ error: err.message });
      throw err;
    }
  },
}));
