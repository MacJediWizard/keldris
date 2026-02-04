import { useEffect, useState } from 'react';
import {
	usePasswordRequirements,
	useValidatePassword,
} from '../hooks/usePasswordPolicy';

interface PasswordRequirementsProps {
	password: string;
	showValidation?: boolean;
}

export function PasswordRequirements({
	password,
	showValidation = true,
}: PasswordRequirementsProps) {
	const { data: requirements, isLoading } = usePasswordRequirements();
	const validatePassword = useValidatePassword();
	const [validationResult, setValidationResult] = useState<{
		valid: boolean;
		errors?: string[];
		warnings?: string[];
	} | null>(null);

	// biome-ignore lint/correctness/useExhaustiveDependencies: validatePassword.mutate is stable from TanStack Query
	useEffect(() => {
		if (password && showValidation && password.length >= 3) {
			const timer = setTimeout(() => {
				validatePassword.mutate(password, {
					onSuccess: (result) => setValidationResult(result),
				});
			}, 300);
			return () => clearTimeout(timer);
		}
		setValidationResult(null);
	}, [password, showValidation]);

	if (isLoading) {
		return (
			<div className="bg-gray-50 rounded-lg p-4">
				<div className="animate-pulse">
					<div className="h-4 bg-gray-200 rounded w-3/4 mb-2" />
					<div className="h-3 bg-gray-200 rounded w-1/2" />
				</div>
			</div>
		);
	}

	if (!requirements) {
		return null;
	}

	const checkRequirements = [
		{
			label: `At least ${requirements.min_length} characters`,
			met: password.length >= requirements.min_length,
		},
		{
			label: 'Uppercase letter (A-Z)',
			met: /[A-Z]/.test(password),
			required: requirements.require_uppercase,
		},
		{
			label: 'Lowercase letter (a-z)',
			met: /[a-z]/.test(password),
			required: requirements.require_lowercase,
		},
		{
			label: 'Number (0-9)',
			met: /[0-9]/.test(password),
			required: requirements.require_number,
		},
		{
			label: 'Special character (!@#$%^&*...)',
			met: /[!@#$%^&*()_+\-=[\]{};':"\\|,.<>/?]/.test(password),
			required: requirements.require_special,
		},
	].filter((req) => req.required !== false);

	return (
		<div className="space-y-3">
			<div className="text-sm font-medium text-gray-700">
				Password Requirements
			</div>
			<ul className="space-y-1.5">
				{checkRequirements.map((req) => (
					<li key={req.label} className="flex items-center gap-2">
						{password.length > 0 ? (
							req.met ? (
								<svg
									className="w-4 h-4 text-green-500"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
									aria-hidden="true"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M5 13l4 4L19 7"
									/>
								</svg>
							) : (
								<svg
									className="w-4 h-4 text-red-500"
									fill="none"
									stroke="currentColor"
									viewBox="0 0 24 24"
									aria-hidden="true"
								>
									<path
										strokeLinecap="round"
										strokeLinejoin="round"
										strokeWidth={2}
										d="M6 18L18 6M6 6l12 12"
									/>
								</svg>
							)
						) : (
							<svg
								className="w-4 h-4 text-gray-300"
								fill="currentColor"
								viewBox="0 0 24 24"
								aria-hidden="true"
							>
								<circle cx="12" cy="12" r="4" />
							</svg>
						)}
						<span
							className={`text-sm ${
								password.length > 0
									? req.met
										? 'text-green-700'
										: 'text-red-700'
									: 'text-gray-600'
							}`}
						>
							{req.label}
						</span>
					</li>
				))}
			</ul>

			{validationResult &&
				!validationResult.valid &&
				validationResult.errors && (
					<div className="mt-3 bg-red-50 border border-red-200 rounded-md p-3">
						<div className="text-sm text-red-700">
							{validationResult.errors.map((error) => (
								<p key={error}>{error}</p>
							))}
						</div>
					</div>
				)}

			{validationResult?.warnings && validationResult.warnings.length > 0 && (
				<div className="mt-3 bg-amber-50 border border-amber-200 rounded-md p-3">
					<div className="text-sm text-amber-700">
						{validationResult.warnings.map((warning) => (
							<p key={warning}>{warning}</p>
						))}
					</div>
				</div>
			)}
		</div>
	);
}
