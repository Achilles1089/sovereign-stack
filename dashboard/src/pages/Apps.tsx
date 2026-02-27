import { useState } from 'react';

interface App {
    name: string;
    displayName: string;
    description: string;
    category: string;
    version: string;
    installed: boolean;
}

const APPS: App[] = [
    { name: 'nextcloud', displayName: 'Nextcloud', description: 'File sync, share, and collaboration', category: 'productivity', version: '29', installed: false },
    { name: 'jellyfin', displayName: 'Jellyfin', description: 'Media streaming server', category: 'media', version: '10.9', installed: false },
    { name: 'immich', displayName: 'Immich', description: 'Self-hosted photo & video management', category: 'media', version: '1.99', installed: false },
    { name: 'adguard-home', displayName: 'AdGuard Home', description: 'Network-wide ad blocking', category: 'network', version: '0.107', installed: false },
    { name: 'vaultwarden', displayName: 'Vaultwarden', description: 'Password manager (Bitwarden)', category: 'security', version: '1.31', installed: false },
    { name: 'gitea', displayName: 'Gitea', description: 'Lightweight Git hosting', category: 'development', version: '1.22', installed: false },
    { name: 'n8n', displayName: 'n8n', description: 'Workflow automation', category: 'automation', version: '1.76', installed: false },
    { name: 'uptime-kuma', displayName: 'Uptime Kuma', description: 'Beautiful uptime monitoring', category: 'monitoring', version: '1.23', installed: false },
    { name: 'stirling-pdf', displayName: 'Stirling PDF', description: 'PDF manipulation tools', category: 'productivity', version: '0.34', installed: false },
    { name: 'portainer', displayName: 'Portainer', description: 'Container management UI', category: 'system', version: '2.21', installed: false },
    { name: 'syncthing', displayName: 'Syncthing', description: 'P2P file synchronization', category: 'productivity', version: '1.27', installed: false },
    { name: 'open-webui', displayName: 'Open WebUI', description: 'ChatGPT-style AI interface', category: 'ai', version: '0.5', installed: false },
];

const CATEGORY_ICONS: Record<string, string> = {
    productivity: '📂', media: '🎬', network: '🌐', security: '🔒',
    development: '💻', automation: '⚡', monitoring: '📈', system: '🔧', ai: '🧠',
};

export default function Apps() {
    const [apps, setApps] = useState(APPS);
    const [filter, setFilter] = useState('all');
    const categories = ['all', ...new Set(apps.map(a => a.category))];

    const filtered = filter === 'all' ? apps : apps.filter(a => a.category === filter);

    const handleInstall = (name: string) => {
        setApps(prev => prev.map(a => a.name === name ? { ...a, installed: true } : a));
    };

    const handleRemove = (name: string) => {
        setApps(prev => prev.map(a => a.name === name ? { ...a, installed: false } : a));
    };

    return (
        <>
            <div className="page-header">
                <h2>App Marketplace</h2>
                <p>One-click self-hosted apps for your sovereign server</p>
            </div>

            {/* Filter */}
            <div style={{ display: 'flex', gap: 6, marginBottom: 20, flexWrap: 'wrap' }}>
                {categories.map(cat => (
                    <button
                        key={cat}
                        className={`btn btn-sm ${filter === cat ? 'btn-primary' : ''}`}
                        onClick={() => setFilter(cat)}
                    >
                        {cat === 'all' ? '🏠 All' : `${CATEGORY_ICONS[cat] || '📦'} ${cat}`}
                    </button>
                ))}
            </div>

            {/* App Grid */}
            <div className="grid-3">
                {filtered.map(app => (
                    <div key={app.name} className="app-card">
                        <div className="app-card-header">
                            <h3>{CATEGORY_ICONS[app.category] || '📦'} {app.displayName}</h3>
                            <span className="mono" style={{ fontSize: 11, color: 'var(--text-muted)' }}>v{app.version}</span>
                        </div>
                        <p>{app.description}</p>
                        <div className="app-card-actions">
                            {app.installed ? (
                                <>
                                    <span className="badge badge-green">Installed</span>
                                    <button className="btn btn-sm btn-danger" onClick={() => handleRemove(app.name)}>Remove</button>
                                </>
                            ) : (
                                <button className="btn btn-sm btn-primary" onClick={() => handleInstall(app.name)}>Install</button>
                            )}
                        </div>
                    </div>
                ))}
            </div>
        </>
    );
}
