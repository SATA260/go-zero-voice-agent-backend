import asyncio
import hashlib
import logging
from pathlib import PurePosixPath
from typing import Any, Dict, List, Optional

from fastapi import HTTPException
from langchain_core.documents import Document
from langchain_core.embeddings import Embeddings
from langchain_text_splitters import RecursiveCharacterTextSplitter
from minio import Minio
from minio.error import S3Error

from app.db.vector_store.async_pg_vector import AsyncPgVector
from app.models.document import (
    CleanupMethod,
    DocumentResponse,
    QueryMultipleBody,
    QueryRequestBody,
)


_vector_store: AsyncPgVector | None = None
_embeddings: Embeddings | None = None
_minio_client: Minio | None = None
_chunk_size: int = 1500
_chunk_overlap: int = 100
_logger: logging.Logger = logging.getLogger("app.services.document")


def configure_document_service(
    *,
    vector_store: AsyncPgVector,
    embeddings: Embeddings,
    minio_client: Minio,
    chunk_size: int,
    chunk_overlap: int,
    logger: logging.Logger,
) -> None:
    """在应用启动阶段注入文档服务所需的运行时依赖。

    通过集中配置向量库、向量嵌入模型、Minio 客户端以及分片参数，
    便于后续的业务函数直接复用这些对象而无需重复传参。
    """
    global _vector_store, _embeddings, _minio_client, _chunk_size, _chunk_overlap, _logger
    _vector_store = vector_store
    _embeddings = embeddings
    _minio_client = minio_client
    _chunk_size = chunk_size
    _chunk_overlap = chunk_overlap
    _logger = logger


def _require_vector_store() -> AsyncPgVector:
    """返回已配置的向量库实例，未配置时抛出 500 错误。"""
    if _vector_store is None:
        raise HTTPException(status_code=500, detail="Vector store not configured")
    return _vector_store


def _require_embeddings() -> Embeddings:
    """返回已配置的嵌入模型实例，未配置时抛出 500 错误。"""
    if _embeddings is None:
        raise HTTPException(status_code=500, detail="Embeddings not configured")
    return _embeddings


async def _stat_object(bucket_name: str, object_path: str):
    """获取 Minio 对象的元数据，用于记录文件大小与时间戳等信息。"""
    if _minio_client is None:
        raise HTTPException(status_code=500, detail="Minio client not configured")
    try:
        return await asyncio.to_thread(
            _minio_client.stat_object,
            bucket_name,
            object_path,
        )
    except S3Error as exc:
        if exc.code == "NoSuchKey":
            raise HTTPException(status_code=404, detail="Object not found in Minio")
        raise HTTPException(status_code=502, detail=f"Minio stat error: {exc.code}")


async def _download_object(bucket_name: str, object_path: str) -> bytes:
    """从 Minio 下载目标对象的二进制内容。"""
    if _minio_client is None:
        raise HTTPException(status_code=500, detail="Minio client not configured")
    try:
        response = await asyncio.to_thread(
            _minio_client.get_object,
            bucket_name,
            object_path,
        )
    except S3Error as exc:
        if exc.code == "NoSuchKey":
            raise HTTPException(status_code=404, detail="Object not found in Minio")
        raise HTTPException(status_code=502, detail=f"Minio get error: {exc.code}")

    try:
        data = await asyncio.to_thread(response.read)
        return data
    finally:
        response.close()
        response.release_conn()


def _bytes_to_text(data: bytes) -> str:
    """尝试以 UTF-8 解码字节串，必要时容错非法字符。"""
    if not data:
        return ""
    try:
        return data.decode("utf-8")
    except UnicodeDecodeError:
        return data.decode("utf-8", errors="ignore")


def _build_metadata(
    *,
    file_id: str,
    bucket_name: str,
    object_path: str,
    filename: str,
    content_type: Optional[str],
    entity_id: Optional[str],
    user_id: str,
    chunk_index: int,
    chunk_digest: str,
    vector_id: str,
    size_bytes: int,
    last_modified: Optional[str],
) -> Dict[str, Any]:
    """构建写入向量存储的元数据，用于描述每个文本分片。"""
    return {
        "file_id": file_id,
        "bucket_name": bucket_name,
        "object_path": object_path,
        "filename": filename,
        "file_content_type": content_type,
        "entity_id": entity_id,
        "user_id": user_id,
        "chunk_index": chunk_index,
        "chunk_digest": chunk_digest,
        "vector_id": vector_id,
        "size_bytes": size_bytes,
        "last_modified": last_modified,
        "source": f"minio://{bucket_name}/{object_path}",
    }


