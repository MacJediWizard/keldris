import { useState } from 'react';
import { Link, Outlet, useLocation } from 'react-router-dom';
import { useLogout, useMe } from '../hooks/useAuth';

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
];

function Sidebar() {
	const location = useLocation();

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
			</nav>
			<div className="p-4 border-t border-gray-800">
				<p className="text-xs text-gray-500">v0.0.1</p>
			</div>
		</aside>
	);
}

function Header() {
	const [showDropdown, setShowDropdown] = useState(false);
	const { data: user } = useMe();
	const logout = useLogout();

	const userInitial =
		user?.name?.charAt(0).toUpperCase() ??
		user?.email?.charAt(0).toUpperCase() ??
		'U';

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
				</button>
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
		<div className="min-h-screen bg-gray-50 flex items-center justify-center">
			<div className="text-center">
				<div className="w-12 h-12 border-4 border-indigo-200 border-t-indigo-600 rounded-full animate-spin mx-auto mb-4" />
				<p className="text-gray-600">Loading...</p>
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
