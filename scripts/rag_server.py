#!/usr/bin/env python3
"""
rag_server.py — RAG (Retrieval-Augmented Generation) server for Sovereign Stack.
Runs on Brain Net, port 8093. 100% local — no external APIs.

Upload PDFs/text → chunk → embed with all-MiniLM-L6-v2 → store locally.
Search queries embed and retrieve top-K relevant chunks via cosine similarity.

Endpoints:
  POST /upload     - Upload a document (multipart/form-data)
  GET  /search     - Search chunks by query (?q=...&k=5)
  GET  /documents  - List uploaded documents
  DELETE /document - Delete a document (?name=...)
  GET  /status     - Health check
"""

import argparse
import json
import os
import hashlib
import time
import struct
import tempfile
from http.server import BaseHTTPRequestHandler
from urllib.parse import urlparse, parse_qs
import socketserver
import numpy as np

# Lazy-loaded globals
_tokenizer = None
_session = None
MODEL_DIR = os.path.expanduser("~/models/all-MiniLM-L6-v2")
STORE_DIR = os.path.expanduser("~/rag-store")

# ─── Embedding Engine ───────────────────────────────────────────────────────

def get_model():
    """Lazy-load ONNX model + tokenizer."""
    global _tokenizer, _session
    if _session is None:
        import onnxruntime as ort
        from tokenizers import Tokenizer

        onnx_path = os.path.join(MODEL_DIR, "onnx", "model.onnx")
        if not os.path.exists(onnx_path):
            # Try the root-level model path
            onnx_path = os.path.join(MODEL_DIR, "model.onnx")
        if not os.path.exists(onnx_path):
            raise FileNotFoundError(f"ONNX model not found in {MODEL_DIR}")

        _session = ort.InferenceSession(onnx_path, providers=["CPUExecutionProvider"])
        _tokenizer = Tokenizer.from_file(os.path.join(MODEL_DIR, "tokenizer.json"))
        _tokenizer.enable_truncation(max_length=512)
        _tokenizer.enable_padding(length=512)
        print(f"[rag] Model loaded: {onnx_path}")
    return _tokenizer, _session


def embed_text(text: str) -> list:
    """Embed a single text string → 384-dim vector."""
    tokenizer, session = get_model()
    encoded = tokenizer.encode(text)

    input_ids = np.array([encoded.ids], dtype=np.int64)
    attention_mask = np.array([encoded.attention_mask], dtype=np.int64)
    token_type_ids = np.zeros_like(input_ids, dtype=np.int64)

    outputs = session.run(None, {
        "input_ids": input_ids,
        "attention_mask": attention_mask,
        "token_type_ids": token_type_ids,
    })

    # Mean pooling over token embeddings
    token_embeddings = outputs[0]  # (1, seq_len, 384)
    mask = attention_mask[..., np.newaxis].astype(np.float32)
    summed = (token_embeddings * mask).sum(axis=1)
    counted = mask.sum(axis=1)
    embedding = (summed / counted)[0]

    # Normalize
    norm = np.linalg.norm(embedding)
    if norm > 0:
        embedding = embedding / norm

    return embedding.tolist()


def cosine_similarity(a, b):
    """Cosine similarity between two vectors (already normalized → dot product)."""
    return float(np.dot(a, b))


# ─── Document Processing ────────────────────────────────────────────────────

def chunk_text(text: str, chunk_size: int = 500, overlap: int = 50) -> list:
    """Split text into overlapping chunks by character count."""
    chunks = []
    start = 0
    while start < len(text):
        end = start + chunk_size
        chunk = text[start:end].strip()
        if chunk:
            chunks.append(chunk)
        start += chunk_size - overlap
    return chunks


def extract_text_from_pdf(path: str) -> str:
    """Extract text from a PDF file."""
    import fitz  # PyMuPDF
    doc = fitz.open(path)
    text = ""
    for page in doc:
        text += page.get_text() + "\n"
    doc.close()
    return text


# ─── Storage ────────────────────────────────────────────────────────────────

