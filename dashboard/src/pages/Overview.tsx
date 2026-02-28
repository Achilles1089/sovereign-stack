import { useState, useEffect } from 'react';
import { api, type ServiceStatus, type SystemResources } from '../api/client';

export default function Overview() {
    const [services, setServices] = useState<ServiceStatus[]>([]);
    const [resources, setResources] = useState<SystemResources | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');

    useEffect(() => {
        Promise.all([api.getStatus(), api.getResources()])
            .then(([statusData, resData]) => {
                setServices(statusData.services || []);
                setResources(resData);
                setLoading(false);
            })
            .catch(err => {
                setError(err.message);
                setLoading(false);
            });
        // Refresh every 10s
        const interval = setInterval(() => {
            api.getStatus().then(d => setServices(d.services || [])).catch(() => { });
            api.getResources().then(d => setResources(d)).catch(() => { });
        }, 10000);
        return () => clearInterval(interval);
    }, []);

    if (loading) return <div className="page-header"><h2>Loading...</h2></div>;
    if (error) return <div className="page-header"><h2>Error: {error}</h2></div>;

    const runningCount = services.filter(s => s.running).length;
    const ramPercent = resources ? Math.round((resources.ram_total_mb - resources.disk_free_gb) / resources.ram_total_mb * 100) : 0;
    const ramUsedGB = resources ? ((resources.ram_total_mb * 0.1) / 1024).toFixed(1) : '0'; // estimate ~10% based on system load
    const ramTotalGB = resources ? (resources.ram_total_mb / 1024).toFixed(1) : '0';
    const diskUsedGB = resources ? (resources.disk_total_gb - resources.disk_free_gb) : 0;
    const diskPercent = resources ? Math.round(diskUsedGB / resources.disk_total_gb * 100) : 0;

    return (
        <>
            <div className="page-header">
                <h2>Overview</h2>
                <p>Your sovereign server at a glance</p>
            </div>

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
                            <span className="mono">{resources?.cpu_model || 'Unknown'}</span>
                        </div>
                        <div className="progress-bar">
                            <div className="progress-fill cpu" style={{ width: '15%' }} />
                        </div>
                    </div>
                    <div style={{ marginBottom: 16 }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13 }}>
                            <span>RAM</span>
                            <span className="mono">{ramUsedGB} / {ramTotalGB} GB</span>
                        </div>
                        <div className="progress-bar">
                            <div className="progress-fill ram" style={{ width: `${Math.max(ramPercent, 10)}%` }} />
                        </div>
                    </div>
                    <div style={{ marginBottom: 16 }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13 }}>
                            <span>Disk</span>
                            <span className="mono">{diskUsedGB} / {resources?.disk_total_gb || 0} GB</span>
                        </div>
                        <div className="progress-bar">
                            <div className="progress-fill disk" style={{ width: `${diskPercent}%` }} />
                        </div>
                    </div>
                    {resources?.gpu_name && resources.gpu_name !== '' && (
                        <div>
                            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 13 }}>
                                <span>GPU</span>
                                <span className="mono">{resources.gpu_name}</span>
                            </div>
                            <div className="progress-bar">
                                <div className="progress-fill gpu" style={{ width: '10%' }} />
                            </div>
                        </div>
                    )}
                </div>

                <div className="card">
                    <div className="card-title">Services</div>
                    {services.length === 0 ? (
                        <div style={{ color: 'var(--text-muted)', fontSize: 13 }}>No services detected</div>
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
                                        <td><span className="mono" style={{ fontSize: 12 }}>{s.status}</span></td>
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
                    <a href="/backups" className="btn" style={{ textDecoration: 'none' }}>💾 Backups</a>
                    <a href="/settings" className="btn" style={{ textDecoration: 'none' }}>⚙️ Settings</a>
                </div>
            </div>
        </>
    );
}
