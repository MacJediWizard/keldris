import { useEffect, useState } from 'react';
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useAlertCount } from '../hooks/useAlerts';
import { useLogout, useMe } from '../hooks/useAuth';
import { useLocale } from '../hooks/useLocale';
import { useOnboardingStatus } from '../hooks/useOnboarding';
import {
	useOrganizations,
	useSwitchOrganization,
} from '../hooks/useOrganizations';
import { LanguageSelector } from './features/LanguageSelector';

interface NavItem {
	path: string;
	labelKey: string;
	icon: React.ReactNode;
}

const navItems: NavItem[] = [
	{
		path: '/',
		labelKey: 'nav.dashboard',
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
		path: '/alerts',
		labelKey: 'nav.alerts',
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
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										isActive
											? 'bg-indigo-600 text-white'
											: 'text-gray-300 hover:bg-gray-800 hover:text-white'
									}`}
								>
									{item.icon}
									<span>{t(item.labelKey)}</span>
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
						</ul>
					</>
				)}
			</nav>
			<div className="p-4 border-t border-gray-800">
				<p className="text-xs text-gray-500">
					{t('common.version', { version: '0.0.1' })}
				</p>
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
		<div className="min-h-screen bg-gray-50 flex">
			<Sidebar />
			<div className="flex-1 flex flex-col">
				<Header />
				<main className="flex-1 p-6">
					<Outlet />
				</main>
			</div>
		</div>
	);
}
