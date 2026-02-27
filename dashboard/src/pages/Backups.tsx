import { useState } from 'react';

interface BackupSnapshot {
    id: string;
    date: string;
    size: string;
    status: 'success' | 'failed';
}

export default function Backups() {
    const [snapshots] = useState<BackupSnapshot[]>([
        { id: 'snap-001', date: '2026-02-27 03:00', size: '2.4 GB', status: 'success' },
        { id: 'snap-002', date: '2026-02-26 03:00', size: '2.3 GB', status: 'success' },
        { id: 'snap-003', date: '2026-02-25 03:00', size: '2.1 GB', status: 'success' },
    ]);

    return (
        <>
            <div className="page-header">
                <h2>Backups</h2>
                <p>Encrypted backups with Restic — your data is always safe</p>
            </div>

            <div className="grid-3" style={{ marginBottom: 24 }}>
                <div className="card">
                    <div className="stat-value" style={{ color: 'var(--accent-green)' }}>3</div>
                    <div className="stat-label">Total Snapshots</div>
                </div>
                <div className="card">
                    <div className="stat-value">6.8 GB</div>
                    <div className="stat-label">Total Backup Size</div>
                </div>
                <div className="card">
                    <div className="stat-value" style={{ color: 'var(--accent-cyan)' }}>Daily 3am</div>
                    <div className="stat-label">Schedule</div>
                </div>
            </div>

            <div className="card" style={{ marginBottom: 24 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
                    <div className="card-title" style={{ marginBottom: 0 }}>Backup History</div>
                    <button className="btn btn-primary">💾 Backup Now</button>
                </div>
                <table>
                    <thead>
                        <tr><th>Snapshot</th><th>Date</th><th>Size</th><th>Status</th><th>Actions</th></tr>
                    </thead>
                    <tbody>
                        {snapshots.map(snap => (
                            <tr key={snap.id}>
                                <td className="mono">{snap.id}</td>
                                <td className="mono">{snap.date}</td>
                                <td>{snap.size}</td>
                                <td><span className={`badge ${snap.status === 'success' ? 'badge-green' : 'badge-red'}`}>{snap.status}</span></td>
                                <td><button className="btn btn-sm">Restore</button></td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            </div>

            <div className="card">
                <div className="card-title">Backup Settings</div>
                <div style={{ display: 'grid', gap: 16 }}>
                    <div>
                        <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>Destination</span>
                        <div className="mono" style={{ marginTop: 4 }}>~/.sovereign/backups/</div>
                    </div>
                    <div>
                        <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>Encryption</span>
                        <div style={{ marginTop: 4 }}><span className="badge badge-green">AES-256 Encrypted</span></div>
                    </div>
                    <div>
                        <span style={{ fontSize: 12, color: 'var(--text-secondary)' }}>Schedule</span>
                        <div className="mono" style={{ marginTop: 4 }}>0 3 * * * (daily at 3:00 AM)</div>
                    </div>
                </div>
            </div>
        </>
    );
}
