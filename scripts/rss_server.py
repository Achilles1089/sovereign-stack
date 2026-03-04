#!/usr/bin/env python3
"""
rss_server.py — RSS News Aggregator for Sovereign Stack.
Runs on Brain Net, port 8094. Fetches and caches RSS/Atom feeds.

Endpoints:
  GET /feeds          - List all feed categories
  GET /articles       - List articles (?feed=tech&page=1&limit=20)
  GET /search         - Search articles (?q=bitcoin)
  POST /refresh       - Force refresh all feeds
  GET /status         - Health check
"""

import argparse
import json
import os
import time
import threading
import hashlib
from http.server import BaseHTTPRequestHandler
from urllib.parse import urlparse, parse_qs
import socketserver
import feedparser
import re
from html import unescape

# ─── Default feeds ───────────────────────────────────────────────────────────

DEFAULT_FEEDS = {
    "tech": [
        {"name": "Hacker News", "url": "https://hnrss.org/frontpage"},
        {"name": "Ars Technica", "url": "https://feeds.arstechnica.com/arstechnica/index"},
        {"name": "The Verge", "url": "https://www.theverge.com/rss/index.xml"},
        {"name": "TechCrunch", "url": "https://techcrunch.com/feed/"},
    ],
    "crypto": [
        {"name": "CoinDesk", "url": "https://www.coindesk.com/arc/outboundfeeds/rss/"},
        {"name": "Cointelegraph", "url": "https://cointelegraph.com/rss"},
        {"name": "The Block", "url": "https://www.theblock.co/rss.xml"},
    ],
    "general": [
        {"name": "Reuters Top News", "url": "https://feeds.reuters.com/reuters/topNews"},
        {"name": "AP News", "url": "https://rsshub.app/apnews/topics/apf-topnews"},
        {"name": "BBC World", "url": "https://feeds.bbci.co.uk/news/world/rss.xml"},
    ],
    "ai": [
        {"name": "AI News", "url": "https://buttondown.com/ainews/rss"},
        {"name": "MIT Tech Review AI", "url": "https://www.technologyreview.com/feed/"},
    ],
}

# ─── Article cache ───────────────────────────────────────────────────────────

articles_cache = {}  # category -> [articles]
last_refresh = 0
refresh_interval = 1800  # 30 minutes
is_refreshing = False

def strip_html(html):
    """Remove HTML tags from a string."""
    clean = re.sub(r'<[^>]+>', '', html or '')
    return unescape(clean).strip()

def fetch_feeds():
    """Fetch all RSS feeds and update the cache."""
    global articles_cache, last_refresh, is_refreshing
    is_refreshing = True
    print("[rss] Refreshing feeds...")
    start = time.time()

    new_cache = {}
    total = 0

    for category, feeds in DEFAULT_FEEDS.items():
        category_articles = []
        for feed_info in feeds:
            try:
                d = feedparser.parse(feed_info["url"])
                for entry in d.entries[:20]:  # Max 20 per feed
                    article = {
                        "id": hashlib.md5((entry.get("link", "") + entry.get("title", "")).encode()).hexdigest()[:12],
                        "title": entry.get("title", "Untitled"),
                        "link": entry.get("link", ""),
                        "source": feed_info["name"],
                        "category": category,
                        "published": entry.get("published", entry.get("updated", "")),
                        "summary": strip_html(entry.get("summary", entry.get("description", "")))[:300],
                    }
                    category_articles.append(article)
                    total += 1
                print(f"[rss]   {feed_info['name']}: {len(d.entries[:20])} articles")
            except Exception as e:
                print(f"[rss]   {feed_info['name']}: ERROR - {e}")

        # Sort by published date (newest first)
        category_articles.sort(key=lambda a: a["published"], reverse=True)
        new_cache[category] = category_articles

    articles_cache = new_cache
    last_refresh = time.time()
    elapsed = time.time() - start
    is_refreshing = False
    print(f"[rss] Done: {total} articles in {elapsed:.1f}s")


def auto_refresh():
    """Background thread to auto-refresh feeds."""
    while True:
        try:
            if time.time() - last_refresh >= refresh_interval:
                fetch_feeds()
        except Exception as e:
            print(f"[rss] Auto-refresh error: {e}")
        time.sleep(60)


# ─── HTTP Server ─────────────────────────────────────────────────────────────

class RSSHandler(BaseHTTPRequestHandler):

    def do_GET(self):
        parsed = urlparse(self.path)
        params = parse_qs(parsed.query)

        if parsed.path == "/status":
            total = sum(len(v) for v in articles_cache.values())
            self._send_json({
                "online": True,
                "feeds": len(DEFAULT_FEEDS),
                "articles": total,
                "last_refresh": int(last_refresh),
                "refreshing": is_refreshing,
            })

        elif parsed.path == "/feeds":
            feeds = []
            for cat, feed_list in DEFAULT_FEEDS.items():
                feeds.append({
                    "category": cat,
                    "feeds": [f["name"] for f in feed_list],
                    "article_count": len(articles_cache.get(cat, [])),
                })
            self._send_json({"feeds": feeds})

        elif parsed.path == "/articles":
            category = params.get("feed", [""])[0] or params.get("category", [""])[0]
            page = int(params.get("page", ["1"])[0])
            limit = int(params.get("limit", ["20"])[0])

            if category and category in articles_cache:
                arts = articles_cache[category]
            else:
                # Return all articles combined
                arts = []
                for cat_articles in articles_cache.values():
                    arts.extend(cat_articles)
                arts.sort(key=lambda a: a["published"], reverse=True)

            start = (page - 1) * limit
            end = start + limit
            self._send_json({
                "articles": arts[start:end],
                "total": len(arts),
                "page": page,
                "pages": (len(arts) + limit - 1) // limit,
            })

        elif parsed.path == "/search":
            query = params.get("q", [""])[0].lower()
            if not query:
                self._send_json({"error": "query required (?q=...)"}, 400)
                return

            results = []
            for cat_articles in articles_cache.values():
                for art in cat_articles:
                    if query in art["title"].lower() or query in art["summary"].lower():
                        results.append(art)

            results.sort(key=lambda a: a["published"], reverse=True)
            self._send_json({"articles": results[:50], "total": len(results), "query": query})

        else:
            self.send_error(404)

    def do_POST(self):
        if self.path == "/refresh":
            if is_refreshing:
                self._send_json({"message": "already refreshing"})
            else:
                threading.Thread(target=fetch_feeds, daemon=True).start()
                self._send_json({"message": "refresh started"})
        else:
            self.send_error(404)

    def do_OPTIONS(self):
        self.send_response(200)
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type")
        self.end_headers()

    def _send_json(self, data, status=200):
        body = json.dumps(data).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, format, *args):
        print(f"[rss] {args[0]} {args[1]}")


class ReusableTCPServer(socketserver.TCPServer):
    allow_reuse_address = True
    allow_reuse_port = True


def main():
    parser = argparse.ArgumentParser(description="RSS News Server")
    parser.add_argument("--port", type=int, default=8094)
    args = parser.parse_args()

    # Initial fetch
    print(f"[rss] Starting on port {args.port}")
    fetch_feeds()

    # Start auto-refresh thread
    t = threading.Thread(target=auto_refresh, daemon=True)
    t.start()

    server = ReusableTCPServer(("0.0.0.0", args.port), RSSHandler)
    print(f"[rss] Listening on 0.0.0.0:{args.port}")
    server.serve_forever()


if __name__ == "__main__":
    main()
