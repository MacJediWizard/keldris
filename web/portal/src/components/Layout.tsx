import { Link, Outlet, useLocation } from 'react-router-dom';
import { useLogout, useMe } from '../hooks/useAuth';

export function Layout() {
	const location = useLocation();
	const { data: customer } = useMe();
	const logout = useLogout();

	const navItems = [
		{ path: '/', label: 'Dashboard' },
		{ path: '/licenses', label: 'Licenses' },
		{ path: '/invoices', label: 'Invoices' },
	];

	const isActive = (path: string) => {
		if (path === '/') return location.pathname === '/';
		return location.pathname.startsWith(path);
	};

	return (
		<div className="min-h-screen bg-gray-50 dark:bg-dark-bg">
			{/* Header */}
			<header className="bg-white dark:bg-dark-card border-b border-gray-200 dark:border-dark-border">
				<div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
					<div className="flex justify-between items-center h-16">
						<div className="flex items-center">
							<Link
								to="/"
								className="text-xl font-bold text-gray-900 dark:text-white"
							>
								Keldris Portal
							</Link>
						</div>
						<nav className="flex items-center space-x-4">
							{navItems.map((item) => (
								<Link
									key={item.path}
									to={item.path}
									className={`px-3 py-2 rounded-md text-sm font-medium ${
										isActive(item.path)
											? 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-200'
											: 'text-gray-600 hover:text-gray-900 dark:text-gray-300 dark:hover:text-white'
									}`}
								>
									{item.label}
								</Link>
							))}
						</nav>
						<div className="flex items-center space-x-4">
							<span className="text-sm text-gray-600 dark:text-gray-300">
								{customer?.name || customer?.email}
							</span>
							<button
								type="button"
								onClick={() => logout.mutate()}
								className="text-sm text-gray-600 hover:text-gray-900 dark:text-gray-300 dark:hover:text-white"
							>
								Logout
							</button>
						</div>
					</div>
				</div>
			</header>

			{/* Main content */}
			<main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
				<Outlet />
			</main>
		</div>
	);
}
