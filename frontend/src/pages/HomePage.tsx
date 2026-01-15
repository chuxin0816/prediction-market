import { MarketList } from '../components/MarketList';

export function HomePage() {
  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Prediction Markets</h1>
      <MarketList />
    </div>
  );
}
