import { useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { useLocale } from '../hooks/useLocale';

interface DocPage {
	slug: string;
	title: string;
	description?: string;
	content?: string;
	html_content?: string;
}

interface DocSearchResult {
	slug: string;
	title: string;
	excerpt: string;
	score: number;
}

const docPages = [
	{ slug: 'getting-started', labelKey: 'help.gettingStarted' },
	{ slug: 'installation', labelKey: 'help.installation' },
	{ slug: 'configuration', labelKey: 'help.configuration' },
	{ slug: 'agent-deployment', labelKey: 'help.agentDeployment' },
	{ slug: 'agent-installation', labelKey: 'help.agentDeployment' },
	{ slug: 'api-reference', labelKey: 'help.apiReference' },
	{ slug: 'troubleshooting', labelKey: 'help.troubleshooting' },
];

function DocSidebar({
	currentSlug,
	onSearch,
}: { currentSlug?: string; onSearch: (query: string) => void }) {
	const { t } = useLocale();
	const [searchQuery, setSearchQuery] = useState('');

	const handleSearch = (e: React.FormEvent) => {
		e.preventDefault();
		onSearch(searchQuery);
	};

	return (
		<div className="w-64 bg-white border-r border-gray-200 p-4">
			<h2 className="text-lg font-semibold text-gray-900 mb-4">
				{t('help.documentation')}
			</h2>

			<form onSubmit={handleSearch} className="mb-4">
				<div className="relative">
					<input
						type="text"
						value={searchQuery}
						onChange={(e) => setSearchQuery(e.target.value)}
						placeholder={t('common.search')}
						className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
					/>
					<button
						type="submit"
						className="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
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
								d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
							/>
						</svg>
					</button>
				</div>
			</form>

			<nav>
				<ul className="space-y-1">
					{docPages.map((page) => (
						<li key={page.slug}>
							<Link
								to={`/docs/${page.slug}`}
								className={`block px-3 py-2 text-sm rounded-lg transition-colors ${
									currentSlug === page.slug
										? 'bg-indigo-100 text-indigo-700 font-medium'
										: 'text-gray-600 hover:bg-gray-100 hover:text-gray-900'
								}`}
							>
								{t(page.labelKey)}
							</Link>
						</li>
					))}
				</ul>
			</nav>

			<div className="mt-6 pt-4 border-t border-gray-200">
				<a
					href="/api/docs"
					target="_blank"
					rel="noopener noreferrer"
					className="flex items-center gap-2 px-3 py-2 text-sm text-gray-600 hover:bg-gray-100 hover:text-gray-900 rounded-lg transition-colors"
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
							d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
						/>
					</svg>
					{t('help.swaggerDocs')}
				</a>
			</div>
		</div>
	);
}

function MarkdownRenderer({ content }: { content: string }) {
	// Basic markdown rendering - in production, use a proper markdown library
	const html = content
		// Headers
		.replace(
			/^#### (.+)$/gm,
			'<h4 class="text-lg font-medium mt-6 mb-2">$1</h4>',
		)
		.replace(
			/^### (.+)$/gm,
			'<h3 class="text-xl font-semibold mt-8 mb-3">$1</h3>',
		)
		.replace(
			/^## (.+)$/gm,
			'<h2 class="text-2xl font-bold mt-10 mb-4 pb-2 border-b">$1</h2>',
		)
		.replace(/^# (.+)$/gm, '<h1 class="text-3xl font-bold mb-6">$1</h1>')
		// Code blocks
		.replace(
			/```(\w+)?\n([\s\S]*?)```/g,
			'<pre class="bg-gray-900 text-gray-100 p-4 rounded-lg overflow-x-auto my-4"><code>$2</code></pre>',
		)
		// Inline code
		.replace(
			/`([^`]+)`/g,
			'<code class="bg-gray-100 px-1.5 py-0.5 rounded text-sm">$1</code>',
		)
		// Bold
		.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
		// Italic
		.replace(/\*([^*]+)\*/g, '<em>$1</em>')
		// Links
		.replace(
			/\[([^\]]+)\]\(([^)]+)\)/g,
			'<a href="$2" class="text-indigo-600 hover:text-indigo-800 underline">$1</a>',
		)
		// Unordered lists
		.replace(/^- (.+)$/gm, '<li class="ml-4">$1</li>')
		// Paragraphs (simple approach)
		.replace(
			/^(?!<[hluop]|```)((?!<).+)$/gm,
			'<p class="my-3 text-gray-700 leading-relaxed">$1</p>',
		);

	return (
		<div
			className="prose prose-indigo max-w-none"
			dangerouslySetInnerHTML={{ __html: html }}
		/>
	);
}

