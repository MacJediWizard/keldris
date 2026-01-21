import { useEffect, useRef, useState } from 'react';
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom';
import { useAlertCount } from '../hooks/useAlerts';
import { useLogout, useMe } from '../hooks/useAuth';
import {
	useOrganizations,
	useSwitchOrganization,
} from '../hooks/useOrganizations';
import { useSearch } from '../hooks/useSearch';
import { useTheme } from '../hooks/useTheme';
import type { SearchResult, SearchResultType } from '../lib/types';
import { MaintenanceBanner } from './features/MaintenanceBanner';

interface NavItem {
	path: string;
	label: string;
	icon: React.ReactNode;
}

const navItems: NavItem[] = [
	{
		path: '/',
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
		path: '/agent-groups',
		label: 'Agent Groups',
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
					d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"
				/>
			</svg>
		),
	},
	{
		path: '/repositories',
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
		path: '/policies',
		label: 'Policies',
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
		path: '/backups',
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
		path: '/dr-runbooks',
		label: 'DR Runbooks',
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
					d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
				/>
			</svg>
		),
	},
	{
		path: '/restore',
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
		path: '/notifications',
		label: 'Notifications',
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
		path: '/reports',
		label: 'Reports',
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
					d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
				/>
			</svg>
		),
	},
	{
		path: '/audit-logs',
		label: 'Audit Logs',
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
					d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"
				/>
			</svg>
		),
	},
];

function Sidebar() {
	const location = useLocation();
	const { data: user } = useMe();
	const isAdmin =
		user?.current_org_role === 'owner' || user?.current_org_role === 'admin';

	return (
		<aside className="w-64 bg-gray-900 text-white flex flex-col">
			<div className="p-6">
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
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										isActive
											? 'bg-indigo-600 text-white'
											: 'text-gray-300 hover:bg-gray-800 hover:text-white'
									}`}
								>
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
							Organization
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
									<span>Members</span>
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
									<span>Settings</span>
								</Link>
							</li>
							<li>
								<Link
									to="/organization/maintenance"
									className={`flex items-center gap-3 px-4 py-3 rounded-lg transition-colors ${
										location.pathname === '/organization/maintenance'
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
								</Link>
							</li>
						</ul>
					</>
				)}
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
				className="flex items-center gap-2 px-3 py-2 text-sm font-medium text-gray-700 dark:text-gray-200 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-600"
			>
				<svg
					aria-hidden="true"
					className="w-4 h-4 text-gray-500 dark:text-gray-400"
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
					{currentOrg?.name ?? 'Select org'}
				</span>
				<svg
					aria-hidden="true"
					className="w-4 h-4 text-gray-400 dark:text-gray-500"
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
					<div className="px-3 py-2 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase">
						Organizations
					</div>
					{organizations.map((org) => (
						<button
							key={org.id}
							type="button"
							onClick={() => handleSwitch(org.id)}
							className={`w-full text-left px-3 py-2 text-sm hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center justify-between ${
								org.id === user?.current_org_id
									? 'bg-indigo-50 dark:bg-indigo-900/50 text-indigo-700 dark:text-indigo-300'
									: 'text-gray-700 dark:text-gray-300'
							}`}
						>
							<span className="truncate">{org.name}</span>
							<span className="text-xs text-gray-400 dark:text-gray-500 capitalize">
								{org.role}
							</span>
						</button>
					))}
					<div className="border-t border-gray-100 dark:border-gray-700 mt-1 pt-1">
						<Link
							to="/organization/new"
							onClick={() => setShowDropdown(false)}
							className="w-full text-left px-3 py-2 text-sm text-indigo-600 dark:text-indigo-400 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
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
							Create organization
						</Link>
					</div>
				</div>
			)}
		</div>
	);
}

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
					className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 dark:text-gray-500"
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
					className="w-64 pl-9 pr-4 py-2 text-sm border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-500 dark:placeholder-gray-400 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
				/>
			</div>

			{isOpen && query.length >= 2 && (
				<div className="absolute top-full left-0 mt-2 w-80 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-2 z-50">
					{isLoading ? (
						<div className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
							Searching...
						</div>
					) : searchData?.results && searchData.results.length > 0 ? (
						<>
							<div className="px-4 py-1 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase">
								{searchData.total} result{searchData.total !== 1 ? 's' : ''}
							</div>
							{searchData.results.map((result, index) => (
								<button
									key={`${result.type}-${result.id}-${index}`}
									type="button"
									onClick={() => handleResultClick(result)}
									className="w-full text-left px-4 py-2 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-3"
								>
									<span className="text-gray-400 dark:text-gray-500">
										{getSearchResultIcon(result.type)}
									</span>
									<div className="flex-1 min-w-0">
										<p className="text-sm font-medium text-gray-900 dark:text-white truncate">
											{result.name}
										</p>
										<p className="text-xs text-gray-500 dark:text-gray-400 truncate">
											{result.type}
											{result.status && ` - ${result.status}`}
										</p>
									</div>
								</button>
							))}
						</>
					) : (
						<div className="px-4 py-3 text-sm text-gray-500 dark:text-gray-400">
							No results found for "{query}"
						</div>
					)}
				</div>
			)}
		</div>
	);
}

