import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Suspense, lazy } from 'react';
import { BrowserRouter, Route, Routes } from 'react-router-dom';
import { Layout } from './components/Layout';
import { LoadingSpinner } from './components/ui/LoadingSpinner';
import { UpgradePromptProvider } from './hooks/useUpgradePrompt';
import { BrandingProvider } from './contexts/BrandingContext';
import { Activity } from './pages/Activity';
import { AdminLogs } from './pages/AdminLogs';
import { AdminSetup } from './pages/AdminSetup';
import { AgentDetails } from './pages/AgentDetails';
import { AgentGroups } from './pages/AgentGroups';
import { Agents } from './pages/Agents';
import { Alerts } from './pages/Alerts';
import { Announcements } from './pages/Announcements';
import { AuditLogs } from './pages/AuditLogs';
import { BackupHookTemplates } from './pages/BackupHookTemplates';
import { Backups } from './pages/Backups';
import { BrandingSettings } from './pages/BrandingSettings';
import { Changelog } from './pages/Changelog';
import { Classifications } from './pages/Classifications';
import { CostEstimation } from './pages/CostEstimation';
import { DRRunbooks } from './pages/DRRunbooks';
import { Dashboard } from './pages/Dashboard';
import { FileDiff } from './pages/FileDiff';
import { FileHistory } from './pages/FileHistory';
import { FileSearch } from './pages/FileSearch';
import { LegalHolds } from './pages/LegalHolds';
import { License } from './pages/License';
import { LifecyclePolicies } from './pages/LifecyclePolicies';
import { Maintenance } from './pages/Maintenance';
import { NewOrganization } from './pages/NewOrganization';
import { Notifications } from './pages/Notifications';
import { Onboarding } from './pages/Onboarding';
import { OrgManagement } from './pages/OrgManagement';
import { OrganizationMembers } from './pages/OrganizationMembers';
import { OrganizationSSOSettings } from './pages/OrganizationSSOSettings';
import { OrganizationSettings } from './pages/OrganizationSettings';
import { Policies } from './pages/Policies';
import Reports from './pages/Reports';
import { Repositories } from './pages/Repositories';
import { RepositoryStatsDetail } from './pages/RepositoryStatsDetail';
import { Restore } from './pages/Restore';
import { Schedules } from './pages/Schedules';
import { Setup } from './pages/Setup';
import { SnapshotCompare } from './pages/SnapshotCompare';
import { StorageStats } from './pages/StorageStats';
import { SystemHealth } from './pages/SystemHealth';
import { Tags } from './pages/Tags';
import { Templates } from './pages/Templates';
import { UserManagement } from './pages/UserManagement';
import { UserSessions } from './pages/UserSessions';
import { LicenseManagement } from './pages/admin/LicenseManagement';

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
const Restore = lazy(() =>
	import('./pages/Restore').then((m) => ({ default: m.Restore })),
);
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
const AdminLogs = lazy(() =>
	import('./pages/AdminLogs').then((m) => ({ default: m.AdminLogs })),
);
const Announcements = lazy(() =>
	import('./pages/Announcements').then((m) => ({ default: m.Announcements })),
);
const Changelog = lazy(() =>
	import('./pages/Changelog').then((m) => ({ default: m.Changelog })),
);
const Classifications = lazy(() =>
	import('./pages/Classifications').then((m) => ({
		default: m.Classifications,
	})),
);
const FileDiff = lazy(() =>
	import('./pages/FileDiff').then((m) => ({ default: m.FileDiff })),
);
const FileSearch = lazy(() =>
	import('./pages/FileSearch').then((m) => ({ default: m.FileSearch })),
);
const IPAllowlistSettings = lazy(() =>
	import('./pages/IPAllowlistSettings').then((m) => ({
		default: m.IPAllowlistSettings,
	})),
);
const LegalHolds = lazy(() =>
	import('./pages/LegalHolds').then((m) => ({ default: m.LegalHolds })),
);
const PasswordPolicies = lazy(() =>
	import('./pages/PasswordPolicies').then((m) => ({
		default: m.PasswordPolicies,
	})),
);
const RateLimitDashboard = lazy(() =>
	import('./pages/RateLimitDashboard').then((m) => ({
		default: m.RateLimitDashboard,
	})),
);
const RateLimits = lazy(() =>
	import('./pages/RateLimits').then((m) => ({ default: m.RateLimits })),
);
const Templates = lazy(() =>
	import('./pages/Templates').then((m) => ({ default: m.Templates })),
);
const UserSessions = lazy(() =>
	import('./pages/UserSessions').then((m) => ({ default: m.UserSessions })),
);
const Activity = lazy(() =>
	import('./pages/Activity').then((m) => ({ default: m.Activity })),
);
const DowntimeHistory = lazy(() =>
	import('./pages/DowntimeHistory').then((m) => ({
		default: m.DowntimeHistory,
	})),
);
const NotificationRules = lazy(() =>
	import('./pages/NotificationRules').then((m) => ({
		default: m.NotificationRules,
	})),
);
const LifecyclePolicies = lazy(() =>
	import('./pages/LifecyclePolicies').then((m) => ({
		default: m.LifecyclePolicies,
	})),
);
const SLA = lazy(() =>
	import('./pages/SLA').then((m) => ({ default: m.SLA })),
);
const DockerLogs = lazy(() =>
	import('./pages/DockerLogs').then((m) => ({ default: m.DockerLogs })),
);
const DockerRegistries = lazy(() =>
	import('./pages/DockerRegistries').then((m) => ({
		default: m.DockerRegistries,
	})),
);
const NotFound = lazy(() =>
	import('./pages/NotFound').then((m) => ({ default: m.NotFound })),
);
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
		<ErrorBoundary>
			<QueryClientProvider client={queryClient}>
				<BrandingProvider>
					<BrowserRouter>
						<Routes>
							{/* Setup route - outside Layout, no auth required */}
							<Route path="/setup" element={<Setup />} />
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
								<Route path="dr-tests" element={<DRTests />} />
								<Route path="restore" element={<Restore />} />
								<Route path="file-history" element={<FileHistory />} />
								<Route path="file-search" element={<FileSearch />} />
								<Route
									path="snapshots/compare"
									element={<SnapshotCompare />}
								/>
								<Route
									path="backup-hook-templates"
									element={<BackupHookTemplates />}
								/>
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
								<Route
									path="stats/:id"
									element={<RepositoryStatsDetail />}
								/>
								<Route path="tags" element={<Tags />} />
								<Route
									path="classifications"
									element={<Classifications />}
								/>
								<Route path="costs" element={<CostEstimation />} />
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
								<Route path="organization/users" element={<UserManagement />} />
								<Route
									path="organization/settings"
									element={<OrganizationSettings />}
								/>
								<Route
									path="organization/sso"
									element={<OrganizationSSOSettings />}
								/>
								<Route
									path="organization/branding"
									element={<Branding />}
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
								<Route
									path="organization/new"
									element={<NewOrganization />}
								/>
								<Route path="admin/logs" element={<AdminLogs />} />
								<Route path="organization/license" element={<License />} />
								<Route
									path="organization/branding"
									element={<BrandingSettings />}
								/>
								<Route path="organization/new" element={<NewOrganization />} />
								<Route path="admin/logs" element={<AdminLogs />} />
								<Route path="admin/docker-logs" element={<DockerLogs />} />
								<Route path="admin/organizations" element={<OrgManagement />} />
								<Route
									path="admin/rate-limits"
									element={<RateLimitDashboard />}
								/>
								<Route
									path="admin/rate-limit-configs"
									element={<RateLimits />}
								/>
								<Route
									path="account/sessions"
									element={<UserSessions />}
								/>
								<Route path="docker-backup" element={<DockerBackup />} />
								<Route path="sla" element={<SLATracking />} />
								<Route path="sla-tracking" element={<SLA />} />
								<Route path="onboarding" element={<Onboarding />} />
								<Route path="changelog" element={<Changelog />} />
								<Route
									path="system/airgap"
									element={<AirGapLicense />}
								/>
								<Route path="license" element={<License />} />
								<Route path="docs/*" element={<Docs />} />
								<Route
									path="organization/docker-registries"
									element={<DockerRegistries />}
								/>
								<Route
									path="admin/docker-logs"
									element={<DockerLogs />}
								/>
								<Route path="*" element={<NotFound />} />
							</Route>
						</Routes>
					</Suspense>
				</UpgradePromptProvider>
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
					</Route>
				</Routes>
			</BrowserRouter>
		</QueryClientProvider>
								<Route path="admin/health" element={<SystemHealth />} />
								<Route path="admin/license" element={<LicenseManagement />} />
								<Route path="admin/setup" element={<AdminSetup />} />
								<Route path="account/sessions" element={<UserSessions />} />
								<Route path="onboarding" element={<Onboarding />} />
								<Route path="changelog" element={<Changelog />} />
								<Route path="*" element={<NotFound />} />
							</Route>
						</Routes>
					</BrowserRouter>
				</BrandingProvider>
			</QueryClientProvider>
		</ErrorBoundary>
	);
}

export default App;
