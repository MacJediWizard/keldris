import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { VerticalStepper } from '../components/ui/Stepper';
import {
	useCompleteSetup,
	useCreateSuperuser,
	useSetupStatus,
	useTestDatabase,
} from '../hooks/useSetup';

const SETUP_STEPS = [
	{
		id: 'database',
		label: 'Database',
		description: 'Verify database connection',
	},
	{ id: 'superuser', label: 'Superuser', description: 'Create admin account' },
];

interface StepProps {
	onComplete: () => void;
	isLoading?: boolean;
}

function DatabaseStep({ onComplete, isLoading }: StepProps) {
	const testDatabase = useTestDatabase();
	const [tested, setTested] = useState(false);

	const handleTest = () => {
		testDatabase.mutate(undefined, {
			onSuccess: (data) => {
				if (data.ok) {
					setTested(true);
					onComplete();
				}
			},
		});
	};

	// biome-ignore lint/correctness/useExhaustiveDependencies: intentionally run only on mount
	useEffect(() => {
		// Auto-test on mount
		handleTest();
	}, []);

	return (
		<div className="py-4">
			<h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-2">
				Database Connection
			</h2>
			<p className="text-gray-600 dark:text-gray-400 mb-6">
				Verifying the database connection to ensure the server can store data.
			</p>

			{testDatabase.isPending && (
				<div className="flex items-center gap-3 p-4 bg-blue-50 dark:bg-blue-900/30 border border-blue-200 dark:border-blue-800 rounded-lg mb-6">
					<div className="w-5 h-5 border-2 border-blue-200 border-t-blue-600 rounded-full animate-spin" />
					<span className="text-blue-800 dark:text-blue-300">
						Testing database connection...
					</span>
				</div>
			)}

			{testDatabase.isError && (
				<div className="p-4 bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-lg mb-6">
					<div className="flex items-center gap-2 text-red-800 dark:text-red-300">
						<svg
							aria-hidden="true"
							className="w-5 h-5"
							fill="currentColor"
							viewBox="0 0 20 20"
						>
							<path
								fillRule="evenodd"
								d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
								clipRule="evenodd"
							/>
						</svg>
						<span className="font-medium">Connection failed</span>
					</div>
					<p className="mt-1 text-sm text-red-700 dark:text-red-400">
						{testDatabase.error instanceof Error
							? testDatabase.error.message
							: 'Unknown error'}
					</p>
				</div>
			)}

			{tested && testDatabase.data?.ok && (
				<div className="p-4 bg-green-50 dark:bg-green-900/30 border border-green-200 dark:border-green-800 rounded-lg mb-6">
					<div className="flex items-center gap-2 text-green-800 dark:text-green-300">
						<svg
							aria-hidden="true"
							className="w-5 h-5"
							fill="currentColor"
							viewBox="0 0 20 20"
						>
							<path
								fillRule="evenodd"
								d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
								clipRule="evenodd"
							/>
						</svg>
						<span className="font-medium">Database connection successful</span>
					</div>
				</div>
			)}

			<div className="flex justify-end">
				<button
					type="button"
					onClick={handleTest}
					disabled={testDatabase.isPending || isLoading}
					className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
				>
					{testDatabase.isPending
						? 'Testing...'
						: tested
							? 'Continue'
							: 'Test Connection'}
				</button>
			</div>
		</div>
	);
}

