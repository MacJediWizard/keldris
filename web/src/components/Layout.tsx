import { useEffect, useState } from 'react';
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useAlertCount } from '../hooks/useAlerts';
import { useLogout, useMe } from '../hooks/useAuth';
import {
	useLatestChanges,
	useNewVersionAvailable,
} from '../hooks/useChangelog';
import { useKeyboardShortcuts } from '../hooks/useKeyboardShortcuts';
import { useLocale } from '../hooks/useLocale';
import { useOnboardingStatus } from '../hooks/useOnboarding';
import {
	useOrganizations,
	useSwitchOrganization,
} from '../hooks/useOrganizations';
import {
	ReadOnlyModeContext,
	useReadOnlyModeValue,
} from '../hooks/useReadOnlyMode';
import { AnnouncementBanner } from './features/AnnouncementBanner';
import { LanguageSelector } from './features/LanguageSelector';
import { MaintenanceCountdown } from './features/MaintenanceCountdown';
import { ShortcutHelpModal } from './features/ShortcutHelpModal';
import { WhatsNewModal } from './features/WhatsNewModal';

interface NavItem {
	path: string;
	labelKey: string;
	icon: React.ReactNode;
	shortcut?: string;
}

const navItems: NavItem[] = [
	{
		path: '/',
		labelKey: 'nav.dashboard',
		shortcut: 'G D',
		icon: (
			<svg
				aria-hidden="true"
				className="w-5 h-5"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					strokeLinecap="round"
					strokeLinejoin="round"
					strokeWidth={2}
					d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"
				/>
			</svg>
		),
	},
	{
		path: '/agents',
		labelKey: 'nav.agents',
		shortcut: 'G A',
		icon: (
			<svg
				aria-hidden="true"
				className="w-5 h-5"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					strokeLinecap="round"
					strokeLinejoin="round"
					strokeWidth={2}
					d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
				/>
			</svg>
		),
	},
	{
		path: '/repositories',
		labelKey: 'nav.repositories',
		shortcut: 'G R',
		icon: (
			<svg
				aria-hidden="true"
				className="w-5 h-5"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					strokeLinecap="round"
					strokeLinejoin="round"
					strokeWidth={2}
					d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"
				/>
			</svg>
		),
	},
	{
		path: '/schedules',
		labelKey: 'nav.schedules',
		shortcut: 'G S',
		icon: (
			<svg
				aria-hidden="true"
				className="w-5 h-5"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					strokeLinecap="round"
					strokeLinejoin="round"
					strokeWidth={2}
					d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
				/>
			</svg>
		),
	},
	{
		path: '/backups',
		labelKey: 'nav.backups',
		shortcut: 'G B',
		icon: (
			<svg
				aria-hidden="true"
				className="w-5 h-5"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					strokeLinecap="round"
					strokeLinejoin="round"
					strokeWidth={2}
					d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"
				/>
			</svg>
		),
	},
	{
		path: '/restore',
		labelKey: 'nav.restore',
		shortcut: 'G E',
		icon: (
			<svg
				aria-hidden="true"
				className="w-5 h-5"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					strokeLinecap="round"
					strokeLinejoin="round"
					strokeWidth={2}
					d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
				/>
			</svg>
		),
	},
	{
		path: '/file-search',
		labelKey: 'nav.fileSearch',
		icon: (
			<svg
				aria-hidden="true"
				className="w-5 h-5"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					strokeLinecap="round"
					strokeLinejoin="round"
					strokeWidth={2}
					d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
				/>
			</svg>
		),
	},
	{
		path: '/file-history',
		labelKey: 'nav.fileHistory',
		icon: (
			<svg
				aria-hidden="true"
				className="w-5 h-5"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					strokeLinecap="round"
					strokeLinejoin="round"
					strokeWidth={2}
					d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
				/>
			</svg>
		),
	},
	{
		path: '/alerts',
		labelKey: 'nav.alerts',
		shortcut: 'G L',
		icon: (
			<svg
				aria-hidden="true"
				className="w-5 h-5"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					strokeLinecap="round"
					strokeLinejoin="round"
					strokeWidth={2}
					d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
				/>
			</svg>
		),
	},
	{
		path: '/notifications',
		labelKey: 'nav.notifications',
		icon: (
			<svg
				aria-hidden="true"
				className="w-5 h-5"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					strokeLinecap="round"
					strokeLinejoin="round"
					strokeWidth={2}
					d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
				/>
			</svg>
		),
	},
	{
		path: '/audit-logs',
		labelKey: 'nav.auditLogs',
		icon: (
			<svg
				aria-hidden="true"
				className="w-5 h-5"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					strokeLinecap="round"
					strokeLinejoin="round"
					strokeWidth={2}
					d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
				/>
			</svg>
		),
	},
	{
		path: '/stats',
		labelKey: 'nav.storageStats',
		icon: (
			<svg
				aria-hidden="true"
				className="w-5 h-5"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					strokeLinecap="round"
					strokeLinejoin="round"
					strokeWidth={2}
					d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
				/>
			</svg>
		),
	},
	{
		path: '/classifications',
		labelKey: 'nav.classifications',
		icon: (
			<svg
				aria-hidden="true"
				className="w-5 h-5"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					strokeLinecap="round"
					strokeLinejoin="round"
					strokeWidth={2}
					d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
				/>
			</svg>
		),
	},
	{
		path: '/costs',
		labelKey: 'nav.costs',
		icon: (
			<svg
				aria-hidden="true"
				className="w-5 h-5"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					strokeLinecap="round"
					strokeLinejoin="round"
					strokeWidth={2}
					d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
				/>
			</svg>
		),
	},
];

