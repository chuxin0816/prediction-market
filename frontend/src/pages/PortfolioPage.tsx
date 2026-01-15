import { useEffect } from 'react';
import { useAccount } from 'wagmi';
import { useAppStore } from '../stores/useAppStore';
import { DepositWithdraw } from '../components/DepositWithdraw';

export function PortfolioPage() {
  const { address, isConnected } = useAccount();
  const { userOrders, fetchUserOrders, cancelOrder } = useAppStore();

  useEffect(() => {
    if (address) {
      fetchUserOrders(address);
    }
  }, [address, fetchUserOrders]);

  if (!isConnected) {
    return (
      <div className="text-center py-8 text-gray-500">
        Please connect your wallet to view your portfolio
      </div>
    );
  }

  const openOrders = userOrders.filter(
    (o) => o.status === 'open' || o.status === 'partial'
  );

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">My Portfolio</h1>

      <div className="mb-6">
        <DepositWithdraw />
      </div>

      <div className="bg-white rounded-lg shadow">
        <div className="p-4 border-b">
          <h2 className="font-semibold">Open Orders</h2>
        </div>

        {openOrders.length === 0 ? (
          <div className="p-4 text-gray-500">No open orders</div>
        ) : (
          <table className="w-full">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-2 text-left">Market</th>
                <th className="px-4 py-2 text-left">Side</th>
                <th className="px-4 py-2 text-left">Price</th>
                <th className="px-4 py-2 text-left">Qty</th>
                <th className="px-4 py-2 text-left">Filled</th>
                <th className="px-4 py-2 text-left">Action</th>
              </tr>
            </thead>
            <tbody>
              {openOrders.map((order) => (
                <tr key={order.id} className="border-t">
                  <td className="px-4 py-2">#{order.market_id}</td>
                  <td className="px-4 py-2">
                    <span
                      className={
                        order.side === 'buy' ? 'text-green-600' : 'text-red-600'
                      }
                    >
                      {order.side.toUpperCase()}
                    </span>
                  </td>
                  <td className="px-4 py-2">{parseFloat(order.price).toFixed(2)}</td>
                  <td className="px-4 py-2">{parseFloat(order.quantity).toFixed(2)}</td>
                  <td className="px-4 py-2">{parseFloat(order.filled_quantity).toFixed(2)}</td>
                  <td className="px-4 py-2">
                    <button
                      onClick={() => cancelOrder(order.id, address!)}
                      className="text-red-600 hover:text-red-800"
                    >
                      Cancel
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}
