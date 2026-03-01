import { useState, useEffect, useRef, useCallback } from 'react';
import { useAccount } from 'wagmi';
import { ammApi } from '../services/api';
import type { AmmQuote } from '../services/api';

interface TradePanelProps {
  marketId: number;
  outcomes: string[];
}

export function TradePanel({ marketId, outcomes }: TradePanelProps) {
  const { address, isConnected } = useAccount();

  const [side, setSide] = useState<'buy' | 'sell'>('buy');
  const [outcome, setOutcome] = useState(1);
  const [amount, setAmount] = useState('');
  const [quote, setQuote] = useState<AmmQuote | null>(null);
  const [quoteLoading, setQuoteLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const fetchQuote = useCallback(
    async (amt: string) => {
      const parsed = parseFloat(amt);
      if (!amt || isNaN(parsed) || parsed <= 0) {
        setQuote(null);
        return;
      }
      setQuoteLoading(true);
      try {
        const res = await ammApi.getQuote({
          market_id: marketId,
          outcome,
          amount: parsed,
          side,
        });
        setQuote(res.data);
      } catch {
        setQuote(null);
      } finally {
        setQuoteLoading(false);
      }
    },
    [marketId, outcome, side],
  );

  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      fetchQuote(amount);
    }, 300);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [amount, fetchQuote]);

  // Reset quote/feedback when switching side or outcome
  useEffect(() => {
    setQuote(null);
    setError('');
    setSuccess('');
  }, [side, outcome]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setSuccess('');

    if (!isConnected || !address) {
      setError('Please connect your wallet');
      return;
    }

    const parsed = parseFloat(amount);
    if (isNaN(parsed) || parsed <= 0) {
      setError('Enter a valid amount');
      return;
    }

    setSubmitting(true);
    try {
      if (side === 'buy') {
        const res = await ammApi.buy(
          { market_id: marketId, outcome, amount: String(parsed) },
          address,
        );
        setSuccess(
          `Bought ${Number(res.data.shares_received).toFixed(2)} shares at avg price ${(Number(res.data.avg_price) * 100).toFixed(1)}%`,
        );
      } else {
        const res = await ammApi.sell(
          { market_id: marketId, outcome, shares: String(parsed) },
          address,
        );
        setSuccess(
          `Sold for ${Number(res.data.usdc_received).toFixed(2)} USDC at avg price ${(Number(res.data.avg_price) * 100).toFixed(1)}%`,
        );
      }
      setAmount('');
      setQuote(null);
    } catch (err: unknown) {
      const axiosErr = err as { response?: { data?: { error?: string } }; message?: string };
      setError(axiosErr.response?.data?.error || axiosErr.message || 'Trade failed');
    } finally {
      setSubmitting(false);
    }
  };

  const isBuy = side === 'buy';
  const themeColor = isBuy ? 'green' : 'red';

  return (
    <div className="bg-white rounded-lg shadow p-4">
      <h3 className="font-semibold mb-4">Trade</h3>

      {/* Buy / Sell Tabs */}
      <div className="flex gap-2 mb-4">
        <button
          type="button"
          onClick={() => setSide('buy')}
          className={`flex-1 py-2 rounded-md font-medium ${
            isBuy
              ? 'bg-green-600 text-white'
              : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
          }`}
        >
          Buy
        </button>
        <button
          type="button"
          onClick={() => setSide('sell')}
          className={`flex-1 py-2 rounded-md font-medium ${
            !isBuy
              ? 'bg-red-600 text-white'
              : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
          }`}
        >
          Sell
        </button>
      </div>

      {/* Outcome selector */}
      <div className="mb-4">
        <label className="block text-sm font-medium text-gray-700 mb-2">
          Outcome
        </label>
        <div className="flex gap-2">
          {outcomes.map((o, i) => (
            <button
              key={i}
              type="button"
              onClick={() => setOutcome(i + 1)}
              className={`flex-1 px-3 py-2 rounded-md text-sm font-medium ${
                outcome === i + 1
                  ? `bg-${themeColor}-100 text-${themeColor}-800 ring-2 ring-${themeColor}-500`
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              {o}
            </button>
          ))}
        </div>
      </div>

      <form onSubmit={handleSubmit} className="space-y-4">
        {/* Amount input */}
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            {isBuy ? 'Amount (USDC)' : 'Shares to sell'}
          </label>
          <input
            type="number"
            step="any"
            min="0"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            placeholder={isBuy ? 'Enter USDC amount' : 'Enter shares'}
            className="w-full border rounded-md p-2 focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          />
        </div>

        {/* Quote display */}
        {quoteLoading && (
          <div className="text-sm text-gray-500">Fetching quote...</div>
        )}
        {quote && !quoteLoading && (
          <div className="bg-gray-50 rounded-md p-3 space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-gray-600">
                {isBuy ? 'Est. shares' : 'Est. USDC out'}
              </span>
              <span className="font-medium">
                {isBuy
                  ? Number(quote.shares_out).toFixed(4)
                  : Number(quote.usdc_out).toFixed(4)}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-600">Avg price</span>
              <span className="font-medium">
                {(Number(quote.avg_price) * 100).toFixed(2)}%
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-gray-600">Price impact</span>
              <span
                className={`font-medium ${
                  Number(quote.price_impact) > 0.05
                    ? 'text-red-600'
                    : 'text-gray-900'
                }`}
              >
                {(Number(quote.price_impact) * 100).toFixed(2)}%
              </span>
            </div>
            {Number(quote.price_impact) > 0.05 && (
              <div className="text-red-600 font-medium text-xs bg-red-50 rounded p-2">
                Warning: High price impact!
              </div>
            )}
          </div>
        )}

        {/* Feedback */}
        {error && <div className="text-sm text-red-600">{error}</div>}
        {success && <div className="text-sm text-green-600">{success}</div>}

        {/* Submit */}
        <button
          type="submit"
          disabled={submitting || !isConnected || !amount}
          className={`w-full py-2 rounded-md text-white font-medium ${
            isBuy
              ? 'bg-green-600 hover:bg-green-700'
              : 'bg-red-600 hover:bg-red-700'
          } disabled:opacity-50`}
        >
          {submitting
            ? 'Processing...'
            : !isConnected
              ? 'Connect Wallet'
              : `${isBuy ? 'Buy' : 'Sell'} ${outcomes[outcome - 1]}`}
        </button>
      </form>
    </div>
  );
}