function SearchResults({
	results,
	query,
	onClose,
}: {
	results: DocSearchResult[];
	query: string;
	onClose: () => void;
}) {
	const { t } = useLocale();

	return (
		<div className="flex-1 p-6">
			<div className="flex items-center justify-between mb-6">
				<h2 className="text-xl font-semibold">Search results for "{query}"</h2>
				<button
					type="button"
					onClick={onClose}
					className="text-gray-500 hover:text-gray-700"
				>
					{t('common.close')}
				</button>
			</div>

			{results.length === 0 ? (
				<p className="text-gray-500">No results found</p>
			) : (
				<div className="space-y-4">
					{results.map((result) => (
						<Link
							key={result.slug}
							to={`/docs/${result.slug}`}
							onClick={onClose}
							className="block p-4 bg-white rounded-lg border border-gray-200 hover:border-indigo-300 hover:shadow-sm transition-all"
						>
							<h3 className="font-medium text-gray-900">{result.title}</h3>
							<p className="text-sm text-gray-500 mt-1">{result.excerpt}</p>
						</Link>
					))}
				</div>
			)}
		</div>
	);
}

export function Documentation() {
	const { slug } = useParams<{ slug: string }>();
	const { t } = useLocale();
	const [doc, setDoc] = useState<DocPage | null>(null);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [searchResults, setSearchResults] = useState<DocSearchResult[] | null>(
		null,
	);
	const [searchQuery, setSearchQuery] = useState('');

	useEffect(() => {
		const fetchDoc = async () => {
			if (!slug) {
				setLoading(false);
				return;
			}

			setLoading(true);
			setError(null);

			try {
				const response = await fetch(`/docs/${slug}`);
				if (!response.ok) {
					throw new Error('Documentation page not found');
				}
				const data = await response.json();
				setDoc(data);
			} catch (err) {
				setError(
					err instanceof Error ? err.message : 'Failed to load documentation',
				);
			} finally {
				setLoading(false);
			}
		};

		fetchDoc();
	}, [slug]);

	const handleSearch = async (query: string) => {
		if (!query.trim()) {
			setSearchResults(null);
			return;
		}

		setSearchQuery(query);

		try {
			const response = await fetch(
				`/docs/search?q=${encodeURIComponent(query)}`,
			);
			if (response.ok) {
				const data = await response.json();
				setSearchResults(data.results);
			}
		} catch (err) {
			console.error('Search failed:', err);
		}
	};

	const clearSearch = () => {
		setSearchResults(null);
		setSearchQuery('');
	};

	// Show documentation index if no slug
	if (!slug) {
		return (
			<div className="flex h-full">
				<DocSidebar currentSlug={undefined} onSearch={handleSearch} />
				<div className="flex-1 p-6">
					<h1 className="text-3xl font-bold mb-6">{t('help.documentation')}</h1>
					<p className="text-gray-600 mb-8">
						Welcome to the Keldris documentation. Select a topic from the
						sidebar to get started.
					</p>

					<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
						{docPages.map((page) => (
							<Link
								key={page.slug}
								to={`/docs/${page.slug}`}
								className="p-4 bg-white rounded-lg border border-gray-200 hover:border-indigo-300 hover:shadow-md transition-all"
							>
								<h3 className="font-medium text-gray-900">
									{t(page.labelKey)}
								</h3>
							</Link>
						))}
					</div>
				</div>
			</div>
		);
	}

	if (searchResults) {
		return (
			<div className="flex h-full">
				<DocSidebar currentSlug={slug} onSearch={handleSearch} />
				<SearchResults
					results={searchResults}
					query={searchQuery}
					onClose={clearSearch}
				/>
			</div>
		);
	}

	if (loading) {
		return (
			<div className="flex h-full">
				<DocSidebar currentSlug={slug} onSearch={handleSearch} />
				<div className="flex-1 flex items-center justify-center">
					<div className="w-8 h-8 border-4 border-indigo-200 border-t-indigo-600 rounded-full animate-spin" />
				</div>
			</div>
		);
	}

	if (error || !doc) {
		return (
			<div className="flex h-full">
				<DocSidebar currentSlug={slug} onSearch={handleSearch} />
				<div className="flex-1 p-6">
					<div className="bg-red-50 border border-red-200 rounded-lg p-4">
						<h2 className="text-red-800 font-medium">
							Documentation not found
						</h2>
						<p className="text-red-600 text-sm mt-1">
							The requested documentation page could not be found.
						</p>
						<Link
							to="/docs"
							className="inline-block mt-3 text-sm text-indigo-600 hover:text-indigo-800"
						>
							View all documentation
						</Link>
					</div>
				</div>
			</div>
		);
	}

	return (
		<div className="flex h-full">
			<DocSidebar currentSlug={slug} onSearch={handleSearch} />
			<div className="flex-1 p-6 overflow-y-auto">
				<MarkdownRenderer content={doc.content || ''} />
			</div>
		</div>
	);
}
