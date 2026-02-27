import { BrowserRouter, Routes, Route, NavLink } from 'react-router-dom';
import './index.css';
import Overview from './pages/Overview';
import Services from './pages/Services';
import Apps from './pages/Apps';
import AI from './pages/AI';
import Backups from './pages/Backups';
import Settings from './pages/Settings';

const NAV_ITEMS = [
  { path: '/', label: 'Overview', icon: '📊' },
  { path: '/services', label: 'Services', icon: '🔧' },
  { path: '/apps', label: 'Apps', icon: '📦' },
  { path: '/ai', label: 'AI', icon: '🧠' },
  { path: '/backups', label: 'Backups', icon: '💾' },
  { path: '/settings', label: 'Settings', icon: '⚙️' },
];

function App() {
  return (
    <BrowserRouter>
      <div className="app-layout">
        <aside className="sidebar">
          <div className="sidebar-brand">
            <h1>Sovereign Stack</h1>
            <span>Personal Cloud</span>
          </div>
          <nav className="sidebar-nav">
            {NAV_ITEMS.map(item => (
              <NavLink
                key={item.path}
                to={item.path}
                end={item.path === '/'}
                className={({ isActive }) => `nav-item ${isActive ? 'active' : ''}`}
              >
                <span className="nav-icon">{item.icon}</span>
                {item.label}
              </NavLink>
            ))}
          </nav>
          <div className="sidebar-footer">
            <div className="mono">v0.1.0-dev</div>
          </div>
        </aside>
        <main className="main-content">
          <Routes>
            <Route path="/" element={<Overview />} />
            <Route path="/services" element={<Services />} />
            <Route path="/apps" element={<Apps />} />
            <Route path="/ai" element={<AI />} />
            <Route path="/backups" element={<Backups />} />
            <Route path="/settings" element={<Settings />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  );
}

export default App;
