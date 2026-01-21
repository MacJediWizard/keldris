interface StepperStep {
	id: string;
	label: string;
	description?: string;
}

interface StepperProps {
	steps: StepperStep[];
	currentStep: string;
	completedSteps: string[];
	onStepClick?: (stepId: string) => void;
}

export function Stepper({
	steps,
	currentStep,
	completedSteps,
	onStepClick,
}: StepperProps) {
	const currentIndex = steps.findIndex((s) => s.id === currentStep);

	return (
		<nav aria-label="Progress" className="w-full">
			<ol className="flex items-center">
				{steps.map((step, index) => {
					const isCompleted = completedSteps.includes(step.id);
					const isCurrent = step.id === currentStep;
					const isClickable = onStepClick && (isCompleted || isCurrent);
					const isLast = index === steps.length - 1;

					return (
						<li key={step.id} className={`relative ${isLast ? '' : 'flex-1'}`}>
							<div className="flex items-center">
								<button
									type="button"
									onClick={() => isClickable && onStepClick?.(step.id)}
									disabled={!isClickable}
									className={`group flex h-10 w-10 items-center justify-center rounded-full border-2 transition-colors ${
										isCompleted
											? 'border-indigo-600 bg-indigo-600 text-white'
											: isCurrent
												? 'border-indigo-600 bg-white text-indigo-600'
												: 'border-gray-300 bg-white text-gray-500'
									} ${isClickable ? 'cursor-pointer hover:bg-indigo-50' : 'cursor-default'}`}
									title={step.label}
								>
									{isCompleted ? (
										<svg
											aria-hidden="true"
											className="h-5 w-5"
											fill="currentColor"
											viewBox="0 0 20 20"
										>
											<path
												fillRule="evenodd"
												d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
												clipRule="evenodd"
											/>
										</svg>
									) : (
										<span className="text-sm font-medium">{index + 1}</span>
									)}
								</button>
								{!isLast && (
									<div
										className={`ml-2 h-0.5 flex-1 ${
											index < currentIndex || isCompleted
												? 'bg-indigo-600'
												: 'bg-gray-300'
										}`}
									/>
								)}
							</div>
							<div className="mt-2 hidden md:block">
								<span
									className={`text-xs font-medium ${
										isCurrent
											? 'text-indigo-600'
											: isCompleted
												? 'text-gray-900'
												: 'text-gray-500'
									}`}
								>
									{step.label}
								</span>
							</div>
						</li>
					);
				})}
			</ol>
		</nav>
	);
}

interface VerticalStepperProps {
	steps: StepperStep[];
	currentStep: string;
	completedSteps: string[];
	onStepClick?: (stepId: string) => void;
}

export function VerticalStepper({
	steps,
	currentStep,
	completedSteps,
	onStepClick,
}: VerticalStepperProps) {
	return (
		<nav aria-label="Progress" className="w-full">
			<ol className="space-y-4">
				{steps.map((step, index) => {
					const isCompleted = completedSteps.includes(step.id);
					const isCurrent = step.id === currentStep;
					const isClickable = onStepClick && (isCompleted || isCurrent);
					const isLast = index === steps.length - 1;

					return (
						<li key={step.id} className="relative">
							<div className="flex items-start">
								{/* Vertical line connector */}
								{!isLast && (
									<div
										className={`absolute left-5 top-10 h-full w-0.5 -translate-x-1/2 ${
											isCompleted ? 'bg-indigo-600' : 'bg-gray-300'
										}`}
									/>
								)}

								<button
									type="button"
									onClick={() => isClickable && onStepClick?.(step.id)}
									disabled={!isClickable}
									className={`relative z-10 flex h-10 w-10 shrink-0 items-center justify-center rounded-full border-2 transition-colors ${
										isCompleted
											? 'border-indigo-600 bg-indigo-600 text-white'
											: isCurrent
												? 'border-indigo-600 bg-white text-indigo-600'
												: 'border-gray-300 bg-white text-gray-500'
									} ${isClickable ? 'cursor-pointer hover:bg-indigo-50' : 'cursor-default'}`}
								>
									{isCompleted ? (
										<svg
											aria-hidden="true"
											className="h-5 w-5"
											fill="currentColor"
											viewBox="0 0 20 20"
										>
											<path
												fillRule="evenodd"
												d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
												clipRule="evenodd"
											/>
										</svg>
									) : (
										<span className="text-sm font-medium">{index + 1}</span>
									)}
								</button>

								<div className="ml-4 min-w-0 flex-1">
									<span
										className={`block text-sm font-medium ${
											isCurrent
												? 'text-indigo-600'
												: isCompleted
													? 'text-gray-900'
													: 'text-gray-500'
										}`}
									>
										{step.label}
									</span>
									{step.description && (
										<span className="mt-0.5 block text-xs text-gray-500">
											{step.description}
										</span>
									)}
								</div>
							</div>
						</li>
					);
				})}
			</ol>
		</nav>
	);
}