class DocumentStore:
    """Simple file-based document + embedding store."""

    def __init__(self, store_dir: str):
        self.store_dir = store_dir
        os.makedirs(store_dir, exist_ok=True)
        self.index_path = os.path.join(store_dir, "index.json")
        self.index = self._load_index()

    def _load_index(self) -> dict:
        if os.path.exists(self.index_path):
            with open(self.index_path, "r") as f:
                return json.load(f)
        return {"documents": {}}

    def _save_index(self):
        with open(self.index_path, "w") as f:
            json.dump(self.index, f, indent=2)

    def add_document(self, name: str, text: str) -> dict:
        """Chunk text, embed chunks, store everything."""
        doc_id = hashlib.md5(name.encode()).hexdigest()[:12]
        chunks = chunk_text(text)

        print(f"[rag] Embedding {len(chunks)} chunks from '{name}'...")
        start = time.time()

        embeddings = []
        for i, chunk in enumerate(chunks):
            emb = embed_text(chunk)
            embeddings.append(emb)
            if (i + 1) % 10 == 0:
                print(f"[rag]   {i+1}/{len(chunks)} chunks embedded")

        elapsed = time.time() - start
        print(f"[rag] Done: {len(chunks)} chunks in {elapsed:.1f}s")

        # Save embeddings as numpy binary
        emb_path = os.path.join(self.store_dir, f"{doc_id}.npy")
        np.save(emb_path, np.array(embeddings, dtype=np.float32))

        # Save chunks as JSON
        chunks_path = os.path.join(self.store_dir, f"{doc_id}_chunks.json")
        with open(chunks_path, "w") as f:
            json.dump(chunks, f)

        # Update index
        self.index["documents"][doc_id] = {
            "name": name,
            "num_chunks": len(chunks),
            "added_at": time.strftime("%Y-%m-%d %H:%M:%S"),
            "embed_time_s": round(elapsed, 1),
        }
        self._save_index()

        return {
            "doc_id": doc_id,
            "name": name,
            "chunks": len(chunks),
            "embed_time_ms": int(elapsed * 1000),
        }

    def search(self, query: str, k: int = 5) -> list:
        """Embed query, search all chunks by cosine similarity."""
        if not self.index["documents"]:
            return []

        query_emb = np.array(embed_text(query), dtype=np.float32)
        results = []

        for doc_id, doc_info in self.index["documents"].items():
            emb_path = os.path.join(self.store_dir, f"{doc_id}.npy")
            chunks_path = os.path.join(self.store_dir, f"{doc_id}_chunks.json")

            if not os.path.exists(emb_path) or not os.path.exists(chunks_path):
                continue

            embeddings = np.load(emb_path)
            with open(chunks_path, "r") as f:
                chunks = json.load(f)

            # Cosine similarity (vectors are normalized → dot product)
            similarities = embeddings @ query_emb
            top_indices = np.argsort(similarities)[::-1][:k]

            for idx in top_indices:
                results.append({
                    "document": doc_info["name"],
                    "chunk": chunks[int(idx)],
                    "score": float(similarities[int(idx)]),
                })

        # Sort all results by score, return top-k
        results.sort(key=lambda x: x["score"], reverse=True)
        return results[:k]

    def list_documents(self) -> list:
        return [
            {"doc_id": did, **info}
            for did, info in self.index["documents"].items()
        ]

    def delete_document(self, name: str) -> bool:
        for doc_id, info in list(self.index["documents"].items()):
            if info["name"] == name:
                # Remove files
                for suffix in [".npy", "_chunks.json"]:
                    path = os.path.join(self.store_dir, f"{doc_id}{suffix}")
                    try:
                        os.unlink(path)
                    except OSError:
                        pass
                del self.index["documents"][doc_id]
                self._save_index()
                return True
        return False


# ─── HTTP Server ─────────────────────────────────────────────────────────────

store = None  # initialized in main()


