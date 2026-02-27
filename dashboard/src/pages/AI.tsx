import { useState, useRef, useEffect } from 'react';

interface Message {
    role: 'user' | 'assistant';
    content: string;
}

const MODELS = [
    { name: 'qwen2.5:32b', size: '20.0 GB', tier: 'ultra' },
    { name: 'qwen2.5:7b', size: '4.7 GB', tier: 'mid' },
    { name: 'deepseek-r1:14b', size: '9.0 GB', tier: 'high' },
];

export default function AI() {
    const [messages, setMessages] = useState<Message[]>([
        { role: 'assistant', content: 'Hello! I\'m your local AI running on Sovereign Stack. How can I help you today?' },
    ]);
    const [input, setInput] = useState('');
    const [isLoading, setIsLoading] = useState(false);
    const [model] = useState('qwen2.5:32b');
    const messagesEndRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
    }, [messages]);

    const handleSend = () => {
        if (!input.trim() || isLoading) return;

        const userMsg: Message = { role: 'user', content: input };
        setMessages(prev => [...prev, userMsg]);
        setInput('');
        setIsLoading(true);

        // Simulate AI response (will connect to Ollama API in production)
        setTimeout(() => {
            const responses = [
                'I\'m running locally on your hardware — no data leaves this machine. Your privacy is absolute.',
                'That\'s a great question! Let me think about it using my local inference engine...',
                'I can help with that. Since I\'m running on your Sovereign Stack, all processing stays on your hardware.',
            ];
            setMessages(prev => [...prev, {
                role: 'assistant',
                content: responses[Math.floor(Math.random() * responses.length)],
            }]);
            setIsLoading(false);
        }, 1000);
    };

    return (
        <>
            <div className="page-header">
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                    <div>
                        <h2>AI Inference</h2>
                        <p>Chat with your local AI — all data stays on your hardware</p>
                    </div>
                    <div style={{ display: 'flex', gap: 8 }}>
                        <span className="badge badge-green">🟢 Ollama Running</span>
                        <span className="badge badge-blue">🧠 {model}</span>
                    </div>
                </div>
            </div>

            <div className="grid-2" style={{ marginBottom: 24 }}>
                {/* Models */}
                <div className="card">
                    <div className="card-title">Installed Models</div>
                    <table>
                        <thead>
                            <tr><th>Model</th><th>Size</th><th>Tier</th><th></th></tr>
                        </thead>
                        <tbody>
                            {MODELS.map(m => (
                                <tr key={m.name}>
                                    <td><strong className="mono">{m.name}</strong></td>
                                    <td className="mono">{m.size}</td>
                                    <td><span className="badge badge-blue">{m.tier}</span></td>
                                    <td><button className="btn btn-sm">Use</button></td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                    <div style={{ marginTop: 12 }}>
                        <button className="btn btn-sm btn-primary">+ Pull New Model</button>
                    </div>
                </div>

                {/* GPU Info */}
                <div className="card">
                    <div className="card-title">Hardware</div>
                    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
                        <div>
                            <span style={{ color: 'var(--text-secondary)', fontSize: 12 }}>GPU</span>
                            <div style={{ fontWeight: 600 }}>Apple M1 Ultra</div>
                        </div>
                        <div>
                            <span style={{ color: 'var(--text-secondary)', fontSize: 12 }}>Unified Memory</span>
                            <div style={{ fontWeight: 600 }}>128 GB</div>
                        </div>
                        <div>
                            <span style={{ color: 'var(--text-secondary)', fontSize: 12 }}>Inference Mode</span>
                            <div style={{ fontWeight: 600 }}>Native (Metal GPU)</div>
                        </div>
                        <div>
                            <span style={{ color: 'var(--text-secondary)', fontSize: 12 }}>Max Model Size</span>
                            <div style={{ fontWeight: 600, color: 'var(--accent-green)' }}>~70B parameters</div>
                        </div>
                    </div>
                </div>
            </div>

            {/* Chat */}
            <div className="card" style={{ height: 'calc(100vh - 440px)', display: 'flex', flexDirection: 'column' }}>
                <div className="card-title">AI Chat</div>
                <div className="chat-messages" style={{ flex: 1, overflowY: 'auto' }}>
                    {messages.map((msg, i) => (
                        <div key={i} className={`chat-message ${msg.role}`}>
                            <div className="chat-bubble">{msg.content}</div>
                        </div>
                    ))}
                    {isLoading && (
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
