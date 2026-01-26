import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Route, Routes } from 'react-router-dom';
import { Layout } from './components/Layout';
import { AdminLogs } from './pages/AdminLogs';
import { AgentDetails } from './pages/AgentDetails';
import { AgentGroups } from './pages/AgentGroups';
import { Agents } from './pages/Agents';
import { Alerts } from './pages/Alerts';
import { Announcements } from './pages/Announcements';
import { AuditLogs } from './pages/AuditLogs';
import { Backups } from './pages/Backups';
import { Changelog } from './pages/Changelog';
import { Classifications } from './pages/Classifications';
import { CostEstimation } from './pages/CostEstimation';
import { DRRunbooks } from './pages/DRRunbooks';
import { Dashboard } from './pages/Dashboard';
import { FileDiff } from './pages/FileDiff';
import { FileHistory } from './pages/FileHistory';
import { FileSearch } from './pages/FileSearch';
import { LegalHolds } from './pages/LegalHolds';
import { Maintenance } from './pages/Maintenance';
import { NewOrganization } from './pages/NewOrganization';
import { NotificationRules } from './pages/NotificationRules';
import { Notifications } from './pages/Notifications';
import { Onboarding } from './pages/Onboarding';
import { OrganizationMembers } from './pages/OrganizationMembers';
import { OrganizationSSOSettings } from './pages/OrganizationSSOSettings';
import { OrganizationSettings } from './pages/OrganizationSettings';
import { Policies } from './pages/Policies';
import Reports from './pages/Reports';
import { Repositories } from './pages/Repositories';
import { RepositoryStatsDetail } from './pages/RepositoryStatsDetail';
import { Restore } from './pages/Restore';
import { Schedules } from './pages/Schedules';
import { SnapshotCompare } from './pages/SnapshotCompare';
import { StorageStats } from './pages/StorageStats';
import { Tags } from './pages/Tags';
import { Templates } from './pages/Templates';

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
						<Route path="agents/:id" element={<AgentDetails />} />
						<Route path="agent-groups" element={<AgentGroups />} />
						<Route path="repositories" element={<Repositories />} />
						<Route path="schedules" element={<Schedules />} />
						<Route path="policies" element={<Policies />} />
						<Route path="templates" element={<Templates />} />
						<Route path="backups" element={<Backups />} />
						<Route path="dr-runbooks" element={<DRRunbooks />} />
						<Route path="restore" element={<Restore />} />
						<Route path="file-history" element={<FileHistory />} />
						<Route path="file-search" element={<FileSearch />} />
						<Route path="snapshots/compare" element={<SnapshotCompare />} />
						<Route path="snapshots/file-diff" element={<FileDiff />} />
						<Route path="alerts" element={<Alerts />} />
						<Route path="notifications" element={<Notifications />} />
						<Route path="notification-rules" element={<NotificationRules />} />
						<Route path="reports" element={<Reports />} />
						<Route path="audit-logs" element={<AuditLogs />} />
						<Route path="legal-holds" element={<LegalHolds />} />
						<Route path="stats" element={<StorageStats />} />
						<Route path="stats/:id" element={<RepositoryStatsDetail />} />
						<Route path="tags" element={<Tags />} />
						<Route path="classifications" element={<Classifications />} />
						<Route path="costs" element={<CostEstimation />} />
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
						<Route path="organization/maintenance" element={<Maintenance />} />
						<Route
							path="organization/announcements"
							element={<Announcements />}
						/>
						<Route path="organization/new" element={<NewOrganization />} />
						<Route path="admin/logs" element={<AdminLogs />} />
						<Route path="onboarding" element={<Onboarding />} />
						<Route path="changelog" element={<Changelog />} />
					</Route>
				</Routes>
			</BrowserRouter>
		</QueryClientProvider>
	);
}

export default App;
