import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useCreateOrganization } from '../hooks/useOrganizations';

export function NewOrganization() {
	const navigate = useNavigate();
	const createOrganization = useCreateOrganization();

	const [name, setName] = useState('');
	const [slug, setSlug] = useState('');
	const [autoSlug, setAutoSlug] = useState(true);

	const handleNameChange = (value: string) => {
		setName(value);
		if (autoSlug) {
			setSlug(
				value
					.toLowerCase()
					.replace(/[^a-z0-9]+/g, '-')
					.replace(/^-|-$/g, ''),
			);
		}
	};

	const handleSlugChange = (value: string) => {
		setAutoSlug(false);
		setSlug(value.toLowerCase().replace(/[^a-z0-9-]/g, '-'));
	};

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		try {
			await createOrganization.mutateAsync({ name, slug });
			navigate('/');
		} catch {
			// Error handled by mutation
		}
	};

	return (
		<div className="max-w-lg mx-auto">
			<div className="mb-6">
				<Link
					to="/"
					className="text-sm text-gray-500 dark:text-gray-400 hover:text-gray-700 inline-flex items-center gap-1"
					className="text-sm text-gray-500 hover:text-gray-700 inline-flex items-center gap-1"
				>
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
							d="M15 19l-7-7 7-7"
						/>
					</svg>
					Back to Dashboard
				</Link>
			</div>

			<div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700">
				<div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
					<h1 className="text-xl font-semibold text-gray-900">
						Create New Organization
					</h1>
					<p className="text-sm text-gray-500 dark:text-gray-400 mt-1">
			<div className="bg-white rounded-lg border border-gray-200">
				<div className="px-6 py-4 border-b border-gray-200">
					<h1 className="text-xl font-semibold text-gray-900">
						Create New Organization
					</h1>
					<p className="text-sm text-gray-500 mt-1">
						Organizations help you manage backup resources separately
					</p>
				</div>

				<form onSubmit={handleSubmit} className="p-6 space-y-4">
					<div>
						<label
							htmlFor="name"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							className="block text-sm font-medium text-gray-700 mb-1"
						>
							Organization Name
						</label>
						<input
							type="text"
							id="name"
							value={name}
							onChange={(e) => handleNameChange(e.target.value)}
							placeholder="My Company"
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							required
						/>
					</div>

					<div>
						<label
							htmlFor="slug"
							className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1"
							className="block text-sm font-medium text-gray-700 mb-1"
						>
							URL Slug
						</label>
						<input
							type="text"
							id="slug"
							value={slug}
							onChange={(e) => handleSlugChange(e.target.value)}
							placeholder="my-company"
							className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-white rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							pattern="[a-z0-9-]+"
							required
						/>
						<p className="mt-1 text-xs text-gray-500 dark:text-gray-400 dark:text-gray-400">
							className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
							pattern="[a-z0-9-]+"
							required
						/>
						<p className="mt-1 text-xs text-gray-500">
							Only lowercase letters, numbers, and hyphens. This will be used in
							URLs.
						</p>
					</div>

					{createOrganization.isError && (
						<div className="bg-red-50 border border-red-200 rounded-lg p-4">
							<p className="text-sm text-red-600 dark:text-red-400">
							<p className="text-sm text-red-600">
								Failed to create organization. The slug may already be taken.
							</p>
						</div>
					)}

					<div className="flex justify-end gap-3 pt-4">
						<Link
							to="/"
							className="px-4 py-2 text-gray-700 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
							className="px-4 py-2 text-gray-700 hover:bg-gray-100 rounded-lg transition-colors"
						>
							Cancel
						</Link>
						<button
							type="submit"
							disabled={createOrganization.isPending || !name || !slug}
							className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
						>
							{createOrganization.isPending
								? 'Creating...'
								: 'Create Organization'}
						</button>
					</div>
				</form>
			</div>

			<div className="mt-6 bg-blue-50 border border-blue-200 rounded-lg p-4">
				<h3 className="text-sm font-medium text-blue-900 mb-2">
					What happens next?
				</h3>
				<ul className="text-sm text-blue-700 space-y-1">
					<li>• You'll be set as the owner of the new organization</li>
					<li>• You can switch between organizations using the dropdown</li>
					<li>• Invite team members from the Members page</li>
					<li>• Each organization has separate agents and backups</li>
				</ul>
			</div>
		</div>
	);
}

export default NewOrganization;
