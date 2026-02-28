import { useState, useEffect } from 'react';
import { api, type ServiceStatus, type SystemResources } from '../api/client';

export default function Overview() {
    const [services, setServices] = useState<ServiceStatus[]>([]);
    const [resources, setResources] = useState<SystemResources | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        // Fetch independently — don't block on both
        api.getResources()
            .then(d => { setResources(d); setLoading(false); })
            .catch(() => setLoading(false));
        api.getStatus()
            .then(d => setServices(d.services || []))
            .catch(() => { });

        // Refresh every 15s (gentler on Celeron)
        const interval = setInterval(() => {
            api.getResources().then(d => setResources(d)).catch(() => { });
            api.getStatus().then(d => setServices(d.services || [])).catch(() => { });
        }, 15000);
        return () => clearInterval(interval);
    }, []);

    const runningCount = services.filter(s => s.running).length;
    const ramTotalGB = resources ? (resources.ram_total_mb / 1024).toFixed(1) : '?';
    const diskUsedGB = resources ? Math.round(resources.disk_total_gb - resources.disk_free_gb) : 0;
    const diskPercent = resources ? Math.round(diskUsedGB / resources.disk_total_gb * 100) : 0;

    return (
        <>
            <div className="page-header">
                <h2>Overview</h2>
                <p>Your sovereign server at a glance</p>
            </div>

            {loading ? (
                <div className="card"><div style={{ padding: 20, textAlign: 'center', color: 'var(--text-muted)' }}>Loading...</div></div>
            ) : (
                <>
                    <div className="grid-4" style={{ marginBottom: 24 }}>
                        <div className="card">
                            <div className="stat-value" style={{ color: 'var(--accent-green)' }}>{runningCount}/{services.length}</div>
                            <div className="stat-label">Services Running</div>
                        </div>
                        <div className="card">
                            <div className="stat-value">{resources?.cpu_cores || 0}</div>
                            <div className="stat-label">CPU Cores</div>
                        </div>
                        <div className="card">
                            <div className="stat-value" style={{ color: 'var(--accent-cyan)' }}>{ramTotalGB} GB</div>
                            <div className="stat-label">Total RAM</div>
                        </div>
                        <div className="card">
                            <div className="stat-value" style={{ color: 'var(--accent-primary)' }}>{resources?.disk_free_gb || 0} GB</div>
                            <div className="stat-label">Disk Free</div>
                        </div>
                    </div>

                    <div className="grid-2" style={{ marginBottom: 24 }}>
                        <div className="card">
                            <div className="card-title">System Resources</div>
                            <div style={{ marginBottom: 16 }}>
                                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13 }}>
                                    <span>CPU</span>
                                    <span className="mono" style={{ fontSize: 11 }}>{resources?.cpu_model || 'Unknown'}</span>
                                </div>
                            </div>
                            <div style={{ marginBottom: 16 }}>
                                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13 }}>
                                    <span>RAM</span>
                                    <span className="mono">{ramTotalGB} GB</span>
                                </div>
                                <div className="progress-bar">
                                    <div className="progress-fill ram" style={{ width: '10%' }} />
                                </div>
                            </div>
                            <div>
                                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13 }}>
                                    <span>Disk</span>
                                    <span className="mono">{diskUsedGB} / {resources?.disk_total_gb || 0} GB</span>
                                </div>
                                <div className="progress-bar">
                                    <div className="progress-fill disk" style={{ width: `${diskPercent}%` }} />
                                </div>
                            </div>
                        </div>

                        <div className="card">
                            <div className="card-title">Services</div>
                            {services.length === 0 ? (
                                <div style={{ color: 'var(--text-muted)', fontSize: 13 }}>Checking Docker services...</div>
                            ) : (
                                <table>
                                    <thead>
                                        <tr><th>Service</th><th>Status</th></tr>
                                    </thead>
                                    <tbody>
                                        {services.map(s => (
                                            <tr key={s.name}>
                                                <td style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                                                    <span className={`status-dot ${s.running ? 'up' : 'down'}`} />
                                                    {s.name.replace('sovereign-', '')}
                                                </td>
                                                <td><span className={`badge ${s.running ? 'badge-green' : 'badge-red'}`}>{s.running ? 'Running' : 'Stopped'}</span></td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            )}
                        </div>
                    </div>

                    <div className="card">
                        <div className="card-title">Quick Actions</div>
                        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                            <a href="/ai" className="btn btn-primary" style={{ textDecoration: 'none' }}>🧠 Chat with AI</a>
                            <a href="/apps" className="btn" style={{ textDecoration: 'none' }}>📦 Install App</a>
                            <a href="/settings" className="btn" style={{ textDecoration: 'none' }}>⚙️ Settings</a>
                        </div>
                    </div>
                </>
            )}
        </>
    );
}
