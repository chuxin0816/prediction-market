import { useEffect, useState } from 'react';
import { adminApi, marketApi, type Market } from '../services/api';

const ADMIN_PIN = import.meta.env.VITE_ADMIN_PIN;
const ADMIN_TOKEN = import.meta.env.VITE_ADMIN_TOKEN;

function PinGate({ onSuccess }: { onSuccess: () => void }) {
  const [pin, setPin] = useState('');
  const [error, setError] = useState('');

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (pin === ADMIN_PIN) {
      sessionStorage.setItem('adminAuth', 'true');
      onSuccess();
    } else {
      setError('Invalid PIN');
      setPin('');
    }
  };

  return (
    <div className="flex items-center justify-center min-h-[60vh]">
      <form onSubmit={handleSubmit} className="bg-white rounded-lg shadow p-8 w-full max-w-sm">
        <h2 className="text-xl font-bold mb-4 text-center">Admin Access</h2>
        <input
          type="password"
          value={pin}
          onChange={(e) => { setPin(e.target.value); setError(''); }}
          placeholder="Enter PIN"
          className="w-full border rounded-lg px-4 py-2 mb-3 focus:outline-none focus:ring-2 focus:ring-blue-500"
          autoFocus
        />
        {error && <p className="text-red-500 text-sm mb-3">{error}</p>}
        <button
          type="submit"
          className="w-full bg-blue-600 text-white py-2 rounded-lg hover:bg-blue-700"
        >
          Enter
        </button>
      </form>
    </div>
  );
}

function CreateMarketForm({ onCreated }: { onCreated: () => void }) {
  const [question, setQuestion] = useState('');
  const [description, setDescription] = useState('');
  const [outcomes, setOutcomes] = useState(['Yes', 'No']);
  const [endTime, setEndTime] = useState('');
  const [resolutionTime, setResolutionTime] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setError('');
    try {
      await adminApi.createMarket(
        {
          question,
          description,
          outcomes: outcomes.filter((o) => o.trim() !== ''),
          end_time: new Date(endTime).toISOString(),
          resolution_time: new Date(resolutionTime).toISOString(),
        },
        ADMIN_TOKEN,
      );
      setQuestion('');
      setDescription('');
      setOutcomes(['Yes', 'No']);
      setEndTime('');
      setResolutionTime('');
      onCreated();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to create market');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <form onSubmit={handleSubmit} className="bg-white rounded-lg shadow p-6 mb-6">
      <h2 className="text-lg font-bold mb-4">Create Market</h2>

      <div className="mb-3">
        <label className="block text-sm font-medium text-gray-700 mb-1">Question</label>
        <input
          type="text"
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          required
          className="w-full border rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      <div className="mb-3">
        <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
        <textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={2}
          className="w-full border rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
        />
      </div>

      <div className="mb-3">
        <label className="block text-sm font-medium text-gray-700 mb-1">Outcomes</label>
        {outcomes.map((outcome, i) => (
          <div key={i} className="flex gap-2 mb-2">
            <input
              type="text"
              value={outcome}
              onChange={(e) => {
                const next = [...outcomes];
                next[i] = e.target.value;
                setOutcomes(next);
              }}
              required
              className="flex-1 border rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            {outcomes.length > 2 && (
              <button
                type="button"
                onClick={() => setOutcomes(outcomes.filter((_, j) => j !== i))}
                className="text-red-500 hover:text-red-700 px-2"
              >
                Remove
              </button>
            )}
          </div>
        ))}
        <button
          type="button"
          onClick={() => setOutcomes([...outcomes, ''])}
          className="text-blue-600 hover:text-blue-800 text-sm"
        >
          + Add outcome
        </button>
      </div>

      <div className="grid grid-cols-2 gap-4 mb-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">End Time</label>
          <input
            type="datetime-local"
            value={endTime}
            onChange={(e) => setEndTime(e.target.value)}
            required
            className="w-full border rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">Resolution Time</label>
          <input
            type="datetime-local"
            value={resolutionTime}
            onChange={(e) => setResolutionTime(e.target.value)}
            required
            className="w-full border rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>
      </div>

      {error && <p className="text-red-500 text-sm mb-3">{error}</p>}

      <button
        type="submit"
        disabled={submitting}
        className="bg-blue-600 text-white px-6 py-2 rounded-lg hover:bg-blue-700 disabled:opacity-50"
      >
        {submitting ? 'Creating...' : 'Create Market'}
      </button>
    </form>
  );
}