class RAGHandler(BaseHTTPRequestHandler):

    def do_GET(self):
        parsed = urlparse(self.path)
        params = parse_qs(parsed.query)

        if parsed.path == "/status":
            try:
                get_model()
                model_ok = True
            except Exception:
                model_ok = False
            self._send_json({
                "online": True,
                "model_loaded": model_ok,
                "documents": len(store.index["documents"]),
                "model": "all-MiniLM-L6-v2",
            })

        elif parsed.path == "/search":
            query = params.get("q", [""])[0]
            k = int(params.get("k", ["5"])[0])
            if not query:
                self._send_json({"error": "query required (?q=...)"}, 400)
                return
            results = store.search(query, k=k)
            self._send_json({"results": results, "query": query})

        elif parsed.path == "/documents":
            self._send_json({"documents": store.list_documents()})

        else:
            self.send_error(404)

    def do_POST(self):
        if self.path == "/upload":
            self._handle_upload()
        else:
            self.send_error(404)

    def do_DELETE(self):
        parsed = urlparse(self.path)
        params = parse_qs(parsed.query)

        if parsed.path == "/document":
            name = params.get("name", [""])[0]
            if not name:
                self._send_json({"error": "name required"}, 400)
                return
            if store.delete_document(name):
                self._send_json({"deleted": name})
            else:
                self._send_json({"error": f"document '{name}' not found"}, 404)
        else:
            self.send_error(404)

    def _handle_upload(self):
        content_type = self.headers.get("Content-Type", "")
        content_len = int(self.headers.get("Content-Length", 0))

        if "multipart/form-data" in content_type:
            # Parse multipart form data
            boundary = content_type.split("boundary=")[1].strip()
            body = self.rfile.read(content_len)
            parts = body.split(f"--{boundary}".encode())

            filename = None
            file_data = None

            for part in parts:
                if b"Content-Disposition" not in part:
                    continue
                header_end = part.find(b"\r\n\r\n")
                if header_end < 0:
                    continue
                header = part[:header_end].decode("utf-8", errors="ignore")
                data = part[header_end + 4:].rstrip(b"\r\n--")

                if 'name="file"' in header or "filename=" in header:
                    # Extract filename
                    for h in header.split("\r\n"):
                        if "filename=" in h:
                            filename = h.split('filename="')[1].rstrip('"')
                    file_data = data

            if not filename or not file_data:
                self._send_json({"error": "no file found in upload"}, 400)
                return

            # Process file
            try:
                if filename.lower().endswith(".pdf"):
                    with tempfile.NamedTemporaryFile(suffix=".pdf", delete=False) as tmp:
                        tmp.write(file_data)
                        tmp_path = tmp.name
                    text = extract_text_from_pdf(tmp_path)
                    os.unlink(tmp_path)
                else:
                    text = file_data.decode("utf-8", errors="ignore")

                if not text.strip():
                    self._send_json({"error": "no text extracted from file"}, 400)
                    return

                result = store.add_document(filename, text)
                self._send_json(result)

            except Exception as e:
                self._send_json({"error": str(e)}, 500)

        elif "application/json" in content_type:
            # Accept JSON with {name, text}
            body = self.rfile.read(content_len)
            try:
                req = json.loads(body)
            except json.JSONDecodeError:
                self._send_json({"error": "invalid JSON"}, 400)
                return

            name = req.get("name", "")
            text = req.get("text", "")
            if not name or not text:
                self._send_json({"error": "name and text required"}, 400)
                return

            result = store.add_document(name, text)
            self._send_json(result)

        else:
            self._send_json({"error": "Content-Type must be multipart/form-data or application/json"}, 400)

    def _send_json(self, data, status=200):
        body = json.dumps(data).encode("utf-8")
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_OPTIONS(self):
        self.send_response(200)
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type")
        self.end_headers()

    def log_message(self, format, *args):
        print(f"[rag] {args[0]} {args[1]}")


class ReusableTCPServer(socketserver.TCPServer):
    allow_reuse_address = True
    allow_reuse_port = True


def main():
    global store, MODEL_DIR
    parser = argparse.ArgumentParser(description="RAG Server — Document Embedding & Search")
    parser.add_argument("--port", type=int, default=8093)
    parser.add_argument("--model-dir", default=MODEL_DIR)
    parser.add_argument("--store-dir", default=STORE_DIR)
    args = parser.parse_args()

    MODEL_DIR = args.model_dir

    store = DocumentStore(args.store_dir)
    print(f"[rag] Store: {args.store_dir} ({len(store.index['documents'])} documents)")

    server = ReusableTCPServer(("0.0.0.0", args.port), RAGHandler)
    print(f"[rag] Listening on 0.0.0.0:{args.port}")
    print(f"[rag] Model: {args.model_dir}")
    server.serve_forever()


if __name__ == "__main__":
    main()