function Sidebar() {
	const location = useLocation();
	const { data: user } = useMe();
	const { t } = useLocale();
	const { hasNewVersion, latestVersion } = useNewVersionAvailable();
	const isAdmin =
		user?.current_org_role === 'owner' || user?.current_org_role === 'admin';

	return (
		<aside className="w-64 bg-gray-900 text-white flex flex-col">
			<div className="p-6">
				<h1 className="text-2xl font-bold">{t('common.appName')}</h1>
				<p className="text-gray-400 text-sm">{t('common.tagline')}</p>
			</div>
			<nav className="flex-1 px-4">
				<ul className="space-y-1">
					{navItems.map((item) => {
						const isActive =
							location.pathname === item.path ||
							(item.path !== '/' && location.pathname.startsWith(item.path));
						return (
							<li key={item.path}>
								<Link
									to={item.path}
									title={
										item.shortcut
											? `${t(item.labelKey)} (${item.shortcut})`
											: t(item.labelKey)
									}
									className={`flex items-center justify-between gap-3 px-4 py-3 rounded-lg transition-colors ${
										isActive
											? 'bg-indigo-600 text-white'
											: 'text-gray-300 hover:bg-gray-800 hover:text-white'
									}`}
								>
									<span className="flex items-center gap-3">
										{item.icon}
										<span>{t(item.labelKey)}</span>
									</span>
									{item.shortcut && (
										<span
											className={`text-xs font-mono ${isActive ? 'text-indigo-200' : 'text-gray-500'}`}
										>
											{item.shortcut}
										</span>
									)}
								</Link>
							</li>
						);
					})}
				</ul>
				{isAdmin && (
					<>
						<div className="mt-6 mb-2 px-4 text-xs font-semibold text-gray-500 uppercase tracking-wider">
							{t('nav.organization')}
						</div>
						<ul className="space-y-1">
							<li>
								<Link
									to="/organization/members"
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										location.pathname === '/organization/members'
											? 'bg-indigo-600 text-white'
											: 'text-gray-300 hover:bg-gray-800 hover:text-white'
									}`}
								>
									<svg
										aria-hidden="true"
										className="w-5 h-5"
										fill="none"
										stroke="currentColor"
										viewBox="0 0 24 24"
									>
										<path
											strokeLinecap="round"
											strokeLinejoin="round"
											strokeWidth={2}
											d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z"
										/>
									</svg>
									<span>{t('nav.members')}</span>
								</Link>
							</li>
							<li>
								<Link
									to="/organization/settings"
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										location.pathname === '/organization/settings'
											? 'bg-indigo-600 text-white'
											: 'text-gray-300 hover:bg-gray-800 hover:text-white'
									}`}
								>
									<svg
										aria-hidden="true"
										className="w-5 h-5"
										fill="none"
										stroke="currentColor"
										viewBox="0 0 24 24"
									>
										<path
											strokeLinecap="round"
											strokeLinejoin="round"
											strokeWidth={2}
											d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
										/>
										<path
											strokeLinecap="round"
											strokeLinejoin="round"
											strokeWidth={2}
											d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
										/>
									</svg>
									<span>{t('nav.settings')}</span>
								</Link>
							</li>
							<li>
								<Link
									to="/organization/sso"
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										location.pathname === '/organization/sso'
											? 'bg-indigo-600 text-white'
											: 'text-gray-300 hover:bg-gray-800 hover:text-white'
									}`}
								>
									<svg
										aria-hidden="true"
										className="w-5 h-5"
										fill="none"
										stroke="currentColor"
										viewBox="0 0 24 24"
									>
										<path
											strokeLinecap="round"
											strokeLinejoin="round"
											strokeWidth={2}
											d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"
										/>
									</svg>
									<span>SSO Group Sync</span>
								</Link>
							</li>
							<li>
								<Link
									to="/organization/announcements"
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										location.pathname === '/organization/announcements'
											? 'bg-indigo-600 text-white'
											: 'text-gray-300 hover:bg-gray-800 hover:text-white'
									}`}
								>
									<svg
										aria-hidden="true"
										className="w-5 h-5"
										fill="none"
										stroke="currentColor"
										viewBox="0 0 24 24"
									>
										<path
											strokeLinecap="round"
											strokeLinejoin="round"
											strokeWidth={2}
											d="M11 5.882V19.24a1.76 1.76 0 01-3.417.592l-2.147-6.15M18 13a3 3 0 100-6M5.436 13.683A4.001 4.001 0 017 6h1.832c4.1 0 7.625-1.234 9.168-3v14c-1.543-1.766-5.067-3-9.168-3H7a3.988 3.988 0 01-1.564-.317z"
										/>
									</svg>
									<span>Announcements</span>
								</Link>
							</li>
							<li>
								<Link
									to="/organization/ip-allowlist"
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										location.pathname === '/organization/ip-allowlist'
											? 'bg-indigo-600 text-white'
											: 'text-gray-300 hover:bg-gray-800 hover:text-white'
									}`}
								>
									<svg
										aria-hidden="true"
										className="w-5 h-5"
										fill="none"
										stroke="currentColor"
										viewBox="0 0 24 24"
									>
										<path
											strokeLinecap="round"
											strokeLinejoin="round"
											strokeWidth={2}
											d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
										/>
									</svg>
									<span>IP Allowlist</span>
								</Link>
							</li>
							<li>
								<Link
									to="/legal-holds"
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										location.pathname === '/legal-holds'
											? 'bg-indigo-600 text-white'
											: 'text-gray-300 hover:bg-gray-800 hover:text-white'
									}`}
								>
									<svg
										aria-hidden="true"
										className="w-5 h-5"
										fill="none"
										stroke="currentColor"
										viewBox="0 0 24 24"
									>
										<path
											strokeLinecap="round"
											strokeLinejoin="round"
											strokeWidth={2}
											d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"
										/>
									</svg>
									<span>Legal Holds</span>
								</Link>
							</li>
							<li>
								<Link
									to="/admin/logs"
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										location.pathname === '/admin/logs'
											? 'bg-indigo-600 text-white'
											: 'text-gray-300 hover:bg-gray-800 hover:text-white'
									}`}
								>
									<svg
										aria-hidden="true"
										className="w-5 h-5"
										fill="none"
										stroke="currentColor"
										viewBox="0 0 24 24"
									>
										<path
											strokeLinecap="round"
											strokeLinejoin="round"
											strokeWidth={2}
											d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
										/>
									</svg>
									<span>Server Logs</span>
								</Link>
							</li>
						</ul>
					</>
				)}
			</nav>
			<div className="p-4 border-t border-gray-800">
				<Link
					to="/changelog"
					className="flex items-center gap-2 text-xs text-gray-500 hover:text-gray-300 transition-colors"
				>
					<span>{t('common.version', { version: '0.0.1' })}</span>
					{hasNewVersion && (
						<span className="px-1.5 py-0.5 text-[10px] font-medium bg-indigo-600 text-white rounded-full">
							v{latestVersion} available
						</span>
					)}
				</Link>
			</div>
		</aside>
	);
}

function OrgSwitcher() {
	const [showDropdown, setShowDropdown] = useState(false);
	const { data: organizations, isLoading } = useOrganizations();
	const { data: user } = useMe();
	const switchOrg = useSwitchOrganization();
	const { t } = useLocale();

	const currentOrg = organizations?.find(
		(org) => org.id === user?.current_org_id,
	);

	const handleSwitch = (orgId: string) => {
		if (orgId !== user?.current_org_id) {
			switchOrg.mutate(orgId);
		}
		setShowDropdown(false);
	};

	if (isLoading || !organizations?.length) {
		return null;
	}

	return (
		<div className="relative">
			<button
				type="button"
				onClick={() => setShowDropdown(!showDropdown)}
				className="flex items-center gap-2 px-3 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50"
			>
				<svg
					aria-hidden="true"
					className="w-4 h-4 text-gray-500"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"
					/>
				</svg>
				<span className="max-w-32 truncate">
					{currentOrg?.name ?? t('common.selectOrg')}
				</span>
				<svg
					aria-hidden="true"
					className="w-4 h-4 text-gray-400"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M19 9l-7 7-7-7"
					/>
				</svg>
			</button>
			{showDropdown && (
				<div className="absolute left-0 mt-2 w-56 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-50">
					<div className="px-3 py-2 text-xs font-semibold text-gray-500 uppercase">
						{t('org.organizations')}
					</div>
					{organizations.map((org) => (
						<button
							key={org.id}
							type="button"
							onClick={() => handleSwitch(org.id)}
							className={`w-full text-left px-3 py-2 text-sm hover:bg-gray-100 flex items-center justify-between ${
								org.id === user?.current_org_id
									? 'bg-indigo-50 text-indigo-700'
									: 'text-gray-700'
							}`}
						>
							<span className="truncate">{org.name}</span>
							<span className="text-xs text-gray-400 capitalize">
								{org.role}
							</span>
						</button>
					))}
					<div className="border-t border-gray-100 mt-1 pt-1">
						<Link
							to="/organization/new"
							onClick={() => setShowDropdown(false)}
							className="w-full text-left px-3 py-2 text-sm text-indigo-600 hover:bg-gray-100 flex items-center gap-2"
						>
							<svg
								aria-hidden="true"
								className="w-4 h-4"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M12 4v16m8-8H4"
								/>
							</svg>
							{t('org.createOrganization')}
						</Link>
					</div>
				</div>
			)}
		</div>
	);
}

function Header() {
	const [showDropdown, setShowDropdown] = useState(false);
	const { data: user } = useMe();
	const { data: alertCount } = useAlertCount();
	const logout = useLogout();
	const { t } = useLocale();

	const userInitial =
		user?.name?.charAt(0).toUpperCase() ??
		user?.email?.charAt(0).toUpperCase() ??
		'U';

	return (
		<header className="h-16 bg-white border-b border-gray-200 flex items-center justify-between px-6">
			<div className="flex items-center gap-4">
				<OrgSwitcher />
			</div>
			<div className="flex items-center gap-4">
				<LanguageSelector />
				<Link
					to="/alerts"
					aria-label={t('nav.alerts')}
					className="relative p-2 text-gray-500 hover:text-gray-700 rounded-lg hover:bg-gray-100"
				>
					<svg
						aria-hidden="true"
						className="w-5 h-5"
						fill="none"
						stroke="currentColor"
						viewBox="0 0 24 24"
					>
						<path
							strokeLinecap="round"
							strokeLinejoin="round"
							strokeWidth={2}
							d="M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9"
						/>
					</svg>
					{alertCount !== undefined && alertCount > 0 && (
						<span className="absolute -top-0.5 -right-0.5 flex h-5 min-w-5 items-center justify-center rounded-full bg-red-500 px-1.5 text-xs font-medium text-white">
							{alertCount > 99 ? '99+' : alertCount}
						</span>
					)}
				</Link>
				<div className="relative">
					<button
						type="button"
						onClick={() => setShowDropdown(!showDropdown)}
						className="flex items-center gap-2"
					>
						<div className="w-8 h-8 bg-indigo-600 rounded-full flex items-center justify-center text-white text-sm font-medium">
							{userInitial}
						</div>
					</button>
					{showDropdown && (
						<div className="absolute right-0 mt-2 w-48 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-50">
							{user && (
								<div className="px-4 py-2 border-b border-gray-100">
									<p className="text-sm font-medium text-gray-900 truncate">
										{user.name}
									</p>
									<p className="text-xs text-gray-500 truncate">{user.email}</p>
								</div>
							)}
							<button
								type="button"
								onClick={() => logout.mutate()}
								className="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
							>
								{t('common.signOut')}
							</button>
						</div>
					)}
				</div>
			</div>
		</header>
	);
}

function LoadingScreen() {
	const { t } = useLocale();
	return (
		<div className="min-h-screen bg-gray-50 flex items-center justify-center">
			<div className="text-center">
				<div className="w-12 h-12 border-4 border-indigo-200 border-t-indigo-600 rounded-full animate-spin mx-auto mb-4" />
				<p className="text-gray-600">{t('common.loading')}</p>
			</div>
		</div>
	);
}

export function Layout() {
	const location = useLocation();
	const navigate = useNavigate();
	const { isLoading, isError } = useMe();
	const { data: onboardingStatus, isLoading: onboardingLoading } =
		useOnboardingStatus();
	const { latestEntry, currentVersion } = useLatestChanges();
	const [showWhatsNew, setShowWhatsNew] = useState(true);
	const readOnlyModeValue = useReadOnlyModeValue();
	const [showShortcutHelp, setShowShortcutHelp] = useState(false);

	const { shortcuts } = useKeyboardShortcuts({
		onShowHelp: () => setShowShortcutHelp(true),
		onCloseModal: () => setShowShortcutHelp(false),
		enabled: !isLoading && !isError,
	});

	// Redirect to onboarding if needed (but not if already on onboarding page)
	useEffect(() => {
		if (
			!onboardingLoading &&
			onboardingStatus?.needs_onboarding &&
			location.pathname !== '/onboarding'
		) {
			navigate('/onboarding');
		}
	}, [
		onboardingLoading,
		onboardingStatus?.needs_onboarding,
		location.pathname,
		navigate,
	]);

	// Show loading state while checking auth
	if (isLoading || onboardingLoading) {
		return <LoadingScreen />;
	}

	// If auth check failed, the API client will redirect to login
	// But we show a loading state just in case
	if (isError) {
		return <LoadingScreen />;
	}

	return (
		<ReadOnlyModeContext.Provider value={readOnlyModeValue}>
			<div className="min-h-screen bg-gray-50 flex flex-col">
				<MaintenanceCountdown />
				<AnnouncementBanner />
				<div className="flex flex-1">
					<Sidebar />
					<div className="flex-1 flex flex-col">
						<Header />
						<main className="flex-1 p-6">
							<Outlet />
						</main>
					</div>
				</div>
				<ShortcutHelpModal
					isOpen={showShortcutHelp}
					onClose={() => setShowShortcutHelp(false)}
					shortcuts={shortcuts}
				/>
				{showWhatsNew && (
					<WhatsNewModal
						entry={latestEntry ?? null}
						currentVersion={currentVersion}
						onDismiss={() => setShowWhatsNew(false)}
					/>
				)}
			</div>
		</ReadOnlyModeContext.Provider>
	);
}
