import { useState, useRef, useEffect, useCallback } from 'react';
import { api, type AIModel, type AIStatus, type SystemResources, type ServiceStatus, type CatalogEntry, type PhoneStatus, type PhoneModel } from '../api/client';

interface Message {
    role: 'user' | 'assistant' | 'system';
    content: string;
}

// Terminal commands
const TERMINAL_COMMANDS = [
    { cmd: '/models', desc: 'List installed models' },
    { cmd: '/status', desc: 'Show system info' },
    { cmd: '/use', desc: 'Switch model (e.g. /use qwen3b)' },
    { cmd: '/clear', desc: 'Clear terminal' },
    { cmd: '/help', desc: 'Show available commands' },
];

const BOOT_LINES = [
    'SOVEREIGN-OS v1.0',
    'Initializing kernel...',
    'Mounting /models...',
    'Loading inference engine...',
    'Establishing secure channel...',
    'SYSTEM READY.',
    '',
];

export default function AI() {
    const [messages, setMessages] = useState<Message[]>([]);
    const [input, setInput] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [models, setModels] = useState<AIModel[]>([]);
    const [aiStatus, setAiStatus] = useState<AIStatus | null>(null);
    const [resources, setResources] = useState<SystemResources | null>(null);
    const [services, setServices] = useState<ServiceStatus[]>([]);
    const [activeModel, setActiveModel] = useState('');

    const [statusMsg, setStatusMsg] = useState('');
    const [catalog, setCatalog] = useState<CatalogEntry[]>([]);
    const [phoneStatus, setPhoneStatus] = useState<PhoneStatus | null>(null);
    const [phoneModels, setPhoneModels] = useState<PhoneModel[]>([]);
    const [switching, setSwitching] = useState(false);
    const [crtColor, setCrtColor] = useState<'green' | 'amber'>(() =>
        (localStorage.getItem('crt-color') as 'green' | 'amber') || 'green'
    );
    const [booted, setBooted] = useState(false);
    const messagesEndRef = useRef<HTMLDivElement>(null);
    const abortRef = useRef<AbortController | null>(null);
    const historyRef = useRef<string[]>([]);
    const historyIndexRef = useRef(-1);

    // Speed optimization: buffer chunks with requestAnimationFrame
    const chunkBufferRef = useRef('');
    const rafRef = useRef<number | null>(null);

    // CRT color CSS variable
    const crtMain = crtColor === 'amber' ? '#ffb000' : '#33ff33';
    const crtDim = crtColor === 'amber' ? '#4a3000' : '#1a5a1a';
    const crtGlow = crtColor === 'amber' ? 'rgba(255,176,0,0.3)' : 'rgba(51,255,51,0.3)';

    // Boot sequence on mount
    useEffect(() => {
        if (booted) return;
        let i = 0;
        const bootInterval = setInterval(() => {
            if (i < BOOT_LINES.length) {
                setMessages(prev => [...prev, { role: 'system', content: BOOT_LINES[i] }]);
                i++;
            } else {
                clearInterval(bootInterval);
                setBooted(true);
            }
        }, 300);
        return () => clearInterval(bootInterval);
    }, [booted]);

    const fetchModels = () => {
        api.getModels().then(d => setModels(d.models || [])).catch(() => { });
    };

    const fetchSystem = () => {
        api.getResources().then(d => setResources(d)).catch(() => { });
        api.getStatus().then(d => setServices(d.services || [])).catch(() => { });
    };

    const fetchPhoneStatus = () => {
        api.getPhoneStatus().then(d => setPhoneStatus(d)).catch(() => setPhoneStatus(null));
        api.getPhoneModels().then(d => setPhoneModels(d.models || [])).catch(() => { });
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

    // Only scroll on send/complete — NOT on every token
    const scrollToBottom = useCallback(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, []);

    useEffect(() => {
        scrollToBottom();
    }, [isLoading, scrollToBottom]);

    // Command history navigation
    const handleKeyDown = (e: React.KeyboardEvent) => {
        if (e.key === 'ArrowUp') {
            e.preventDefault();
            if (historyRef.current.length > 0) {
                const newIdx = Math.min(historyIndexRef.current + 1, historyRef.current.length - 1);
                historyIndexRef.current = newIdx;
                setInput(historyRef.current[historyRef.current.length - 1 - newIdx]);
            }
        } else if (e.key === 'ArrowDown') {
            e.preventDefault();
            const newIdx = historyIndexRef.current - 1;
            if (newIdx < 0) {
                historyIndexRef.current = -1;
                setInput('');
            } else {
                historyIndexRef.current = newIdx;
                setInput(historyRef.current[historyRef.current.length - 1 - newIdx]);
            }
        } else if (e.key === 'Escape') {
            setInput('');
            historyIndexRef.current = -1;
        } else if (e.key === 'Enter') {
            handleSend();
        }
    };

    // Inline terminal command handler
    const handleTerminalCommand = (cmd: string): boolean => {
        const parts = cmd.trim().split(/\s+/);
        const command = parts[0].toLowerCase();

        if (command === '/clear') {
            setMessages([{ role: 'system', content: 'Terminal cleared.' }]);
            return true;
        }
        if (command === '/help') {
            const helpText = TERMINAL_COMMANDS.map(c => `  ${c.cmd.padEnd(12)} ${c.desc}`).join('\n');
            setMessages(prev => [...prev, { role: 'system', content: `Available commands:\n${helpText}` }]);
            return true;
        }
        if (command === '/models') {
            if (phoneModels.length === 0) {
                setMessages(prev => [...prev, { role: 'system', content: 'No models installed.' }]);
            } else {
                const modelList = phoneModels.map(m => {
                    const active = phoneStatus?.model === m.name ? ' [ACTIVE]' : '';
                    const display = catalog.find(c => c.filename === m.name)?.display_name || m.name;
                    return `  ${display} (${(m.size_mb / 1024).toFixed(1)}G)${active}`;
                }).join('\n');
                setMessages(prev => [...prev, { role: 'system', content: `Installed models:\n${modelList}` }]);
            }
            return true;
        }
        if (command === '/status') {
            const lines = [
                `System Status:`,
                `  Model: ${phoneStatus?.display_name || phoneStatus?.model || 'None'}`,
                `  Engine: ${phoneStatus?.running ? 'ONLINE' : 'OFFLINE'}`,
                `  RAM: ${phoneStatus?.phone_ram_available_mb ? (phoneStatus.phone_ram_available_mb / 1024).toFixed(1) + 'G free' : 'N/A'}`,
                `  SoC: ${phoneStatus?.soc || 'Unknown'}`,
                `  Battery: ${phoneStatus?.battery_pct != null && phoneStatus.battery_pct >= 0 ? phoneStatus.battery_pct + '%' : 'N/A'}`,
            ];
            setMessages(prev => [...prev, { role: 'system', content: lines.join('\n') }]);
            return true;
        }
        if (command === '/use') {
            const query = parts.slice(1).join(' ').toLowerCase();
            if (!query) {
                setMessages(prev => [...prev, { role: 'system', content: 'Usage: /use <model name>\nExample: /use qwen3b' }]);
                return true;
            }
            // Fuzzy match
            const match = phoneModels.find(m => {
                const display = catalog.find(c => c.filename === m.name)?.display_name || m.name;
                return m.name.toLowerCase().includes(query) || display.toLowerCase().includes(query);
            });
            if (match) {
                const displayName = catalog.find(c => c.filename === match.name)?.display_name || match.name;
                setMessages(prev => [...prev, { role: 'system', content: `Switching to ${displayName}...` }]);
                // Trigger switch (same logic as Use button)
                setSwitching(true);
                api.switchPhoneModel(match.name).then(async () => {
                    let loaded = false;
                    for (let i = 0; i < 30; i++) {
                        await new Promise(r => setTimeout(r, 3000));
                        setMessages(prev => {
                            const updated = [...prev];
                            updated[updated.length - 1] = { role: 'system', content: `Switching to ${displayName}... ${(i + 1) * 3}s` };
                            return updated;
                        });
                        try {
                            const s = await api.getAIStatus();
                            if (s.running) { loaded = true; break; }
                        } catch { /* still loading */ }
                    }
                    fetchPhoneStatus();
                    setMessages(prev => [...prev, { role: 'system', content: loaded ? `${displayName} ready.` : `${displayName} still loading...` }]);
                    setSwitching(false);
                }).catch(() => {
                    setMessages(prev => [...prev, { role: 'system', content: 'Switch failed.' }]);
                    setSwitching(false);
                });
            } else {
                setMessages(prev => [...prev, { role: 'system', content: `Model not found: ${query}` }]);
            }
            return true;
        }
        return false;
    };

    const handleSend = async () => {
        if (!input.trim() || isLoading) return;

        // Store in history
        historyRef.current.push(input);
        historyIndexRef.current = -1;

        // Check for terminal commands
        if (input.startsWith('/')) {
            const userMsg: Message = { role: 'user', content: input };
            setMessages(prev => [...prev, userMsg]);
            handleTerminalCommand(input);
            setInput('');
            return;
        }

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
            // Limit context to last 10 messages — phone CPU prefill is slow on long history
            const contextWindow = chatMessages.slice(-10);
            await api.chat(phoneStatus?.model || '', contextWindow, (chunk) => {
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
                // User stopped — keep whatever was generated so far
            } else {
                setMessages(prev => {
                    const updated = [...prev];
                    updated[updated.length - 1] = { role: 'assistant', content: '[!] Could not reach llama-server. Is it running?' };
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


    const handleDelete = async (modelName: string) => {
        if (!confirm(`Delete ${modelName}?`)) return;
        try {
            const res = await api.deleteModel(modelName);
            if (res.error) { setStatusMsg(`[!] ${res.error}`); }
            else {
                setStatusMsg(`[OK] ${modelName} deleted`);
                fetchModels();
                if (activeModel === modelName) setActiveModel(models.find(m => m.name !== modelName)?.name || '');
            }
        } catch { setStatusMsg('[!] Delete failed'); }
        setTimeout(() => setStatusMsg(''), 4000);
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



    const runningCount = services.filter(s => s.running).length;
    const ramGB = resources ? (resources.ram_total_mb / 1024).toFixed(1) : '?';
    const diskUsedGB = resources ? Math.round(resources.disk_total_gb - resources.disk_free_gb) : 0;
    const diskPercent = resources ? Math.round(diskUsedGB / resources.disk_total_gb * 100) : 0;

    return (
        <div className="ai-layout">
            {/* LEFT PANEL — System + Services + AI Engine */}
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

                {/* Phone Hardware */}
                {phoneStatus?.phone_model && (
                    <div className="card compact">
                        <div className="card-title">Phone Hardware</div>
                        <div style={{ fontSize: 12, lineHeight: 1.8 }}>
                            <div style={{ color: 'var(--text-muted)' }}>Device</div>
                            <div className="mono" style={{ fontSize: 11, color: 'var(--accent-cyan)' }}>{phoneStatus.phone_model}</div>
                            {phoneStatus.soc && (
                                <>
                                    <div style={{ color: 'var(--text-muted)', marginTop: 6 }}>SoC</div>
                                    <div className="mono" style={{ fontSize: 11 }}>{phoneStatus.soc}</div>
                                </>
                            )}
                            {phoneStatus.phone_cpu_cores && phoneStatus.phone_cpu_cores > 0 && (
                                <>
                                    <div style={{ color: 'var(--text-muted)', marginTop: 6 }}>CPU</div>
                                    <div className="mono" style={{ fontSize: 11 }}>{phoneStatus.phone_cpu_cores} cores</div>
                                </>
                            )}
                            {phoneStatus.phone_ram_total_mb && phoneStatus.phone_ram_total_mb > 0 && (
                                <>
                                    <div style={{ color: 'var(--text-muted)', marginTop: 6 }}>RAM</div>
                                    <div className="mono" style={{ fontSize: 11 }}>
                                        {(phoneStatus.phone_ram_total_mb / 1024).toFixed(1)}G
                                        {phoneStatus.phone_ram_available_mb ? (
                                            <span style={{ color: 'var(--text-muted)' }}> / {(phoneStatus.phone_ram_available_mb / 1024).toFixed(1)}G free</span>
                                        ) : null}
                                    </div>
                                </>
                            )}
                            {phoneStatus.phone_storage_free_gb != null && phoneStatus.phone_storage_free_gb > 0 && (
                                <>
                                    <div style={{ color: 'var(--text-muted)', marginTop: 6 }}>Storage</div>
                                    <div className="mono" style={{ fontSize: 11 }}>{phoneStatus.phone_storage_free_gb}G free</div>
                                </>
                            )}
                            {phoneStatus.battery_pct != null && phoneStatus.battery_pct >= 0 && (
                                <>
                                    <div style={{ color: 'var(--text-muted)', marginTop: 6 }}>Battery</div>
                                    <div className="mono" style={{
                                        fontSize: 11,
                                        color: phoneStatus.battery_pct > 50 ? 'var(--accent-green)' :
                                            phoneStatus.battery_pct > 20 ? 'var(--accent-primary)' : 'var(--accent-red)'
                                    }}>
                                        {phoneStatus.battery_pct}%
                                    </div>
                                </>
                            )}
                            {phoneStatus.android_version && (
                                <>
                                    <div style={{ color: 'var(--text-muted)', marginTop: 6 }}>Android</div>
                                    <div className="mono" style={{ fontSize: 11 }}>{phoneStatus.android_version}</div>
                                </>
                            )}
                        </div>
                    </div>
                )}
            </aside>

            {/* CENTER — AI Chat */}
            <div className="ai-panel-center">
                {statusMsg && (
                    <div style={{
                        padding: '6px 12px', marginBottom: 12, borderRadius: 8, fontSize: 12,
                        background: statusMsg.startsWith('[OK]') ? 'rgba(34,197,94,0.12)' : 'rgba(234,179,8,0.12)',
                        border: `1px solid ${statusMsg.startsWith('[OK]') ? 'rgba(34,197,94,0.25)' : 'rgba(234,179,8,0.25)'}`,
                    }}>
                        {statusMsg}
                    </div>
                )}

                <div className="terminal-chat" style={{ '--crt-main': crtMain, '--crt-dim': crtDim, '--crt-glow': crtGlow } as React.CSSProperties}>
                    <div className="terminal-header">
                        <span className="terminal-header-title">SOVEREIGN-OS v1.0 ─── {phoneStatus?.running ? 'READY' : 'OFFLINE'}</span>
                        <div className="terminal-header-status">
                            {phoneStatus && (phoneStatus.phone_ram_available_mb ?? 0) > 0 && (
                                <span style={{ color: crtDim, fontSize: 10 }}>
                                    RAM:{((phoneStatus.phone_ram_available_mb ?? 0) / 1024).toFixed(1)}G
                                    {phoneStatus.soc ? ` │ ${phoneStatus.soc}` : ''}
                                    {phoneStatus.battery_pct != null && phoneStatus.battery_pct >= 0 ? ` │ ${phoneStatus.battery_pct}%` : ''}
                                </span>
                            )}
                            {phoneStatus?.running && (
                                <span style={{ color: crtMain }}>{phoneStatus.display_name || phoneStatus.model}</span>
                            )}
                            <span className={`terminal-dot ${phoneStatus?.running ? 'on' : 'off'}`} />
                            <button
                                onClick={() => {
                                    const next = crtColor === 'green' ? 'amber' : 'green';
                                    setCrtColor(next);
                                    localStorage.setItem('crt-color', next);
                                }}
                                title={`Switch to ${crtColor === 'green' ? 'amber' : 'green'} CRT`}
                                style={{
                                    background: 'transparent', border: 'none', cursor: 'pointer', fontSize: 12,
                                    color: crtColor === 'green' ? '#ffb000' : '#33ff33', padding: '0 4px',
                                }}
                            >
                                ◉
                            </button>
                        </div>
                    </div>

                    <div className="chat-messages">
                        {messages.map((msg, i) => {
                            if (msg.role === 'system') {
                                return (
                                    <div key={i} className="chat-message assistant">
                                        <div className="chat-bubble terminal-system-msg">{msg.content}</div>
                                    </div>
                                );
                            }
                            const isStreaming = isLoading && msg.role === 'assistant' && i === messages.length - 1;
                            return (
                                <div key={i} className={`chat-message ${msg.role}`}>
                                    <div className="chat-bubble">
                                        {msg.role === 'user' ? (
                                            <><span className="terminal-prompt">{'> '}</span>{msg.content}</>
                                        ) : (
                                            <>{msg.content || '...'}{isStreaming && <span className="terminal-cursor" />}</>
                                        )}
                                    </div>
                                </div>
                            );
                        })}
                        {isLoading && messages[messages.length - 1]?.content === '' && (
                            <div className="chat-message assistant">
                                <div className="chat-bubble" style={{ color: '#1a8a1a' }}>
                                    Processing<span className="terminal-cursor" />
                                </div>
                            </div>
                        )}
                        <div ref={messagesEndRef} />
                    </div>

                    <div className="terminal-input-container">
                        <span className="terminal-input-prefix">C:\&gt;&nbsp;</span>
                        <input
                            className="chat-input"
                            placeholder={phoneStatus?.running ? 'enter command...' : 'waiting for system...'}
                            value={input}
                            onChange={e => setInput(e.target.value)}
                            onKeyDown={handleKeyDown}
                            disabled={isLoading || !phoneStatus?.running}
                        />
                        <button className="btn btn-primary" onClick={handleSend} disabled={isLoading || !phoneStatus?.running}>SEND</button>
                        {isLoading && (
                            <button className="btn btn-danger" onClick={handleStop} style={{ minWidth: 60 }}>■ STOP</button>
                        )}
                    </div>
                </div>
            </div>

            {/* RIGHT PANEL — AI Engine + Models */}
            <aside className="ai-panel-right">
                {/* AI Engine — moved from left sidebar */}
                <div className="card compact">
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <div className="card-title" style={{ marginBottom: 0 }}>AI Engine</div>
                        <span className={`badge badge-sm ${phoneStatus?.running ? 'badge-green' : 'badge-red'}`}>
                            {phoneStatus?.running ? <span className="status-dot up" /> : <span className="status-dot down" />}
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
                        <div style={{ marginTop: 8 }}>
                            <div style={{ color: 'var(--text-muted)', fontSize: 11, marginBottom: 8 }}>
                                {switching ? 'Starting AI engine...' : 'llama-server not reachable'}
                            </div>
                            {!switching && (
                                <button
                                    className="btn btn-sm btn-primary"
                                    style={{ fontSize: 11, padding: '4px 12px', width: '100%' }}
                                    onClick={async () => {
                                        setSwitching(true);
                                        setStatusMsg('Starting AI engine via USB...');
                                        try {
                                            await api.startPhone();
                                            // Wait for llama-server to boot
                                            await new Promise(r => setTimeout(r, 8000));
                                            fetchPhoneStatus();
                                            setStatusMsg('AI engine started!');
                                        } catch {
                                            setStatusMsg('Failed to start');
                                        } finally {
                                            setSwitching(false);
                                            setTimeout(() => setStatusMsg(''), 3000);
                                        }
                                    }}
                                >
                                    ▶ Start AI Engine
                                </button>
                            )}
                        </div>
                    )}
                </div>

                <div className="card compact">
                    <div className="card-title">Installed Models</div>
                    {phoneModels.length === 0 ? (
                        <div style={{ color: 'var(--text-muted)', fontSize: 12 }}>No models on phone. Pull one below.</div>
                    ) : (
                        <div className="model-list">
                            {phoneModels.map(m => {
                                const isActive = phoneStatus?.model === m.name;
                                const displayName = catalog.find(c => c.filename === m.name)?.display_name || m.name.replace('.gguf', '');
                                return (
                                    <div key={m.name} className={`model-row ${isActive ? 'active' : ''}`}>
                                        <div>
                                            <div className="mono" style={{ fontSize: 12, fontWeight: 600 }}>{displayName}</div>
                                            <div style={{ fontSize: 10, color: 'var(--text-muted)' }}>{(m.size_mb / 1024).toFixed(1)} GB</div>
                                        </div>
                                        <div style={{ display: 'flex', gap: 4 }}>
                                            <button
                                                className={`btn btn-sm ${isActive ? 'btn-primary' : ''}`}
                                                disabled={switching || isActive}
                                                onClick={async () => {
                                                    setSwitching(true);
                                                    const estTime = m.size_mb > 3000 ? '30-60s' : m.size_mb > 1500 ? '15-30s' : '5-15s';
                                                    setStatusMsg(`Loading ${displayName}... (est. ${estTime})`);
                                                    try {
                                                        await api.switchPhoneModel(m.name);
                                                        // Poll health every 3s for up to 90s
                                                        let loaded = false;
                                                        for (let i = 0; i < 30; i++) {
                                                            await new Promise(r => setTimeout(r, 3000));
                                                            setStatusMsg(`Loading ${displayName}... ${(i + 1) * 3}s`);
                                                            try {
                                                                const s = await api.getAIStatus();
                                                                if (s.running) {
                                                                    loaded = true;
                                                                    setAiStatus(s);
                                                                    setActiveModel(s.model || '');
                                                                    break;
                                                                }
                                                            } catch { /* still loading */ }
                                                        }
                                                        fetchPhoneStatus();
                                                        setStatusMsg(loaded ? `${displayName} ready!` : `${displayName} still loading — check dashboard`);
                                                    } catch {
                                                        setStatusMsg('Switch failed');
                                                    } finally {
                                                        setSwitching(false);
                                                        setTimeout(() => setStatusMsg(''), 5000);
                                                    }
                                                }}
                                                style={{ fontSize: 10, padding: '2px 8px' }}
                                            >
                                                {isActive ? '● Active' : switching ? '⏳' : 'Use'}
                                            </button>
                                            <button
                                                className="btn btn-sm"
                                                onClick={() => handleDelete(m.name)}
                                                style={{ fontSize: 10, padding: '2px 6px', color: 'var(--accent-red)' }}
                                            >
                                                ✕
                                            </button>
                                        </div>
                                    </div>
                                );
                            })}
                        </div>
                    )}
                </div>

                <div className="card compact" style={{ background: '#0a0a0a', border: `1px solid ${crtMain}33` }}>
                    <div className="card-title" style={{ color: crtMain, fontFamily: '"IBM Plex Mono", monospace', fontSize: 11, letterSpacing: 1 }}>COMMANDS</div>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
                        {TERMINAL_COMMANDS.map(c => (
                            <button
                                key={c.cmd}
                                onClick={() => {
                                    setInput(c.cmd + ' ');
                                    if (c.cmd !== '/use') {
                                        // Auto-execute commands that don't need args
                                        const userMsg: Message = { role: 'user', content: c.cmd };
                                        setMessages(prev => [...prev, userMsg]);
                                        handleTerminalCommand(c.cmd);
                                        setInput('');
                                    }
                                }}
                                style={{
                                    background: 'transparent',
                                    border: `1px solid ${crtMain}33`,
                                    color: crtMain,
                                    fontFamily: '"IBM Plex Mono", monospace',
                                    fontSize: 11,
                                    padding: '6px 10px',
                                    borderRadius: 2,
                                    cursor: 'pointer',
                                    textAlign: 'left',
                                    transition: 'all 0.15s',
                                }}
                                onMouseEnter={e => {
                                    e.currentTarget.style.background = `${crtMain}11`;
                                    e.currentTarget.style.borderColor = `${crtMain}88`;
                                }}
                                onMouseLeave={e => {
                                    e.currentTarget.style.background = 'transparent';
                                    e.currentTarget.style.borderColor = `${crtMain}33`;
                                }}
                            >
                                <span style={{ color: crtMain }}>{c.cmd}</span>
                                <span style={{ color: crtDim, marginLeft: 8, fontSize: 10 }}>{c.desc}</span>
                            </button>
                        ))}
                    </div>
                </div>
            </aside>
        </div>
    );
}
