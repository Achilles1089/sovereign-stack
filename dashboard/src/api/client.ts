const API_BASE = '/api';

export interface ServiceStatus {
    name: string;
    running: boolean;
    status: string;
    ports: string;
    image: string;
}

export interface SystemResources {
    cpu_percent: number;
    ram_used_mb: number;
    ram_total_mb: number;
    disk_used_gb: number;
    disk_total_gb: number;
    gpu_name: string;
    gpu_type: string;
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

export interface ChatMessage {
    role: 'user' | 'assistant';
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
    installApp: (name: string) => fetch(API_BASE + `/apps/${name}/install`, { method: 'POST' }),
    removeApp: (name: string) => fetch(API_BASE + `/apps/${name}`, { method: 'DELETE' }),
    getModels: () => fetchJSON<{ models: AIModel[] }>('/ai/models'),
    chat: async (model: string, messages: ChatMessage[], onChunk: (text: string) => void) => {
        const res = await fetch(API_BASE + '/ai/chat', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ model, messages }),
        });
        const reader = res.body?.getReader();
        if (!reader) return;
        const decoder = new TextDecoder();
        while (true) {
            const { done, value } = await reader.read();
            if (done) break;
            onChunk(decoder.decode(value));
        }
    },
    getBackups: () => fetchJSON<{ snapshots: any[] }>('/backups'),
    triggerBackup: () => fetch(API_BASE + '/backups', { method: 'POST' }),
};
