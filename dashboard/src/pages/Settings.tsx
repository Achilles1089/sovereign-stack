import { useState, useEffect } from 'react';
import { api, type AIStatus, type SystemResources } from '../api/client';

export default function Settings() {
    const [aiStatus, setAiStatus] = useState<AIStatus | null>(null);
    const [resources, setResources] = useState<SystemResources | null>(null);

    useEffect(() => {
        api.getAIStatus().then(d => setAiStatus(d)).catch(() => { });
        api.getResources().then(d => setResources(d)).catch(() => { });
    }, []);

    return (
        <>
            <div className="page-header">
                <h2>Settings</h2>
                <p>Configure your sovereign server</p>
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
                <div className="card">
                    <div className="card-title">Server</div>
                    <div style={{ display: 'grid', gap: 16 }}>
                        <div>
                            <label style={{ fontSize: 13, color: 'var(--text-secondary)', display: 'block', marginBottom: 6 }}>Platform</label>
                            <span className="mono">
                                {resources?.cpu_model || 'Unknown CPU'} — {resources?.cpu_cores || '?'} cores
                            </span>
                        </div>
                        <div>
                            <label style={{ fontSize: 13, color: 'var(--text-secondary)', display: 'block', marginBottom: 6 }}>GPU</label>
                            <span className="mono">{resources?.gpu_name || 'CPU Only'} ({resources?.gpu_type || 'none'})</span>
                        </div>
                        <div>
                            <label style={{ fontSize: 13, color: 'var(--text-secondary)', display: 'block', marginBottom: 6 }}>Storage</label>
                            <span className="mono">{resources?.disk_free_gb || '?'} GB free / {resources?.disk_total_gb || '?'} GB total</span>
                        </div>
                        <div>
                            <label style={{ fontSize: 13, color: 'var(--text-secondary)', display: 'block', marginBottom: 6 }}>SSL</label>
                            <span className="badge badge-green">Auto-SSL via Caddy</span>
                        </div>
                    </div>
                </div>

                <div className="card">
                    <div className="card-title">AI Inference</div>
                    <div style={{ display: 'grid', gap: 16 }}>
                        <div>
                            <label style={{ fontSize: 13, color: 'var(--text-secondary)', display: 'block', marginBottom: 6 }}>Default Model</label>
                            <span className="mono">{aiStatus?.model || 'none configured'}</span>
                        </div>
                        <div>
                            <label style={{ fontSize: 13, color: 'var(--text-secondary)', display: 'block', marginBottom: 6 }}>Recommended</label>
                            <span className="mono">{aiStatus?.recommended || '—'}</span>
                        </div>
                        <div>
                            <label style={{ fontSize: 13, color: 'var(--text-secondary)', display: 'block', marginBottom: 6 }}>Ollama Mode</label>
                            <span className="mono">{aiStatus?.mode || '—'}</span>
                        </div>
                        <div>
                            <label style={{ fontSize: 13, color: 'var(--text-secondary)', display: 'block', marginBottom: 6 }}>Ollama Host</label>
                            <span className="mono">{aiStatus?.host || '—'}</span>
                        </div>
                        <div>
                            <label style={{ fontSize: 13, color: 'var(--text-secondary)', display: 'block', marginBottom: 6 }}>GPU Tier</label>
                            <span className={`badge ${aiStatus?.gpu_tier === 'cpu' ? 'badge-amber' : 'badge-green'}`}>
                                {aiStatus?.gpu_tier || '—'}
                            </span>
                        </div>
                    </div>
                </div>

                <div className="card">
                    <div className="card-title">Configuration</div>
                    <div style={{ display: 'grid', gap: 16 }}>
                        <div>
                            <label style={{ fontSize: 13, color: 'var(--text-secondary)', display: 'block', marginBottom: 6 }}>Config Path</label>
                            <code>~/.sovereign/config.yaml</code>
                        </div>
                        <div>
                            <label style={{ fontSize: 13, color: 'var(--text-secondary)', display: 'block', marginBottom: 6 }}>Compose Path</label>
                            <code>~/.sovereign/docker-compose.yml</code>
                        </div>
                        <div>
                            <label style={{ fontSize: 13, color: 'var(--text-secondary)', display: 'block', marginBottom: 6 }}>Data Directory</label>
                            <code>~/.sovereign/data/</code>
                        </div>
                    </div>
                </div>
            </div>
        </>
    );
}
