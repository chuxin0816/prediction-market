import { useEffect, useState } from 'react';
import { marketApi } from '../services/api';
import type { PriceLevel } from '../services/api';

interface OrderBookProps {
  marketId: number;
  outcome: number;
}

export function OrderBook({ marketId, outcome }: OrderBookProps) {
  const [buys, setBuys] = useState<PriceLevel[]>([]);
  const [sells, setSells] = useState<PriceLevel[]>([]);

  useEffect(() => {
    const fetchOrderBook = async () => {
      try {
        const res = await marketApi.getOrderBook(marketId, outcome);
        setBuys(res.data.buys || []);
        setSells(res.data.sells || []);
      } catch (err) {
        console.error('Failed to fetch order book:', err);
      }
    };

    fetchOrderBook();
    const interval = setInterval(fetchOrderBook, 5000);
    return () => clearInterval(interval);
  }, [marketId, outcome]);

  return (
    <div className="bg-white rounded-lg shadow p-4">
      <h3 className="font-semibold mb-4">Order Book</h3>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <div className="text-sm font-medium text-green-600 mb-2">Buys</div>
          {buys.length === 0 ? (
            <div className="text-sm text-gray-400">No buy orders</div>
          ) : (
            buys.slice(0, 5).map((level, i) => (
              <div key={i} className="flex justify-between text-sm py-1">
                <span className="text-green-600">{parseFloat(level.price).toFixed(2)}</span>
                <span>{parseFloat(level.quantity).toFixed(2)}</span>
              </div>
            ))
          )}
        </div>

        <div>
          <div className="text-sm font-medium text-red-600 mb-2">Sells</div>
          {sells.length === 0 ? (
            <div className="text-sm text-gray-400">No sell orders</div>
          ) : (
            sells.slice(0, 5).map((level, i) => (
              <div key={i} className="flex justify-between text-sm py-1">
                <span className="text-red-600">{parseFloat(level.price).toFixed(2)}</span>
                <span>{parseFloat(level.quantity).toFixed(2)}</span>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
