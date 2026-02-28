import { useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';

interface AuthStatus {
	oidc_enabled: boolean;
	password_enabled: boolean;
}

interface ErrorResponse {
	error: string;
	code?: string;
}

export default function LoginPage() {
	const navigate = useNavigate();

	const [email, setEmail] = useState('');
	const [password, setPassword] = useState('');
	const [error, setError] = useState<string | null>(null);
	const [loading, setLoading] = useState(false);
	const [authStatus, setAuthStatus] = useState<AuthStatus | null>(null);
	const [authStatusLoading, setAuthStatusLoading] = useState(true);

	// Fetch auth status on mount to determine which login methods are available
	useEffect(() => {
		const fetchAuthStatus = async () => {
			try {
				const response = await fetch('/auth/status');
				if (response.ok) {
					const data: AuthStatus = await response.json();
					setAuthStatus(data);
				}
			} catch {
				// If we can't fetch auth status, default to showing password login
				setAuthStatus({ oidc_enabled: false, password_enabled: true });
			} finally {
				setAuthStatusLoading(false);
			}
		};

		fetchAuthStatus();
	}, []);

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		setLoading(true);
		setError(null);

		try {
			const response = await fetch('/auth/login/password', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				credentials: 'include',
				body: JSON.stringify({ email, password }),
			});

			if (response.ok) {
				navigate('/');
				return;
			}

			const data: ErrorResponse = await response.json();

			if (response.status === 401) {
				setError('Invalid email or password.');
			} else if (response.status === 403) {
				setError(
					data.error ||
						'Your email address has not been verified. Please check your inbox.',
				);
			} else {
				setError(
					data.error || 'An unexpected error occurred. Please try again.',
				);
			}
		} catch {
			setError('Unable to connect to the server. Please try again later.');
		} finally {
			setLoading(false);
		}
	};

	const handleSSOLogin = () => {
		window.location.href = '/auth/login';
	};

	const showPasswordForm = authStatus?.password_enabled !== false;
	const showSSO = authStatus?.oidc_enabled === true;

	return (
		<div className="flex min-h-screen flex-col justify-center bg-gray-50 py-12 dark:bg-gray-900 sm:px-6 lg:px-8">
			<div className="sm:mx-auto sm:w-full sm:max-w-md">
				{/* Keldris branding */}
				<div className="flex justify-center">
					<svg
						className="h-12 w-12 text-blue-600"
						viewBox="0 0 48 48"
						fill="none"
						xmlns="http://www.w3.org/2000/svg"
						aria-hidden="true"
					>
						<rect
							x="4"
							y="8"
							width="40"
							height="32"
							rx="4"
							stroke="currentColor"
							strokeWidth="3"
						/>
						<path d="M4 16h40" stroke="currentColor" strokeWidth="3" />
						<circle cx="10" cy="12" r="1.5" fill="currentColor" />
						<circle cx="15" cy="12" r="1.5" fill="currentColor" />
						<circle cx="20" cy="12" r="1.5" fill="currentColor" />
						<path
							d="M16 26l4 4 8-8"
							stroke="currentColor"
							strokeWidth="2.5"
							strokeLinecap="round"
							strokeLinejoin="round"
						/>
					</svg>
				</div>
				<h1 className="mt-4 text-center text-3xl font-bold tracking-tight text-gray-900 dark:text-white">
					Keldris
				</h1>
				<p className="mt-1 text-center text-sm text-gray-600 dark:text-gray-400">
					Sign in to your account
				</p>
			</div>

			<div className="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
				<div className="bg-white px-4 py-8 shadow dark:bg-gray-800 sm:rounded-lg sm:px-10">
					{authStatusLoading ? (
						<div className="flex justify-center py-8">
							<div className="h-8 w-8 animate-spin rounded-full border-b-2 border-blue-600" />
						</div>
					) : (
						<>
							{/* Error message */}
							{error && (
								<div className="mb-6 rounded-md bg-red-50 p-4 dark:bg-red-900/30">
									<div className="flex">
										<div className="flex-shrink-0">
											<svg
												className="h-5 w-5 text-red-400"
												viewBox="0 0 20 20"
												fill="currentColor"
												aria-hidden="true"
											>
												<path
													fillRule="evenodd"
													d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
													clipRule="evenodd"
												/>
											</svg>
										</div>
										<div className="ml-3">
											<p className="text-sm text-red-700 dark:text-red-400">
												{error}
											</p>
										</div>
									</div>
								</div>
							)}

							{/* Password login form */}
							{showPasswordForm && (
								<form onSubmit={handleSubmit} className="space-y-6">
									<div>
										<label
											htmlFor="email"
											className="block text-sm font-medium text-gray-700 dark:text-gray-300"
										>
											Email address
										</label>
										<div className="mt-1">
											<input
												id="email"
												name="email"
												type="email"
												autoComplete="email"
												required
												value={email}
												onChange={(e) => setEmail(e.target.value)}
												className="block w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-blue-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"
												placeholder="you@example.com"
											/>
										</div>
									</div>

									<div>
										<label
											htmlFor="password"
											className="block text-sm font-medium text-gray-700 dark:text-gray-300"
										>
											Password
										</label>
										<div className="mt-1">
											<input
												id="password"
												name="password"
												type="password"
												autoComplete="current-password"
												required
												value={password}
												onChange={(e) => setPassword(e.target.value)}
												className="block w-full rounded-md border border-gray-300 bg-white px-3 py-2 text-sm shadow-sm focus:border-blue-500 focus:outline-none focus:ring-blue-500 dark:border-gray-700 dark:bg-gray-800 dark:text-white"
												placeholder="Enter your password"
											/>
										</div>
									</div>

									<div className="flex items-center justify-end">
										<Link
											to="/reset-password"
											className="text-sm font-medium text-blue-600 hover:text-blue-500"
										>
											Forgot your password?
										</Link>
									</div>

									<div>
										<button
											type="submit"
											disabled={loading}
											className="w-full rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 disabled:opacity-50"
										>
											{loading ? 'Signing in...' : 'Sign in'}
										</button>
									</div>
								</form>
							)}

							{/* Divider between password form and SSO */}
							{showPasswordForm && showSSO && (
								<div className="relative mt-6">
									<div className="absolute inset-0 flex items-center">
										<div className="w-full border-t border-gray-300 dark:border-gray-600" />
									</div>
									<div className="relative flex justify-center text-sm">
										<span className="bg-white px-2 text-gray-500 dark:bg-gray-800 dark:text-gray-400">
											Or continue with
										</span>
									</div>
								</div>
							)}

							{/* SSO login button */}
							{showSSO && (
								<div className={showPasswordForm ? 'mt-6' : ''}>
									<button
										type="button"
										onClick={handleSSOLogin}
										className="flex w-full items-center justify-center gap-2 rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-200 dark:hover:bg-gray-600"
									>
										<svg
											className="h-5 w-5"
											viewBox="0 0 20 20"
											fill="currentColor"
											aria-hidden="true"
										>
											<path
												fillRule="evenodd"
												d="M10 1a4.5 4.5 0 00-4.5 4.5V9H5a2 2 0 00-2 2v6a2 2 0 002 2h10a2 2 0 002-2v-6a2 2 0 00-2-2h-.5V5.5A4.5 4.5 0 0010 1zm3 8V5.5a3 3 0 10-6 0V9h6z"
												clipRule="evenodd"
											/>
										</svg>
										Sign in with SSO
									</button>
								</div>
							)}

							{/* Fallback when neither method is available */}
							{!showPasswordForm && !showSSO && (
								<div className="py-4 text-center">
									<p className="text-sm text-gray-600 dark:text-gray-400">
										No login methods are currently configured. Please contact
										your administrator.
									</p>
								</div>
							)}
						</>
					)}
				</div>
			</div>
		</div>
	);
}
