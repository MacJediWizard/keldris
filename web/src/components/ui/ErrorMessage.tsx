interface ErrorMessageProps {
	message: string;
}

export function ErrorMessage({ message }: ErrorMessageProps) {
	return (
		<div className="rounded-md border border-red-200 bg-red-50 p-4">
			<div className="flex items-center gap-3">
				<svg
					className="h-5 w-5 text-red-400"
					fill="currentColor"
					viewBox="0 0 20 20"
					aria-hidden="true"
				>
					<path
						fillRule="evenodd"
						d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
						clipRule="evenodd"
					/>
				</svg>
				<p className="text-sm font-medium text-red-800" role="alert">
					{message}
				</p>
			</div>
		</div>
	);
}
