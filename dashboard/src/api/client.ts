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
}

export interface ChatMessage {
    role: 'user' | 'assistant' | 'system';
    content: string;
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
};
