import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Route, Routes } from 'react-router-dom';
import { Layout } from './components/Layout';
import { Agents } from './pages/Agents';
import { Alerts } from './pages/Alerts';
import { AuditLogs } from './pages/AuditLogs';
import { Backups } from './pages/Backups';
import { Dashboard } from './pages/Dashboard';
import { NewOrganization } from './pages/NewOrganization';
import { Notifications } from './pages/Notifications';
import { OrganizationMembers } from './pages/OrganizationMembers';
import { OrganizationSettings } from './pages/OrganizationSettings';
import { OrganizationSSOSettings } from './pages/OrganizationSSOSettings';
import { Repositories } from './pages/Repositories';
import { RepositoryStatsDetail } from './pages/RepositoryStatsDetail';
import { Restore } from './pages/Restore';
import { Schedules } from './pages/Schedules';
import { StorageStats } from './pages/StorageStats';

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
						<Route path="restore" element={<Restore />} />
						<Route path="alerts" element={<Alerts />} />
						<Route path="notifications" element={<Notifications />} />
						<Route path="audit-logs" element={<AuditLogs />} />
						<Route path="stats" element={<StorageStats />} />
						<Route path="stats/:id" element={<RepositoryStatsDetail />} />
						<Route
							path="organization/members"
							element={<OrganizationMembers />}
						/>
						<Route
							path="organization/settings"
							element={<OrganizationSettings />}
						/>
						<Route
							path="organization/sso"
							element={<OrganizationSSOSettings />}
						/>
						<Route path="organization/new" element={<NewOrganization />} />
					</Route>
				</Routes>
			</BrowserRouter>
		</QueryClientProvider>
	);
}

export default App;
