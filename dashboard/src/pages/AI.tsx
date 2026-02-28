import { useState, useRef, useEffect } from 'react';
import { api, type AIModel, type AIStatus, type SystemResources } from '../api/client';

interface Message {
    role: 'user' | 'assistant';
    content: string;
}

export default function AI() {
    const [messages, setMessages] = useState<Message[]>([
        { role: 'assistant', content: 'Hello! I\'m your local AI running on Sovereign Stack. Ask me anything about your server.' },
    ]);
    const [input, setInput] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [models, setModels] = useState<AIModel[]>([]);
    const [aiStatus, setAiStatus] = useState<AIStatus | null>(null);
    const [resources, setResources] = useState<SystemResources | null>(null);
    const [activeModel, setActiveModel] = useState('');
    const messagesEndRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        api.getModels().then(d => setModels(d.models || [])).catch(() => { });
        api.getAIStatus().then(d => {
            setAiStatus(d);
            setActiveModel(d.model || '');
        }).catch(() => { });
        api.getResources().then(d => setResources(d)).catch(() => { });
    }, []);

    useEffect(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, [messages]);

    const handleSend = async () => {
        if (!input.trim() || isLoading) return;

        const userMsg: Message = { role: 'user', content: input };
        setMessages(prev => [...prev, userMsg]);
        setInput('');
        setIsLoading(true);

        // Add empty assistant message that we'll stream into
        setMessages(prev => [...prev, { role: 'assistant', content: '' }]);

        try {
            await api.serverChat(input, activeModel, (chunk) => {
                setMessages(prev => {
                    const updated = [...prev];
                    const last = updated[updated.length - 1];
                    if (last.role === 'assistant') {
                        updated[updated.length - 1] = { ...last, content: last.content + chunk };
                    }
                    return updated;
                });
            });
        } catch {
            setMessages(prev => {
                const updated = [...prev];
                updated[updated.length - 1] = {
                    role: 'assistant',
                    content: '⚠️ Could not reach Ollama. Make sure the container is running: `sudo docker compose -f /root/.sovereign/docker-compose.yml up -d`',
                };
                return updated;
            });
        }
        setIsLoading(false);
    };

    const formatSize = (bytes: number) => {
        const gb = bytes / (1024 * 1024 * 1024);
        return gb >= 1 ? `${gb.toFixed(1)} GB` : `${(bytes / (1024 * 1024)).toFixed(0)} MB`;
    };

    return (
        <>
            <div className="page-header">
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: 8 }}>
                    <div>
                        <h2>AI Inference</h2>
                        <p>Chat with your local AI — all data stays on your hardware</p>
                    </div>
                    <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
                        <span className={`badge ${aiStatus?.running ? 'badge-green' : 'badge-red'}`}>
                            {aiStatus?.running ? '🟢 Ollama Running' : '🔴 Ollama Offline'}
                        </span>
                        <span className="badge badge-blue">🧠 {activeModel || 'no model'}</span>
                    </div>
                </div>
            </div>

            <div className="grid-2" style={{ marginBottom: 24 }}>
                <div className="card">
                    <div className="card-title">Installed Models</div>
                    {models.length === 0 ? (
                        <div style={{ color: 'var(--text-muted)', fontSize: 13 }}>
                            No models installed. Pull one via SSH: <code>sudo docker exec sovereign-ollama ollama pull qwen2.5:0.5b</code>
                        </div>
                    ) : (
                        <table>
                            <thead>
                                <tr><th>Model</th><th>Size</th><th></th></tr>
                            </thead>
                            <tbody>
                                {models.map(m => (
                                    <tr key={m.name}>
                                        <td><strong className="mono">{m.name}</strong></td>
                                        <td className="mono">{formatSize(m.size)}</td>
                                        <td>
                                            <button
                                                className={`btn btn-sm ${activeModel === m.name ? 'btn-primary' : ''}`}
                                                onClick={() => setActiveModel(m.name)}
                                            >
                                                {activeModel === m.name ? 'Active' : 'Use'}
                                            </button>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    )}
                </div>

                <div className="card">
                    <div className="card-title">Hardware</div>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                        <div>
                            <span style={{ color: 'var(--text-secondary)', fontSize: 12 }}>CPU</span>
                            <div style={{ fontWeight: 600, fontSize: 14 }}>{resources?.cpu_model || 'Unknown'}</div>
                        </div>
                        <div>
                            <span style={{ color: 'var(--text-secondary)', fontSize: 12 }}>RAM</span>
                            <div style={{ fontWeight: 600 }}>{resources ? (resources.ram_total_mb / 1024).toFixed(1) : '?'} GB</div>
                        </div>
                        <div>
                            <span style={{ color: 'var(--text-secondary)', fontSize: 12 }}>GPU</span>
                            <div style={{ fontWeight: 600 }}>{resources?.gpu_name || 'CPU Only'}</div>
                        </div>
                        <div>
                            <span style={{ color: 'var(--text-secondary)', fontSize: 12 }}>GPU Tier</span>
                            <div style={{ fontWeight: 600, color: 'var(--accent-green)' }}>{aiStatus?.gpu_tier || 'cpu'}</div>
                        </div>
                    </div>
                </div>
            </div>

            <div className="card" style={{ height: 'calc(100vh - 440px)', minHeight: 300, display: 'flex', flexDirection: 'column' }}>
                <div className="card-title">AI Chat</div>
                <div className="chat-messages" style={{ flex: 1, overflowY: 'auto' }}>
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
                        placeholder="Ask your sovereign AI anything..."
                        value={input}
                        onChange={e => setInput(e.target.value)}
                        onKeyDown={e => e.key === 'Enter' && handleSend()}
                        disabled={isLoading}
                    />
                    <button className="btn btn-primary" onClick={handleSend} disabled={isLoading}>Send</button>
                </div>
            </div>
        </>
    );
}
