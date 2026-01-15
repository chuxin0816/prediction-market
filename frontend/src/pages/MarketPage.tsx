import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { useAppStore } from '../stores/useAppStore';
import { TradePanel } from '../components/TradePanel';
import { OrderBook } from '../components/OrderBook';

export function MarketPage() {
  const { id } = useParams<{ id: string }>();
  const { currentMarket, fetchMarket, isLoading } = useAppStore();
  const [selectedOutcome, setSelectedOutcome] = useState(1);

  useEffect(() => {
    if (id) {
      fetchMarket(parseInt(id));
    }
  }, [id, fetchMarket]);

  if (isLoading || !currentMarket) {
    return <div className="text-center py-8">Loading...</div>;
  }

  const outcomes = typeof currentMarket.outcomes === 'string'
    ? JSON.parse(currentMarket.outcomes)
    : currentMarket.outcomes;

  return (
    <div>
      <h1 className="text-2xl font-bold mb-2">{currentMarket.question}</h1>
      {currentMarket.description && (
        <p className="text-gray-600 mb-6">{currentMarket.description}</p>
      )}

      <div className="flex gap-2 mb-6">
        {outcomes.map((outcome: string, index: number) => (
          <button
            key={index}
            onClick={() => setSelectedOutcome(index + 1)}
            className={`px-4 py-2 rounded-full ${
              selectedOutcome === index + 1
                ? 'bg-blue-600 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            {outcome}
          </button>
        ))}
      </div>

      <div className="grid md:grid-cols-2 gap-6">
        <OrderBook marketId={currentMarket.id} outcome={selectedOutcome} />
        <TradePanel marketId={currentMarket.id} outcomes={outcomes} />
      </div>
    </div>
  );
}
