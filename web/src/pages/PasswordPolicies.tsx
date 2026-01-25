import { useEffect, useState } from 'react';
import { useMe } from '../hooks/useAuth';
import {
	usePasswordPolicy,
	useUpdatePasswordPolicy,
} from '../hooks/usePasswordPolicy';
import type { OrgRole, UpdatePasswordPolicyRequest } from '../lib/types';

export function PasswordPolicies() {
	const { data: user } = useMe();
	const { data: policyResponse, isLoading, isError } = usePasswordPolicy();
	const updatePolicy = useUpdatePasswordPolicy();

	const [formData, setFormData] = useState<UpdatePasswordPolicyRequest>({
		min_length: 8,
		require_uppercase: true,
		require_lowercase: true,
		require_number: true,
		require_special: false,
		max_age_days: undefined,
		history_count: 0,
	});

	const [hasChanges, setHasChanges] = useState(false);

	useEffect(() => {
		if (policyResponse?.policy) {
			setFormData({
				min_length: policyResponse.policy.min_length,
				require_uppercase: policyResponse.policy.require_uppercase,
				require_lowercase: policyResponse.policy.require_lowercase,
				require_number: policyResponse.policy.require_number,
				require_special: policyResponse.policy.require_special,
				max_age_days: policyResponse.policy.max_age_days,
				history_count: policyResponse.policy.history_count,
			});
		}
	}, [policyResponse]);

	const currentUserRole = (user?.current_org_role ?? 'member') as OrgRole;
	const isAdmin = currentUserRole === 'owner' || currentUserRole === 'admin';

	const handleChange = (
		field: keyof UpdatePasswordPolicyRequest,
		value: boolean | number | undefined,
	) => {
		setFormData({ ...formData, [field]: value });
		setHasChanges(true);
	};

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			await updatePolicy.mutateAsync(formData);
			setHasChanges(false);
		} catch {
			// Error handled by mutation
		}
	};

	const handleReset = () => {
		if (policyResponse?.policy) {
			setFormData({
				min_length: policyResponse.policy.min_length,
				require_uppercase: policyResponse.policy.require_uppercase,
				require_lowercase: policyResponse.policy.require_lowercase,
				require_number: policyResponse.policy.require_number,
				require_special: policyResponse.policy.require_special,
				max_age_days: policyResponse.policy.max_age_days,
				history_count: policyResponse.policy.history_count,
			});
			setHasChanges(false);
		}
	};

	if (!isAdmin) {
		return (
			<div className="space-y-6">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">Password Policy</h1>
					<p className="text-gray-600 mt-1">
						Configure password requirements for non-OIDC users
					</p>
				</div>
				<div className="bg-amber-50 border border-amber-200 rounded-lg p-4">
					<p className="text-amber-800">
						Only administrators can manage password policies.
					</p>
				</div>
			</div>
		);
	}

	if (isLoading) {
		return (
			<div className="space-y-6">
				<div>
					<div className="h-8 w-48 bg-gray-200 rounded animate-pulse" />
					<div className="h-4 w-64 bg-gray-200 rounded animate-pulse mt-2" />
				</div>
				<div className="bg-white rounded-lg border border-gray-200 p-6">
					<div className="space-y-4">
						{[1, 2, 3, 4, 5].map((i) => (
							<div key={i} className="h-10 bg-gray-200 rounded animate-pulse" />
						))}
					</div>
				</div>
			</div>
		);
	}

	if (isError) {
		return (
			<div className="space-y-6">
				<div>
					<h1 className="text-2xl font-bold text-gray-900">Password Policy</h1>
				</div>
				<div className="bg-red-50 border border-red-200 rounded-lg p-4">
					<p className="text-red-800">Failed to load password policy</p>
				</div>
			</div>
		);
	}

	return (
		<div className="space-y-6">
			<div>
				<h1 className="text-2xl font-bold text-gray-900">Password Policy</h1>
				<p className="text-gray-600 mt-1">
					Configure password requirements for non-OIDC users
				</p>
			</div>

			{policyResponse?.requirements && (
				<div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
					<h3 className="text-sm font-medium text-blue-800 mb-2">
						Current Requirements
					</h3>
					<p className="text-sm text-blue-700">
						{policyResponse.requirements.description}
					</p>
				</div>
			)}

			<div className="bg-white rounded-lg border border-gray-200 p-6">
				<form onSubmit={handleSubmit}>
					<div className="space-y-6">
						{/* Password Length */}
						<div>
							<label
								htmlFor="min_length"
								className="block text-sm font-medium text-gray-700"
							>
								Minimum Password Length
							</label>
							<div className="mt-1 flex items-center gap-4">
								<input
									type="number"
									id="min_length"
									value={formData.min_length}
									onChange={(e) =>
										handleChange(
											'min_length',
											Number.parseInt(e.target.value, 10),
										)
									}
									min={6}
									max={128}
									className="block w-24 rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
								/>
								<span className="text-sm text-gray-500">
									characters (6-128)
								</span>
							</div>
						</div>

						{/* Character Requirements */}
						<div>
							<h3 className="text-sm font-medium text-gray-700 mb-3">
								Character Requirements
							</h3>
							<div className="space-y-3">
								<div className="flex items-center">
									<input
										type="checkbox"
										id="require_uppercase"
										checked={formData.require_uppercase}
										onChange={(e) =>
											handleChange('require_uppercase', e.target.checked)
										}
										className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
									/>
									<label
										htmlFor="require_uppercase"
										className="ml-3 text-sm text-gray-900"
									>
										Require uppercase letter (A-Z)
									</label>
								</div>

								<div className="flex items-center">
									<input
										type="checkbox"
										id="require_lowercase"
										checked={formData.require_lowercase}
										onChange={(e) =>
											handleChange('require_lowercase', e.target.checked)
										}
										className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
									/>
									<label
										htmlFor="require_lowercase"
										className="ml-3 text-sm text-gray-900"
									>
										Require lowercase letter (a-z)
									</label>
								</div>

								<div className="flex items-center">
									<input
										type="checkbox"
										id="require_number"
										checked={formData.require_number}
										onChange={(e) =>
											handleChange('require_number', e.target.checked)
										}
										className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
									/>
									<label
										htmlFor="require_number"
										className="ml-3 text-sm text-gray-900"
									>
										Require number (0-9)
									</label>
								</div>

								<div className="flex items-center">
									<input
										type="checkbox"
										id="require_special"
										checked={formData.require_special}
										onChange={(e) =>
											handleChange('require_special', e.target.checked)
										}
										className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
									/>
									<label
										htmlFor="require_special"
										className="ml-3 text-sm text-gray-900"
									>
										Require special character (!@#$%^&amp;*...)
									</label>
								</div>
							</div>
						</div>

						{/* Password Expiration */}
						<div>
							<label
								htmlFor="max_age_days"
								className="block text-sm font-medium text-gray-700"
							>
								Password Expiration
							</label>
							<div className="mt-1 flex items-center gap-4">
								<input
									type="number"
									id="max_age_days"
									value={formData.max_age_days ?? ''}
									onChange={(e) => {
										const value = e.target.value;
										handleChange(
											'max_age_days',
											value === '' ? undefined : Number.parseInt(value, 10),
										);
									}}
									min={0}
									max={365}
									placeholder="Never"
									className="block w-24 rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
								/>
								<span className="text-sm text-gray-500">
									days (leave empty for no expiration)
								</span>
							</div>
							<p className="mt-1 text-xs text-gray-500">
								Users will be prompted to change their password when it expires.
							</p>
						</div>

						{/* Password History */}
						<div>
							<label
								htmlFor="history_count"
								className="block text-sm font-medium text-gray-700"
							>
								Password History
							</label>
							<div className="mt-1 flex items-center gap-4">
								<input
									type="number"
									id="history_count"
									value={formData.history_count}
									onChange={(e) =>
										handleChange(
											'history_count',
											Number.parseInt(e.target.value, 10),
										)
									}
									min={0}
									max={24}
									className="block w-24 rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm"
								/>
								<span className="text-sm text-gray-500">
									previous passwords to remember (0-24)
								</span>
							</div>
							<p className="mt-1 text-xs text-gray-500">
								Prevents users from reusing recent passwords. Set to 0 to
								disable.
							</p>
						</div>

						{/* Actions */}
						<div className="flex justify-end gap-3 pt-4 border-t border-gray-200">
							{hasChanges && (
								<button
									type="button"
									onClick={handleReset}
									className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
								>
									Reset
								</button>
							)}
							<button
								type="submit"
								disabled={!hasChanges || updatePolicy.isPending}
								className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 border border-transparent rounded-md hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
							>
								{updatePolicy.isPending ? 'Saving...' : 'Save Changes'}
							</button>
						</div>
					</div>
				</form>
			</div>

			{/* Information Section */}
			<div className="bg-gray-50 rounded-lg border border-gray-200 p-6">
				<h3 className="text-sm font-medium text-gray-900 mb-3">
					About Password Policies
				</h3>
				<ul className="text-sm text-gray-600 space-y-2">
					<li className="flex items-start gap-2">
						<svg
							className="w-5 h-5 text-gray-400 mt-0.5 flex-shrink-0"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
						<span>
							These policies apply to users who log in with email and password,
							not OIDC/SSO users.
						</span>
					</li>
					<li className="flex items-start gap-2">
						<svg
							className="w-5 h-5 text-gray-400 mt-0.5 flex-shrink-0"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
						<span>
							Changes take effect immediately for new password changes.
						</span>
					</li>
					<li className="flex items-start gap-2">
						<svg
							className="w-5 h-5 text-gray-400 mt-0.5 flex-shrink-0"
							fill="none"
							stroke="currentColor"
							viewBox="0 0 24 24"
							aria-hidden="true"
						>
							<path
								strokeLinecap="round"
								strokeLinejoin="round"
								strokeWidth={2}
								d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
							/>
						</svg>
						<span>
							Existing passwords are not affected until the user changes their
							password.
						</span>
					</li>
				</ul>
			</div>
		</div>
	);
}
