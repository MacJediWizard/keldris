import { useEffect, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';

interface ValidateResponse {
	valid: boolean;
	email: string;
}

interface ErrorResponse {
	error: string;
	code?: string;
}

export function PasswordReset() {
	const [searchParams] = useSearchParams();
	const token = searchParams.get('token');

	// States for request form
	const [email, setEmail] = useState('');
	const [requestSubmitted, setRequestSubmitted] = useState(false);
	const [requestLoading, setRequestLoading] = useState(false);
	const [requestError, setRequestError] = useState<string | null>(null);

	// States for reset form
	const [newPassword, setNewPassword] = useState('');
	const [confirmPassword, setConfirmPassword] = useState('');
	const [resetSubmitted, setResetSubmitted] = useState(false);
	const [resetLoading, setResetLoading] = useState(false);
	const [resetError, setResetError] = useState<string | null>(null);

	// Token validation states
	const [tokenValid, setTokenValid] = useState<boolean | null>(null);
	const [tokenEmail, setTokenEmail] = useState<string | null>(null);
	const [validating, setValidating] = useState(false);

	// Validate token on mount
	useEffect(() => {
		if (token) {
			validateToken(token);
		}
	}, [token]);

	const validateToken = async (token: string) => {
		setValidating(true);
		try {
			const response = await fetch(`/auth/reset-password/validate/${token}`);
			const data = await response.json();

			if (response.ok) {
				setTokenValid(true);
				setTokenEmail((data as ValidateResponse).email);
			} else {
				setTokenValid(false);
				setResetError((data as ErrorResponse).error || 'Invalid reset link');
			}
		} catch {
			setTokenValid(false);
			setResetError('Failed to validate reset link');
		} finally {
			setValidating(false);
		}
	};

	const handleRequestSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		setRequestLoading(true);
		setRequestError(null);

		try {
			const response = await fetch('/auth/reset-password/request', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ email }),
			});

			const data = await response.json();

			if (response.ok) {
				setRequestSubmitted(true);
			} else if (response.status === 429) {
				setRequestError('Too many requests. Please try again later.');
			} else {
				setRequestError(
					(data as ErrorResponse).error || 'Failed to request password reset',
				);
			}
		} catch {
			setRequestError('Failed to send request. Please try again.');
		} finally {
			setRequestLoading(false);
		}
	};

	const handleResetSubmit = async (e: React.FormEvent) => {
		e.preventDefault();

		if (newPassword !== confirmPassword) {
			setResetError('Passwords do not match');
			return;
		}

		if (newPassword.length < 8) {
			setResetError('Password must be at least 8 characters');
			return;
		}

		setResetLoading(true);
		setResetError(null);

		try {
			const response = await fetch('/auth/reset-password/reset', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ token, new_password: newPassword }),
			});

			const data = await response.json();

			if (response.ok) {
				setResetSubmitted(true);
			} else {
				setResetError(
					(data as ErrorResponse).error || 'Failed to reset password',
				);
			}
		} catch {
			setResetError('Failed to reset password. Please try again.');
		} finally {
			setResetLoading(false);
		}
	};

	// Show token validation loading
	if (token && validating) {
		return (
			<div className="min-h-screen bg-gray-50 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
				<div className="sm:mx-auto sm:w-full sm:max-w-md">
					<div className="text-center">
						<div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto" />
						<p className="mt-4 text-gray-600">Validating reset link...</p>
					</div>
				</div>
			</div>
		);
	}

	// Show reset form if we have a valid token
	if (token) {
		// Show success message after reset
		if (resetSubmitted) {
			return (
				<div className="min-h-screen bg-gray-50 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
					<div className="sm:mx-auto sm:w-full sm:max-w-md">
						<h2 className="mt-6 text-center text-3xl font-bold text-gray-900">
							Password Reset Complete
						</h2>
					</div>

					<div className="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
						<div className="bg-white py-8 px-4 shadow sm:rounded-lg sm:px-10">
							<div className="rounded-md bg-green-50 p-4 mb-6">
								<div className="flex">
									<div className="flex-shrink-0">
										<svg
											className="h-5 w-5 text-green-400"
											viewBox="0 0 20 20"
											fill="currentColor"
											aria-hidden="true"
										>
											<path
												fillRule="evenodd"
												d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
												clipRule="evenodd"
											/>
										</svg>
									</div>
									<div className="ml-3">
										<p className="text-sm text-green-700">
											Your password has been reset successfully. You can now log
											in with your new password.
										</p>
									</div>
								</div>
							</div>
							<Link
								to="/"
								className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
							>
								Go to Login
							</Link>
						</div>
					</div>
				</div>
			);
		}

		// Show invalid token error
		if (tokenValid === false) {
			return (
				<div className="min-h-screen bg-gray-50 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
					<div className="sm:mx-auto sm:w-full sm:max-w-md">
						<h2 className="mt-6 text-center text-3xl font-bold text-gray-900">
							Invalid Reset Link
						</h2>
					</div>

					<div className="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
						<div className="bg-white py-8 px-4 shadow sm:rounded-lg sm:px-10">
							<div className="rounded-md bg-red-50 p-4 mb-6">
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
										<p className="text-sm text-red-700">{resetError}</p>
									</div>
								</div>
							</div>
							<Link
								to="/reset-password"
								className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
							>
								Request New Reset Link
							</Link>
						</div>
					</div>
				</div>
			);
		}

		// Show reset form
		return (
			<div className="min-h-screen bg-gray-50 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
				<div className="sm:mx-auto sm:w-full sm:max-w-md">
					<h2 className="mt-6 text-center text-3xl font-bold text-gray-900">
						Reset Your Password
					</h2>
					{tokenEmail && (
						<p className="mt-2 text-center text-sm text-gray-600">
							for {tokenEmail}
						</p>
					)}
				</div>

				<div className="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
					<div className="bg-white py-8 px-4 shadow sm:rounded-lg sm:px-10">
						<form onSubmit={handleResetSubmit} className="space-y-6">
							{resetError && (
								<div className="rounded-md bg-red-50 p-4">
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
											<p className="text-sm text-red-700">{resetError}</p>
										</div>
									</div>
								</div>
							)}

							<div>
								<label
									htmlFor="newPassword"
									className="block text-sm font-medium text-gray-700"
								>
									New Password
								</label>
								<div className="mt-1">
									<input
										id="newPassword"
										name="newPassword"
										type="password"
										autoComplete="new-password"
										required
										minLength={8}
										value={newPassword}
										onChange={(e) => setNewPassword(e.target.value)}
										className="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm placeholder-gray-400 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
									/>
								</div>
							</div>

							<div>
								<label
									htmlFor="confirmPassword"
									className="block text-sm font-medium text-gray-700"
								>
									Confirm Password
								</label>
								<div className="mt-1">
									<input
										id="confirmPassword"
										name="confirmPassword"
										type="password"
										autoComplete="new-password"
										required
										minLength={8}
										value={confirmPassword}
										onChange={(e) => setConfirmPassword(e.target.value)}
										className="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm placeholder-gray-400 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
									/>
								</div>
							</div>

							<div>
								<button
									type="submit"
									disabled={resetLoading}
									className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
								>
									{resetLoading ? 'Resetting...' : 'Reset Password'}
								</button>
							</div>
						</form>
					</div>
				</div>
			</div>
		);
	}

	// Show success message after request
	if (requestSubmitted) {
		return (
			<div className="min-h-screen bg-gray-50 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
				<div className="sm:mx-auto sm:w-full sm:max-w-md">
					<h2 className="mt-6 text-center text-3xl font-bold text-gray-900">
						Check Your Email
					</h2>
				</div>

				<div className="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
					<div className="bg-white py-8 px-4 shadow sm:rounded-lg sm:px-10">
						<div className="rounded-md bg-blue-50 p-4 mb-6">
							<div className="flex">
								<div className="flex-shrink-0">
									<svg
										className="h-5 w-5 text-blue-400"
										viewBox="0 0 20 20"
										fill="currentColor"
										aria-hidden="true"
									>
										<path d="M2.003 5.884L10 9.882l7.997-3.998A2 2 0 0016 4H4a2 2 0 00-1.997 1.884z" />
										<path d="M18 8.118l-8 4-8-4V14a2 2 0 002 2h12a2 2 0 002-2V8.118z" />
									</svg>
								</div>
								<div className="ml-3">
									<p className="text-sm text-blue-700">
										If an account exists with that email, you will receive a
										password reset link shortly. The link will expire in 1 hour.
									</p>
								</div>
							</div>
						</div>
						<Link
							to="/"
							className="w-full flex justify-center py-2 px-4 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
						>
							Return to Login
						</Link>
					</div>
				</div>
			</div>
		);
	}

	// Show request form
	return (
		<div className="min-h-screen bg-gray-50 flex flex-col justify-center py-12 sm:px-6 lg:px-8">
			<div className="sm:mx-auto sm:w-full sm:max-w-md">
				<h2 className="mt-6 text-center text-3xl font-bold text-gray-900">
					Reset Your Password
				</h2>
				<p className="mt-2 text-center text-sm text-gray-600">
					Enter your email address and we'll send you a link to reset your
					password.
				</p>
			</div>

			<div className="mt-8 sm:mx-auto sm:w-full sm:max-w-md">
				<div className="bg-white py-8 px-4 shadow sm:rounded-lg sm:px-10">
					<form onSubmit={handleRequestSubmit} className="space-y-6">
						{requestError && (
							<div className="rounded-md bg-red-50 p-4">
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
										<p className="text-sm text-red-700">{requestError}</p>
									</div>
								</div>
							</div>
						)}

						<div>
							<label
								htmlFor="email"
								className="block text-sm font-medium text-gray-700"
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
									className="appearance-none block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm placeholder-gray-400 focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
								/>
							</div>
						</div>

						<div>
							<button
								type="submit"
								disabled={requestLoading}
								className="w-full flex justify-center py-2 px-4 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
							>
								{requestLoading ? 'Sending...' : 'Send Reset Link'}
							</button>
						</div>
					</form>

					<div className="mt-6">
						<div className="relative">
							<div className="absolute inset-0 flex items-center">
								<div className="w-full border-t border-gray-300" />
							</div>
							<div className="relative flex justify-center text-sm">
								<span className="px-2 bg-white text-gray-500">Or</span>
							</div>
						</div>

						<div className="mt-6">
							<Link
								to="/"
								className="w-full flex justify-center py-2 px-4 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
							>
								Return to Login
							</Link>
						</div>
					</div>
				</div>
			</div>
		</div>
	);
}
