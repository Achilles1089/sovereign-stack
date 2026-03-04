import { BrowserRouter, Routes, Route, NavLink } from 'react-router-dom';
import { Component, type ReactNode } from 'react';
import './index.css';
import Apps from './pages/Apps';
import AI from './pages/AI';
import Backups from './pages/Backups';
import Settings from './pages/Settings';

class ErrorBoundary extends Component<{ children: ReactNode }, { error: Error | null }> {
  state = { error: null as Error | null };
  static getDerivedStateFromError(error: Error) { return { error }; }
  render() {
    if (this.state.error) {
      return (
        <div style={{ padding: 40, color: '#ff6b6b', fontFamily: 'monospace', background: '#1a1a2e', minHeight: '50vh' }}>
          <h2>⚠️ Component Crash</h2>
          <pre style={{ whiteSpace: 'pre-wrap', fontSize: 13 }}>{this.state.error.message}</pre>
          <pre style={{ whiteSpace: 'pre-wrap', fontSize: 11, color: '#888', marginTop: 12 }}>{this.state.error.stack}</pre>
          <button onClick={() => this.setState({ error: null })} style={{ marginTop: 20, padding: '8px 16px', cursor: 'pointer' }}>
            Retry
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}

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
          <ErrorBoundary>
            <Routes>
              <Route path="/" element={<AI />} />
              <Route path="/apps" element={<Apps />} />
              <Route path="/backups" element={<Backups />} />
              <Route path="/settings" element={<Settings />} />
            </Routes>
          </ErrorBoundary>
        </main>
      </div>
    </BrowserRouter>
  );
}

export default App;
