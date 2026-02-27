import { useState } from 'react';

const SERVICES = [
    { name: 'postgres', image: 'postgres:16-alpine', running: true, uptime: '2h 34m', ports: '5432', cpu: '2.1%', ram: '128 MB' },
    { name: 'caddy', image: 'caddy:2-alpine', running: true, uptime: '2h 34m', ports: '80, 443', cpu: '0.5%', ram: '24 MB' },
    { name: 'ollama', image: 'ollama/ollama', running: true, uptime: '2h 30m', ports: '11434', cpu: '8.3%', ram: '2.1 GB' },
];

export default function Services() {
    const [services] = useState(SERVICES);

    return (
        <>
            <div className="page-header">
                <h2>Services</h2>
                <p>Manage your sovereign infrastructure services</p>
            </div>

            <div className="card">
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
                    <div className="card-title" style={{ marginBottom: 0 }}>Core Services</div>
                    <div style={{ display: 'flex', gap: 8 }}>
                        <button className="btn btn-sm">🔄 Restart All</button>
                        <button className="btn btn-sm">📋 Compose</button>
                    </div>
                </div>
                <div className="table-container">
                    <table>
                        <thead>
                            <tr>
                                <th>Service</th>
                                <th>Status</th>
                                <th>Image</th>
                                <th>Uptime</th>
                                <th>Ports</th>
                                <th>CPU</th>
                                <th>RAM</th>
                                <th>Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            {services.map(s => (
                                <tr key={s.name}>
                                    <td style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                                        <span className={`status-dot ${s.running ? 'up' : 'down'}`} />
                                        <strong>{s.name}</strong>
                                    </td>
                                    <td><span className={`badge ${s.running ? 'badge-green' : 'badge-red'}`}>{s.running ? 'Running' : 'Stopped'}</span></td>
                                    <td><code>{s.image}</code></td>
                                    <td className="mono">{s.uptime}</td>
                                    <td className="mono">{s.ports}</td>
                                    <td className="mono">{s.cpu}</td>
                                    <td className="mono">{s.ram}</td>
                                    <td>
                                        <div style={{ display: 'flex', gap: 4 }}>
                                            <button className="btn btn-sm">🔄</button>
                                            <button className="btn btn-sm">📋</button>
                                            <button className="btn btn-sm btn-danger">⏹</button>
                                        </div>
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            </div>
        </>
    );
}
