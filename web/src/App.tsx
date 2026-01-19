import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Route, Routes } from 'react-router-dom';
import { Layout } from './components/Layout';
import { Agents } from './pages/Agents';
import { Alerts } from './pages/Alerts';
import { Backups } from './pages/Backups';
import { Dashboard } from './pages/Dashboard';
import { Repositories } from './pages/Repositories';
import { Schedules } from './pages/Schedules';

const queryClient = new QueryClient({
	defaultOptions: {
		queries: {
			retry: (failureCount, error) => {
				// Don't retry on 4xx errors
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
					<Route path="/" element={<Layout />}>
						<Route index element={<Dashboard />} />
						<Route path="agents" element={<Agents />} />
						<Route path="repositories" element={<Repositories />} />
						<Route path="schedules" element={<Schedules />} />
						<Route path="backups" element={<Backups />} />
						<Route path="alerts" element={<Alerts />} />
					</Route>
				</Routes>
			</BrowserRouter>
		</QueryClientProvider>
	);
}

export default App;
