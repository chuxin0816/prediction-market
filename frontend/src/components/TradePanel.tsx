import { useState } from 'react';
import { useAccount } from 'wagmi';
import { useAppStore } from '../stores/useAppStore';

interface TradePanelProps {
  marketId: number;
  outcomes: string[];
}

export function TradePanel({ marketId, outcomes }: TradePanelProps) {
  const { address, isConnected } = useAccount();
  const { placeOrder, isLoading } = useAppStore();

  const [outcome, setOutcome] = useState(1);
  const [side, setSide] = useState<'buy' | 'sell'>('buy');
  const [price, setPrice] = useState('0.50');
  const [quantity, setQuantity] = useState('10');
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    if (!isConnected || !address) {
      setError('Please connect your wallet');
      return;
    }

    try {
      await placeOrder(marketId, outcome, side, price, quantity, address);
      setQuantity('10');
    } catch (err: any) {
      setError(err.response?.data?.error || err.message);
    }
  };

  return (
    <div className="bg-white rounded-lg shadow p-4">
      <h3 className="font-semibold mb-4">Place Order</h3>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Outcome
          </label>
          <select
            value={outcome}
            onChange={(e) => setOutcome(Number(e.target.value))}
            className="w-full border rounded-md p-2"
          >
            {outcomes.map((o, i) => (
              <option key={i} value={i + 1}>
                {o}
              </option>
            ))}
          </select>
        </div>

        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => setSide('buy')}
            className={`flex-1 py-2 rounded-md ${
              side === 'buy'
                ? 'bg-green-600 text-white'
                : 'bg-gray-100 text-gray-700'
            }`}
          >
            Buy
          </button>
          <button
            type="button"
            onClick={() => setSide('sell')}
            className={`flex-1 py-2 rounded-md ${
              side === 'sell'
                ? 'bg-red-600 text-white'
                : 'bg-gray-100 text-gray-700'
            }`}
          >
            Sell
          </button>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Price (0.01 - 0.99)
          </label>
          <input
            type="number"
            step="0.01"
            min="0.01"
            max="0.99"
            value={price}
            onChange={(e) => setPrice(e.target.value)}
            className="w-full border rounded-md p-2"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Quantity
          </label>
          <input
            type="number"
            step="1"
            min="1"
            value={quantity}
            onChange={(e) => setQuantity(e.target.value)}
            className="w-full border rounded-md p-2"
          />
        </div>

        <div className="text-sm text-gray-600">
          Total: {(parseFloat(price) * parseFloat(quantity)).toFixed(2)} USDC
        </div>

        {error && (
          <div className="text-sm text-red-600">{error}</div>
        )}

        <button
          type="submit"
          disabled={isLoading || !isConnected}
          className={`w-full py-2 rounded-md text-white ${
            side === 'buy' ? 'bg-green-600 hover:bg-green-700' : 'bg-red-600 hover:bg-red-700'
          } disabled:opacity-50`}
        >
          {isLoading ? 'Processing...' : isConnected ? `${side === 'buy' ? 'Buy' : 'Sell'} ${outcomes[outcome - 1]}` : 'Connect Wallet'}
        </button>
      </form>
    </div>
  );
}
