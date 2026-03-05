const API_BASE = '/api';

export interface ServiceStatus {
    name: string;
    running: boolean;
    status: string;
    ports: string;
    image: string;
}

export interface SystemResources {
    cpu_model: string;
    cpu_cores: number;
    ram_total_mb: number;
    disk_total_gb: number;
    disk_free_gb: number;
    gpu_type: string;
    gpu_name: string;
    gpu_memory_mb: number;
}

export interface AppInfo {
    name: string;
    display_name: string;
    description: string;
    category: string;
    version: string;
    installed: boolean;
}

export interface AIModel {
    name: string;
    size: number;
    modified_at: string;
}

export interface AIStatus {
    running: boolean;
    host: string;
    mode: string;
    model: string;
    gpu_tier: string;
    recommended: string;
    engine: string;
    models_dir: string;
}

export interface CatalogEntry {
    name: string;
    display_name: string;
    filename: string;
    size_gb: number;
    min_ram_mb: number;
    tier: string;
    architecture: string;
    description: string;
    url: string;
    installed: boolean;
}

export interface PhoneStatus {
    model: string;
    display_name: string;
    params: number;
    vocab: number;
    context: number;
    size_bytes: number;
    engine: string;
    running: boolean;
    // Phone hardware (from sysinfo companion)
    phone_model?: string;
    soc?: string;
    android_version?: string;
    phone_cpu_cores?: number;
    phone_ram_total_mb?: number;
    phone_ram_available_mb?: number;
    phone_storage_free_gb?: number;
    battery_pct?: number;
}

export interface PhoneModel {
    name: string;
    path: string;
    size_mb: number;
}

export interface ChatMessage {
    role: 'user' | 'assistant' | 'system';
    content: string;
}

export interface ImageGenResponse {
    image: string;
    time_ms: number;
}

export interface ImageGenStatus {
    online: boolean;
    model: string;
}

export interface BrainNetLive {
    ram_total_mb: number;
    ram_available_mb: number;
    load_1m: number;
    load_5m: number;
    load_15m: number;
    uptime_secs: number;
    temp_c: number;
    disk_total_gb: number;
    disk_free_gb: number;
}

export interface EnvySysinfo {
    online: boolean;
    cpu_model?: string;
    cpu_cores?: number;
    load_1m?: number;
    load_5m?: number;
    load_15m?: number;
    ram_total_mb?: number;
    ram_available_mb?: number;
    disk_total_gb?: number;
    disk_free_gb?: number;
    temp_c?: number;
    uptime_secs?: number;
}

export interface AgentStatus {
    online: boolean;
    model: string;
    display_name: string;
    tools: number;
    memory_messages: number;
}

async function fetchJSON<T>(path: string): Promise<T> {
    const res = await fetch(API_BASE + path);
    if (!res.ok) throw new Error(`API error: ${res.status}`);
    return res.json();
}

