import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Suspense, lazy } from 'react';
import { BrowserRouter, Route, Routes } from 'react-router-dom';
import { Layout } from './components/Layout';
import { LoadingSpinner } from './components/ui/LoadingSpinner';
import { UpgradePromptProvider } from './hooks/useUpgradePrompt';

const Dashboard = lazy(() => import('./pages/Dashboard'));
const Agents = lazy(() => import('./pages/Agents'));
const AgentDetails = lazy(() => import('./pages/AgentDetails'));
const AgentGroups = lazy(() => import('./pages/AgentGroups'));
const Repositories = lazy(() => import('./pages/Repositories'));
const Schedules = lazy(() => import('./pages/Schedules'));
const Policies = lazy(() => import('./pages/Policies'));
const Backups = lazy(() => import('./pages/Backups'));
const DRRunbooks = lazy(() => import('./pages/DRRunbooks'));
const DRTests = lazy(() => import('./pages/DRTests'));
const Restore = lazy(() => import('./pages/Restore'));
const FileHistory = lazy(() => import('./pages/FileHistory'));
const SnapshotCompare = lazy(() => import('./pages/SnapshotCompare'));
const Alerts = lazy(() => import('./pages/Alerts'));
const Notifications = lazy(() => import('./pages/Notifications'));
const Reports = lazy(() => import('./pages/Reports'));
const AuditLogs = lazy(() => import('./pages/AuditLogs'));
const StorageStats = lazy(() => import('./pages/StorageStats'));
const RepositoryStatsDetail = lazy(
	() => import('./pages/RepositoryStatsDetail'),
);
const Tags = lazy(() => import('./pages/Tags'));
const CostEstimation = lazy(() => import('./pages/CostEstimation'));
const OrganizationMembers = lazy(() => import('./pages/OrganizationMembers'));
const OrganizationSettings = lazy(() => import('./pages/OrganizationSettings'));
const OrganizationSSOSettings = lazy(
	() => import('./pages/OrganizationSSOSettings'),
);
const Branding = lazy(() => import('./pages/Branding'));
const Maintenance = lazy(() => import('./pages/Maintenance'));
const NewOrganization = lazy(() => import('./pages/NewOrganization'));
const Onboarding = lazy(() => import('./pages/Onboarding'));
const SLATracking = lazy(() => import('./pages/SLATracking'));
const DockerBackup = lazy(() => import('./pages/DockerBackup'));
const AirGapLicense = lazy(() => import('./pages/AirGapLicense'));
const License = lazy(() => import('./pages/License'));
const Docs = lazy(() => import('./pages/Docs'));

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
				<UpgradePromptProvider>
					<Suspense fallback={<LoadingSpinner />}>
						<Routes>
							<Route path="/" element={<Layout />}>
								<Route index element={<Dashboard />} />
								<Route path="agents" element={<Agents />} />
								<Route path="agents/:id" element={<AgentDetails />} />
								<Route path="agent-groups" element={<AgentGroups />} />
								<Route path="repositories" element={<Repositories />} />
								<Route path="schedules" element={<Schedules />} />
								<Route path="policies" element={<Policies />} />
								<Route path="backups" element={<Backups />} />
								<Route path="dr-runbooks" element={<DRRunbooks />} />
								<Route path="dr-tests" element={<DRTests />} />
								<Route path="restore" element={<Restore />} />
								<Route path="file-history" element={<FileHistory />} />
								<Route path="snapshots/compare" element={<SnapshotCompare />} />
								<Route path="alerts" element={<Alerts />} />
								<Route path="notifications" element={<Notifications />} />
								<Route path="reports" element={<Reports />} />
								<Route path="audit-logs" element={<AuditLogs />} />
								<Route path="stats" element={<StorageStats />} />
								<Route path="stats/:id" element={<RepositoryStatsDetail />} />
								<Route path="tags" element={<Tags />} />
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
								<Route path="organization/branding" element={<Branding />} />
								<Route
									path="organization/maintenance"
									element={<Maintenance />}
								/>
								<Route path="organization/new" element={<NewOrganization />} />
								<Route path="docker-backup" element={<DockerBackup />} />
								<Route path="sla" element={<SLATracking />} />
								<Route path="onboarding" element={<Onboarding />} />
								<Route path="system/airgap" element={<AirGapLicense />} />
								<Route path="license" element={<License />} />
								<Route path="docs/*" element={<Docs />} />
							</Route>
						</Routes>
					</Suspense>
				</UpgradePromptProvider>
			</BrowserRouter>
		</QueryClientProvider>
	);
}

export default App;
