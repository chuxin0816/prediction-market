import { Link } from 'react-router-dom';
import type { Market } from '../services/api';

interface MarketCardProps {
  market: Market;
}

export function MarketCard({ market }: MarketCardProps) {
  const outcomes = typeof market.outcomes === 'string'
    ? JSON.parse(market.outcomes)
    : market.outcomes;

  const statusColors: Record<string, string> = {
    active: 'bg-green-100 text-green-800',
    pending: 'bg-yellow-100 text-yellow-800',
    resolved: 'bg-blue-100 text-blue-800',
    cancelled: 'bg-red-100 text-red-800',
  };

  return (
    <Link
      to={`/market/${market.id}`}
      className="block p-6 bg-white rounded-lg shadow hover:shadow-md transition-shadow"
    >
      <div className="flex justify-between items-start mb-4">
        <h3 className="text-lg font-semibold text-gray-900">{market.question}</h3>
        <span className={`px-2 py-1 rounded text-sm ${statusColors[market.status]}`}>
          {market.status}
        </span>
      </div>

      <div className="flex gap-2 mb-4">
        {outcomes.map((outcome: string, index: number) => (
          <span
            key={index}
            className="px-3 py-1 bg-gray-100 rounded-full text-sm text-gray-700"
          >
            {outcome}
          </span>
        ))}
      </div>

      <div className="text-sm text-gray-500">
        Ends: {new Date(market.end_time).toLocaleDateString()}
      </div>
    </Link>
  );
}
