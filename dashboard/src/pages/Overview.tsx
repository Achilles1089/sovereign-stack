import { useState, useEffect } from 'react';

// Mock data for development — will be replaced with real API calls
const MOCK_SERVICES = [
    { name: 'sovereign-postgres', running: true, status: 'Up 2 hours', ports: '5432', image: 'postgres:16-alpine' },
    { name: 'sovereign-caddy', running: true, status: 'Up 2 hours', ports: '80, 443', image: 'caddy:2-alpine' },
    { name: 'sovereign-ollama', running: true, status: 'Up 2 hours', ports: '11434', image: 'ollama/ollama' },
];

const MOCK_RESOURCES = {
    cpu: 23,
    ram: { used: 4200, total: 8192 },
    disk: { used: 120, total: 500 },
    gpu: { name: 'Apple M1 Ultra', memory: 131072, type: 'apple_silicon' },
};

export default function Overview() {
    const [uptime] = useState('2h 34m');
    const [services] = useState(MOCK_SERVICES);
    const [resources] = useState(MOCK_RESOURCES);

    const ramPercent = Math.round((resources.ram.used / resources.ram.total) * 100);
    const diskPercent = Math.round((resources.disk.used / resources.disk.total) * 100);
    const runningCount = services.filter(s => s.running).length;

    return (
        <>
            <div className="page-header">
                <h2>Overview</h2>
                <p>Your sovereign server at a glance</p>
            </div>

            {/* Stats Grid */}
            <div className="grid-4" style={{ marginBottom: 24 }}>
                <div className="card">
                    <div className="stat-value" style={{ color: 'var(--accent-green)' }}>{runningCount}/{services.length}</div>
                    <div className="stat-label">Services Running</div>
                </div>
                <div className="card">
                    <div className="stat-value">{uptime}</div>
                    <div className="stat-label">Uptime</div>
                </div>
                <div className="card">
                    <div className="stat-value" style={{ color: 'var(--accent-cyan)' }}>12</div>
                    <div className="stat-label">Apps Available</div>
                </div>
                <div className="card">
                    <div className="stat-value" style={{ color: 'var(--accent-primary)' }}>1</div>
                    <div className="stat-label">AI Models</div>
                </div>
            </div>

            {/* Resources */}
            <div className="grid-2" style={{ marginBottom: 24 }}>
                <div className="card">
                    <div className="card-title">System Resources</div>

                    <div style={{ marginBottom: 16 }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13 }}>
                            <span>CPU</span>
                            <span className="mono">{resources.cpu}%</span>
                        </div>
                        <div className="progress-bar">
                            <div className="progress-fill cpu" style={{ width: `${resources.cpu}%` }} />
                        </div>
                    </div>

                    <div style={{ marginBottom: 16 }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13 }}>
                            <span>RAM</span>
                            <span className="mono">{(resources.ram.used / 1024).toFixed(1)} / {(resources.ram.total / 1024).toFixed(1)} GB</span>
                        </div>
                        <div className="progress-bar">
                            <div className="progress-fill ram" style={{ width: `${ramPercent}%` }} />
                        </div>
                    </div>

                    <div style={{ marginBottom: 16 }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13 }}>
                            <span>Disk</span>
                            <span className="mono">{resources.disk.used} / {resources.disk.total} GB</span>
                        </div>
                        <div className="progress-bar">
                            <div className="progress-fill disk" style={{ width: `${diskPercent}%` }} />
                        </div>
                    </div>

                    {resources.gpu.type !== 'none' && (
                        <div>
                            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13 }}>
                                <span>GPU</span>
                                <span className="mono">{resources.gpu.name}</span>
                            </div>
                            <div className="progress-bar">
                                <div className="progress-fill gpu" style={{ width: '15%' }} />
                            </div>
                        </div>
                    )}
                </div>

                <div className="card">
                    <div className="card-title">Services</div>
                    <table>
                        <thead>
                            <tr>
                                <th>Service</th>
                                <th>Status</th>
                                <th>Ports</th>
                            </tr>
                        </thead>
                        <tbody>
                            {services.map(s => (
                                <tr key={s.name}>
                                    <td style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                                        <span className={`status-dot ${s.running ? 'up' : 'down'}`} />
                                        {s.name.replace('sovereign-', '')}
                                    </td>
                                    <td><span className="mono" style={{ fontSize: 12 }}>{s.status}</span></td>
                                    <td><span className="mono">{s.ports}</span></td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            </div>

            {/* Quick Actions */}
            <div className="card">
                <div className="card-title">Quick Actions</div>
                <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                    <button className="btn btn-primary">🧠 Chat with AI</button>
                    <button className="btn">📦 Install App</button>
                    <button className="btn">💾 Create Backup</button>
                    <button className="btn">🔄 Update All</button>
                </div>
            </div>
        </>
    );
}
