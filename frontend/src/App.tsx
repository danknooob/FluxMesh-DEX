import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { AuthProvider } from './auth/AuthContext';
import { ProtectedRoute } from './auth/ProtectedRoute';
import { NotificationProvider } from './components/NotificationProvider';
import { PublicLayout } from './layouts/PublicLayout';
import { TraderLayout } from './layouts/TraderLayout';
import { AdminLayout } from './layouts/AdminLayout';
import { Login } from './pages/Login';
import { Home } from './pages/Home';
import { Markets } from './pages/Markets';
import { OrderBook } from './pages/OrderBook';
import { Balances } from './pages/Balances';
import { Profile } from './pages/Profile';
import { AdminMarkets } from './pages/admin/AdminMarkets';
import { AdminHealth } from './pages/admin/AdminHealth';

function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <NotificationProvider>
        <Routes>
          {/* Public routes */}
          <Route path="/login" element={<Login />} />
          <Route path="/" element={<PublicLayout />}>
            <Route index element={<Home />} />
          </Route>

          {/* Trader routes — require valid JWT */}
          <Route
            path="/trade"
            element={
              <ProtectedRoute>
                <TraderLayout />
              </ProtectedRoute>
            }
          >
            <Route index element={<Markets />} />
            <Route path="markets" element={<Markets />} />
            <Route path="markets/:marketId" element={<OrderBook />} />
            <Route path="balances" element={<Balances />} />
            <Route path="profile" element={<Profile />} />
          </Route>

          {/* Admin routes — require JWT + admin role */}
          <Route
            path="/admin"
            element={
              <ProtectedRoute requireAdmin>
                <AdminLayout />
              </ProtectedRoute>
            }
          >
            <Route index element={<AdminMarkets />} />
            <Route path="markets" element={<AdminMarkets />} />
            <Route path="health" element={<AdminHealth />} />
          </Route>
        </Routes>
        </NotificationProvider>
      </AuthProvider>
    </BrowserRouter>
  );
}

export default App;