function ResolveModal({
  market,
  onClose,
  onResolved,
}: {
  market: Market;
  onClose: () => void;
  onResolved: () => void;
}) {
  const [selectedOutcome, setSelectedOutcome] = useState<number | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  const outcomes: string[] =
    typeof market.outcomes === 'string' ? JSON.parse(market.outcomes) : market.outcomes;

  const handleResolve = async () => {
    if (selectedOutcome === null) return;
    setSubmitting(true);
    setError('');
    try {
      await adminApi.resolveMarket(market.id, selectedOutcome, ADMIN_TOKEN);
      onResolved();
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to resolve market');
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl p-6 w-full max-w-md">
        <h3 className="text-lg font-bold mb-2">Resolve Market</h3>
        <p className="text-gray-600 mb-4">{market.question}</p>

        <div className="mb-4">
          <label className="block text-sm font-medium text-gray-700 mb-2">
            Select winning outcome
          </label>
          {outcomes.map((outcome, i) => (
            <button
              key={i}
              onClick={() => setSelectedOutcome(i + 1)}
              className={`block w-full text-left px-4 py-2 rounded-lg mb-2 border ${
                selectedOutcome === i + 1
                  ? 'border-blue-600 bg-blue-50 text-blue-700'
                  : 'border-gray-200 hover:bg-gray-50'
              }`}
            >
              {outcome}
            </button>
          ))}
        </div>

        {error && <p className="text-red-500 text-sm mb-3">{error}</p>}

        <div className="flex gap-3 justify-end">
          <button
            onClick={onClose}
            className="px-4 py-2 rounded-lg border border-gray-300 hover:bg-gray-50"
          >
            Cancel
          </button>
          <button
            onClick={handleResolve}
            disabled={selectedOutcome === null || submitting}
            className="bg-green-600 text-white px-4 py-2 rounded-lg hover:bg-green-700 disabled:opacity-50"
          >
            {submitting ? 'Resolving...' : 'Confirm Resolve'}
          </button>
        </div>
      </div>
    </div>
  );
}

function AdminDashboard() {
  const [markets, setMarkets] = useState<Market[]>([]);
  const [loading, setLoading] = useState(true);
  const [resolveTarget, setResolveTarget] = useState<Market | null>(null);

  const fetchMarkets = async () => {
    try {
      const res = await marketApi.list();
      setMarkets(res.data);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchMarkets();
  }, []);

  const statusColor: Record<string, string> = {
    active: 'bg-green-100 text-green-800',
    pending: 'bg-yellow-100 text-yellow-800',
    resolved: 'bg-gray-100 text-gray-800',
    cancelled: 'bg-red-100 text-red-800',
  };

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Admin Dashboard</h1>

      <CreateMarketForm onCreated={fetchMarkets} />

      <div className="bg-white rounded-lg shadow">
        <div className="px-6 py-4 border-b">
          <h2 className="text-lg font-bold">Markets</h2>
        </div>

        {loading ? (
          <div className="p-6 text-center text-gray-500">Loading...</div>
        ) : markets.length === 0 ? (
          <div className="p-6 text-center text-gray-500">No markets yet</div>
        ) : (
          <table className="w-full">
            <thead>
              <tr className="text-left text-sm text-gray-500 border-b">
                <th className="px-6 py-3">ID</th>
                <th className="px-6 py-3">Question</th>
                <th className="px-6 py-3">Status</th>
                <th className="px-6 py-3">End Time</th>
                <th className="px-6 py-3">Actions</th>
              </tr>
            </thead>
            <tbody>
              {markets.map((market) => (
                <tr key={market.id} className="border-b last:border-b-0 hover:bg-gray-50">
                  <td className="px-6 py-3 text-sm">{market.id}</td>
                  <td className="px-6 py-3 text-sm">{market.question}</td>
                  <td className="px-6 py-3">
                    <span
                      className={`text-xs px-2 py-1 rounded-full ${statusColor[market.status] || ''}`}
                    >
                      {market.status}
                    </span>
                  </td>
                  <td className="px-6 py-3 text-sm text-gray-500">
                    {new Date(market.end_time).toLocaleString()}
                  </td>
                  <td className="px-6 py-3">
                    {market.status === 'active' && (
                      <button
                        onClick={() => setResolveTarget(market)}
                        className="text-sm text-blue-600 hover:text-blue-800"
                      >
                        Resolve
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {resolveTarget && (
        <ResolveModal
          market={resolveTarget}
          onClose={() => setResolveTarget(null)}
          onResolved={() => {
            setResolveTarget(null);
            fetchMarkets();
          }}
        />
      )}
    </div>
  );
}

export function AdminPage() {
  const [authed, setAuthed] = useState(() => sessionStorage.getItem('adminAuth') === 'true');

  if (!authed) {
    return <PinGate onSuccess={() => setAuthed(true)} />;
  }

  return <AdminDashboard />;
}
