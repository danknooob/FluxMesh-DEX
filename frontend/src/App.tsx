import { BrowserRouter, Routes, Route, NavLink } from 'react-router-dom';
import { TraderLayout } from './layouts/TraderLayout';
import { AdminLayout } from './layouts/AdminLayout';
import { Markets } from './pages/Markets';
import { OrderBook } from './pages/OrderBook';
import { Balances } from './pages/Balances';
import { AdminMarkets } from './pages/admin/AdminMarkets';
import { AdminHealth } from './pages/admin/AdminHealth';

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<TraderLayout />}>
          <Route index element={<Markets />} />
          <Route path="markets" element={<Markets />} />
          <Route path="markets/:marketId" element={<OrderBook />} />
          <Route path="balances" element={<Balances />} />
        </Route>
        <Route path="/admin" element={<AdminLayout />}>
          <Route index element={<AdminMarkets />} />
          <Route path="markets" element={<AdminMarkets />} />
          <Route path="health" element={<AdminHealth />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;
