import { useState, useEffect } from 'react';
import { api, type AppInfo } from '../api/client';

const CATEGORY_ICONS: Record<string, string> = {
    productivity: '📂', media: '🎬', network: '🌐', security: '🔒',
    development: '💻', automation: '⚡', monitoring: '📈', system: '🔧', ai: '🧠',
    database: '🗄️', communication: '💬', finance: '💰', knowledge: '📚',
    photos: '📷', music: '🎵', food: '🍽️', tools: '🛠️', storage: '📦',
};

export default function Apps() {
    const [apps, setApps] = useState<AppInfo[]>([]);
    const [loading, setLoading] = useState(true);
    const [filter, setFilter] = useState('all');
    const [error, setError] = useState('');

    useEffect(() => {
        api.getApps()
            .then(data => {
                setApps(data.apps || []);
                setLoading(false);
            })
            .catch(err => {
                setError(err.message);
                setLoading(false);
            });
    }, []);

    const categories = ['all', ...new Set(apps.map(a => a.category))];
    const filtered = filter === 'all' ? apps : apps.filter(a => a.category === filter);

    if (loading) return <div className="page-header"><h2>Loading apps...</h2></div>;

    return (
        <>
            <div className="page-header">
                <h2>App Marketplace</h2>
                <p>{apps.length} self-hosted apps available — {apps.filter(a => a.installed).length} installed</p>
            </div>

            {error && (
                <div className="card" style={{ marginBottom: 16, borderColor: 'rgba(239, 68, 68, 0.3)' }}>
                    <span style={{ color: 'var(--accent-red)' }}>⚠️ {error}</span>
                </div>
            )}

            <div style={{ display: 'flex', gap: 6, marginBottom: 20, flexWrap: 'wrap' }}>
                {categories.map(cat => (
                    <button
                        key={cat}
                        className={`btn btn-sm ${filter === cat ? 'btn-primary' : ''}`}
                        onClick={() => setFilter(cat)}
                    >
                        {cat === 'all' ? `🏠 All (${apps.length})` : `${CATEGORY_ICONS[cat] || '📦'} ${cat}`}
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
                        <div className="app-card-actions">
                            {app.installed ? (
                                <span className="badge badge-green">Installed</span>
                            ) : (
                                <span className="badge" style={{ background: 'rgba(255,255,255,0.05)', color: 'var(--text-muted)' }}>Available</span>
                            )}
                        </div>
                    </div>
                ))}
            </div>
        </>
    );
}
