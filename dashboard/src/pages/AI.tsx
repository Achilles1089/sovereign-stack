import { useState, useRef, useEffect, useCallback } from 'react';
import { api, type AIModel, type AIStatus, type SystemResources, type ServiceStatus, type CatalogEntry, type PhoneStatus, type PhoneModel, type BrainNetLive, type EnvySysinfo } from '../api/client';

interface Message {
    role: 'user' | 'assistant' | 'system';
    content: string;
}

// Terminal commands
const TERMINAL_COMMANDS = [
    { cmd: '/models', desc: 'List installed models' },
    { cmd: '/status', desc: 'Show system info' },
    { cmd: '/use', desc: 'Switch model (e.g. /use qwen3b)' },
    { cmd: '/imagine', desc: 'Generate image (e.g. /imagine sunset)' },
    { cmd: '/voice', desc: 'Toggle voice mode (auto-speak responses)' },
    { cmd: '/music', desc: 'Generate music (e.g. /music ambient piano)' },
    { cmd: '/doc', desc: 'Document chat (upload/list/search/ask/delete)' },
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
    const [_aiStatus, setAiStatus] = useState<AIStatus | null>(null);
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
    const [isHackerMode, setIsHackerMode] = useState(() => localStorage.getItem('hacker-mode') === 'true');

    // Save hacker mode preference
    useEffect(() => {
        localStorage.setItem('hacker-mode', String(isHackerMode));
    }, [isHackerMode]);

    const messagesEndRef = useRef<HTMLDivElement>(null);
    const abortRef = useRef<AbortController | null>(null);
    const historyRef = useRef<string[]>([]);
    const historyIndexRef = useRef(-1);

    // Speed optimization: buffer chunks with requestAnimationFrame
    const chunkBufferRef = useRef('');
    const rafRef = useRef<number | null>(null);

    // Context window slider — persisted to localStorage
    const [contextSize, setContextSize] = useState(() =>
        parseInt(localStorage.getItem('context-size') || '10', 10)
    );
    const [imageResolution, setImageResolution] = useState(() =>
        (localStorage.getItem('image-resolution') || '256') as '256' | '512'
    );
    // Image generation in-flight flag (used to disable input during gen)
    const [isGeneratingImage, setIsGeneratingImage] = useState(false);
    // Image gen node status
    const [imageNodeOnline, setImageNodeOnline] = useState(false);
    const [brainLive, setBrainLive] = useState<BrainNetLive | null>(null);
    const [envySysinfo, setEnvySysinfo] = useState<EnvySysinfo | null>(null);
    // Voice pipeline state
    const [isRecording, setIsRecording] = useState(false);
    const [voiceEnabled, setVoiceEnabled] = useState(() => localStorage.getItem('voice-mode') === 'true');
    const mediaRecorderRef = useRef<MediaRecorder | null>(null);
    const audioChunksRef = useRef<Blob[]>([]);
    const audioRef = useRef<HTMLAudioElement | null>(null);
    const fileInputRef = useRef<HTMLInputElement | null>(null);
    const [musicNodeOnline, setMusicNodeOnline] = useState(false);

    // Gallery state
    const [showGallery, setShowGallery] = useState(false);
    const [galleryImages, setGalleryImages] = useState<Array<{ id: string; prompt: string; width: number; height: number; created_at: string; size_bytes: number }>>([]);
    const [selectedImage, setSelectedImage] = useState<string | null>(null);
    const [galleryLoading, setGalleryLoading] = useState(false);

    // CRT color CSS variable
    const crtMain = crtColor === 'amber' ? '#cc8800' : '#22bb22';
    const crtDim = crtColor === 'amber' ? '#4a3000' : '#1a4a1a';
    const crtGlow = crtColor === 'amber' ? 'rgba(204,136,0,0.15)' : 'rgba(34,187,34,0.15)';

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

        // Initial live stats
        api.getBrainNetLive().then(d => setBrainLive(d)).catch(() => { });
        api.getEnvySysinfo().then(d => setEnvySysinfo(d)).catch(() => { });

        const interval = setInterval(() => {
            fetchSystem();
            fetchPhoneStatus();
            api.getImageStatus().then(d => setImageNodeOnline(d.online)).catch(() => setImageNodeOnline(false));
            api.getMusicStatus().then(d => setMusicNodeOnline(d.online)).catch(() => setMusicNodeOnline(false));
            api.getBrainNetLive().then(d => setBrainLive(d)).catch(() => { });
            api.getEnvySysinfo().then(d => setEnvySysinfo(d)).catch(() => { });
        }, 15000);
        // Initial image node check
        api.getImageStatus().then(d => setImageNodeOnline(d.online)).catch(() => setImageNodeOnline(false));
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
        if (command === '/imagine') {
            const prompt = parts.slice(1).join(' ');
            if (!prompt) {
                setMessages(prev => [...prev, { role: 'system', content: 'Usage: /imagine <prompt>\nExample: /imagine a sunset over mountains' }]);
                return true;
            }
            if (!imageNodeOnline) {
                setMessages(prev => [...prev, { role: 'system', content: '[!] Image gen node offline. Start sd_server on the mini PC.' }]);
                return true;
            }
            // Show progress message
            const imgSize = parseInt(imageResolution);
            const estTime = imgSize === 256 ? '~20s' : '~75s';
            setMessages(prev => [...prev, { role: 'system', content: `🎨 Generating: ${prompt} (${imgSize}x${imgSize}, ${estTime})...\n<IMAGEPROGRESS>` }]);
            setIsGeneratingImage(true);
            const startTime = Date.now();
            api.generateImage(prompt, imgSize, imgSize).then(result => {
                const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
                setMessages(prev => {
                    const updated = [...prev];
                    // Find the progress message (loop backwards for compat)
                    let progressIdx = -1;
                    for (let j = updated.length - 1; j >= 0; j--) {
                        if (updated[j].content.includes('<IMAGEPROGRESS>')) { progressIdx = j; break; }
                    }
                    if (progressIdx >= 0) {
                        updated[progressIdx] = { role: 'system', content: `<IMAGE>${result.image}</IMAGE>\n${prompt} — ${elapsed}s` };
                    }
                    return updated;
                });
            }).catch(() => {
                setMessages(prev => {
                    const updated = [...prev];
                    let progressIdx = -1;
                    for (let j = updated.length - 1; j >= 0; j--) {
                        if (updated[j].content.includes('<IMAGEPROGRESS>')) { progressIdx = j; break; }
                    }
                    if (progressIdx >= 0) {
                        updated[progressIdx] = { role: 'system', content: '[!] Image generation failed. Check mini PC connection.' };
                    }
                    return updated;
                });
            }).finally(() => {
                setIsGeneratingImage(false);
            });
            return true;
        }
        if (command === '/voice') {
            const next = !voiceEnabled;
            setVoiceEnabled(next);
            localStorage.setItem('voice-mode', String(next));
            setMessages(prev => [...prev, { role: 'system', content: `Voice mode ${next ? 'ON — responses will be spoken' : 'OFF'}` }]);
            return true;
        }
        if (command === '/music') {
            const prompt = parts.slice(1).join(' ');
            if (!prompt) {
                setMessages(prev => [...prev, { role: 'system', content: 'Usage: /music <description>\nExample: /music ambient piano with rain sounds' }]);
                return true;
            }
            if (!musicNodeOnline) {
                setMessages(prev => [...prev, { role: 'system', content: '[!] Music gen node offline. Start music_server on Envy.' }]);
                return true;
            }
            setMessages(prev => [...prev, { role: 'system', content: `🎵 Generating: ${prompt}...\n<MUSICPROGRESS>` }]);
            setIsGeneratingImage(true);
            const startTime = Date.now();
            api.generateMusic(prompt).then(result => {
                const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
                setMessages(prev => {
                    const updated = [...prev];
                    let idx = -1;
                    for (let j = updated.length - 1; j >= 0; j--) {
                        if (updated[j].content.includes('<MUSICPROGRESS>')) { idx = j; break; }
                    }
                    if (idx >= 0) {
                        updated[idx] = { role: 'system', content: `<MUSIC>${result.audio}</MUSIC>\n🎵 ${prompt} — ${elapsed}s (${result.duration_s}s audio)` };
                    }
                    return updated;
                });
            }).catch(() => {
                setMessages(prev => {
                    const updated = [...prev];
                    let idx = -1;
                    for (let j = updated.length - 1; j >= 0; j--) {
                        if (updated[j].content.includes('<MUSICPROGRESS>')) { idx = j; break; }
                    }
                    if (idx >= 0) {
                        updated[idx] = { role: 'system', content: '[!] Music generation failed.' };
                    }
                    return updated;
                });
            }).finally(() => setIsGeneratingImage(false));
            return true;
        }
        if (command === '/doc') {
            const sub = (parts[1] || '').toLowerCase();
            if (sub === 'upload') {
                fileInputRef.current?.click();
                return true;
            }
            if (sub === 'list') {
                setMessages(prev => [...prev, { role: 'system', content: '📄 Loading documents...' }]);
                api.listDocuments().then(res => {
                    setMessages(prev => prev.filter(m => m.content !== '📄 Loading documents...'));
                    if (res.documents.length === 0) {
                        setMessages(prev => [...prev, { role: 'system', content: 'No documents uploaded. Use /doc upload to add files.' }]);
                    } else {
                        const list = res.documents.map(d => `  ${d.name} — ${d.num_chunks} chunks (${d.added_at})`).join('\n');
                        setMessages(prev => [...prev, { role: 'system', content: `Documents:\n${list}` }]);
                    }
                }).catch(() => {
                    setMessages(prev => prev.filter(m => m.content !== '📄 Loading documents...'));
                    setMessages(prev => [...prev, { role: 'system', content: '[!] Failed to list documents. RAG server may be offline.' }]);
                });
                return true;
            }
            if (sub === 'search') {
                const query = parts.slice(2).join(' ');
                if (!query) {
                    setMessages(prev => [...prev, { role: 'system', content: 'Usage: /doc search <query>' }]);
                    return true;
                }
                setMessages(prev => [...prev, { role: 'system', content: `🔍 Searching: ${query}...` }]);
                api.searchDocuments(query).then(res => {
                    setMessages(prev => prev.filter(m => m.content.startsWith('🔍 Searching:')));
                    if (res.results.length === 0) {
                        setMessages(prev => [...prev, { role: 'system', content: 'No results found.' }]);
                    } else {
                        const results = res.results.map((r, i) => `${i + 1}. [${r.document}] (${(r.score * 100).toFixed(0)}%)\n   ${r.chunk.slice(0, 200)}...`).join('\n\n');
                        setMessages(prev => [...prev, { role: 'system', content: `Results:\n${results}` }]);
                    }
                }).catch(() => {
                    setMessages(prev => [...prev, { role: 'system', content: '[!] Search failed.' }]);
                });
                return true;
            }
            if (sub === 'delete') {
                const name = parts.slice(2).join(' ');
                if (!name) {
                    setMessages(prev => [...prev, { role: 'system', content: 'Usage: /doc delete <filename>' }]);
                    return true;
                }
                api.deleteDocument(name).then(() => {
                    setMessages(prev => [...prev, { role: 'system', content: `Deleted: ${name}` }]);
                }).catch(() => {
                    setMessages(prev => [...prev, { role: 'system', content: `[!] Failed to delete: ${name}` }]);
                });
                return true;
            }
            if (sub === 'ask') {
                const query = parts.slice(2).join(' ');
                if (!query) {
                    setMessages(prev => [...prev, { role: 'system', content: 'Usage: /doc ask <question>\nSearches your documents and asks the LLM with context.' }]);
                    return true;
                }
                // RAG-augmented query: search docs → inject context → send to LLM
                setMessages(prev => [...prev, { role: 'user', content: query }, { role: 'system', content: '📄 Searching documents...' }]);
                setIsLoading(true);
                api.searchDocuments(query, 3).then(async (res) => {
                    setMessages(prev => prev.filter(m => m.content !== '📄 Searching documents...'));
                    const context = res.results.map(r => r.chunk).join('\n\n---\n\n');
                    const ragPrompt = context
                        ? `Based on the following document excerpts, answer the question.\n\nDocument excerpts:\n${context}\n\nQuestion: ${query}`
                        : query;
                    // Send to LLM via streaming chat
                    setInput('');
                    const newMsg: Message = { role: 'assistant', content: '' };
                    setMessages(prev => [...prev, newMsg]);
                    try {
                        const response = await fetch('/api/ai/chat', {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: JSON.stringify({ message: ragPrompt, context_size: contextSize }),
                        });
                        const reader = response.body?.getReader();
                        const decoder = new TextDecoder();
                        let fullText = '';
                        if (reader) {
                            while (true) {
                                const { done, value } = await reader.read();
                                if (done) break;
                                const chunk = decoder.decode(value, { stream: true });
                                for (const line of chunk.split('\n')) {
                                    if (!line.startsWith('data: ')) continue;
                                    const data = line.slice(6);
                                    if (data === '[DONE]') break;
                                    try {
                                        const parsed = JSON.parse(data);
                                        const content = parsed.choices?.[0]?.delta?.content || '';
                                        fullText += content;
                                        setMessages(prev => {
                                            const updated = [...prev];
                                            updated[updated.length - 1] = { role: 'assistant', content: fullText };
                                            return updated;
                                        });
                                    } catch { /* skip */ }
                                }
                            }
                        }
                    } catch {
                        setMessages(prev => [...prev, { role: 'system', content: '[!] LLM query failed.' }]);
                    }
                }).catch(() => {
                    setMessages(prev => prev.filter(m => m.content !== '📄 Searching documents...'));
                    setMessages(prev => [...prev, { role: 'system', content: '[!] Document search failed.' }]);
                }).finally(() => setIsLoading(false));
                return true;
            }
            // Default /doc help
            setMessages(prev => [...prev, { role: 'system', content: 'Document commands:\n  /doc upload    Upload a PDF or text file\n  /doc list      List uploaded documents\n  /doc search    Search documents (e.g. /doc search quantum physics)\n  /doc ask       Ask a question using document context\n  /doc delete    Remove a document (e.g. /doc delete notes.pdf)' }]);
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
            const contextWindow = chatMessages.slice(-contextSize);
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

        // Auto-speak the response if voice mode is on
        if (voiceEnabled) {
            // We need to get the latest message from state — use a callback pattern
            setMessages(prev => {
                const assistantMsg = prev[prev.length - 1];
                if (assistantMsg?.role === 'assistant' && assistantMsg.content) {
                    api.speak(assistantMsg.content.slice(0, 500)).then(result => {
                        if (result.audio && audioRef.current) {
                            audioRef.current.src = result.audio;
                            audioRef.current.play().catch(() => { });
                        }
                    }).catch(() => { });
                }
                return prev;
            });
        }
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



    const formatUptime = (secs: number) => {
        const d = Math.floor(secs / 86400);
        const h = Math.floor((secs % 86400) / 3600);
        const m = Math.floor((secs % 3600) / 60);
        if (d > 0) return `${d}d ${h}h`;
        if (h > 0) return `${h}h ${m}m`;
        return `${m}m`;
    };

    const UsageBar = ({ label, used, total, unit }: { label: string; used: number; total: number; unit: string }) => {
        const pct = total > 0 ? Math.round((used / total) * 100) : 0;
        const color = pct > 80 ? 'var(--accent-red)' : pct > 60 ? 'var(--accent-amber)' : 'var(--accent-green)';
        return (
            <div style={{ margin: '4px 0 0' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 10, color: 'var(--text-muted)' }}>
                    <span>{label}</span>
                    <span className="mono">{used}{unit} / {total}{unit}</span>
                </div>
                <div className="progress-bar" style={{ marginTop: 2, height: 4 }}>
                    <div className="progress-fill" style={{ width: `${pct}%`, background: color }} />
                </div>
            </div>
        );
    };

    const runningCount = services.filter(s => s.running).length;
    const ramGB = resources ? (resources.ram_total_mb / 1024).toFixed(1) : '?';

    // Brain Net live computed values
    const brainRamUsed = brainLive ? Math.round((brainLive.ram_total_mb - brainLive.ram_available_mb) / 1024 * 10) / 10 : null;
    const brainRamTotal = brainLive ? Math.round(brainLive.ram_total_mb / 1024 * 10) / 10 : null;
    const brainDiskUsed = brainLive ? brainLive.disk_total_gb - brainLive.disk_free_gb : null;

    // Envy computed values
    const envyRamUsed = envySysinfo?.ram_total_mb && envySysinfo?.ram_available_mb
        ? Math.round((envySysinfo.ram_total_mb - envySysinfo.ram_available_mb) / 1024 * 10) / 10 : null;
    const envyRamTotal = envySysinfo?.ram_total_mb ? Math.round(envySysinfo.ram_total_mb / 1024 * 10) / 10 : null;
    const envyDiskUsed = envySysinfo?.disk_total_gb && envySysinfo?.disk_free_gb
        ? envySysinfo.disk_total_gb - envySysinfo.disk_free_gb : null;

    // Phone computed values
    const phoneRamUsed = phoneStatus?.phone_ram_total_mb && phoneStatus?.phone_ram_available_mb
        ? Math.round((phoneStatus.phone_ram_total_mb - phoneStatus.phone_ram_available_mb) / 1024 * 10) / 10 : null;
    const phoneRamTotal = phoneStatus?.phone_ram_total_mb ? Math.round(phoneStatus.phone_ram_total_mb / 1024 * 10) / 10 : null;
    const phoneStorageUsed = phoneStatus?.phone_storage_free_gb != null
        ? 112 - phoneStatus.phone_storage_free_gb : null;

    return (
        <div className="ai-layout">
            {/* LEFT PANEL — Cluster Nodes */}
            <aside className="ai-panel-left">
                <div style={{ fontSize: 9, fontWeight: 600, textTransform: 'uppercase', letterSpacing: '1.5px', color: 'var(--text-muted)', marginBottom: -4 }}>
                    Cluster Nodes
                </div>

                {/* 🔧 Brain Net — OPS + VOICE */}
                <div className="node-card node-card-brain">
                    <div className="node-header">
                        <div>
                            <div className="node-name">🔧 Brain Net</div>
                            <div className="node-role">OPS + VOICE</div>
                        </div>
                        <span className={`badge badge-sm ${runningCount > 0 ? 'badge-green' : 'badge-red'}`}>
                            <span className={`status-dot ${runningCount > 0 ? 'up' : 'down'}`} />
                        </span>
                    </div>
                    <div className="node-specs">Celeron N4100 · {ramGB}G RAM</div>
                    <div className="node-service-row">
                        <span className="service-label">Dashboard</span>
                        <span className="service-status">
                            <span className="status-dot up" /> Online
                        </span>
                    </div>
                    <div className="node-service-row">
                        <span className="service-label">Whisper STT</span>
                        <span className="service-status">
                            <span className={`status-dot ${services.find(s => s.name.includes('whisper'))?.running !== false ? 'up' : 'down'}`} />
                            {services.find(s => s.name.includes('whisper'))?.running !== false ? 'Ready' : 'Offline'}
                        </span>
                    </div>
                    <div className="node-service-row">
                        <span className="service-label">Piper TTS</span>
                        <span className="service-status">
                            <span className={`status-dot ${services.find(s => s.name.includes('piper'))?.running !== false ? 'up' : 'down'}`} />
                            {services.find(s => s.name.includes('piper'))?.running !== false ? 'Ready' : 'Offline'}
                        </span>
                    </div>
                    {brainRamUsed != null && brainRamTotal != null && (
                        <UsageBar label="RAM" used={brainRamUsed} total={brainRamTotal} unit="G" />
                    )}
                    {brainDiskUsed != null && brainLive && (
                        <UsageBar label="Disk" used={brainDiskUsed} total={brainLive.disk_total_gb} unit="G" />
                    )}
                    {brainLive && (
                        <div className="node-service-row" style={{ marginTop: 4 }}>
                            <span className="service-label">Load</span>
                            <span className="service-status mono" style={{ color: 'var(--text-muted)', fontSize: 10 }}>
                                {brainLive.load_1m.toFixed(2)}
                            </span>
                        </div>
                    )}
                    {brainLive && brainLive.temp_c > 0 && (
                        <div className="node-service-row">
                            <span className="service-label">Temp</span>
                            <span className="service-status mono" style={{
                                color: brainLive.temp_c > 70 ? 'var(--accent-red)' : brainLive.temp_c > 55 ? 'var(--accent-amber)' : 'var(--text-muted)',
                                fontSize: 10
                            }}>{brainLive.temp_c}°C</span>
                        </div>
                    )}
                    {brainLive && (
                        <div className="node-service-row">
                            <span className="service-label">Uptime</span>
                            <span className="service-status mono" style={{ color: 'var(--text-muted)', fontSize: 10 }}>
                                {formatUptime(brainLive.uptime_secs)}
                            </span>
                        </div>
                    )}
                </div>

                {/* 🖼️ Envy — MEDIA */}
                <div className="node-card node-card-envy">
                    <div className="node-header">
                        <div>
                            <div className="node-name">🖼️ Envy</div>
                            <div className="node-role">MEDIA</div>
                        </div>
                        <span className={`badge badge-sm ${imageNodeOnline || musicNodeOnline ? 'badge-green' : 'badge-red'}`}>
                            <span className={`status-dot ${imageNodeOnline || musicNodeOnline ? 'up' : 'down'}`} />
                        </span>
                    </div>
                    <div className="node-specs">{envySysinfo?.cpu_model ? envySysinfo.cpu_model.replace('Intel(R) Core(TM) ', '').replace(' CPU', '') : 'i5-6200U'} · {envyRamTotal || '7.8'}G RAM</div>
                    <div className="node-service-row">
                        <span className="service-label">Image Gen</span>
                        <span className="service-status" style={{ color: imageNodeOnline ? 'var(--accent-green)' : 'var(--accent-red)' }}>
                            <span className={`status-dot ${imageNodeOnline ? 'up' : 'down'}`} />
                            {imageNodeOnline ? 'Online' : 'Offline'}
                        </span>
                    </div>
                    <div className="node-service-row">
                        <span className="service-label">Music Gen</span>
                        <span className="service-status" style={{ color: musicNodeOnline ? 'var(--accent-green)' : 'var(--accent-red)' }}>
                            <span className={`status-dot ${musicNodeOnline ? 'up' : 'down'}`} />
                            {musicNodeOnline ? 'Online' : 'Offline'}
                        </span>
                    </div>
                    {envyRamUsed != null && envyRamTotal != null && (
                        <UsageBar label="RAM" used={envyRamUsed} total={envyRamTotal} unit="G" />
                    )}
                    {envyDiskUsed != null && envySysinfo?.disk_total_gb && (
                        <UsageBar label="Disk" used={envyDiskUsed} total={envySysinfo.disk_total_gb} unit="G" />
                    )}
                    {envySysinfo?.load_1m != null && (
                        <div className="node-service-row" style={{ marginTop: 4 }}>
                            <span className="service-label">Load</span>
                            <span className="service-status mono" style={{ color: 'var(--text-muted)', fontSize: 10 }}>
                                {envySysinfo.load_1m.toFixed(2)}
                            </span>
                        </div>
                    )}
                    {envySysinfo?.temp_c != null && envySysinfo.temp_c > 0 && (
                        <div className="node-service-row">
                            <span className="service-label">Temp</span>
                            <span className="service-status mono" style={{
                                color: envySysinfo.temp_c > 70 ? 'var(--accent-red)' : envySysinfo.temp_c > 55 ? 'var(--accent-amber)' : 'var(--text-muted)',
                                fontSize: 10
                            }}>{envySysinfo.temp_c}°C</span>
                        </div>
                    )}
                    {envySysinfo?.uptime_secs != null && (
                        <div className="node-service-row">
                            <span className="service-label">Uptime</span>
                            <span className="service-status mono" style={{ color: 'var(--text-muted)', fontSize: 10 }}>
                                {formatUptime(envySysinfo.uptime_secs)}
                            </span>
                        </div>
                    )}
                    <div className="node-service-row" style={{ marginTop: 2 }}>
                        <span className="service-label">Link</span>
                        <span className="service-status" style={{ color: 'var(--text-muted)' }}>Ethernet 1Gbps</span>
                    </div>
                </div>

                {/* 🧠 Phone — LLM */}
                <div className="node-card node-card-phone">
                    <div className="node-header">
                        <div>
                            <div className="node-name">🧠 Phone</div>
                            <div className="node-role">LLM</div>
                        </div>
                        <span className={`badge badge-sm ${phoneStatus?.running ? 'badge-green' : 'badge-red'}`}>
                            <span className={`status-dot ${phoneStatus?.running ? 'up' : 'down'}`} />
                        </span>
                    </div>
                    <div className="node-specs">
                        {phoneStatus?.soc || 'T616'} · {phoneStatus?.phone_ram_total_mb ? `${(phoneStatus.phone_ram_total_mb / 1024).toFixed(1)}G RAM` : '6G RAM'}
                    </div>
                    <div className="node-service-row">
                        <span className="service-label">Active Model</span>
                        <span className="service-status" style={{ color: phoneStatus?.running ? 'var(--accent-green)' : 'var(--accent-red)' }}>
                            <span className={`status-dot ${phoneStatus?.running ? 'up' : 'down'}`} />
                            {phoneStatus?.display_name || (phoneStatus?.running ? 'Running' : 'Offline')}
                        </span>
                    </div>
                    {phoneStatus?.phone_cpu_cores && phoneStatus.phone_cpu_cores > 0 && (
                        <div className="node-service-row">
                            <span className="service-label">CPU</span>
                            <span className="service-status" style={{ color: 'var(--text-muted)' }}>{phoneStatus.phone_cpu_cores} cores</span>
                        </div>
                    )}
                    {phoneStatus?.android_version && (
                        <div className="node-service-row">
                            <span className="service-label">Android</span>
                            <span className="service-status" style={{ color: 'var(--text-muted)' }}>{phoneStatus.android_version}</span>
                        </div>
                    )}
                    {phoneStatus?.battery_pct != null && phoneStatus.battery_pct >= 0 && (
                        <div className="node-service-row">
                            <span className="service-label">Battery</span>
                            <span className="service-status" style={{
                                color: phoneStatus.battery_pct > 50 ? 'var(--accent-green)' :
                                    phoneStatus.battery_pct > 20 ? 'var(--accent-amber)' : 'var(--accent-red)'
                            }}>{phoneStatus.battery_pct}%</span>
                        </div>
                    )}
                    {phoneRamUsed != null && phoneRamTotal != null && (
                        <UsageBar label="RAM" used={phoneRamUsed} total={phoneRamTotal} unit="G" />
                    )}
                    {phoneStorageUsed != null && (
                        <UsageBar label="Storage" used={phoneStorageUsed} total={112} unit="G" />
                    )}
                    <div className="node-service-row" style={{ marginTop: 2 }}>
                        <span className="service-label">Link</span>
                        <span className="service-status" style={{ color: 'var(--text-muted)' }}>USB / ADB</span>
                    </div>
                </div>

                {/* Services */}
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
                            {phoneStatus?.running && (
                                <span style={{ color: crtDim, fontSize: 10 }}>
                                    {formatParams(phoneStatus.params)} │ {formatContext(phoneStatus.context)}ctx
                                    {phoneStatus.model ? ` │ ${phoneStatus.model.replace('.gguf', '').split('-').pop()?.toUpperCase()}` : ''}
                                </span>
                            )}
                            {imageNodeOnline && (
                                <span style={{ color: crtDim, fontSize: 10 }}>│ 🎨</span>
                            )}
                            {phoneStatus?.running && (
                                <span style={{ color: crtMain }}>{phoneStatus.display_name || phoneStatus.model}</span>
                            )}
                            <span className={`terminal-dot ${phoneStatus?.running ? 'on' : 'off'}`} />
                            <button
                                className="clear-chat-btn"
                                onClick={() => setMessages([{ role: 'system', content: 'Terminal cleared.' }])}
                                title="Clear chat"
                            >
                                CLR
                            </button>
                            <button
                                onClick={() => setIsHackerMode(!isHackerMode)}
                                title={`Switch to ${isHackerMode ? 'Premium' : 'Hacker'} Mode`}
                                style={{
                                    background: isHackerMode ? 'rgba(34,187,34,0.1)' : 'transparent',
                                    border: isHackerMode ? '1px solid #22bb22' : '1px solid var(--border-primary)',
                                    borderRadius: '4px',
                                    color: isHackerMode ? '#22bb22' : 'var(--text-muted)',
                                    cursor: 'pointer', fontSize: 10, padding: '2px 6px',
                                    fontFamily: '"IBM Plex Mono", monospace',
                                    textTransform: 'uppercase'
                                }}
                            >
                                {isHackerMode ? 'Hack' : 'Prem'}
                            </button>
                            {isHackerMode && (
                                <button
                                    onClick={() => {
                                        const next = crtColor === 'green' ? 'amber' : 'green';
                                        setCrtColor(next);
                                        localStorage.setItem('crt-color', next);
                                    }}
                                    title={`Switch to ${crtColor === 'green' ? 'amber' : 'green'} CRT`}
                                    style={{
                                        background: 'transparent', border: 'none', cursor: 'pointer', fontSize: 12,
                                        color: crtColor === 'green' ? '#cc8800' : '#22bb22', padding: '0 4px',
                                    }}
                                >
                                    ◉
                                </button>
                            )}
                        </div>
                    </div>

                    {showGallery ? (
                        <div className="chat-messages" style={{ padding: 16, overflow: 'auto' }}>
                            {galleryLoading ? (
                                <div style={{ textAlign: 'center', padding: 40, color: 'var(--text-muted)' }}>Loading gallery...</div>
                            ) : galleryImages.length === 0 ? (
                                <div style={{ textAlign: 'center', padding: 40, color: 'var(--text-muted)' }}>
                                    <div style={{ fontSize: 48, marginBottom: 12 }}>🖼️</div>
                                    <div>No images yet. Use <code>/imagine &lt;prompt&gt;</code> to generate art.</div>
                                </div>
                            ) : (
                                <div style={{
                                    display: 'grid',
                                    gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))',
                                    gap: 12,
                                    padding: 4,
                                }}>
                                    {galleryImages.map(img => (
                                        <div
                                            key={img.id}
                                            style={{
                                                position: 'relative',
                                                borderRadius: 8,
                                                overflow: 'hidden',
                                                cursor: 'pointer',
                                                background: 'rgba(0,0,0,0.3)',
                                                border: '1px solid rgba(255,255,255,0.1)',
                                                transition: 'transform 0.2s, box-shadow 0.2s',
                                            }}
                                            onClick={() => setSelectedImage(img.id)}
                                            onMouseEnter={e => { (e.currentTarget as HTMLElement).style.transform = 'scale(1.03)'; (e.currentTarget as HTMLElement).style.boxShadow = '0 4px 20px rgba(0,0,0,0.5)'; }}
                                            onMouseLeave={e => { (e.currentTarget as HTMLElement).style.transform = 'scale(1)'; (e.currentTarget as HTMLElement).style.boxShadow = 'none'; }}
                                        >
                                            <img
                                                src={api.getGalleryImageUrl(img.id)}
                                                alt={img.prompt}
                                                loading="lazy"
                                                style={{ width: '100%', display: 'block', aspectRatio: `${img.width}/${img.height}` }}
                                            />
                                            <div style={{
                                                position: 'absolute', bottom: 0, left: 0, right: 0,
                                                background: 'linear-gradient(transparent, rgba(0,0,0,0.8))',
                                                padding: '20px 8px 8px',
                                                fontSize: 11,
                                                color: '#ccc',
                                            }}>
                                                {img.prompt?.slice(0, 60) || 'Untitled'}
                                            </div>
                                            <button
                                                onClick={e => { e.stopPropagation(); api.deleteGalleryImage(img.id).then(() => setGalleryImages(prev => prev.filter(g => g.id !== img.id))); }}
                                                style={{
                                                    position: 'absolute', top: 4, right: 4,
                                                    background: 'rgba(255,0,0,0.6)', border: 'none', color: '#fff',
                                                    borderRadius: 4, width: 24, height: 24, cursor: 'pointer',
                                                    fontSize: 12, display: 'flex', alignItems: 'center', justifyContent: 'center',
                                                    opacity: 0.6, transition: 'opacity 0.2s',
                                                }}
                                                onMouseEnter={e => (e.currentTarget.style.opacity = '1')}
                                                onMouseLeave={e => (e.currentTarget.style.opacity = '0.6')}
                                                title="Delete image"
                                            >✕</button>
                                        </div>
                                    ))}
                                </div>
                            )}
                        </div>
                    ) : (
                        <div className="chat-messages">
                            {messages.map((msg, i) => {
                                if (msg.role === 'system') {
                                    const content = msg.content || '';
                                    // Inline image rendering
                                    if (content.includes('<IMAGE>')) {
                                        const imageMatch = content.match(/<IMAGE>(.*?)<\/IMAGE>/);
                                        const caption = content.replace(/<IMAGE>.*?<\/IMAGE>\n?/, '').trim();
                                        return (
                                            <div key={i} className="chat-message assistant">
                                                <div className="chat-bubble terminal-system-msg">
                                                    <div className="terminal-tool-call">
                                                        <span className="terminal-tool-icon">🎨</span>
                                                        <span>Image Generated</span>
                                                    </div>
                                                    {imageMatch && (
                                                        <div className="terminal-image">
                                                            <img src={imageMatch[1]} alt={caption} />
                                                            <div className="terminal-image-meta">
                                                                <span>{caption}</span>
                                                            </div>
                                                        </div>
                                                    )}
                                                </div>
                                            </div>
                                        );
                                    }
                                    // Image generation progress
                                    if (content.includes('<IMAGEPROGRESS>')) {
                                        const promptText = content.replace('\n<IMAGEPROGRESS>', '').replace('🎨 Generating: ', '').replace('...', '');
                                        return (
                                            <div key={i} className="chat-message assistant">
                                                <div className="chat-bubble terminal-system-msg">
                                                    <div className="terminal-tool-call">
                                                        <span className="terminal-tool-icon">🎨</span>
                                                        <span>Generating: {promptText}...</span>
                                                    </div>
                                                    <div className="gen-progress-bar">
                                                        <div className="gen-progress-fill" />
                                                    </div>
                                                </div>
                                            </div>
                                        );
                                    }
                                    // Inline music player
                                    if (content.includes('<MUSIC>')) {
                                        const audioMatch = content.match(/<MUSIC>(.*?)<\/MUSIC>/);
                                        const caption = content.replace(/<MUSIC>.*?<\/MUSIC>\n?/, '').trim();
                                        return (
                                            <div key={i} className="chat-message assistant">
                                                <div className="chat-bubble terminal-system-msg">
                                                    <div className="terminal-tool-call">
                                                        <span className="terminal-tool-icon">🎵</span>
                                                        <span>Music Generated</span>
                                                    </div>
                                                    {audioMatch && (
                                                        <div style={{ margin: '8px 0' }}>
                                                            <audio controls src={audioMatch[1]} style={{ width: '100%', height: 32 }} />
                                                            <div style={{ fontSize: 10, color: 'var(--text-muted)', marginTop: 4 }}>{caption}</div>
                                                        </div>
                                                    )}
                                                </div>
                                            </div>
                                        );
                                    }
                                    // Music generation progress
                                    if (content.includes('<MUSICPROGRESS>')) {
                                        const promptText = content.replace('\n<MUSICPROGRESS>', '').replace('🎵 Generating: ', '').replace('...', '');
                                        return (
                                            <div key={i} className="chat-message assistant">
                                                <div className="chat-bubble terminal-system-msg">
                                                    <div className="terminal-tool-call">
                                                        <span className="terminal-tool-icon">🎵</span>
                                                        <span>Generating: {promptText}...</span>
                                                    </div>
                                                    <div className="gen-progress-bar">
                                                        <div className="gen-progress-fill" />
                                                    </div>
                                                </div>
                                            </div>
                                        );
                                    }
                                    return (
                                        <div key={i} className={`chat-message assistant ${!isHackerMode ? 'premium-chat-message' : ''}`}>
                                            <div className={`chat-bubble terminal-system-msg ${!isHackerMode ? 'premium-chat-bubble' : ''}`}>{content}</div>
                                        </div>
                                    );
                                }
                                const isStreaming = isLoading && msg.role === 'assistant' && i === messages.length - 1;

                                if (!isHackerMode) {
                                    // Premium Bubble Render
                                    return (
                                        <div key={i} className={`premium-chat-message ${msg.role}`}>
                                            <div className="premium-chat-bubble">
                                                {msg.content || (isStreaming ? '...' : '')}
                                            </div>
                                        </div>
                                    );
                                }

                                // Hacker CRT Terminal Render
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
                                <div className={`chat-message assistant ${!isHackerMode ? 'premium-chat-message' : ''}`}>
                                    <div className={`chat-bubble ${!isHackerMode ? 'premium-chat-bubble' : ''}`} style={isHackerMode ? { color: '#1a8a1a' } : {}}>
                                        Processing{isHackerMode && <span className="terminal-cursor" />}
                                    </div>
                                </div>
                            )}
                            <div ref={messagesEndRef} />
                        </div>
                    )}

                    <div className="terminal-input-container" style={!isHackerMode ? { background: 'var(--bg-card)', borderTop: '1px solid var(--border-primary)', padding: '16px' } : {}}>
                        {isHackerMode && <span className="terminal-input-prefix">C:\&gt;&nbsp;</span>}
                        <input
                            className="chat-input"
                            style={!isHackerMode ? { background: 'var(--bg-secondary)', border: '1px solid var(--border-primary)', borderRadius: 'var(--radius-md)', padding: '12px 16px', color: 'var(--text-primary)', textShadow: 'none', fontFamily: '"Inter", sans-serif' } : {}}
                            placeholder={'enter command or message...'}
                            value={input}
                            onChange={e => setInput(e.target.value)}
                            onKeyDown={handleKeyDown}
                            disabled={isLoading || isGeneratingImage}
                        />
                        <button
                            className={`btn ${isRecording ? 'btn-danger' : 'btn-secondary'}`}
                            onClick={async () => {
                                if (isRecording) {
                                    // Stop recording
                                    mediaRecorderRef.current?.stop();
                                    setIsRecording(false);
                                } else {
                                    // Start recording — voice chat mode
                                    try {
                                        const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
                                        const recorder = new MediaRecorder(stream, { mimeType: 'audio/webm' });
                                        audioChunksRef.current = [];
                                        recorder.ondataavailable = (e) => audioChunksRef.current.push(e.data);
                                        recorder.onstop = async () => {
                                            stream.getTracks().forEach(t => t.stop());
                                            const blob = new Blob(audioChunksRef.current, { type: 'audio/webm' });
                                            setMessages(prev => [...prev, { role: 'system', content: '🎙️ Processing voice...' }]);
                                            setIsLoading(true);
                                            try {
                                                const result = await api.voiceChat(blob);
                                                // Remove processing message
                                                setMessages(prev => prev.filter(m => m.content !== '🎙️ Processing voice...'));
                                                if (result.error && !result.transcript) {
                                                    setMessages(prev => [...prev, { role: 'system', content: `[!] ${result.error}` }]);
                                                } else {
                                                    // Add user transcript and assistant response to chat
                                                    if (result.transcript) {
                                                        setMessages(prev => [...prev, { role: 'user', content: result.transcript }]);
                                                    }
                                                    if (result.response) {
                                                        setMessages(prev => [...prev, { role: 'assistant', content: result.response }]);
                                                    }
                                                    // Play TTS audio
                                                    if (result.audio && audioRef.current) {
                                                        audioRef.current.src = result.audio;
                                                        audioRef.current.play().catch(() => { });
                                                    }
                                                }
                                            } catch {
                                                setMessages(prev => prev.filter(m => m.content !== '🎙️ Processing voice...'));
                                                setMessages(prev => [...prev, { role: 'system', content: '[!] Voice chat failed' }]);
                                            } finally {
                                                setIsLoading(false);
                                            }
                                        };
                                        recorder.start();
                                        mediaRecorderRef.current = recorder;
                                        setIsRecording(true);
                                    } catch {
                                        setMessages(prev => [...prev, { role: 'system', content: '[!] Microphone access denied' }]);
                                    }
                                }
                            }}
                            disabled={isLoading || isGeneratingImage}
                            title={isRecording ? 'Stop recording' : 'Voice chat'}
                            style={{ minWidth: 40, fontSize: 16 }}
                        >
                            {isRecording ? '⏹' : '🎙️'}
                        </button>
                        <button className="btn btn-primary" onClick={handleSend} disabled={isLoading || isGeneratingImage}>SEND</button>
                        {isLoading && (
                            <button className="btn btn-danger" onClick={handleStop} style={{ minWidth: 60 }}>■ STOP</button>
                        )}
                        {voiceEnabled && <span style={{ fontSize: 10, color: 'var(--accent-green)', alignSelf: 'center' }}>🔊</span>}
                        <button
                            onClick={() => {
                                const next = !showGallery;
                                setShowGallery(next);
                                if (next) {
                                    setGalleryLoading(true);
                                    api.getGalleryImages().then(res => setGalleryImages(res.images || [])).catch(() => { }).finally(() => setGalleryLoading(false));
                                }
                            }}
                            style={{
                                background: showGallery ? 'var(--accent-blue)' : 'transparent',
                                border: '1px solid var(--border-primary)',
                                color: showGallery ? '#fff' : 'var(--text-muted)',
                                borderRadius: 6, padding: '4px 8px', cursor: 'pointer', fontSize: 14,
                                alignSelf: 'center',
                            }}
                            title={showGallery ? 'Back to chat' : 'Open gallery'}
                        >{showGallery ? '💬 Chat' : '🖼️ Gallery'}</button>
                    </div>
                    <audio ref={audioRef} style={{ display: 'none' }} />

                    {/* Image viewer overlay */}
                    {selectedImage && (
                        <div
                            style={{
                                position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
                                background: 'rgba(0,0,0,0.9)', zIndex: 9999,
                                display: 'flex', alignItems: 'center', justifyContent: 'center',
                                cursor: 'pointer',
                            }}
                            onClick={() => setSelectedImage(null)}
                        >
                            <img
                                src={api.getGalleryImageUrl(selectedImage)}
                                alt=""
                                style={{ maxWidth: '90vw', maxHeight: '90vh', borderRadius: 8, boxShadow: '0 0 40px rgba(0,0,0,0.8)' }}
                                onClick={e => e.stopPropagation()}
                            />
                            <button
                                style={{
                                    position: 'absolute', top: 20, right: 20,
                                    background: 'rgba(255,255,255,0.15)', border: 'none', color: '#fff',
                                    borderRadius: 8, padding: '8px 16px', cursor: 'pointer', fontSize: 16,
                                }}
                                onClick={() => setSelectedImage(null)}
                            >✕ Close</button>
                            <button
                                style={{
                                    position: 'absolute', bottom: 20, right: 20,
                                    background: 'rgba(255,60,60,0.7)', border: 'none', color: '#fff',
                                    borderRadius: 8, padding: '8px 16px', cursor: 'pointer', fontSize: 14,
                                }}
                                onClick={e => {
                                    e.stopPropagation();
                                    api.deleteGalleryImage(selectedImage).then(() => {
                                        setGalleryImages(prev => prev.filter(g => g.id !== selectedImage));
                                        setSelectedImage(null);
                                    });
                                }}
                            >🗑️ Delete</button>
                        </div>
                    )}
                    <input
                        ref={fileInputRef}
                        type="file"
                        accept=".pdf,.txt,.md,.json,.csv"
                        style={{ display: 'none' }}
                        onChange={async (e) => {
                            const file = e.target.files?.[0];
                            if (!file) return;
                            setMessages(prev => [...prev, { role: 'system', content: `📄 Uploading ${file.name}...` }]);
                            try {
                                const result = await api.uploadDocument(file);
                                setMessages(prev => prev.filter(m => !m.content.startsWith('📄 Uploading')));
                                setMessages(prev => [...prev, {
                                    role: 'system',
                                    content: `✅ ${result.name} uploaded\n   ${result.chunks} chunks embedded in ${(result.embed_time_ms / 1000).toFixed(1)}s`
                                }]);
                            } catch {
                                setMessages(prev => prev.filter(m => !m.content.startsWith('📄 Uploading')));
                                setMessages(prev => [...prev, { role: 'system', content: '[!] Upload failed. RAG server may be offline.' }]);
                            }
                            e.target.value = '';
                        }}
                    />
                </div>
            </div>

            {/* RIGHT PANEL — Models */}
            <aside className="ai-panel-right">

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
                                                        // Poll phone-status every 3s for up to 90s
                                                        let loaded = false;
                                                        for (let i = 0; i < 30; i++) {
                                                            await new Promise(r => setTimeout(r, 3000));
                                                            setStatusMsg(`Loading ${displayName}... ${(i + 1) * 3}s`);
                                                            try {
                                                                const ps = await api.getPhoneStatus();
                                                                if (ps.running && ps.model === m.name) {
                                                                    loaded = true;
                                                                    setPhoneStatus(ps);
                                                                    setActiveModel(ps.model || '');
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

                {/* Context Window Slider */}
                <div className="card compact">
                    <div className="card-title">Performance</div>
                    <div className="setting-block">
                        <div className="setting-label-row">
                            <span className="setting-label">Context Window</span>
                            <span className="setting-value">{contextSize} msgs</span>
                        </div>
                        <input
                            type="range"
                            className="setting-slider"
                            min={2}
                            max={20}
                            value={contextSize}
                            onChange={e => {
                                const val = parseInt(e.target.value, 10);
                                setContextSize(val);
                                localStorage.setItem('context-size', String(val));
                            }}
                        />
                        <div className="setting-hint">Fewer = faster responses on small models</div>
                    </div>
                    <div className="setting-block" style={{ marginTop: 12 }}>
                        <div className="setting-label-row">
                            <span className="setting-label">Image Resolution</span>
                            <span className="setting-value">{imageResolution}x{imageResolution}</span>
                        </div>
                        <div style={{ display: 'flex', gap: 6, marginTop: 6 }}>
                            <button
                                className={`btn btn-sm ${imageResolution === '256' ? 'btn-primary' : 'btn-outline'}`}
                                onClick={() => { setImageResolution('256'); localStorage.setItem('image-resolution', '256'); }}
                                style={{ flex: 1, fontSize: 11, padding: '4px 8px' }}
                            >
                                256 · Fast (~20s)
                            </button>
                            <button
                                className={`btn btn-sm ${imageResolution === '512' ? 'btn-primary' : 'btn-outline'}`}
                                onClick={() => { setImageResolution('512'); localStorage.setItem('image-resolution', '512'); }}
                                style={{ flex: 1, fontSize: 11, padding: '4px 8px' }}
                            >
                                512 · Quality (~75s)
                            </button>
                        </div>
                        <div className="setting-hint">/imagine resolution on brain net</div>
                    </div>
                    {imageNodeOnline && (
                        <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 11, color: 'var(--accent-green)', marginTop: 8 }}>
                            <span className="status-dot up" />
                            <span>Image Gen Node</span>
                        </div>
                    )}
                    {!imageNodeOnline && (
                        <div style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 11, color: 'var(--text-muted)', marginTop: 8 }}>
                            <span className="status-dot down" />
                            <span>Image Gen Offline</span>
                        </div>
                    )}
                </div>
            </aside>
        </div>
    );
}
