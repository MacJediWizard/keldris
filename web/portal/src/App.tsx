import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom';
import { Layout } from './components/Layout';
import { ProtectedRoute } from './components/ProtectedRoute';
import { Dashboard } from './pages/Dashboard';
import { Invoices } from './pages/Invoices';
import { InvoiceDetail } from './pages/InvoiceDetail';
import { Licenses } from './pages/Licenses';
import { LicenseDetail } from './pages/LicenseDetail';
import { Login } from './pages/Login';
import { Register } from './pages/Register';

const queryClient = new QueryClient({
	defaultOptions: {
		queries: {
			retry: (failureCount, error) => {
				if (error instanceof Error && 'status' in error) {
					const status = (error as { status: number }).status;
					if (status >= 400 && status < 500) return false;
				}
				return failureCount < 3;
			},
			refetchOnWindowFocus: false,
		},
	},
});

function App() {
	return (
		<QueryClientProvider client={queryClient}>
			<BrowserRouter>
				<Routes>
					<Route path="/login" element={<Login />} />
					<Route path="/register" element={<Register />} />
					<Route
						path="/"
						element={
							<ProtectedRoute>
								<Layout />
							</ProtectedRoute>
						}
					>
						<Route index element={<Dashboard />} />
						<Route path="licenses" element={<Licenses />} />
						<Route path="licenses/:id" element={<LicenseDetail />} />
						<Route path="invoices" element={<Invoices />} />
						<Route path="invoices/:id" element={<InvoiceDetail />} />
					</Route>
					<Route path="*" element={<Navigate to="/" replace />} />
				</Routes>
			</BrowserRouter>
		</QueryClientProvider>
	);
}

export default App;
