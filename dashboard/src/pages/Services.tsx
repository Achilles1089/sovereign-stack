import { useState, useEffect } from 'react';
import { api, type ServiceStatus } from '../api/client';

export default function Services() {
    const [services, setServices] = useState<ServiceStatus[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState('');

    const fetchServices = () => {
        api.getStatus()
            .then(data => {
                setServices(data.services || []);
                setLoading(false);
            })
            .catch(err => {
                setError(err.message);
                setLoading(false);
            });
    };

    useEffect(() => {
        fetchServices();
        const interval = setInterval(fetchServices, 10000);
        return () => clearInterval(interval);
    }, []);

    if (loading) return <div className="page-header"><h2>Loading services...</h2></div>;

    return (
        <>
            <div className="page-header">
                <h2>Services</h2>
                <p>Manage your sovereign infrastructure services</p>
            </div>

            {error && (
                <div className="card" style={{ marginBottom: 16, borderColor: 'rgba(239, 68, 68, 0.3)' }}>
                    <span style={{ color: 'var(--accent-red)' }}>⚠️ {error}</span>
                </div>
            )}

            <div className="card">
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
                    <div className="card-title" style={{ marginBottom: 0 }}>Docker Services</div>
                    <div style={{ display: 'flex', gap: 8 }}>
                        <button className="btn btn-sm" onClick={fetchServices}>🔄 Refresh</button>
                    </div>
                </div>
                {services.length === 0 ? (
                    <div style={{ color: 'var(--text-muted)', padding: 20, textAlign: 'center' }}>
                        No Docker services detected. Run <code>sovereign init</code> to set up.
                    </div>
                ) : (
                    <div className="table-container">
                        <table>
                            <thead>
                                <tr>
                                    <th>Service</th>
                                    <th>Status</th>
                                    <th>Image</th>
                                    <th>Ports</th>
                                </tr>
                            </thead>
                            <tbody>
                                {services.map(s => (
                                    <tr key={s.name}>
                                        <td style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                                            <span className={`status-dot ${s.running ? 'up' : 'down'}`} />
                                            <strong>{s.name}</strong>
                                        </td>
                                        <td>
                                            <span className={`badge ${s.running ? 'badge-green' : 'badge-red'}`}>
                                                {s.running ? 'Running' : 'Stopped'}
                                            </span>
                                        </td>
                                        <td><code>{s.image || '—'}</code></td>
                                        <td className="mono">{s.ports || '—'}</td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>
        </>
    );
}
