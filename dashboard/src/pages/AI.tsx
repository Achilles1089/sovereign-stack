import { useState, useRef, useEffect, useCallback } from 'react';
import { api, type AIModel, type AIStatus, type SystemResources, type ServiceStatus, type CatalogEntry, type PhoneStatus } from '../api/client';

interface Message {
    role: 'user' | 'assistant';
    content: string;
}

export default function AI() {
    const [messages, setMessages] = useState<Message[]>([
        { role: 'assistant', content: 'Hello! I\'m your sovereign AI. Ask me anything \u2014 all data stays on your hardware.' },
    ]);
    const [input, setInput] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [models, setModels] = useState<AIModel[]>([]);
    const [aiStatus, setAiStatus] = useState<AIStatus | null>(null);
    const [resources, setResources] = useState<SystemResources | null>(null);
    const [services, setServices] = useState<ServiceStatus[]>([]);
    const [activeModel, setActiveModel] = useState('');
    const [pulling, setPulling] = useState<string | null>(null);
    const [pullProgress, setPullProgress] = useState('');
    const [statusMsg, setStatusMsg] = useState('');
    const [catalog, setCatalog] = useState<CatalogEntry[]>([]);
    const [phoneStatus, setPhoneStatus] = useState<PhoneStatus | null>(null);
    const messagesEndRef = useRef<HTMLDivElement>(null);
    const abortRef = useRef<AbortController | null>(null);

    // Speed optimization: buffer chunks with requestAnimationFrame
    const chunkBufferRef = useRef('');
    const rafRef = useRef<number | null>(null);

    const fetchModels = () => {
        api.getModels().then(d => setModels(d.models || [])).catch(() => { });
    };

    const fetchSystem = () => {
        api.getResources().then(d => setResources(d)).catch(() => { });
        api.getStatus().then(d => setServices(d.services || [])).catch(() => { });
    };

    const fetchPhoneStatus = () => {
        api.getPhoneStatus().then(d => setPhoneStatus(d)).catch(() => setPhoneStatus(null));
    };

    useEffect(() => {
        fetchModels();
        fetchSystem();
        fetchPhoneStatus();
        api.getAIStatus().then(d => {
            setAiStatus(d);
            setActiveModel(d.model || '');
        }).catch(() => { });
        api.getCatalog().then(d => setCatalog(d.catalog || [])).catch(() => { });

        const interval = setInterval(() => {
            fetchSystem();
            fetchPhoneStatus();
        }, 15000);
        return () => clearInterval(interval);
    }, []);

    // Only scroll on send/complete \u2014 NOT on every token
    const scrollToBottom = useCallback(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, []);

    useEffect(() => {
        scrollToBottom();
    }, [isLoading, scrollToBottom]);

    const handleSend = async () => {
        if (!input.trim() || isLoading) return;
        const userMsg: Message = { role: 'user', content: input };
        setMessages(prev => [...prev, userMsg]);
        setInput('');
        setIsLoading(true);
        setMessages(prev => [...prev, { role: 'assistant', content: '' }]);
        scrollToBottom();

        const controller = new AbortController();
        abortRef.current = controller;

        try {
            const chatMessages = messages.filter(m => m.content).map(m => ({ role: m.role, content: m.content }));
            chatMessages.push({ role: 'user' as const, content: input });
            // Limit context to last 10 messages \u2014 phone CPU prefill is slow on long history
            const contextWindow = chatMessages.slice(-10);
            await api.chat(activeModel, contextWindow, (chunk) => {
                // Buffer chunks and batch state updates via requestAnimationFrame
                chunkBufferRef.current += chunk;
                if (!rafRef.current) {
                    rafRef.current = requestAnimationFrame(() => {
                        const buffered = chunkBufferRef.current;
                        chunkBufferRef.current = '';
                        rafRef.current = null;
                        setMessages(prev => {
                            const updated = [...prev];
                            const last = updated[updated.length - 1];
                            if (last.role === 'assistant') {
                                updated[updated.length - 1] = { ...last, content: last.content + buffered };
                            }
                            return updated;
                        });
                    });
                }
            }, controller.signal);
        } catch (e) {
            if ((e as Error).name === 'AbortError') {
                // User stopped \u2014 keep whatever was generated so far
            } else {
                setMessages(prev => {
                    const updated = [...prev];
                    updated[updated.length - 1] = { role: 'assistant', content: '\u26a0\ufe0f Could not reach llama-server. Is it running?' };
                    return updated;
                });
            }
        }
        // Flush any remaining buffered content
        if (chunkBufferRef.current) {
            const remaining = chunkBufferRef.current;
            chunkBufferRef.current = '';
            setMessages(prev => {
                const updated = [...prev];
                const last = updated[updated.length - 1];
                if (last.role === 'assistant') {
                    updated[updated.length - 1] = { ...last, content: last.content + remaining };
                }
                return updated;
            });
        }
        if (rafRef.current) {
            cancelAnimationFrame(rafRef.current);
            rafRef.current = null;
        }
        abortRef.current = null;
        setIsLoading(false);
    };

    const handleStop = () => {
        abortRef.current?.abort();
    };

    const handlePull = async (modelName: string) => {
        setPulling(modelName);
        setPullProgress('Starting download...');
        try {
            await api.pullModel(modelName, (text) => {
                setPullProgress(text.trim().split('\n').pop() || '');
            });
            setPullProgress('');
            setStatusMsg(`\u2705 ${modelName} pulled!`);
            fetchModels();
            if (!activeModel) setActiveModel(modelName);
        } catch {
            setStatusMsg(`\u26a0\ufe0f Failed to pull ${modelName}`);
        }
        setPulling(null);
        setTimeout(() => setStatusMsg(''), 4000);
    };

    const handleDelete = async (modelName: string) => {
        if (!confirm(`Delete ${modelName}?`)) return;
        try {
            const res = await api.deleteModel(modelName);
            if (res.error) { setStatusMsg(`\u26a0\ufe0f ${res.error}`); }
            else {
                setStatusMsg(`\u2705 ${modelName} deleted`);
                fetchModels();
                if (activeModel === modelName) setActiveModel(models.find(m => m.name !== modelName)?.name || '');
            }
        } catch { setStatusMsg('\u26a0\ufe0f Delete failed'); }
        setTimeout(() => setStatusMsg(''), 4000);
    };

    const formatSize = (bytes: number) => {
        const gb = bytes / (1024 * 1024 * 1024);
        return gb >= 1 ? `${gb.toFixed(1)} GB` : `${(bytes / (1024 * 1024)).toFixed(0)} MB`;
    };

    const formatParams = (params: number) => {
        if (params >= 1e9) return `${(params / 1e9).toFixed(2)}B`;
        if (params >= 1e6) return `${(params / 1e6).toFixed(0)}M`;
        return `${params}`;
    };

    const formatContext = (ctx: number) => {
        if (ctx >= 1e6) return `${(ctx / 1e6).toFixed(0)}M`;
        if (ctx >= 1e3) return `${(ctx / 1e3).toFixed(0)}K`;
        return `${ctx}`;
    };

    const archBadge = (arch: string) => {
        const colors: Record<string, string> = {
            rwkv: '#10b981',    // green \u2014 flagship
            qwen2: '#3b82f6',   // blue
            llama: '#f59e0b',   // amber
            phi3: '#8b5cf6',    // purple
        };
        return colors[arch] || '#6b7280';
    };

    const runningCount = services.filter(s => s.running).length;
    const ramGB = resources ? (resources.ram_total_mb / 1024).toFixed(1) : '?';
    const diskUsedGB = resources ? Math.round(resources.disk_total_gb - resources.disk_free_gb) : 0;
    const diskPercent = resources ? Math.round(diskUsedGB / resources.disk_total_gb * 100) : 0;

    return (
        <div className="ai-layout">
            {/* LEFT PANEL \u2014 System + Services + AI Engine */}
            <aside className="ai-panel-left">
                <div className="card compact">
                    <div className="card-title">System</div>
                    <div className="mini-stats">
                        <div className="mini-stat">
                            <span className="mini-stat-value" style={{ color: 'var(--accent-green)' }}>{runningCount}/{services.length}</span>
                            <span className="mini-stat-label">Services</span>
                        </div>
                        <div className="mini-stat">
                            <span className="mini-stat-value">{resources?.cpu_cores || '?'}</span>
                            <span className="mini-stat-label">CPU Cores</span>
                        </div>
                        <div className="mini-stat">
                            <span className="mini-stat-value" style={{ color: 'var(--accent-cyan)' }}>{ramGB}G</span>
                            <span className="mini-stat-label">RAM</span>
                        </div>
                        <div className="mini-stat">
                            <span className="mini-stat-value" style={{ color: 'var(--accent-primary)' }}>{resources?.disk_free_gb || '?'}G</span>
                            <span className="mini-stat-label">Disk Free</span>
                        </div>
                    </div>

                    <div style={{ margin: '12px 0 8px' }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 11, color: 'var(--text-secondary)' }}>
                            <span>Disk</span>
                            <span className="mono">{diskUsedGB}/{resources?.disk_total_gb || 0}GB</span>
                        </div>
                        <div className="progress-bar" style={{ marginTop: 4 }}>
                            <div className="progress-fill disk" style={{ width: `${diskPercent}%` }} />
                        </div>
                    </div>
                </div>

                <div className="card compact">
                    <div className="card-title">Services</div>
                    {services.length === 0 ? (
                        <div style={{ color: 'var(--text-muted)', fontSize: 12 }}>Loading...</div>
                    ) : (
                        <div className="service-list">
                            {services.map(s => (
                                <div key={s.name} className="service-row">
                                    <span className={`status-dot ${s.running ? 'up' : 'down'}`} />
                                    <span className="service-name">{s.name.replace('sovereign-', '')}</span>
                                    <span className={`badge badge-sm ${s.running ? 'badge-green' : 'badge-red'}`}>
                                        {s.running ? 'Up' : 'Down'}
                                    </span>
                                </div>
                            ))}
                        </div>
                    )}
                </div>

                <div className="card compact">
                    <div className="card-title">Hardware</div>
                    <div style={{ fontSize: 12, lineHeight: 1.8 }}>
                        <div style={{ color: 'var(--text-muted)' }}>CPU</div>
                        <div className="mono" style={{ fontSize: 11 }}>{resources?.cpu_model || 'Unknown'}</div>
                        <div style={{ color: 'var(--text-muted)', marginTop: 6 }}>GPU</div>
                        <div style={{ fontSize: 12 }}>{resources?.gpu_name || 'CPU Only'}</div>
                        <div style={{ color: 'var(--text-muted)', marginTop: 6 }}>Tier</div>
                        <div style={{ color: 'var(--accent-green)', fontWeight: 600 }}>{aiStatus?.gpu_tier || 'cpu'}</div>
                    </div>
                </div>

                {/* AI Engine \u2014 Phone Status */}
                <div className="card compact">
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <div className="card-title" style={{ marginBottom: 0 }}>\ud83d\udcf1 AI Engine</div>
                        <span className={`badge badge-sm ${phoneStatus?.running ? 'badge-green' : 'badge-red'}`}>
                            {phoneStatus?.running ? '\ud83d\udfe2' : '\ud83d\udd34'}
                        </span>
                    </div>
                    {phoneStatus?.running ? (
                        <div style={{ fontSize: 12, lineHeight: 2, marginTop: 8 }}>
                            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                                <span style={{ color: 'var(--text-muted)' }}>Model</span>
                                <span className="mono" style={{ color: 'var(--accent-green)', fontWeight: 600, fontSize: 11 }}>
                                    {phoneStatus.display_name || phoneStatus.model}
                                </span>
                            </div>
                            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                                <span style={{ color: 'var(--text-muted)' }}>Params</span>
                                <span className="mono" style={{ fontSize: 11 }}>{formatParams(phoneStatus.params)}</span>
                            </div>
                            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                                <span style={{ color: 'var(--text-muted)' }}>Context</span>
                                <span className="mono" style={{ fontSize: 11 }}>{formatContext(phoneStatus.context)} tokens</span>
                            </div>
                            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                                <span style={{ color: 'var(--text-muted)' }}>Engine</span>
                                <span className="mono" style={{ fontSize: 11 }}>{phoneStatus.engine}</span>
                            </div>
                        </div>
                    ) : (
                        <div style={{ color: 'var(--text-muted)', fontSize: 11, marginTop: 8 }}>
                            llama-server not reachable
                        </div>
                    )}
                </div>
            </aside>

            {/* CENTER \u2014 AI Chat */}
            <div className="ai-panel-center">
                {statusMsg && (
                    <div style={{
                        padding: '6px 12px', marginBottom: 12, borderRadius: 8, fontSize: 12,
                        background: statusMsg.startsWith('\u2705') ? 'rgba(34,197,94,0.12)' : 'rgba(234,179,8,0.12)',
                        border: `1px solid ${statusMsg.startsWith('\u2705') ? 'rgba(34,197,94,0.25)' : 'rgba(234,179,8,0.25)'}`,
                    }}>
                        {statusMsg}
                    </div>
                )}

                <div className="chat-card">
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 12 }}>
                        <div className="card-title" style={{ marginBottom: 0 }}>AI Chat</div>
                        <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                            <span className={`badge badge-sm ${aiStatus?.running ? 'badge-green' : 'badge-red'}`}>
                                {aiStatus?.running ? '\ud83d\udfe2 Online' : '\ud83d\udd34 Offline'}
                            </span>
                            {models.length > 0 && (
                                <select
                                    value={activeModel}
                                    onChange={e => setActiveModel(e.target.value)}
                                    className="model-select"
                                >
                                    {models.map(m => (
                                        <option key={m.name} value={m.name}>{m.name}</option>
                                    ))}
                                </select>
                            )}
                        </div>
                    </div>

                    <div className="chat-messages">
                        {messages.map((msg, i) => (
                            <div key={i} className={`chat-message ${msg.role}`}>
                                <div className="chat-bubble">{msg.content || '...'}</div>
                            </div>
                        ))}
                        {isLoading && messages[messages.length - 1]?.content === '' && (
                            <div className="chat-message assistant">
                                <div className="chat-bubble" style={{ color: 'var(--text-muted)' }}>Thinking...</div>
                            </div>
                        )}
                        <div ref={messagesEndRef} />
                    </div>

                    <div className="chat-input-container">
                        <input
                            className="chat-input"
                            placeholder={activeModel ? `Chat with ${activeModel}...` : 'Waiting for llama-server...'}
                            value={input}
                            onChange={e => setInput(e.target.value)}
                            onKeyDown={e => e.key === 'Enter' && handleSend()}
                            disabled={isLoading || !aiStatus?.running}
                        />
                        <button className="btn btn-primary" onClick={handleSend} disabled={isLoading || !aiStatus?.running}>Send</button>
                        {isLoading && (
                            <button className="btn btn-danger" onClick={handleStop} style={{ minWidth: 60 }}>\u25a0 Stop</button>
                        )}
                    </div>
                </div>
            </div>

            {/* RIGHT PANEL \u2014 Models */}
            <aside className="ai-panel-right">
                <div className="card compact">
                    <div className="card-title">Installed Models</div>
                    {models.length === 0 ? (
                        <div style={{ color: 'var(--text-muted)', fontSize: 12 }}>No models. Pull one below.</div>
                    ) : (
                        <div className="model-list">
                            {models.map(m => (
                                <div key={m.name} className={`model-row ${activeModel === m.name ? 'active' : ''}`}>
                                    <div>
                                        <div className="mono" style={{ fontSize: 12, fontWeight: 600 }}>{m.name}</div>
                                        <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>{formatSize(m.size)}</div>
                                    </div>
                                    <div style={{ display: 'flex', gap: 4 }}>
                                        <button
                                            className={`btn btn-sm ${activeModel === m.name ? 'btn-primary' : ''}`}
                                            onClick={() => setActiveModel(m.name)}
                                            style={{ fontSize: 10, padding: '2px 8px' }}
                                        >
                                            {activeModel === m.name ? '\u25cf' : 'Use'}
                                        </button>
                                        <button
                                            className="btn btn-sm"
                                            onClick={() => handleDelete(m.name)}
                                            style={{ fontSize: 10, padding: '2px 6px', color: 'var(--accent-red)' }}
                                        >
                                            \u2715
                                        </button>
                                    </div>
                                </div>
                            ))}
                        </div>
                    )}
                </div>

                <div className="card compact">
                    <div className="card-title">GGUF Catalog</div>
                    <div className="model-list">
                        {catalog.map(m => (
                            <div key={m.name} className="model-row">
                                <div style={{ flex: 1 }}>
                                    <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                                        <span className="mono" style={{ fontSize: 11, fontWeight: 500 }}>{m.display_name}</span>
                                        <span style={{
                                            fontSize: 8, padding: '1px 5px', borderRadius: 4,
                                            background: archBadge(m.architecture) + '22',
                                            color: archBadge(m.architecture),
                                            fontWeight: 700, textTransform: 'uppercase',
                                        }}>
                                            {m.architecture}
                                        </span>
                                    </div>
                                    <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>{m.description} \u00b7 {m.size_gb} GB</div>
                                </div>
                                <div>
                                    {m.installed ? (
                                        <span style={{ fontSize: 10, color: 'var(--accent-green)' }}>\u2713</span>
                                    ) : (
                                        <button
                                            className="btn btn-sm btn-primary"
                                            onClick={() => handlePull(m.name)}
                                            disabled={!!pulling}
                                            style={{ fontSize: 10, padding: '2px 8px' }}
                                        >
                                            {pulling === m.name ? '\u23f3' : 'Pull'}
                                        </button>
                                    )}
                                </div>
                            </div>
                        ))}
                    </div>
                    {pulling && (
                        <div style={{
                            marginTop: 8, padding: '4px 8px', background: 'var(--bg-secondary)',
                            borderRadius: 6, fontSize: 10, fontFamily: 'var(--font-mono)', color: 'var(--text-secondary)',
                        }}>
                            \u23f3 {pullProgress || 'Connecting...'}
                        </div>
                    )}
                </div>
            </aside>
        </div>
    );
}