function SuperuserStep({ onComplete, isLoading }: StepProps) {
	const createSuperuser = useCreateSuperuser();
	const [email, setEmail] = useState('');
	const [name, setName] = useState('');
	const [password, setPassword] = useState('');
	const [confirmPassword, setConfirmPassword] = useState('');
	const [error, setError] = useState('');
	const [showConfetti, setShowConfetti] = useState(false);

	useEffect(() => {
		if (showConfetti) {
			const timer = setTimeout(() => setShowConfetti(false), 5000);
			return () => clearTimeout(timer);
		}
	}, [showConfetti]);

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();
		setError('');

		if (password !== confirmPassword) {
			setError('Passwords do not match');
			return;
		}

		if (password.length < 8) {
			setError('Password must be at least 8 characters');
			return;
		}

		createSuperuser.mutate(
			{ email, password, name },
			{
				onSuccess: () => {
					setShowConfetti(true);
					onComplete();
				},
				onError: (err) =>
					setError(
						err instanceof Error ? err.message : 'Failed to create superuser',
					),
			},
		);
	};

	return (
		<div className="py-4">
			{showConfetti && <Confetti />}
			<h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-2">
				Create Superuser Account
			</h2>
			<p className="text-gray-600 dark:text-gray-400 mb-6">
				Create the administrator account that will have full access to manage
				the server.
			</p>

			<form onSubmit={handleSubmit} className="space-y-4">
				<div>
					<label
						htmlFor="name"
						className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
					>
						Name <span className="text-red-500">*</span>
					</label>
					<input
						id="name"
						type="text"
						value={name}
						onChange={(e) => setName(e.target.value)}
						required
						className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-800 dark:border-gray-600 dark:text-white"
						placeholder="Admin User"
					/>
				</div>

				<div>
					<label
						htmlFor="email"
						className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
					>
						Email <span className="text-red-500">*</span>
					</label>
					<input
						id="email"
						type="email"
						value={email}
						onChange={(e) => setEmail(e.target.value)}
						required
						className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-800 dark:border-gray-600 dark:text-white"
						placeholder="admin@example.com"
					/>
				</div>

				<div>
					<label
						htmlFor="password"
						className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
					>
						Password <span className="text-red-500">*</span>
					</label>
					<input
						id="password"
						type="password"
						value={password}
						onChange={(e) => setPassword(e.target.value)}
						required
						minLength={8}
						className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-800 dark:border-gray-600 dark:text-white"
						placeholder="Minimum 8 characters"
					/>
				</div>

				<div>
					<label
						htmlFor="confirmPassword"
						className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
					>
						Confirm Password <span className="text-red-500">*</span>
					</label>
					<input
						id="confirmPassword"
						type="password"
						value={confirmPassword}
						onChange={(e) => setConfirmPassword(e.target.value)}
						required
						className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-800 dark:border-gray-600 dark:text-white"
					/>
				</div>

				{(error || createSuperuser.isError) && (
					<div className="p-3 bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-lg text-sm text-red-700 dark:text-red-400">
						{error ||
							(createSuperuser.error instanceof Error
								? createSuperuser.error.message
								: 'An error occurred')}
					</div>
				)}

				<div className="flex justify-end pt-4">
					<button
						type="submit"
						disabled={createSuperuser.isPending || isLoading}
						className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
					>
						{createSuperuser.isPending ? 'Creating...' : 'Create Account'}
					</button>
				</div>
			</form>
		</div>
	);
}

function CompleteSetupStep() {
	const completeSetup = useCompleteSetup();
	const [showConfetti, setShowConfetti] = useState(false);

	const handleComplete = () => {
		setShowConfetti(true);
		completeSetup.mutate();
	};

	useEffect(() => {
		if (showConfetti) {
			const timer = setTimeout(() => setShowConfetti(false), 5000);
			return () => clearTimeout(timer);
		}
	}, [showConfetti]);

	return (
		<div className="text-center py-8 relative">
			{showConfetti && <Confetti />}

			<div className="w-20 h-20 bg-green-100 dark:bg-green-900/30 rounded-full flex items-center justify-center mx-auto mb-6">
				<svg
					aria-hidden="true"
					className="w-10 h-10 text-green-600"
					fill="none"
					stroke="currentColor"
					viewBox="0 0 24 24"
				>
					<path
						strokeLinecap="round"
						strokeLinejoin="round"
						strokeWidth={2}
						d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
					/>
				</svg>
			</div>

			<h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-4">
				Setup Complete!
			</h2>

			<p className="text-gray-600 dark:text-gray-400 max-w-md mx-auto mb-8">
				Your Keldris server is now configured and ready to use. Click the button
				below to start using your backup system.
			</p>

			<div className="bg-green-50 dark:bg-green-900/30 border border-green-200 dark:border-green-800 rounded-lg p-4 mb-8 max-w-md mx-auto text-left">
				<h3 className="text-sm font-semibold text-green-900 dark:text-green-200 mb-2">
					What's next?
				</h3>
				<ul className="text-sm text-green-700 dark:text-green-400 space-y-1">
					<li>- Log in with your superuser account</li>
					<li>- Create repositories to store backups</li>
					<li>- Install agents on systems to back up</li>
					<li>- Configure backup schedules</li>
				</ul>
			</div>

			{completeSetup.isError && (
				<div className="p-3 bg-red-50 dark:bg-red-900/30 border border-red-200 dark:border-red-800 rounded-lg text-sm text-red-700 dark:text-red-400 mb-4 max-w-md mx-auto">
					{completeSetup.error instanceof Error
						? completeSetup.error.message
						: 'Failed to complete setup'}
				</div>
			)}

			<button
				type="button"
				onClick={handleComplete}
				disabled={completeSetup.isPending}
				className="inline-flex items-center gap-2 px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
			>
				{completeSetup.isPending ? 'Completing...' : 'Go to Login'}
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
			</button>
		</div>
	);
}

