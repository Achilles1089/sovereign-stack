import { BrowserRouter, Routes, Route, NavLink } from 'react-router-dom';
import './index.css';
import Apps from './pages/Apps';
import AI from './pages/AI';
import Backups from './pages/Backups';
import Settings from './pages/Settings';

const TABS = [
  { path: '/', label: 'AI', icon: '🧠' },
  { path: '/apps', label: 'Apps', icon: '📦' },
  { path: '/backups', label: 'Backups', icon: '💾' },
  { path: '/settings', label: 'Settings', icon: '⚙️' },
];

function App() {
  return (
    <BrowserRouter>
      <div className="app-layout">
        <header className="top-bar">
          <div className="top-bar-brand">
            <h1>Sovereign Stack</h1>
            <span>Personal Cloud</span>
          </div>
          <nav className="top-bar-tabs">
            {TABS.map(item => (
              <NavLink
                key={item.path}
                to={item.path}
                end={item.path === '/'}
                className={({ isActive }) => `tab-item ${isActive ? 'active' : ''}`}
              >
                <span className="tab-icon">{item.icon}</span>
                {item.label}
              </NavLink>
            ))}
          </nav>
          <div className="top-bar-spacer" />
        </header>
        <main className="main-content">
          <Routes>
            <Route path="/" element={<AI />} />
            <Route path="/apps" element={<Apps />} />
            <Route path="/backups" element={<Backups />} />
            <Route path="/settings" element={<Settings />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  );
}

export default App;