export const api = {
    getStatus: () => fetchJSON<{ services: ServiceStatus[] }>('/status'),
    getResources: () => fetchJSON<SystemResources>('/resources'),
    getApps: () => fetchJSON<{ apps: AppInfo[] }>('/apps'),
    getAIStatus: () => fetchJSON<AIStatus>('/ai/status'),
    getModels: () => fetchJSON<{ models: AIModel[] }>('/ai/models'),
    getCatalog: () => fetchJSON<{ catalog: CatalogEntry[] }>('/ai/catalog'),
    getPhoneStatus: () => fetchJSON<PhoneStatus>('/ai/phone-status'),
    getPhoneModels: () => fetchJSON<{ models: PhoneModel[]; active: string | null }>('/ai/phone-models'),
    getImageStatus: () => fetchJSON<ImageGenStatus>('/ai/image-status'),
    getBrainNetLive: () => fetchJSON<BrainNetLive>('/resources/live'),
    getEnvySysinfo: () => fetchJSON<EnvySysinfo>('/envy/sysinfo'),
    switchPhoneModel: (model: string) => fetch(API_BASE + '/ai/phone-switch', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ model }),
    }).then(r => r.json()),
    startPhone: (model?: string) => fetch(API_BASE + '/ai/phone-start', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ model: model || '' }),
    }).then(r => r.json()),

    installApp: (name: string) => fetch(API_BASE + '/apps/install', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name }),
    }).then(r => r.json()),
    removeApp: (name: string) => fetch(API_BASE + '/apps/remove', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name }),
    }).then(r => r.json()),

    generateImage: async (prompt: string, width = 512, height = 512): Promise<ImageGenResponse> => {
        const res = await fetch(API_BASE + '/ai/image-generate', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ prompt, width, height }),
        });
        if (!res.ok) throw new Error(`Image gen error: ${res.status}`);
        return res.json();
    },

    serverChat: async (message: string, model: string, onChunk: (text: string) => void) => {
        const res = await fetch(API_BASE + '/ai/server-chat', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ message, model }),
        });
        if (!res.ok) throw new Error(`Chat error: ${res.status}`);
        const reader = res.body?.getReader();
        if (!reader) return;
        const decoder = new TextDecoder();
        while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            onChunk(decoder.decode(value, { stream: true }));
        }
    },

    chat: async (model: string, messages: ChatMessage[], onChunk: (text: string) => void, signal?: AbortSignal) => {
        const res = await fetch(API_BASE + '/ai/chat', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ model, messages }),
            signal,
        });
        if (!res.ok) throw new Error(`Chat error: ${res.status}`);
        const reader = res.body?.getReader();
        if (!reader) return;
        const decoder = new TextDecoder();
        try {
            while (true) {
                const { done, value } = await reader.read();
                if (done) break;
                onChunk(decoder.decode(value, { stream: true }));
            }
        } catch (e) {
            reader.cancel();
            throw e;
        }
    },

    pullModel: async (model: string, onProgress: (text: string) => void) => {
        const res = await fetch(API_BASE + '/ai/pull', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ model }),
        });
        if (!res.ok) throw new Error(`Pull error: ${res.status}`);
        const reader = res.body?.getReader();
        if (!reader) return;
        const decoder = new TextDecoder();
        while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            onProgress(decoder.decode(value));
        }
    },

    deleteModel: (model: string) => fetch(API_BASE + '/ai/delete', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ model }),
    }).then(r => r.json()),

    // Voice pipeline (Brain Net local — Whisper STT + Piper TTS)
    getVoiceStatus: () => fetchJSON<{ stt_online: boolean; tts_online: boolean }>('/ai/voice-status'),

    transcribe: async (audioBlob: Blob): Promise<{ text: string; time_ms: number }> => {
        const res = await fetch(API_BASE + '/ai/transcribe', {
            method: 'POST',
            headers: { 'Content-Type': audioBlob.type || 'audio/wav' },
            body: audioBlob,
        });
        if (!res.ok) throw new Error(`Transcribe error: ${res.status}`);
        return res.json();
    },

    speak: async (text: string): Promise<{ audio: string; time_ms: number }> => {
        const res = await fetch(API_BASE + '/ai/speak', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ text }),
        });
        if (!res.ok) throw new Error(`Speak error: ${res.status}`);
        return res.json();
    },

    // Music generation (Envy local — spectrogram → audio)
    getMusicStatus: () => fetchJSON<{ online: boolean; engine: string }>('/ai/music-status'),

    generateMusic: async (prompt: string): Promise<{ audio: string; prompt: string; total_time_ms: number; duration_s: number }> => {
        const res = await fetch(API_BASE + '/ai/music-generate', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ prompt }),
        });
        if (!res.ok) throw new Error(`Music gen error: ${res.status}`);
        return res.json();
    },

    // Voice chat — full loop: audio → STT → LLM → TTS → audio response
    voiceChat: async (audioBlob: Blob): Promise<{ transcript: string; response: string; audio: string; stt_ms: number; tts_ms: number; error?: string }> => {
        const res = await fetch(API_BASE + '/ai/voice-chat', {
            method: 'POST',
            headers: { 'Content-Type': audioBlob.type || 'audio/webm' },
            body: audioBlob,
        });
        if (!res.ok) throw new Error(`Voice chat error: ${res.status}`);
        return res.json();
    },

    // RAG Document Chat — upload docs, search, manage
    getRAGStatus: () => fetchJSON<{ online: boolean; model_loaded: boolean; documents: number; model: string }>('/ai/rag-status'),

    uploadDocument: async (file: File): Promise<{ doc_id: string; name: string; chunks: number; embed_time_ms: number }> => {
        const formData = new FormData();
        formData.append('file', file);
        const res = await fetch(API_BASE + '/ai/rag-upload', {
            method: 'POST',
            body: formData,
        });
        if (!res.ok) throw new Error(`Upload error: ${res.status}`);
        return res.json();
    },

    searchDocuments: async (query: string, k: number = 5): Promise<{ results: Array<{ document: string; chunk: string; score: number }>; query: string }> => {
        const res = await fetch(API_BASE + `/ai/rag-search?q=${encodeURIComponent(query)}&k=${k}`);
        if (!res.ok) throw new Error(`Search error: ${res.status}`);
        return res.json();
    },

    listDocuments: () => fetchJSON<{ documents: Array<{ doc_id: string; name: string; num_chunks: number; added_at: string }> }>('/ai/rag-documents'),

    deleteDocument: async (name: string): Promise<{ deleted: string }> => {
        const res = await fetch(API_BASE + `/ai/rag-delete?name=${encodeURIComponent(name)}`, { method: 'POST' });
        if (!res.ok) throw new Error(`Delete error: ${res.status}`);
        return res.json();
    },

    // Gallery
    getGalleryImages: () => fetchJSON<{ images: Array<{ id: string; prompt: string; width: number; height: number; created_at: string; size_bytes: number }> }>('/gallery'),
    getGalleryImageUrl: (id: string) => API_BASE + `/gallery/image/${id}`,
    deleteGalleryImage: async (id: string): Promise<{ deleted: string }> => {
        const res = await fetch(API_BASE + `/gallery/delete/${id}`, { method: 'POST' });
        if (!res.ok) throw new Error(`Delete error: ${res.status}`);
        return res.json();
    },

    // News
    getNewsFeeds: () => fetchJSON<{ feeds: Array<{ category: string; feeds: string[]; article_count: number }> }>('/news/feeds'),
    getNewsArticles: (category?: string, page: number = 1) => {
        const params = new URLSearchParams({ page: String(page), limit: '20' });
        if (category) params.set('feed', category);
        return fetchJSON<{ articles: Array<{ id: string; title: string; link: string; source: string; category: string; published: string; summary: string }>; total: number; page: number; pages: number }>(`/news/articles?${params}`);
    },
    searchNews: (query: string) => fetchJSON<{ articles: Array<{ id: string; title: string; link: string; source: string; category: string; published: string; summary: string }>; total: number }>(`/news/search?q=${encodeURIComponent(query)}`),
    refreshNews: async () => { const res = await fetch(API_BASE + '/news/refresh', { method: 'POST' }); return res.json(); },
    getNewsStatus: () => fetchJSON<{ online: boolean; feeds: number; articles: number; refreshing: boolean }>('/news/status'),

    // Achilles Agent (Claude Sonnet 4.6)
    getAgentStatus: () => fetchJSON<AgentStatus>('/agent/status'),
    clearAgentHistory: () => fetch(API_BASE + '/agent/clear').then(r => r.json()),
    agentChat: async (message: string, onChunk: (text: string) => void, signal?: AbortSignal) => {
        const res = await fetch(API_BASE + '/agent/chat', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ message }),
            signal,
        });
        if (!res.ok) throw new Error(`Agent error: ${res.status}`);
        const reader = res.body?.getReader();
        if (!reader) return;
        const decoder = new TextDecoder();
        try {
            while (true) {
                const { done, value } = await reader.read();
                if (done) break;
                onChunk(decoder.decode(value, { stream: true }));
            }
        } catch (e) {
            reader.cancel();
            throw e;
        }
    },
};
