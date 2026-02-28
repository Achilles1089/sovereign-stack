import { useState, useEffect } from 'react';
import { api, type AppInfo } from '../api/client';

const CATEGORY_ICONS: Record<string, string> = {
    productivity: '📂', media: '🎬', network: '🌐', security: '🔒',
    development: '💻', automation: '⚡', monitoring: '📈', system: '🔧', ai: '🧠',
    'smart-home': '🏠', analytics: '📊', lifestyle: '🍽️', finance: '💰',
};

export default function Apps() {
    const [apps, setApps] = useState<AppInfo[]>([]);
    const [loading, setLoading] = useState(true);
    const [filter, setFilter] = useState('all');
    const [installing, setInstalling] = useState<string | null>(null);
    const [message, setMessage] = useState('');

    const fetchApps = () => {
        api.getApps()
            .then(data => { setApps(data.apps || []); setLoading(false); })
            .catch(() => setLoading(false));
    };

    useEffect(() => { fetchApps(); }, []);

    const handleInstall = async (name: string) => {
        setInstalling(name);
        setMessage('');
        try {
            const res = await api.installApp(name);
            if (res.error) {
                setMessage(`⚠️ ${res.error}`);
            } else {
                setMessage(`✅ ${res.message}`);
                fetchApps(); // Refresh list
            }
        } catch {
            setMessage('⚠️ Install failed — check server logs');
        }
        setInstalling(null);
    };

    const handleRemove = async (name: string) => {
        if (!confirm(`Remove ${name}? This will stop and delete the container.`)) return;
        setInstalling(name);
        setMessage('');
        try {
            const res = await api.removeApp(name);
            if (res.error) {
                setMessage(`⚠️ ${res.error}`);
            } else {
                setMessage(`✅ ${res.message}`);
                fetchApps();
            }
        } catch {
            setMessage('⚠️ Remove failed');
        }
        setInstalling(null);
    };

    const categories = ['all', ...new Set(apps.map(a => a.category))];
    const filtered = filter === 'all' ? apps : apps.filter(a => a.category === filter);

    if (loading) return <div className="page-header"><h2>Loading apps...</h2></div>;

    return (
        <>
            <div className="page-header">
                <h2>App Marketplace</h2>
                <p>{apps.length} self-hosted apps — {apps.filter(a => a.installed).length} installed</p>
            </div>

            {message && (
                <div className="card" style={{ marginBottom: 16, padding: '12px 16px' }}>
                    {message}
                </div>
            )}

            <div style={{ display: 'flex', gap: 6, marginBottom: 20, flexWrap: 'wrap' }}>
                {categories.map(cat => (
                    <button
                        key={cat}
                        className={`btn btn-sm ${filter === cat ? 'btn-primary' : ''}`}
                        onClick={() => setFilter(cat)}
                    >
                        {cat === 'all' ? `All (${apps.length})` : `${CATEGORY_ICONS[cat] || '📦'} ${cat}`}
                    </button>
                ))}
            </div>

            <div className="grid-3">
                {filtered.map(app => (
                    <div key={app.name} className="app-card">
                        <div className="app-card-header">
                            <h3>{CATEGORY_ICONS[app.category] || '📦'} {app.display_name}</h3>
                            <span className="mono" style={{ fontSize: 11, color: 'var(--text-muted)' }}>v{app.version}</span>
                        </div>
                        <p>{app.description}</p>
                        <div className="app-card-actions" style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                            {app.installed ? (
                                <>
                                    <span className="badge badge-green">Installed</span>
                                    <button
                                        className="btn btn-sm"
                                        style={{ color: 'var(--accent-red)', fontSize: 11 }}
                                        onClick={() => handleRemove(app.name)}
                                        disabled={installing === app.name}
                                    >
                                        {installing === app.name ? '...' : 'Remove'}
                                    </button>
                                </>
                            ) : (
                                <button
                                    className="btn btn-sm btn-primary"
                                    onClick={() => handleInstall(app.name)}
                                    disabled={installing !== null}
                                >
                                    {installing === app.name ? '⏳ Installing...' : '📦 Install'}
                                </button>
                            )}
                        </div>
                    </div>
                ))}
            </div>
        </>
    );
}