async def embed_file(
    *,
    file_id: str,
    bucket_name: str,
    object_path: str,
    filename: Optional[str],
    content_type: Optional[str],
    entity_id: Optional[str],
    user_id: str,
    cleanup_method: CleanupMethod,
    executor=None,
) -> Dict[str, Any]:
    """从 Minio 拉取文件、完成文本切片并写入向量库。"""
    store = _require_vector_store()

    object_info = await _stat_object(bucket_name, object_path)
    raw_bytes = await _download_object(bucket_name, object_path)
    text = _bytes_to_text(raw_bytes)
    if not text.strip():
        raise HTTPException(status_code=400, detail="Object content is empty")

    display_name = filename or PurePosixPath(object_path).name
    _logger.info(
        "Embedding Minio object",
        extra={
            "file_id": file_id,
            "bucket": bucket_name,
            "object_path": object_path,
            "cleanup_method": cleanup_method.value,
        },
    )

    if cleanup_method == CleanupMethod.full:
        deleted = await store.adelete_by_file_ids([file_id], executor=executor)
        _logger.debug(
            "Removed previous vectors",
            extra={"file_id": file_id, "deleted": deleted},
        )
        known_digests: set[str] = set()
    else:
        existing = await store.aget_chunk_digests_by_file_id(
            file_id,
            executor=executor,
        )
        known_digests = set(existing.values())

    splitter = RecursiveCharacterTextSplitter(
        chunk_size=_chunk_size,
        chunk_overlap=_chunk_overlap,
    )
    chunks = splitter.split_text(text)
    if not chunks:
        raise HTTPException(status_code=400, detail="No chunks generated from object")

    documents: List[Document] = []
    doc_ids: List[str] = []
    skipped = 0

    for index, chunk in enumerate(chunks):
        normalized = chunk.strip()
        if not normalized:
            continue
        digest = hashlib.md5(normalized.encode("utf-8")).hexdigest()
        if digest in known_digests:
            skipped += 1
            continue
        known_digests.add(digest)
        vector_id = f"{file_id}:{digest}"
        metadata = _build_metadata(
            file_id=file_id,
            bucket_name=bucket_name,
            object_path=object_path,
            filename=display_name,
            content_type=content_type,
            entity_id=entity_id,
            user_id=user_id,
            chunk_index=index,
            chunk_digest=digest,
            vector_id=vector_id,
            size_bytes=object_info.size,
            last_modified=object_info.last_modified.isoformat()
            if object_info.last_modified
            else None,
        )
        documents.append(Document(page_content=normalized, metadata=metadata))
        doc_ids.append(vector_id)

    if not documents:
        return {
            "file_id": file_id,
            "embedded_chunks": 0,
            "skipped_chunks": skipped,
            "message": "No new chunks to embed",
        }

    stored = await store.aadd_documents(
        documents,
        ids=doc_ids,
        executor=executor,
    )

    return {
        "file_id": file_id,
        "embedded_chunks": len(stored),
        "skipped_chunks": skipped,
        "vector_ids": stored,
    }


async def get_documents_by_ids(
    *,
    ids: List[str],
    executor=None,
) -> Dict[str, Any]:
    """根据向量自定义 ID 列表返回存储的文档内容及元数据。"""
    store = _require_vector_store()
    if not ids:
        raise HTTPException(status_code=400, detail="ids list cannot be empty")

    documents = await store.get_documents_by_ids(ids, executor=executor)
    return {
        "documents": [
            DocumentResponse(
                page_content=doc.page_content,
                metadata=doc.metadata,
            ).dict()
            for doc in documents
        ]
    }


async def delete_documents(
    *,
    ids: List[str],
    executor=None,
) -> Dict[str, Any]:
    """按自定义 ID 删除 pgvector 中的文档记录。"""
    store = _require_vector_store()
    if not ids:
        raise HTTPException(status_code=400, detail="ids list cannot be empty")

    existing = await store.get_filtered_ids(ids, executor=executor)
    await store.delete(ids=existing, executor=executor)
    return {"deleted_count": len(existing)}


async def list_chunks(
    *,
    page: int,
    page_size: int,
    file_id: Optional[str] = None,
    entity_id: Optional[str] = None,
    user_id: Optional[str] = None,
    order_by: str = "chunk_index",
    sort: str = "asc",
    executor=None,
) -> Dict[str, Any]:
    """分页查询 pgvector 中的文本切片记录。"""
    store = _require_vector_store()
    return await store.aget_chunks_paginated(
        page=page,
        page_size=page_size,
        file_id=file_id,
        entity_id=entity_id,
        user_id=user_id,
        order_by=order_by,
        sort=sort,
        executor=executor,
    )


def _build_filter(file_id: Optional[str], entity_id: Optional[str]) -> Optional[Dict[str, Any]]:
    """构建 pgvector 相似度检索可识别的过滤条件。"""
    payload: Dict[str, Any] = {}
    if file_id:
        payload["file_id"] = file_id
    if entity_id:
        payload["entity_id"] = entity_id
    return payload or None


async def query_documents(
    *,
    body: QueryRequestBody,
    executor=None,
) -> Dict[str, Any]:
    """对查询语句进行向量化，并在指定文件范围内返回最相似的结果。"""
    store = _require_vector_store()
    embeddings = _require_embeddings()
    if not body.query.strip():
        raise HTTPException(status_code=400, detail="Query cannot be empty")

    embedding_vector = await embeddings.aembed_query(body.query)
    results = await store.asimilarity_search_with_score_by_vector(
        embedding_vector,
        k=body.top_k,
        filter=_build_filter(body.file_id, body.entity_id),
        executor=executor,
    )

    return {
        "results": [
            {
                "page_content": document.page_content,
                "metadata": document.metadata,
                "score": score,
            }
            for document, score in results
        ]
    }


async def query_multiple_documents(
    *,
    body: QueryMultipleBody,
    executor=None,
) -> Dict[str, Any]:
    """向量化查询语句，并在多个文件范围内检索最相似内容。"""
    store = _require_vector_store()
    embeddings = _require_embeddings()
    if not body.query.strip():
        raise HTTPException(status_code=400, detail="Query cannot be empty")
    if not body.file_ids:
        raise HTTPException(status_code=400, detail="file_ids cannot be empty")

    embedding_vector = await embeddings.aembed_query(body.query)
    results = await store.asimilarity_search_with_score_by_vector(
        embedding_vector,
        k=body.top_k,
        filter={"file_id": {"$in": body.file_ids}},
        executor=executor,
    )

    return {
        "results": [
            {
                "page_content": document.page_content,
                "metadata": document.metadata,
                "score": score,
            }
            for document, score in results
        ]
    }
