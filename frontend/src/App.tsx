import { Routes, Route, Link } from 'react-router-dom';
import { WalletButton } from './components/WalletButton';
import { HomePage } from './pages/HomePage';
import { MarketPage } from './pages/MarketPage';
import { PortfolioPage } from './pages/PortfolioPage';

function App() {
  return (
    <div className="min-h-screen bg-gray-100">
      <nav className="bg-white shadow-sm">
        <div className="max-w-7xl mx-auto px-4 py-3 flex justify-between items-center">
          <div className="flex items-center gap-6">
            <Link to="/" className="text-xl font-bold text-blue-600">
              PredictX
            </Link>
            <Link to="/" className="text-gray-600 hover:text-gray-900">
              Markets
            </Link>
            <Link to="/portfolio" className="text-gray-600 hover:text-gray-900">
              Portfolio
            </Link>
          </div>
          <WalletButton />
        </div>
      </nav>

      <main className="max-w-7xl mx-auto px-4 py-8">
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/market/:id" element={<MarketPage />} />
          <Route path="/portfolio" element={<PortfolioPage />} />
        </Routes>
      </main>
    </div>
  );
}

export default App;
