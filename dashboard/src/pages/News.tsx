import { useState, useEffect, useCallback } from 'react';
import { api } from '../api/client';

interface Article {
    id: string;
    title: string;
    link: string;
    source: string;
    category: string;
    published: string;
    summary: string;
}

interface FeedCategory {
    category: string;
    feeds: string[];
    article_count: number;
}

const CATEGORY_ICONS: Record<string, string> = {
    tech: '💻',
    crypto: '₿',
    general: '🌍',
    ai: '🤖',
};

export default function News() {
    const [articles, setArticles] = useState<Article[]>([]);
    const [feeds, setFeeds] = useState<FeedCategory[]>([]);
    const [selectedCategory, setSelectedCategory] = useState<string>('');
    const [searchQuery, setSearchQuery] = useState('');
    const [page, setPage] = useState(1);
    const [totalPages, setTotalPages] = useState(1);
    const [loading, setLoading] = useState(true);
    const [refreshing, setRefreshing] = useState(false);
    const [selectedArticle, setSelectedArticle] = useState<Article | null>(null);

    const loadArticles = useCallback(async (cat?: string, p: number = 1) => {
        setLoading(true);
        try {
            const res = await api.getNewsArticles(cat || undefined, p);
            setArticles(res.articles || []);
            setTotalPages(res.pages || 1);
            setPage(res.page || 1);
        } catch {
            setArticles([]);
        }
        setLoading(false);
    }, []);

    const handleSearch = async () => {
        if (!searchQuery.trim()) {
            loadArticles(selectedCategory);
            return;
        }
        setLoading(true);
        try {
            const res = await api.searchNews(searchQuery);
            setArticles(res.articles || []);
            setTotalPages(1);
            setPage(1);
        } catch {
            setArticles([]);
        }
        setLoading(false);
    };

    useEffect(() => {
        api.getNewsFeeds().then(res => setFeeds(res.feeds || [])).catch(() => { });
        loadArticles();
    }, [loadArticles]);

    const handleCategoryClick = (cat: string) => {
        const newCat = cat === selectedCategory ? '' : cat;
        setSelectedCategory(newCat);
        setSearchQuery('');
        loadArticles(newCat);
    };

    const handleRefresh = async () => {
        setRefreshing(true);
        try {
            await api.refreshNews();
            // Wait a bit for first articles to come in
            setTimeout(async () => {
                await loadArticles(selectedCategory);
                setRefreshing(false);
            }, 3000);
        } catch {
            setRefreshing(false);
        }
    };

    const timeAgo = (dateStr: string) => {
        if (!dateStr) return '';
        try {
            const diff = Date.now() - new Date(dateStr).getTime();
            const mins = Math.floor(diff / 60000);
            if (mins < 60) return `${mins}m ago`;
            const hrs = Math.floor(mins / 60);
            if (hrs < 24) return `${hrs}h ago`;
            const days = Math.floor(hrs / 24);
            return `${days}d ago`;
        } catch { return ''; }
    };

    return (
        <div style={{ display: 'flex', height: '100%', gap: 0, overflow: 'hidden' }}>
            {/* Sidebar — categories */}
            <aside style={{
                width: 220, minWidth: 220, background: 'var(--bg-card)',
                borderRight: '1px solid var(--border-primary)', padding: '16px 0',
                display: 'flex', flexDirection: 'column', overflow: 'auto',
            }}>
                <div style={{ padding: '0 16px 12px', fontSize: 18, fontWeight: 700, color: 'var(--text-primary)' }}>
                    📰 News
                </div>

                <button
                    onClick={() => handleCategoryClick('')}
                    style={{
                        display: 'flex', alignItems: 'center', gap: 8,
                        padding: '10px 16px', border: 'none', cursor: 'pointer',
                        background: selectedCategory === '' ? 'var(--accent-blue)' : 'transparent',
                        color: selectedCategory === '' ? '#fff' : 'var(--text-secondary)',
                        fontSize: 14, textAlign: 'left', width: '100%',
                        transition: 'background 0.15s',
                    }}
                >
                    <span>📋</span> All News
                </button>

                {feeds.map(f => (
                    <button
                        key={f.category}
                        onClick={() => handleCategoryClick(f.category)}
                        style={{
                            display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                            padding: '10px 16px', border: 'none', cursor: 'pointer',
                            background: selectedCategory === f.category ? 'var(--accent-blue)' : 'transparent',
                            color: selectedCategory === f.category ? '#fff' : 'var(--text-secondary)',
                            fontSize: 14, textAlign: 'left', width: '100%',
                            transition: 'background 0.15s',
                        }}
                    >
                        <span>{CATEGORY_ICONS[f.category] || '📄'} {f.category.charAt(0).toUpperCase() + f.category.slice(1)}</span>
                        <span style={{ fontSize: 11, opacity: 0.6 }}>{f.article_count}</span>
                    </button>
                ))}

                <div style={{ flex: 1 }} />

                <button
                    onClick={handleRefresh}
                    disabled={refreshing}
                    style={{
                        margin: '8px 16px', padding: '8px', border: '1px solid var(--border-primary)',
                        background: 'transparent', color: 'var(--text-muted)', cursor: 'pointer',
                        borderRadius: 6, fontSize: 12,
                    }}
                >
                    {refreshing ? '⟳ Refreshing...' : '⟳ Refresh Feeds'}
                </button>
            </aside>

            {/* Main content */}
            <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
                {/* Search bar */}
                <div style={{
                    padding: '12px 20px', borderBottom: '1px solid var(--border-primary)',
                    display: 'flex', gap: 8, background: 'var(--bg-card)',
                }}>
                    <input
                        value={searchQuery}
                        onChange={e => setSearchQuery(e.target.value)}
                        onKeyDown={e => e.key === 'Enter' && handleSearch()}
                        placeholder="Search articles..."
                        style={{
                            flex: 1, padding: '8px 12px', background: 'var(--bg-secondary)',
                            border: '1px solid var(--border-primary)', borderRadius: 6,
                            color: 'var(--text-primary)', fontSize: 14, outline: 'none',
                        }}
                    />
                    <button
                        onClick={handleSearch}
                        style={{
                            padding: '8px 16px', background: 'var(--accent-blue)', border: 'none',
                            color: '#fff', borderRadius: 6, cursor: 'pointer', fontSize: 13,
                        }}
                    >🔍</button>
                </div>

                {/* Article list / reader */}
                <div style={{ flex: 1, overflow: 'auto', padding: '0' }}>
                    {loading ? (
                        <div style={{ textAlign: 'center', padding: 60, color: 'var(--text-muted)' }}>Loading articles...</div>
                    ) : articles.length === 0 ? (
                        <div style={{ textAlign: 'center', padding: 60, color: 'var(--text-muted)' }}>
                            <div style={{ fontSize: 48, marginBottom: 12 }}>📰</div>
                            <div>No articles found. RSS server may be starting up — try refreshing.</div>
                        </div>
                    ) : selectedArticle ? (
                        /* Article reader view */
                        <div style={{ maxWidth: 720, margin: '0 auto', padding: '24px 20px' }}>
                            <button
                                onClick={() => setSelectedArticle(null)}
                                style={{
                                    background: 'transparent', border: 'none', color: 'var(--accent-blue)',
                                    cursor: 'pointer', fontSize: 14, marginBottom: 16, padding: 0,
                                }}
                            >← Back to articles</button>
                            <h2 style={{ fontSize: 22, fontWeight: 700, color: 'var(--text-primary)', marginBottom: 8, lineHeight: 1.3 }}>
                                {selectedArticle.title}
                            </h2>
                            <div style={{ display: 'flex', gap: 12, fontSize: 12, color: 'var(--text-muted)', marginBottom: 16 }}>
                                <span>{selectedArticle.source}</span>
                                <span>•</span>
                                <span>{timeAgo(selectedArticle.published)}</span>
                                <span>•</span>
                                <span style={{ textTransform: 'capitalize' }}>{selectedArticle.category}</span>
                            </div>
                            <p style={{ fontSize: 15, lineHeight: 1.7, color: 'var(--text-secondary)', marginBottom: 20 }}>
                                {selectedArticle.summary}
                            </p>
                            <a
                                href={selectedArticle.link}
                                target="_blank"
                                rel="noopener noreferrer"
                                style={{
                                    display: 'inline-block', padding: '10px 20px',
                                    background: 'var(--accent-blue)', color: '#fff', borderRadius: 6,
                                    textDecoration: 'none', fontSize: 14,
                                }}
                            >Read Full Article →</a>
                        </div>
                    ) : (
                        /* Article list */
                        <div>
                            {articles.map(art => (
                                <div
                                    key={art.id}
                                    onClick={() => setSelectedArticle(art)}
                                    style={{
                                        padding: '14px 20px', cursor: 'pointer',
                                        borderBottom: '1px solid var(--border-primary)',
                                        transition: 'background 0.15s',
                                    }}
                                    onMouseEnter={e => (e.currentTarget.style.background = 'var(--bg-secondary)')}
                                    onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                                >
                                    <div style={{ display: 'flex', alignItems: 'flex-start', gap: 12 }}>
                                        <div style={{ flex: 1 }}>
                                            <h3 style={{
                                                fontSize: 15, fontWeight: 600, color: 'var(--text-primary)',
                                                marginBottom: 4, lineHeight: 1.3,
                                            }}>{art.title}</h3>
                                            <p style={{
                                                fontSize: 13, color: 'var(--text-muted)', lineHeight: 1.4,
                                                overflow: 'hidden', textOverflow: 'ellipsis',
                                                display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical',
                                            }}>{art.summary}</p>
                                        </div>
                                    </div>
                                    <div style={{ display: 'flex', gap: 12, marginTop: 6, fontSize: 11, color: 'var(--text-muted)' }}>
                                        <span style={{ fontWeight: 500 }}>{art.source}</span>
                                        <span>{timeAgo(art.published)}</span>
                                        <span style={{
                                            padding: '1px 6px', borderRadius: 4,
                                            background: 'rgba(100,100,255,0.15)', color: 'var(--accent-blue)',
                                            textTransform: 'capitalize', fontSize: 10,
                                        }}>{art.category}</span>
                                    </div>
                                </div>
                            ))}

                            {/* Pagination */}
                            {totalPages > 1 && (
                                <div style={{
                                    display: 'flex', justifyContent: 'center', gap: 8,
                                    padding: '16px', borderTop: '1px solid var(--border-primary)',
                                }}>
                                    <button
                                        onClick={() => loadArticles(selectedCategory, page - 1)}
                                        disabled={page <= 1}
                                        style={{
                                            padding: '6px 12px', border: '1px solid var(--border-primary)',
                                            background: 'transparent', color: 'var(--text-secondary)',
                                            borderRadius: 4, cursor: page > 1 ? 'pointer' : 'default',
                                            opacity: page > 1 ? 1 : 0.4,
                                        }}
                                    >← Prev</button>
                                    <span style={{ padding: '6px 12px', color: 'var(--text-muted)', fontSize: 13 }}>
                                        Page {page} of {totalPages}
                                    </span>
                                    <button
                                        onClick={() => loadArticles(selectedCategory, page + 1)}
                                        disabled={page >= totalPages}
                                        style={{
                                            padding: '6px 12px', border: '1px solid var(--border-primary)',
                                            background: 'transparent', color: 'var(--text-secondary)',
                                            borderRadius: 4, cursor: page < totalPages ? 'pointer' : 'default',
                                            opacity: page < totalPages ? 1 : 0.4,
                                        }}
                                    >Next →</button>
                                </div>
                            )}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}
