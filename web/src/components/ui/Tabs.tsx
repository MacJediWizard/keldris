import { type ReactNode, createContext, useContext, useState } from 'react';

interface TabsContextValue {
	activeTab: string;
	setActiveTab: (id: string) => void;
}

const TabsContext = createContext<TabsContextValue | null>(null);

function useTabsContext() {
	const context = useContext(TabsContext);
	if (!context) throw new Error('Tabs components must be used within <Tabs>');
	return context;
}

interface TabsProps {
	defaultTab: string;
	children: ReactNode;
	onChange?: (tabId: string) => void;
}

export function Tabs({ defaultTab, children, onChange }: TabsProps) {
	const [activeTab, setActiveTab] = useState(defaultTab);

	function handleChange(tabId: string) {
		setActiveTab(tabId);
		onChange?.(tabId);
	}

	return (
		<TabsContext.Provider value={{ activeTab, setActiveTab: handleChange }}>
			{children}
		</TabsContext.Provider>
	);
}

interface TabListProps {
	children: ReactNode;
}

export function TabList({ children }: TabListProps) {
	return (
		<div className="border-b border-gray-200" role="tablist">
			<nav className="-mb-px flex space-x-8">{children}</nav>
		</div>
	);
}

interface TabProps {
	id: string;
	children: ReactNode;
}

export function Tab({ id, children }: TabProps) {
	const { activeTab, setActiveTab } = useTabsContext();
	const isActive = activeTab === id;

	return (
		<button
			type="button"
			role="tab"
			aria-selected={isActive}
			aria-controls={`tabpanel-${id}`}
			onClick={() => setActiveTab(id)}
			className={`whitespace-nowrap border-b-2 px-1 py-3 text-sm font-medium ${
				isActive
					? 'border-indigo-500 text-indigo-600'
					: 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700'
			}`}
		>
			{children}
		</button>
	);
}

interface TabPanelProps {
	id: string;
	children: ReactNode;
}

export function TabPanel({ id, children }: TabPanelProps) {
	const { activeTab } = useTabsContext();

	if (activeTab !== id) return null;

	return (
		<div role="tabpanel" id={`tabpanel-${id}`} className="py-4">
			{children}
		</div>
	);
}
