import { useState } from 'react';

export default function Settings() {
    const [domain, setDomain] = useState('localhost');
    const [aiModel, setAiModel] = useState('qwen2.5:32b');

    return (
        <>
            <div className="page-header">
                <h2>Settings</h2>
                <p>Configure your sovereign server</p>
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 20 }}>
                {/* Server */}
                <div className="card">
                    <div className="card-title">Server</div>
                    <div style={{ display: 'grid', gap: 16 }}>
                        <div>
                            <label style={{ fontSize: 13, color: 'var(--text-secondary)', display: 'block', marginBottom: 6 }}>Domain</label>
                            <input
                                className="chat-input"
                                value={domain}
                                onChange={e => setDomain(e.target.value)}
                                placeholder="myserver.example.com"
                                style={{ maxWidth: 400 }}
                            />
                        </div>
                        <div>
                            <label style={{ fontSize: 13, color: 'var(--text-secondary)', display: 'block', marginBottom: 6 }}>SSL</label>
                            <span className="badge badge-green">Auto-SSL via Caddy</span>
                        </div>
                        <div>
                            <label style={{ fontSize: 13, color: 'var(--text-secondary)', display: 'block', marginBottom: 6 }}>Platform</label>
                            <span className="mono">macOS (arm64) — Personal Mode</span>
                        </div>
                    </div>
                </div>

                {/* AI */}
                <div className="card">
                    <div className="card-title">AI Inference</div>
                    <div style={{ display: 'grid', gap: 16 }}>
                        <div>
                            <label style={{ fontSize: 13, color: 'var(--text-secondary)', display: 'block', marginBottom: 6 }}>Default Model</label>
                            <select
                                className="chat-input"
                                value={aiModel}
                                onChange={e => setAiModel(e.target.value)}
                                style={{ maxWidth: 400 }}
                            >
                                <option value="qwen2.5:32b">qwen2.5:32b (Flagship)</option>
                                <option value="qwen2.5:14b">qwen2.5:14b (Strong)</option>
                                <option value="qwen2.5:7b">qwen2.5:7b (Medium)</option>
                                <option value="qwen2.5:0.5b">qwen2.5:0.5b (Lightweight)</option>
                            </select>
                        </div>
                        <div>
                            <label style={{ fontSize: 13, color: 'var(--text-secondary)', display: 'block', marginBottom: 6 }}>Ollama Mode</label>
                            <span className="mono">native (Metal GPU acceleration)</span>
                        </div>
                        <div>
                            <label style={{ fontSize: 13, color: 'var(--text-secondary)', display: 'block', marginBottom: 6 }}>Ollama Host</label>
                            <span className="mono">localhost:11434</span>
                        </div>
                    </div>
                </div>

                {/* Config File */}
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

                {/* Danger Zone */}
                <div className="card" style={{ borderColor: 'rgba(239, 68, 68, 0.2)' }}>
                    <div className="card-title" style={{ color: 'var(--accent-red)' }}>Danger Zone</div>
                    <div style={{ display: 'flex', gap: 12 }}>
                        <button className="btn btn-danger">🔄 Reset Configuration</button>
                        <button className="btn btn-danger">🗑️ Remove All Data</button>
                    </div>
                </div>
            </div>
        </>
    );
}
