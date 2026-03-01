import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useAccount } from 'wagmi';
import { useAppStore } from '../stores/useAppStore';
import { ammApi } from '../services/api';
import type { AmmPrices, UserPosition } from '../services/api';
import { TradePanel } from '../components/TradePanel';

export function MarketPage() {
  const { id } = useParams<{ id: string }>();
  const { currentMarket, fetchMarket, isLoading } = useAppStore();
  const { address, isConnected } = useAccount();

  const [prices, setPrices] = useState<AmmPrices | null>(null);
  const [positions, setPositions] = useState<UserPosition[]>([]);

  const marketId = id ? parseInt(id) : 0;

  useEffect(() => {
    if (marketId) {
      fetchMarket(marketId);
    }
  }, [marketId, fetchMarket]);

  // Fetch AMM prices
  useEffect(() => {
    if (!marketId) return;
    const fetchPrices = async () => {
      try {
        const res = await ammApi.getPrices(marketId);
        setPrices(res.data);
      } catch {
        // Prices not available yet
      }
    };
    fetchPrices();
    const interval = setInterval(fetchPrices, 5000);
    return () => clearInterval(interval);
  }, [marketId]);

  // Fetch user positions
  useEffect(() => {
    if (!marketId || !isConnected || !address) {
      setPositions([]);
      return;
    }
    const fetchPositions = async () => {
      try {
        const res = await ammApi.getUserPositions(address, marketId);
        setPositions(res.data || []);
      } catch {
        setPositions([]);
      }
    };
    fetchPositions();
    const interval = setInterval(fetchPositions, 10000);
    return () => clearInterval(interval);
  }, [marketId, isConnected, address]);

  if (isLoading || !currentMarket) {
    return <div className="text-center py-8">Loading...</div>;
  }

  const outcomes: string[] =
    typeof currentMarket.outcomes === 'string'
      ? JSON.parse(currentMarket.outcomes)
      : currentMarket.outcomes;

  const OUTCOME_COLORS = [
    { bg: 'bg-blue-100', text: 'text-blue-800', border: 'border-blue-300' },
    { bg: 'bg-purple-100', text: 'text-purple-800', border: 'border-purple-300' },
    { bg: 'bg-orange-100', text: 'text-orange-800', border: 'border-orange-300' },
    { bg: 'bg-teal-100', text: 'text-teal-800', border: 'border-teal-300' },
  ];

  return (
    <div>
      <h1 className="text-2xl font-bold mb-2">{currentMarket.question}</h1>
      {currentMarket.description && (
        <p className="text-gray-600 mb-6">{currentMarket.description}</p>
      )}

      {/* AMM Prices */}
      {prices && (
        <div className="grid grid-cols-2 gap-4 mb-6">
          {outcomes.map((name, i) => {
            const color = OUTCOME_COLORS[i % OUTCOME_COLORS.length];
            const price = prices.prices[i];
            return (
              <div
                key={i}
                className={`rounded-lg border ${color.border} ${color.bg} p-4 text-center`}
              >
                <div className={`text-sm font-medium ${color.text} mb-1`}>
                  {name}
                </div>
                <div className={`text-3xl font-bold ${color.text}`}>
                  {price != null ? `${(Number(price) * 100).toFixed(1)}%` : '--'}
                </div>
                <div className="text-xs text-gray-500 mt-1">
                  Pool: {parseFloat(prices.pool_reserves[i] || '0').toFixed(2)}
                </div>
              </div>
            );
          })}
        </div>
      )}

      <div className="grid md:grid-cols-2 gap-6">
        {/* Trade Panel */}
        <TradePanel marketId={currentMarket.id} outcomes={outcomes} />

        {/* User Positions */}
        <div className="bg-white rounded-lg shadow p-4">
          <h3 className="font-semibold mb-4">Your Positions</h3>
          {!isConnected ? (
            <p className="text-sm text-gray-500">
              Connect your wallet to see positions
            </p>
          ) : positions.length === 0 ? (
            <p className="text-sm text-gray-500">
              No positions in this market
            </p>
          ) : (
            <div className="space-y-3">
              {positions.map((pos) => {
                const outcomeName = outcomes[pos.outcome - 1] || `Outcome ${pos.outcome}`;
                const shares = parseFloat(pos.shares);
                const color = OUTCOME_COLORS[(pos.outcome - 1) % OUTCOME_COLORS.length];
                return (
                  <div
                    key={pos.id}
                    className={`flex justify-between items-center p-3 rounded-md border ${color.border} ${color.bg}`}
                  >
                    <span className={`font-medium ${color.text}`}>
                      {outcomeName}
                    </span>
                    <span className={`font-bold ${color.text}`}>
                      {Number(shares).toFixed(2)} shares
                    </span>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
