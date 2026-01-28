import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Route, Routes } from 'react-router-dom';
import { ErrorBoundary } from './components/ErrorBoundary';
import { Layout } from './components/Layout';
import { Activity } from './pages/Activity';
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
import { DockerLogs } from './pages/DockerLogs';
import { DockerRegistries } from './pages/DockerRegistries';
import { DowntimeHistory } from './pages/DowntimeHistory';
import { FileDiff } from './pages/FileDiff';
import { FileHistory } from './pages/FileHistory';
import { FileSearch } from './pages/FileSearch';
import { IPAllowlistSettings } from './pages/IPAllowlistSettings';
import { LegalHolds } from './pages/LegalHolds';
import { LifecyclePolicies } from './pages/LifecyclePolicies';
import { Maintenance } from './pages/Maintenance';
import { NewOrganization } from './pages/NewOrganization';
import { NotFound } from './pages/NotFound';
import { NotificationRules } from './pages/NotificationRules';
import { Notifications } from './pages/Notifications';
import { Onboarding } from './pages/Onboarding';
import { OrganizationMembers } from './pages/OrganizationMembers';
import { OrganizationSSOSettings } from './pages/OrganizationSSOSettings';
import { OrganizationSettings } from './pages/OrganizationSettings';
import { PasswordPolicies } from './pages/PasswordPolicies';
import { Policies } from './pages/Policies';
import { RateLimitDashboard } from './pages/RateLimitDashboard';
import { RateLimits } from './pages/RateLimits';
import Reports from './pages/Reports';
import { Repositories } from './pages/Repositories';
import { RepositoryStatsDetail } from './pages/RepositoryStatsDetail';
import { Restore } from './pages/Restore';
import { SLA } from './pages/SLA';
import { Schedules } from './pages/Schedules';
import { SnapshotCompare } from './pages/SnapshotCompare';
import { StorageStats } from './pages/StorageStats';
import { Tags } from './pages/Tags';
import { Templates } from './pages/Templates';
import { UserSessions } from './pages/UserSessions';

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
		<ErrorBoundary>
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
							<Route path="activity" element={<Activity />} />
							<Route path="alerts" element={<Alerts />} />
							<Route path="downtime" element={<DowntimeHistory />} />
							<Route path="notifications" element={<Notifications />} />
							<Route
								path="notification-rules"
								element={<NotificationRules />}
							/>
							<Route path="reports" element={<Reports />} />
							<Route path="audit-logs" element={<AuditLogs />} />
							<Route path="legal-holds" element={<LegalHolds />} />
							<Route
								path="lifecycle-policies"
								element={<LifecyclePolicies />}
							/>
							<Route path="stats" element={<StorageStats />} />
							<Route path="stats/:id" element={<RepositoryStatsDetail />} />
							<Route path="tags" element={<Tags />} />
							<Route path="classifications" element={<Classifications />} />
							<Route path="costs" element={<CostEstimation />} />
							<Route path="sla" element={<SLA />} />
							<Route
								path="organization/docker-registries"
								element={<DockerRegistries />}
							/>
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
							<Route
								path="organization/maintenance"
								element={<Maintenance />}
							/>
							<Route
								path="organization/announcements"
								element={<Announcements />}
							/>
							<Route
								path="organization/ip-allowlist"
								element={<IPAllowlistSettings />}
							/>
							<Route
								path="organization/password-policies"
								element={<PasswordPolicies />}
							/>
							<Route path="organization/new" element={<NewOrganization />} />
							<Route path="admin/logs" element={<AdminLogs />} />
							<Route path="admin/docker-logs" element={<DockerLogs />} />
							<Route
								path="admin/rate-limits"
								element={<RateLimitDashboard />}
							/>
							<Route path="admin/rate-limit-configs" element={<RateLimits />} />
							<Route path="account/sessions" element={<UserSessions />} />
							<Route path="onboarding" element={<Onboarding />} />
							<Route path="changelog" element={<Changelog />} />
							<Route path="*" element={<NotFound />} />
						</Route>
					</Routes>
				</BrowserRouter>
			</QueryClientProvider>
		</ErrorBoundary>
	);
}

export default App;