function Confetti() {
	const confettiPieces = Array.from({ length: 50 }, (_, i) => ({
		id: i,
		left: `${Math.random() * 100}%`,
		delay: `${Math.random() * 2}s`,
		duration: `${2 + Math.random() * 2}s`,
		color: ['#6366f1', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6'][
			Math.floor(Math.random() * 5)
		],
	}));

	return (
		<div className="fixed inset-0 pointer-events-none overflow-hidden z-50">
			{confettiPieces.map((piece) => (
				<div
					key={piece.id}
					className="absolute w-3 h-3 animate-confetti"
					style={{
						left: piece.left,
						top: '-12px',
						backgroundColor: piece.color,
						animationDelay: piece.delay,
						animationDuration: piece.duration,
					}}
				/>
			))}
			<style>{`
				@keyframes confetti {
					0% {
						transform: translateY(0) rotate(0deg);
						opacity: 1;
					}
					100% {
						transform: translateY(100vh) rotate(720deg);
						opacity: 0;
					}
				}
				.animate-confetti {
					animation: confetti linear forwards;
				}
			`}</style>
		</div>
	);
}

export function Setup() {
	const navigate = useNavigate();
	const { data: setupStatus, isLoading: statusLoading } = useSetupStatus();

	const currentStep = setupStatus?.current_step ?? 'database';
	const completedSteps = setupStatus?.completed_steps ?? [];

	// Redirect to login if setup is already complete
	useEffect(() => {
		if (setupStatus?.setup_completed) {
			navigate('/login');
		}
	}, [setupStatus?.setup_completed, navigate]);

	// Handle step completion by refetching status (which advances the step)
	const handleStepComplete = () => {
		// The hooks already invalidate the query, so the status will update automatically
	};

	if (statusLoading) {
		return (
			<div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex items-center justify-center">
				<div className="text-center">
					<div className="w-12 h-12 border-4 border-indigo-200 border-t-indigo-600 rounded-full animate-spin mx-auto mb-4" />
					<p className="text-gray-600 dark:text-gray-400">Loading setup...</p>
				</div>
			</div>
		);
	}

	const renderStep = () => {
		switch (currentStep) {
			case 'database':
				return <DatabaseStep onComplete={handleStepComplete} />;
			case 'superuser':
				return <SuperuserStep onComplete={handleStepComplete} />;
			case 'complete':
				return <CompleteSetupStep />;
			default:
				return null;
		}
	};

	return (
		<div className="min-h-screen bg-gray-50 dark:bg-gray-900 flex items-center justify-center py-12 px-4">
			<div className="max-w-4xl w-full">
				{/* Header */}
				<div className="text-center mb-8">
					<img
						src="https://cdn.macjediwizard.com/cdn/Keldris%20Branding%20Images/keldris-webicon-727336cd.png"
						alt="Keldris"
						className="w-64 h-64 mx-auto mb-4"
					/>
					<h1 className="text-3xl font-bold text-gray-900 dark:text-white">
						Keldris Server Setup
					</h1>
					<p className="text-gray-600 dark:text-gray-400 mt-2">
						Configure your Keldris backup server
					</p>
				</div>

				<div className="flex gap-8">
					{/* Sidebar Stepper */}
					<div className="w-64 shrink-0">
						<div className="sticky top-6">
							<VerticalStepper
								steps={SETUP_STEPS}
								currentStep={currentStep}
								completedSteps={completedSteps}
							/>
						</div>
					</div>

					{/* Main Content */}
					<div className="flex-1">
						<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 shadow-sm p-6">
							{renderStep()}
						</div>
					</div>
				</div>
			</div>
		</div>
	);
}
