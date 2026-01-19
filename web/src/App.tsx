import { BrowserRouter, Route, Routes } from 'react-router-dom';
import { Layout } from './components/Layout';
import { Agents } from './pages/Agents';
import { Backups } from './pages/Backups';
import { Dashboard } from './pages/Dashboard';
import { Repositories } from './pages/Repositories';
import { Schedules } from './pages/Schedules';

function App() {
	return (
		<BrowserRouter>
			<Routes>
				<Route path="/" element={<Layout />}>
					<Route index element={<Dashboard />} />
					<Route path="agents" element={<Agents />} />
					<Route path="repositories" element={<Repositories />} />
					<Route path="schedules" element={<Schedules />} />
					<Route path="backups" element={<Backups />} />
				</Route>
			</Routes>
		</BrowserRouter>
	);
}

export default App;
