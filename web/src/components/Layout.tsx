import { useEffect, useState } from 'react';
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useAirGapStatus } from '../hooks/useAirGap';
import { useBranding } from '../contexts/BrandingContext';
import { useEffect, useRef, useState } from 'react';
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useBranding } from '../contexts/BrandingContext';
import { useAlertCount } from '../hooks/useAlerts';
import { useLogout, useMe } from '../hooks/useAuth';
import { useBranding } from '../hooks/useBranding';
import { useLicense } from '../hooks/useLicense';
import {
	useLatestChanges,
	useNewVersionAvailable,
} from '../hooks/useChangelog';
import { useKeyboardShortcuts } from '../hooks/useKeyboardShortcuts';
import { useLocale } from '../hooks/useLocale';
import { useTheme } from '../hooks/useTheme';
import { useVersion } from '../hooks/useVersion';
import { useOnboardingStatus } from '../hooks/useOnboarding';
import { useState } from 'react';
import { Link, Outlet, useLocation } from 'react-router-dom';
import { useAlertCount } from '../hooks/useAlerts';
import { useLogout, useMe } from '../hooks/useAuth';
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
import { PasswordExpirationBanner } from './PasswordExpirationBanner';
import { AirGapIndicator } from './features/AirGapIndicator';
import { AnnouncementBanner } from './features/AnnouncementBanner';
import { GlobalSearchBar } from './features/GlobalSearchBar';
import { AnnouncementBanner } from './features/AnnouncementBanner';
import { LanguageSelector } from './features/LanguageSelector';
import { TierBadge } from './features/TierBadge';
import { LicenseBanner } from './features/LicenseBanner';
import { MaintenanceCountdown } from './features/MaintenanceCountdown';
import { RecentItemsDropdown } from './features/RecentItems';
import { AnnouncementBanner } from './features/AnnouncementBanner';
import { LanguageSelector } from './features/LanguageSelector';
import { MaintenanceCountdown } from './features/MaintenanceCountdown';
import { ShortcutHelpModal } from './features/ShortcutHelpModal';
import { TrialBanner } from './features/TrialBanner';
import { WhatsNewModal } from './features/WhatsNewModal';
import { Breadcrumbs } from './ui/Breadcrumbs';
import { useSearch } from '../hooks/useSearch';
import type { SearchResult, SearchResultType } from '../lib/types';
import { WhatsNewModal } from './features/WhatsNewModal';
import { useState } from 'react';
import { Link, Outlet, useLocation } from 'react-router-dom';
import { useLogout, useMe } from '../hooks/useAuth';
import { LanguageSelector } from './features/LanguageSelector';
import { ToastProvider } from './ui/Toast';
import { ShortcutHelpModal } from './features/ShortcutHelpModal';
import { MaintenanceBanner } from './features/MaintenanceBanner';
import {
	useOrganizations,
	useSwitchOrganization,
} from '../hooks/useOrganizations';

interface NavItem {
	path: string;
	labelKey: string;
	icon: React.ReactNode;
	shortcut?: string;
import { Link, Outlet, useLocation } from 'react-router-dom';

interface NavItem {
	path: string;
	label: string;
	icon: React.ReactNode;
}

const navItems: NavItem[] = [
	{
		path: '/',
		labelKey: 'nav.dashboard',
		shortcut: 'G D',
		label: 'Dashboard',
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
		label: 'Agents',
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
		label: 'Repositories',
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
		label: 'Schedules',
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
		label: 'Backups',
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
		label: 'Restore',
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
		path: '/alerts',
		label: 'Alerts',
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
		path: '/downtime',
		labelKey: 'nav.downtime',
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
		path: '/notifications',
		labelKey: 'nav.notifications',
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
		label: 'Storage Stats',
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
		path: '/tags',
		label: 'Tags',
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
					d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"
				/>
			</svg>
		),
	},
];

