"""扩展 LangChain 提供的 PGVector 功能，补充监控与便捷方法。"""

import logging
import os
import time
from typing import Optional, Any, Dict, List, Union
from sqlalchemy import event, delete, select, asc, desc, cast
from sqlalchemy.orm import Session
from sqlalchemy.engine import Engine
from sqlalchemy.types import Integer
from langchain_core.documents import Document
from langchain_community.vectorstores.pgvector import PGVector


class ExtendedPgVector(PGVector):
    """在原有 PGVector 基础上新增查询日志、自定义删除等能力。"""
    _query_logging_setup = False

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.setup_query_logging()
    
    @staticmethod
    def _sanitize_parameters_for_logging(
        parameters: Union[Dict, List, tuple, Any]
    ) -> Any:
        """清理日志参数，避免输出高维向量或超长字符串。"""
        if parameters is None:
            return parameters

        if isinstance(parameters, dict):
            sanitized = {}
            for key, value in parameters.items():
                # Check if the key contains 'embedding' or if the value looks like an embedding vector
                if "embedding" in str(key).lower() or (
                    isinstance(value, (list, tuple))
                    and len(value) > 10
                    and all(isinstance(x, (int, float)) for x in value[:10])
                ):
                    sanitized[key] = f"<embedding vector of length {len(value)}>"
                elif isinstance(value, str) and len(value) > 500:
                    sanitized[key] = value[:500] + "... (truncated)"
                elif isinstance(value, (dict, list, tuple)):
                    sanitized[key] = ExtendedPgVector._sanitize_parameters_for_logging(
                        value
                    )
                else:
                    sanitized[key] = value
            return sanitized
        elif isinstance(parameters, (list, tuple)):
            sanitized = []
            # Check if this is a list of embeddings
            if len(parameters) > 0 and all(
                isinstance(item, (list, tuple))
                and len(item) > 10
                and all(isinstance(x, (int, float)) for x in item[: min(10, len(item))])
                for item in parameters
            ):
                return f"<{len(parameters)} embedding vectors>"

            for item in parameters:
                if (
                    isinstance(item, (list, tuple))
                    and len(item) > 10
                    and all(isinstance(x, (int, float)) for x in item[:10])
                ):
                    sanitized.append(f"<embedding vector of length {len(item)}>")
                elif isinstance(item, str) and len(item) > 500:
                    sanitized.append(item[:500] + "... (truncated)")
                elif isinstance(item, (dict, list, tuple)):
                    sanitized.append(
                        ExtendedPgVector._sanitize_parameters_for_logging(item)
                    )
                else:
                    sanitized.append(item)
            return type(parameters)(sanitized)
        else:
            return parameters

    def setup_query_logging(self):
        """按需开启 SQL 执行日志，辅助调试性能问题。"""

        # Only setup logging if the environment variable is set to a truthy value
        debug_queries = os.getenv("DEBUG_PGVECTOR_QUERIES", "").lower()
        if debug_queries not in ["true", "1", "yes", "on"]:
            return

        # Only setup once per class
        if ExtendedPgVector._query_logging_setup:
            return

        logger = logging.getLogger("pgvector.queries")
        logger.setLevel(logging.INFO)

        # Create handler if it doesn't exist
        if not logger.handlers:
            handler = logging.StreamHandler()
            formatter = logging.Formatter("%(asctime)s - PGVECTOR QUERY - %(message)s")
            handler.setFormatter(formatter)
            logger.addHandler(handler)

        @event.listens_for(Engine, "before_cursor_execute")
        def receive_before_cursor_execute(
            conn, cursor, statement, parameters, context, executemany
        ):
            if "langchain_pg_embedding" in statement:
                context._query_start_time = time.time()
                logger.info(f"STARTING QUERY: {statement}")
                sanitized_params = ExtendedPgVector._sanitize_parameters_for_logging(
                    parameters
                )
                logger.info(f"PARAMETERS: {sanitized_params}")

        @event.listens_for(Engine, "after_cursor_execute")
        def receive_after_cursor_execute(
            conn, cursor, statement, parameters, context, executemany
        ):
            if "langchain_pg_embedding" in statement:
                total = time.time() - context._query_start_time
                logger.info(f"COMPLETED QUERY in {total:.4f}s")
                logger.info("-" * 50)

        ExtendedPgVector._query_logging_setup = True

    def get_all_ids(self) -> List[str]:
        """获取所有存储向量的自定义 ID 列表。"""
        with Session(self._bind) as session:
            results = session.execute(self.EmbeddingStore.custom_id).all()
            return [result[0] for result in results if result[0] is not None]
        
    def get_filtered_ids(self, ids: List[str]) -> List[str]:
        """筛选输入列表，仅返回数据库中存在的 ID。"""
        with Session(self._bind) as session:
            query = session.query(self.EmbeddingStore.custom_id).filter(
                self.EmbeddingStore.custom_id.in_(ids)
            )
            results = query.all()
            return [result[0] for result in results if result[0] is not None]
        
    def get_documents_by_ids(self, ids: list[str]) -> list[Document]:
        """根据自定义 ID 列表返回 Document 对象。"""
        with Session(self._bind) as session:
            results = (
                session.query(self.EmbeddingStore)
                .filter(self.EmbeddingStore.custom_id.in_(ids))
                .all()
            )
            return [
                Document(page_content=result.document, metadata=result.cmetadata or {})
                for result in results
                if result.custom_id in ids
            ]
        
    def _delete_multiple(
        self, ids: Optional[list[str]] = None, collection_only: bool = False
    ) -> None:
        """支持按 ID 或集合批量删除嵌入记录。"""
        with Session(self._bind) as session:
            if ids is not None:
                self.logger.debug(
                    "Trying to delete vectors by ids (represented by the model "
                    "using the custom ids field)"
                )
                stmt = delete(self.EmbeddingStore)
                if collection_only:
                    collection = self.get_collection(session)
                    if not collection:
                        self.logger.warning("Collection not found")
                        return
                    stmt = stmt.where(
                        self.EmbeddingStore.collection_id == collection.uuid
                    )
                stmt = stmt.where(self.EmbeddingStore.custom_id.in_(ids))
                session.execute(stmt)
            session.commit()

    def delete_by_file_ids(self, file_ids: List[str]) -> int:
        """根据文件 ID 删除对应的所有向量记录。"""
        if not file_ids:
            return 0

        with Session(self._bind) as session:
            stmt = delete(self.EmbeddingStore).where(
                self.EmbeddingStore.cmetadata["file_id"].astext.in_(file_ids)
            )
            result = session.execute(stmt)
            session.commit()
            return result.rowcount if result is not None else 0

    def get_chunk_digests_by_file_id(self, file_id: str) -> Dict[str, str]:
        """返回指定文件 ID 下的自定义 ID 与分片哈希映射。"""
        if not file_id:
            return {}

        with Session(self._bind) as session:
            stmt = select(
                self.EmbeddingStore.custom_id,
                self.EmbeddingStore.cmetadata,
            ).where(self.EmbeddingStore.cmetadata["file_id"].astext == file_id)
            rows = session.execute(stmt).all()

            digests: Dict[str, str] = {}
            for custom_id, metadata in rows:
                if not custom_id or not isinstance(metadata, dict):
                    continue
                digest = metadata.get("chunk_digest")
                if isinstance(digest, str) and digest:
                    digests[custom_id] = digest
            return digests

    def get_chunks_paginated(
        self,
        *,
        page: int,
        page_size: int,
        file_id: Optional[str] = None,
        entity_id: Optional[str] = None,
        user_id: Optional[str] = None,
        order_by: str = "chunk_index",
        sort: str = "asc",
    ) -> Dict[str, Any]:
        """分页查询存储的文本切片。

        支持按 file_id / entity_id / user_id 过滤，默认按 chunk_index 升序。
        返回 items 与 total，便于前端分页展示。
        """

        safe_page = max(page, 1)
        safe_size = max(min(page_size, 200), 1)
        offset = (safe_page - 1) * safe_size

        order_key = (order_by or "chunk_index").lower()
        sort_dir = desc if str(sort).lower() == "desc" else asc

        with Session(self._bind) as session:
            query = session.query(self.EmbeddingStore)

            if file_id:
                query = query.filter(
                    self.EmbeddingStore.cmetadata["file_id"].astext == file_id
                )
            if entity_id:
                query = query.filter(
                    self.EmbeddingStore.cmetadata["entity_id"].astext == entity_id
                )
            if user_id:
                query = query.filter(
                    self.EmbeddingStore.cmetadata["user_id"].astext == user_id
                )

            total = query.count()

            order_map = {
                "id": self.EmbeddingStore.id
                if hasattr(self.EmbeddingStore, "id")
                else None,
                "custom_id": self.EmbeddingStore.custom_id,
                "chunk_index": cast(
                    self.EmbeddingStore.cmetadata["chunk_index"].astext, Integer
                ),
                "created_at": getattr(self.EmbeddingStore, "created_at", None),
            }

            order_col = order_map.get(order_key)
            if order_col is None:
                order_col = self.EmbeddingStore.custom_id

            rows = (
                query.order_by(sort_dir(order_col))
                .offset(offset)
                .limit(safe_size)
                .all()
            )

            items: List[Dict[str, Any]] = []
            for row in rows:
                items.append(
                    {
                        "custom_id": getattr(row, "custom_id", None),
                        "page_content": getattr(row, "document", None),
                        "metadata": getattr(row, "cmetadata", {}) or {},
                    }
                )

            return {
                "items": items,
                "total": total,
                "page": safe_page,
                "page_size": safe_size,
            }