function ThemeToggle() {
	const { theme, toggleTheme } = useTheme();

	const getIcon = () => {
		if (theme === 'light') {
			return (
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
						d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"
					/>
				</svg>
			);
		}
		if (theme === 'dark') {
			return (
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
						d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"
					/>
				</svg>
			);
		}
		// System theme
		return (
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
					d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
				/>
			</svg>
		);
	};

	const getLabel = () => {
		if (theme === 'light') return 'Light mode';
		if (theme === 'dark') return 'Dark mode';
		return 'System theme';
	};

	return (
		<button
			type="button"
			onClick={toggleTheme}
			aria-label={getLabel()}
			title={getLabel()}
			className="p-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700"
		>
			{getIcon()}
		</button>
	);
}

function Header() {
	const [showDropdown, setShowDropdown] = useState(false);
	const { data: user } = useMe();
	const { data: alertCount } = useAlertCount();
	const logout = useLogout();

	const userInitial =
		user?.name?.charAt(0).toUpperCase() ??
		user?.email?.charAt(0).toUpperCase() ??
		'U';

	return (
		<header className="h-16 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between px-6">
			<div className="flex items-center gap-4">
				<OrgSwitcher />
				<GlobalSearch />
			</div>
			<div className="flex items-center gap-4">
				<ThemeToggle />
				<Link
					to="/alerts"
					aria-label="Alerts"
					className="relative p-2 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700"
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
						<div className="absolute right-0 mt-2 w-48 bg-white dark:bg-gray-800 rounded-lg shadow-lg border border-gray-200 dark:border-gray-700 py-1 z-50">
							{user && (
								<div className="px-4 py-2 border-b border-gray-100 dark:border-gray-700">
									<p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
										{user.name}
									</p>
									<p className="text-xs text-gray-500 dark:text-gray-400 truncate">
										{user.email}
									</p>
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
								className="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
							>
								Sign out
							</button>
						</div>
					)}
				</div>
			</div>
		</header>
	);
}

function LoadingScreen() {
	return (
		<div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex items-center justify-center">
			<div className="text-center">
				<div className="w-12 h-12 border-4 border-indigo-200 dark:border-indigo-800 border-t-indigo-600 dark:border-t-indigo-400 rounded-full animate-spin mx-auto mb-4" />
				<p className="text-gray-600 dark:text-gray-400">Loading...</p>
			</div>
		</div>
	);
}

export function Layout() {
	const { isLoading, isError } = useMe();

	// Show loading state while checking auth
	if (isLoading) {
		return <LoadingScreen />;
	}

	// If auth check failed, the API client will redirect to login
	// But we show a loading state just in case
	if (isError) {
		return <LoadingScreen />;
	}

	return (
		<div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex">
			<Sidebar />
			<div className="flex-1 flex flex-col">
				<Header />
				<MaintenanceBanner />
				<main className="flex-1 p-6">
					<Outlet />
				</main>
			</div>
		</div>
	);
}