function Sidebar() {
	const location = useLocation();
	const { data: user } = useMe();
	const { t } = useLocale();
	const { data: brandingData } = useBranding();
	const { data: onboardingStatus } = useOnboardingStatus();
	const { data: license } = useLicense();
	const { data: versionInfo } = useVersion();
	const { theme, toggleTheme } = useTheme();
	const { hasNewVersion, latestVersion } = useNewVersionAvailable();
	const { productName, logoUrl, branding } = useBranding();
	const { hasNewVersion, latestVersion } = useNewVersionAvailable();
	const { productName, logoUrl, branding } = useBranding();
	const isAdmin =
		user?.current_org_role === 'owner' || user?.current_org_role === 'admin';

	const displayName = brandingData?.product_name || t('common.appName');
	const isAdmin =
		user?.current_org_role === 'owner' || user?.current_org_role === 'admin';

	return (
		<aside className="w-64 bg-gray-900 text-white flex flex-col">
			<div className="p-6">
				{brandingData?.logo_url ? (
					<img
						src={brandingData.logo_url}
						alt={displayName}
						className="h-8 mb-1 object-contain"
					/>
				) : (
					<h1 className="text-2xl font-bold">{displayName}</h1>
				{logoUrl && branding?.enabled ? (
					<img src={logoUrl} alt={productName} className="h-8 mb-2" />
				) : (
					<h1 className="text-2xl font-bold">
						{branding?.enabled ? productName : t('common.appName')}
					</h1>
				)}
				<h1 className="text-2xl font-bold">{t('common.appName')}</h1>
				<p className="text-gray-400 text-sm">{t('common.tagline')}</p>
				<h1 className="text-2xl font-bold">Keldris</h1>
				<p className="text-gray-400 text-sm">Keeper of your data</p>
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
									to="/organization/license"
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										location.pathname === '/organization/license'
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
									<span>License</span>
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
									to="/organization/branding"
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										location.pathname === '/organization/branding'
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
											d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z"
										/>
									</svg>
									<span>Branding</span>
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
									to="/organization/password-policies"
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										location.pathname === '/organization/password-policies'
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
									<span>Password Policy</span>
								</Link>
							</li>
							<li>
								<Link
									to="/organization/branding"
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										location.pathname === '/organization/branding'
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
											d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z"
										/>
									</svg>
									<span>Branding</span>
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
									to="/lifecycle-policies"
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										location.pathname === '/lifecycle-policies'
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
											d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
										/>
									</svg>
									<span>Lifecycle Policies</span>
								</Link>
							</li>
							<li>
								<Link
									to="/organization/docker-registries"
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										location.pathname === '/organization/docker-registries'
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
											d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"
										/>
									</svg>
									<span>Docker Registries</span>
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
							<li>
								<Link
									to="/admin/docker-logs"
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										location.pathname === '/admin/docker-logs'
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
											d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"
										/>
									</svg>
									<span>Docker Logs</span>
								</Link>
							</li>
							<li>
								<Link
									to="/admin/rate-limits"
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										location.pathname === '/admin/rate-limits'
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
											d="M13 10V3L4 14h7v7l9-11h-7z"
										/>
									</svg>
									<span>Rate Limits</span>
								</Link>
							</li>
						</ul>
					</>
				)}
				{user?.is_superuser && (
					<>
						<div className="mt-6 mb-2 px-4 text-xs font-semibold text-gray-500 uppercase tracking-wider">
							Superuser
						</div>
						<ul className="space-y-1">
							<li>
								<Link
									to="/admin/organizations"
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										location.pathname === '/admin/organizations'
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										isActive
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
											d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"
										/>
									</svg>
									<span>Organizations</span>
								</Link>
							</li>
									{item.icon}
									<span>{t(item.labelKey)}</span>
									{item.icon}
									<span>{item.label}</span>
								</Link>
							</li>
						);
					})}
				</ul>
				{isAdmin && (
					<>
						<div className="mt-6 mb-2 px-4 text-xs font-semibold text-gray-500 uppercase tracking-wider">
							{t('nav.organization')}
							Organization
						</div>
						<ul className="space-y-1">
							<li>
								<Link
									to="/admin/license"
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										location.pathname === '/admin/license'
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
											d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
										/>
									</svg>
									<span>License</span>
									<span>{t('nav.members')}</span>
											d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197M13 7a4 4 0 11-8 0 4 4 0 018 0z"
										/>
									</svg>
									<span>Members</span>
								</Link>
							</li>
							<li>
								<Link
									to="/admin/setup"
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										location.pathname === '/admin/setup'
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
									<span>Server Setup</span>
								</Link>
							</li>
							<li>
								<Link
									to="/admin/migration"
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										location.pathname === '/admin/migration'
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
											d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"
										/>
									</svg>
									<span>Migration</span>
									<span>{t('nav.settings')}</span>
								</Link>
							</li>
							<li>
								<Link
									to="/organization/maintenance"
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										location.pathname === '/organization/maintenance'
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
											d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
										/>
									</svg>
									<span>Maintenance</span>
									<span>Settings</span>
											d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"
										/>
									</svg>
									<span>SSO Group Sync</span>
								</Link>
							</li>
							<li>
								<Link
									to="/admin/setup"
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										location.pathname === '/admin/setup'
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
									<span>Server Setup</span>
								</Link>
							</li>
						</ul>
					</>
				)}
			</nav>
			{onboardingStatus?.needs_onboarding &&
				location.pathname !== '/onboarding' && (
					<div className="px-4 pb-2">
						<Link
							to="/onboarding"
							className="flex items-center gap-2 px-3 py-2 bg-indigo-600 rounded-lg text-sm font-medium text-white hover:bg-indigo-500 transition-colors"
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
									d="M9 5l7 7-7 7"
								/>
							</svg>
							Continue Setup
						</Link>
					</div>
				)}
			<div className="p-4 border-t border-gray-800">
				<div className="flex items-center justify-between mb-2">
					{license && (
						<Link to="/license">
							<TierBadge tier={license.tier} />
						</Link>
					)}
					<button
						type="button"
						onClick={toggleTheme}
						title={`Theme: ${theme}`}
						className="p-1.5 rounded-lg text-gray-400 hover:text-white hover:bg-gray-800 transition-colors"
					>
						{theme === 'light' ? (
							<svg aria-hidden="true" className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z" />
							</svg>
						) : theme === 'dark' ? (
							<svg aria-hidden="true" className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z" />
							</svg>
						) : (
							<svg aria-hidden="true" className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
							</svg>
						)}
					</button>
				</div>
				<p className="text-xs text-gray-500">
					{t('common.version', { version: versionInfo?.version ?? '...' })}
				</p>
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
			<div className="p-4 border-t border-gray-800">
				<p className="text-xs text-gray-500">
					{t('common.version', { version: '0.0.1' })}
				</p>
			<div className="p-4 border-t border-gray-800">
				<p className="text-xs text-gray-500">
					{t('common.version', { version: '0.0.1' })}
				</p>
			</nav>
			<div className="p-4 border-t border-gray-800">
				<p className="text-xs text-gray-500">v0.0.1</p>
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
				className="flex items-center gap-2 px-3 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 dark:text-gray-200 dark:bg-gray-800 dark:border-gray-600 dark:hover:bg-gray-700"
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
					{currentOrg?.name ?? 'Select org'}
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
				<div className="absolute left-0 mt-2 w-56 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 z-50">
				<div className="absolute left-0 mt-2 w-56 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-50">
					<div className="px-3 py-2 text-xs font-semibold text-gray-500 uppercase">
						{t('org.organizations')}
				<div className="absolute left-0 mt-2 w-56 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-50">
					<div className="px-3 py-2 text-xs font-semibold text-gray-500 uppercase">
						Organizations
				<div className="absolute left-0 mt-2 w-56 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-50">
					<div className="px-3 py-2 text-xs font-semibold text-gray-500 uppercase">
						{t('org.organizations')}
				<div className="absolute left-0 mt-2 w-56 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-50">
					<div className="px-3 py-2 text-xs font-semibold text-gray-500 uppercase">
						Organizations
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
									? 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300'
									: 'text-gray-700 dark:text-gray-200'
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
							className="w-full text-left px-3 py-2 text-sm text-indigo-600 hover:bg-gray-100 dark:text-indigo-400 dark:hover:bg-gray-700 flex items-center gap-2"
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
							Create organization
						</Link>
					</div>
				</div>
			)}
		</div>
	);
}

