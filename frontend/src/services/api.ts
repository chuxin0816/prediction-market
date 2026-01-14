import axios from 'axios';

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080/api',
});

export interface Market {
  id: number;
  question: string;
  description: string;
  outcomes: string[];
  end_time: string;
  resolution_time: string;
  resolved_outcome: number | null;
  status: 'pending' | 'active' | 'resolved' | 'cancelled';
}

export interface Order {
  id: number;
  market_id: number;
  user_address: string;
  outcome: number;
  side: 'buy' | 'sell';
  price: string;
  quantity: string;
  filled_quantity: string;
  status: 'open' | 'filled' | 'partial' | 'cancelled';
  created_at: string;
}

export interface Trade {
  id: number;
  market_id: number;
  price: string;
  quantity: string;
  created_at: string;
}

export interface PriceLevel {
  price: string;
  quantity: string;
}

export interface OrderBookData {
  buys: PriceLevel[];
  sells: PriceLevel[];
}

export const marketApi = {
  list: () => api.get<Market[]>('/markets'),
  get: (id: number) => api.get<Market>(`/markets/${id}`),
  getTrades: (id: number) => api.get<Trade[]>(`/markets/${id}/trades`),
  getOrderBook: (id: number, outcome: number) =>
    api.get<OrderBookData>(`/markets/${id}/orderbook`, { params: { outcome } }),
};

export const orderApi = {
  place: (data: {
    market_id: number;
    outcome: number;
    side: 'buy' | 'sell';
    price: string;
    quantity: string;
  }, walletAddress: string) =>
    api.post('/orders', data, {
      headers: { 'X-Wallet-Address': walletAddress },
    }),
  cancel: (id: number, walletAddress: string) =>
    api.delete(`/orders/${id}`, {
      headers: { 'X-Wallet-Address': walletAddress },
    }),
  getUserOrders: (walletAddress: string) =>
    api.get<Order[]>('/user/orders', {
      headers: { 'X-Wallet-Address': walletAddress },
    }),
};

export default api;
