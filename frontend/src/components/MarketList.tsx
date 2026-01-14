import { useEffect } from 'react';
import { useAppStore } from '../stores/useAppStore';
import { MarketCard } from './MarketCard';

export function MarketList() {
  const { markets, isLoading, error, fetchMarkets } = useAppStore();

  useEffect(() => {
    fetchMarkets();
  }, [fetchMarkets]);

  if (isLoading) {
    return <div className="text-center py-8">Loading markets...</div>;
  }

  if (error) {
    return <div className="text-center py-8 text-red-600">Error: {error}</div>;
  }

  if (markets.length === 0) {
    return <div className="text-center py-8 text-gray-500">No markets available</div>;
  }

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      {markets.map((market) => (
        <MarketCard key={market.id} market={market} />
      ))}
    </div>
  );
}