function AirGapIndicator() {
	const { data: status } = useAirGapStatus();
	const { t } = useLocale();

	if (!status?.enabled) return null;

	return (
		<Link
			to="/system/airgap"
			className="flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium text-amber-700 bg-amber-50 border border-amber-200 rounded-full hover:bg-amber-100 transition-colors dark:text-amber-300 dark:bg-amber-900/30 dark:border-amber-700 dark:hover:bg-amber-900/50"
			className="flex items-center gap-1.5 px-2.5 py-1 text-xs font-medium text-amber-700 bg-amber-50 border border-amber-200 rounded-full hover:bg-amber-100 transition-colors"
		>
			<svg
				aria-hidden="true"
				className="w-3.5 h-3.5"
				fill="none"
				stroke="currentColor"
				viewBox="0 0 24 24"
			>
				<path
					strokeLinecap="round"
					strokeLinejoin="round"
					strokeWidth={2}
					d="M18.364 5.636a9 9 0 010 12.728M5.636 5.636a9 9 0 000 12.728M12 12h.01"
				/>
			</svg>
			{t('airGap.indicator')}
		</Link>
	);
}

function Header() {
function HelpMenu({ onShowShortcuts }: { onShowShortcuts: () => void }) {
	const [showDropdown, setShowDropdown] = useState(false);
	const { t } = useLocale();

	const helpLinks = [
		{ path: '/docs/getting-started', label: t('help.gettingStarted') },
		{ path: '/docs/configuration', label: t('help.configuration') },
		{ path: '/docs/agent-deployment', label: t('help.agentDeployment') },
		{ path: '/docs/api-reference', label: t('help.apiReference') },
		{ path: '/docs/troubleshooting', label: t('help.troubleshooting') },
	];

	return (
		<div className="relative">
			<button
				type="button"
				onClick={() => setShowDropdown(!showDropdown)}
				className="p-2 text-gray-500 hover:text-gray-700 rounded-lg hover:bg-gray-100"
				aria-label={t('help.title')}
			>
				<svg
					aria-hidden="true"
					className="w-5 h-5"
function getSearchResultIcon(type: SearchResultType) {
	switch (type) {
		case 'agent':
			return (
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
						d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
					/>
				</svg>
			);
		case 'backup':
			return (
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
						d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"
					/>
				</svg>
			);
		case 'schedule':
			return (
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
						d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				</svg>
			);
		case 'repository':
			return (
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
						d="M5 8h14M5 8a2 2 0 110-4h14a2 2 0 110 4M5 8v10a2 2 0 002 2h10a2 2 0 002-2V8m-9 4h4"
					/>
				</svg>
			);
		default:
			return (
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
						d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
					/>
				</svg>
			);
	}
}

function getSearchResultPath(result: SearchResult): string {
	switch (result.type) {
		case 'agent':
			return '/agents';
		case 'backup':
			return '/backups';
		case 'schedule':
			return '/schedules';
		case 'repository':
			return '/repositories';
		default:
			return '/';
	}
}

function GlobalSearch() {
	const [query, setQuery] = useState('');
	const [isOpen, setIsOpen] = useState(false);
	const inputRef = useRef<HTMLInputElement>(null);
	const dropdownRef = useRef<HTMLDivElement>(null);
	const navigate = useNavigate();

	const { data: searchData, isLoading } = useSearch(
		query.length >= 2 ? { q: query, limit: 5 } : null,
	);

	useEffect(() => {
		function handleClickOutside(event: MouseEvent) {
			if (
				dropdownRef.current &&
				!dropdownRef.current.contains(event.target as Node)
			) {
				setIsOpen(false);
			}
		}
		document.addEventListener('mousedown', handleClickOutside);
		return () => document.removeEventListener('mousedown', handleClickOutside);
	}, []);

	useEffect(() => {
		function handleKeyDown(event: KeyboardEvent) {
			if ((event.metaKey || event.ctrlKey) && event.key === 'k') {
				event.preventDefault();
				inputRef.current?.focus();
				setIsOpen(true);
			}
			if (event.key === 'Escape') {
				setIsOpen(false);
				inputRef.current?.blur();
			}
		}
		document.addEventListener('keydown', handleKeyDown);
		return () => document.removeEventListener('keydown', handleKeyDown);
	}, []);

	const handleResultClick = (result: SearchResult) => {
		setIsOpen(false);
		setQuery('');
		navigate(getSearchResultPath(result));
	};

	return (
		<div className="relative" ref={dropdownRef}>
			<div className="relative">
				<svg
					aria-hidden="true"
					className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				</svg>
			</button>
			{showDropdown && (
				<div className="absolute right-0 mt-2 w-64 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-50">
					<div className="px-4 py-2 border-b border-gray-100">
						<p className="text-sm font-semibold text-gray-900">
							{t('help.title')}
						</p>
					</div>
					<div className="py-1">
						{helpLinks.map((link) => (
							<Link
								key={link.path}
								to={link.path}
								onClick={() => setShowDropdown(false)}
								className="block px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
							>
								{link.label}
							</Link>
						))}
					</div>
					<div className="border-t border-gray-100 py-1">
						<button
							type="button"
							onClick={() => {
								setShowDropdown(false);
								onShowShortcuts();
							}}
							className="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 flex items-center justify-between"
						>
							<span>{t('help.keyboardShortcuts')}</span>
							<span className="text-xs text-gray-400 font-mono">?</span>
						</button>
						<a
							href="/api/docs"
							target="_blank"
							rel="noopener noreferrer"
							onClick={() => setShowDropdown(false)}
							className="block px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 flex items-center gap-2"
						>
							{t('help.swaggerDocs')}
							<svg
								aria-hidden="true"
								className="w-3 h-3 text-gray-400"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
								/>
							</svg>
						</a>
					</div>
						d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
					/>
				</svg>
				<input
					ref={inputRef}
					type="text"
					value={query}
					onChange={(e) => {
						setQuery(e.target.value);
						setIsOpen(true);
					}}
					onFocus={() => setIsOpen(true)}
					placeholder="Search... (Cmd+K)"
					className="w-64 pl-9 pr-4 py-2 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
				/>
			</div>

			{isOpen && query.length >= 2 && (
				<div className="absolute top-full left-0 mt-2 w-80 bg-white rounded-lg shadow-lg border border-gray-200 py-2 z-50">
					{isLoading ? (
						<div className="px-4 py-3 text-sm text-gray-500">Searching...</div>
					) : searchData?.results && searchData.results.length > 0 ? (
						<>
							<div className="px-4 py-1 text-xs font-semibold text-gray-500 uppercase">
								{searchData.total} result{searchData.total !== 1 ? 's' : ''}
							</div>
							{searchData.results.map((result, index) => (
								<button
									key={`${result.type}-${result.id}-${index}`}
									type="button"
									onClick={() => handleResultClick(result)}
									className="w-full text-left px-4 py-2 hover:bg-gray-100 flex items-center gap-3"
								>
									<span className="text-gray-400">
										{getSearchResultIcon(result.type)}
									</span>
									<div className="flex-1 min-w-0">
										<p className="text-sm font-medium text-gray-900 truncate">
											{result.name}
										</p>
										<p className="text-xs text-gray-500 truncate">
											{result.type}
											{result.status && ` - ${result.status}`}
										</p>
									</div>
								</button>
							))}
						</>
					) : (
						<div className="px-4 py-3 text-sm text-gray-500">
							No results found for "{query}"
						</div>
					)}
				</div>
			)}
		</div>
	);
}

function Header({ onShowShortcuts }: { onShowShortcuts: () => void }) {
function Header() {
	const [showDropdown, setShowDropdown] = useState(false);
	const { data: user } = useMe();
	const { data: alertCount } = useAlertCount();
	const logout = useLogout();
	const { t } = useLocale();
	const logout = useLogout();

	const userInitial =
		user?.name?.charAt(0).toUpperCase() ??
		user?.email?.charAt(0).toUpperCase() ??
		'U';

	const isSuperuser = user?.is_superuser;
	const isImpersonating = user?.is_impersonating;

	return (
		<header className="h-16 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between px-6">
	return (
		<header className="h-16 bg-white border-b border-gray-200 flex items-center justify-between px-6">
			<div className="flex items-center gap-4">
				<OrgSwitcher />
				<AirGapIndicator />
			</div>
			<div className="flex-1 max-w-xl mx-4">
				<GlobalSearchBar placeholder={t('common.search')} />
				{isImpersonating && (
					<div className="flex items-center gap-2 px-3 py-1.5 bg-amber-100 border border-amber-300 rounded-lg">
						<svg
							aria-hidden="true"
							className="w-4 h-4 text-amber-600"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
							/>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
							/>
						</svg>
						<span className="text-sm font-medium text-amber-800">
							Impersonating
						</span>
					</div>
				)}
			</div>
			<div className="flex-1 max-w-xl mx-4">
				<GlobalSearchBar placeholder={t('common.search')} />
			</div>
			<div className="flex items-center gap-4">
				{isSuperuser && (
					<div
						className="flex items-center gap-1.5 px-2.5 py-1 bg-purple-100 border border-purple-200 rounded-full"
						title="Superuser"
					>
						<svg
							aria-hidden="true"
							className="w-4 h-4 text-purple-600"
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
						<span className="text-xs font-semibold text-purple-700">
							Superuser
						</span>
					</div>
				)}
				<AirGapIndicator showDetails />
				<LanguageSelector />
				<RecentItemsDropdown />
				<HelpMenu onShowShortcuts={onShowShortcuts} />
				<Link
					to="/alerts"
					aria-label={t('nav.alerts')}
					className="relative p-2 text-gray-500 hover:text-gray-700 rounded-lg hover:bg-gray-100 dark:text-gray-400 dark:hover:text-gray-200 dark:hover:bg-gray-700"
			<div className="flex items-center gap-4">
				<LanguageSelector />
				<Link
					to="/alerts"
					aria-label={t('nav.alerts')}
				<GlobalSearch />
			</div>
			<div className="flex items-center gap-4">
				<Link
					to="/alerts"
					aria-label="Alerts"
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
				<Link
					to="/alerts"
					aria-label="Alerts"
					className="relative p-2 text-gray-500 hover:text-gray-700 rounded-lg hover:bg-gray-100"
function Header() {
	return (
		<header className="h-16 bg-white border-b border-gray-200 flex items-center justify-between px-6">
			<div className="flex items-center gap-4">
				<h2 className="text-lg font-semibold text-gray-900">
					Backup Management
				</h2>
			</div>
			<div className="flex items-center gap-4">
				<button
					type="button"
					aria-label="Notifications"
					className="p-2 text-gray-500 hover:text-gray-700 rounded-lg hover:bg-gray-100"
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
				</button>
				<div className="relative">
					<button
						type="button"
						onClick={() => setShowDropdown(!showDropdown)}
						className="flex items-center gap-2"
					>
						<div
							className={`w-8 h-8 rounded-full flex items-center justify-center text-white text-sm font-medium ${
								isSuperuser ? 'bg-purple-600' : 'bg-indigo-600'
							}`}
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
									<p className="text-xs text-gray-500 dark:text-gray-400 truncate">{user.email}</p>
						<div className="absolute right-0 mt-2 w-56 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-50">
							{user && (
								<div className="px-4 py-2 border-b border-gray-100">
									<div className="flex items-center gap-2">
										<p className="text-sm font-medium text-gray-900 truncate">
											{user.name}
										</p>
										{isSuperuser && (
											<span className="px-1.5 py-0.5 text-[10px] font-semibold bg-purple-100 text-purple-700 rounded">
												Superuser
											</span>
										)}
									</div>
									<p className="text-xs text-gray-500 truncate">{user.email}</p>
								</div>
							)}
							{isSuperuser && (
								<>
									<Link
										to="/superuser"
										onClick={() => setShowDropdown(false)}
										className="w-full text-left px-4 py-2 text-sm text-purple-700 hover:bg-purple-50 flex items-center gap-2"
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
												d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
											/>
										</svg>
										Superuser Console
									</Link>
									<div className="border-t border-gray-100 my-1" />
								</>
							)}
							<Link
								to="/account/sessions"
								onClick={() => setShowDropdown(false)}
								className="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 flex items-center gap-2"
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
										d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
									/>
								</svg>
								Active Sessions
							</Link>
							<a
								href="/portal"
								target="_blank"
								rel="noopener noreferrer"
								onClick={() => setShowDropdown(false)}
								className="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 flex items-center gap-2"
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
										d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"
									/>
								</svg>
								Manage License
							</a>
							<button
								type="button"
								onClick={() => logout.mutate()}
								className="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
							>
								{t('common.signOut')}
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
									<p className="text-xs text-gray-500 truncate">{user.email}</p>
								</div>
							)}
							<a
								href="/api/docs/index.html"
								target="_blank"
								rel="noopener noreferrer"
								className="block w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
							>
								API Documentation
							</a>
							<button
								type="button"
								onClick={() => logout.mutate()}
								className="w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
							>
								{t('common.signOut')}
							</button>
						</div>
					)}
				</button>
				<div className="w-8 h-8 bg-indigo-600 rounded-full flex items-center justify-center text-white text-sm font-medium">
					U
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
				<p className="text-gray-600 dark:text-gray-400">{t('common.loading')}</p>
	return (
		<div className="min-h-screen bg-gray-50 flex items-center justify-center">
			<div className="text-center">
				<div className="w-12 h-12 border-4 border-indigo-200 border-t-indigo-600 rounded-full animate-spin mx-auto mb-4" />
				<p className="text-gray-600">Loading...</p>
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
	const { data: brandingData } = useBranding();

	// Apply branding CSS variables and custom CSS
	useEffect(() => {
		if (!brandingData) return;

		const root = document.documentElement;
		if (brandingData.primary_color) {
			root.style.setProperty('--brand-primary', brandingData.primary_color);
		} else {
			root.style.removeProperty('--brand-primary');
		}
		if (brandingData.secondary_color) {
			root.style.setProperty('--brand-secondary', brandingData.secondary_color);
		} else {
			root.style.removeProperty('--brand-secondary');
		}

		// Apply custom CSS
		let styleEl = document.getElementById('keldris-custom-css');
		if (brandingData.custom_css) {
			if (!styleEl) {
				styleEl = document.createElement('style');
				styleEl.id = 'keldris-custom-css';
				document.head.appendChild(styleEl);
			}
			styleEl.textContent = brandingData.custom_css;
		} else if (styleEl) {
			styleEl.remove();
		}

		// Apply favicon
		if (brandingData.favicon_url) {
			let link = document.querySelector<HTMLLinkElement>("link[rel~='icon']");
			if (!link) {
				link = document.createElement('link');
				link.rel = 'icon';
				document.head.appendChild(link);
			}
			link.href = brandingData.favicon_url;
		}

		// Apply product name to document title
		if (brandingData.product_name) {
			document.title = brandingData.product_name;
		}

		return () => {
			root.style.removeProperty('--brand-primary');
			root.style.removeProperty('--brand-secondary');
		};
	}, [brandingData]);
	const { latestEntry, currentVersion } = useLatestChanges();
	const [showWhatsNew, setShowWhatsNew] = useState(true);
	const readOnlyModeValue = useReadOnlyModeValue();
	const [showShortcutHelp, setShowShortcutHelp] = useState(false);
	const readOnlyModeValue = useReadOnlyModeValue();
	const [showShortcutHelp, setShowShortcutHelp] = useState(false);

	const { shortcuts } = useKeyboardShortcuts({
		onShowHelp: () => setShowShortcutHelp(true),
		onCloseModal: () => setShowShortcutHelp(false),
		enabled: !isLoading && !isError,
	});

	const { shortcuts } = useKeyboardShortcuts({
		onShowHelp: () => setShowShortcutHelp(true),
		onCloseModal: () => setShowShortcutHelp(false),
		enabled: !isLoading && !isError,
	});

	// Redirect to onboarding if needed (only from dashboard)
	// Allow access to all other pages so users can complete onboarding steps
	// (e.g. creating repositories, configuring notifications, viewing docs)
	useEffect(() => {
		if (
			!onboardingLoading &&
			onboardingStatus?.needs_onboarding &&
			location.pathname === '/'
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
		<ReadOnlyModeContext.Provider value={readOnlyModeValue}>
			<div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex flex-col">
				<MaintenanceCountdown />
				<AnnouncementBanner />
				<LicenseBanner />
				<PasswordExpirationBanner />
				<TrialBanner />
				<div className="flex flex-1">
					<Sidebar />
					<div className="flex-1 flex flex-col">
						<Header onShowShortcuts={() => setShowShortcutHelp(true)} />
						<main className="flex-1 p-6">
							<Breadcrumbs />
							<Outlet />
						</main>
		<ToastProvider>
			<ReadOnlyModeContext.Provider value={readOnlyModeValue}>
				<div className="min-h-screen bg-gray-50 flex flex-col">
					<MaintenanceCountdown />
					<AnnouncementBanner />
					<LicenseBanner />
					<PasswordExpirationBanner />
					<TrialBanner />
					<div className="flex flex-1">
						<Sidebar />
						<div className="flex-1 flex flex-col">
							<Header onShowShortcuts={() => setShowShortcutHelp(true)} />
							<main className="flex-1 p-6">
								<Breadcrumbs />
								<Outlet />
							</main>
						</div>
					</div>
					<ShortcutHelpModal
						isOpen={showShortcutHelp}
						onClose={() => setShowShortcutHelp(false)}
						shortcuts={shortcuts}
					/>
				)}
		<div className="min-h-screen bg-gray-50 flex">
			<Sidebar />
			<div className="flex-1 flex flex-col">
				<AnnouncementBanner />
				<Header />
				<MaintenanceBanner />
export function Layout() {
	return (
		<div className="min-h-screen bg-gray-50 flex">
			<Sidebar />
			<div className="flex-1 flex flex-col">
				<Header />
				<main className="flex-1 p-6">
					<Outlet />
				</main>
			</div>
		</ReadOnlyModeContext.Provider>
					{showWhatsNew && (
						<WhatsNewModal
							entry={latestEntry ?? null}
							currentVersion={currentVersion}
							onDismiss={() => setShowWhatsNew(false)}
						/>
					)}
				</div>
			</ReadOnlyModeContext.Provider>
		</ToastProvider>
			<ShortcutHelpModal
				isOpen={showShortcutHelp}
				onClose={() => setShowShortcutHelp(false)}
				shortcuts={shortcuts}
			/>
		</div>
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
			</div>
		</ReadOnlyModeContext.Provider>
	);
}
