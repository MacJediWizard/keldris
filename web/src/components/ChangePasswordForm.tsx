import { useState } from 'react';
import { useChangePassword } from '../hooks/usePasswordPolicy';
import { PasswordRequirements } from './PasswordRequirements';

interface ChangePasswordFormProps {
	onSuccess?: () => void;
	onCancel?: () => void;
	showCancel?: boolean;
}

export function ChangePasswordForm({
	onSuccess,
	onCancel,
	showCancel = true,
}: ChangePasswordFormProps) {
	const [currentPassword, setCurrentPassword] = useState('');
	const [newPassword, setNewPassword] = useState('');
	const [confirmPassword, setConfirmPassword] = useState('');
	const [showCurrentPassword, setShowCurrentPassword] = useState(false);
	const [showNewPassword, setShowNewPassword] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [success, setSuccess] = useState(false);

	const changePassword = useChangePassword();

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		setError(null);
		setSuccess(false);

		if (newPassword !== confirmPassword) {
			setError('New passwords do not match');
			return;
		}

		try {
			await changePassword.mutateAsync({
				current_password: currentPassword,
				new_password: newPassword,
			});
			setSuccess(true);
			setCurrentPassword('');
			setNewPassword('');
			setConfirmPassword('');
			if (onSuccess) {
				onSuccess();
			}
		} catch (err) {
			if (err instanceof Error) {
				setError(err.message);
			} else {
				setError('Failed to change password');
			}
		}
	};

	return (
		<form onSubmit={handleSubmit} className="space-y-6">
			{error && (
				<div className="bg-red-50 border border-red-200 rounded-lg p-4">
					<p className="text-sm text-red-700">{error}</p>
				</div>
			)}

			{success && (
				<div className="bg-green-50 border border-green-200 rounded-lg p-4">
					<p className="text-sm text-green-700">
						Password changed successfully!
					</p>
				</div>
			)}

			<div>
				<label
					htmlFor="current_password"
					className="block text-sm font-medium text-gray-700"
				>
					Current Password
				</label>
				<div className="mt-1 relative">
					<input
						type={showCurrentPassword ? 'text' : 'password'}
						id="current_password"
						value={currentPassword}
						onChange={(e) => setCurrentPassword(e.target.value)}
						required
						className="block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm pr-10"
					/>
					<button
						type="button"
						onClick={() => setShowCurrentPassword(!showCurrentPassword)}
						className="absolute inset-y-0 right-0 pr-3 flex items-center"
					>
						{showCurrentPassword ? (
							<svg
								className="w-5 h-5 text-gray-400"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
								aria-hidden="true"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"
								/>
							</svg>
						) : (
							<svg
								className="w-5 h-5 text-gray-400"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
								aria-hidden="true"
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
						)}
					</button>
				</div>
			</div>

			<div>
				<label
					htmlFor="new_password"
					className="block text-sm font-medium text-gray-700"
				>
					New Password
				</label>
				<div className="mt-1 relative">
					<input
						type={showNewPassword ? 'text' : 'password'}
						id="new_password"
						value={newPassword}
						onChange={(e) => setNewPassword(e.target.value)}
						required
						className="block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm pr-10"
					/>
					<button
						type="button"
						onClick={() => setShowNewPassword(!showNewPassword)}
						className="absolute inset-y-0 right-0 pr-3 flex items-center"
					>
						{showNewPassword ? (
							<svg
								className="w-5 h-5 text-gray-400"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
								aria-hidden="true"
							>
								<path
									strokeLinecap="round"
									strokeLinejoin="round"
									strokeWidth={2}
									d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21"
								/>
							</svg>
						) : (
							<svg
								className="w-5 h-5 text-gray-400"
								fill="none"
								stroke="currentColor"
								viewBox="0 0 24 24"
								aria-hidden="true"
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
						)}
					</button>
				</div>
			</div>

			<PasswordRequirements password={newPassword} />

			<div>
				<label
					htmlFor="confirm_password"
					className="block text-sm font-medium text-gray-700"
				>
					Confirm New Password
				</label>
				<div className="mt-1">
					<input
						type="password"
						id="confirm_password"
						value={confirmPassword}
						onChange={(e) => setConfirmPassword(e.target.value)}
						required
						className={`block w-full rounded-md shadow-sm sm:text-sm ${
							confirmPassword && newPassword !== confirmPassword
								? 'border-red-300 focus:border-red-500 focus:ring-red-500'
								: 'border-gray-300 focus:border-indigo-500 focus:ring-indigo-500'
						}`}
					/>
				</div>
				{confirmPassword && newPassword !== confirmPassword && (
					<p className="mt-1 text-sm text-red-600">Passwords do not match</p>
				)}
			</div>

			<div className="flex justify-end gap-3">
				{showCancel && onCancel && (
					<button
						type="button"
						onClick={onCancel}
						className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
					>
						Cancel
					</button>
				)}
				<button
					type="submit"
					disabled={
						changePassword.isPending ||
						!currentPassword ||
						!newPassword ||
						!confirmPassword ||
						newPassword !== confirmPassword
					}
					className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 border border-transparent rounded-md hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
				>
					{changePassword.isPending ? 'Changing...' : 'Change Password'}
				</button>
			</div>
		</form>
	);
}
