"""为 pgvector 提供异步友好的封装。"""

from typing import Optional, List, Tuple, Dict, Any
import asyncio
from langchain_core.documents import Document
from langchain_core.runnables.config import run_in_executor
from .extended_pg_vector import ExtendedPgVector


class AsyncPgVector(ExtendedPgVector):
    """基于线程池将同步 pgvector 操作包装为异步接口。"""

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self._thread_pool = None
    
    def _get_thread_pool(self):
        """获取当前事件循环默认线程池以复用执行资源。"""
        if self._thread_pool is None:
            try:
                # Try to get the thread pool from FastAPI app state
                import contextvars
                from fastapi import Request
                # This is a fallback - in practice, we'll pass the executor explicitly
                loop = asyncio.get_running_loop()
                self._thread_pool = getattr(loop, '_default_executor', None)
            except:
                pass
        return self._thread_pool
    
    async def get_all_ids(self, executor=None) -> list[str]:
        """异步返回所有存储的自定义 ID。"""
        executor = executor or self._get_thread_pool()
        return await run_in_executor(executor, super().get_all_ids)
    
    async def get_filtered_ids(self, ids: list[str], executor=None) -> list[str]:
        """异步过滤输入列表，仅返回已存在的自定义 ID。"""
        executor = executor or self._get_thread_pool()
        return await run_in_executor(executor, super().get_filtered_ids, ids)

    async def get_documents_by_ids(self, ids: list[str], executor=None) -> list[Document]:
        """异步获取指定 ID 对应的文档对象。"""
        executor = executor or self._get_thread_pool()
        return await run_in_executor(executor, super().get_documents_by_ids, ids)

    async def delete(
        self, ids: Optional[list[str]] = None, collection_only: bool = False, executor=None
    ) -> None:
        """异步删除指定 ID 或集合下的文档记录。"""
        executor = executor or self._get_thread_pool()
        await run_in_executor(executor, self._delete_multiple, ids, collection_only)
    
    async def asimilarity_search_with_score_by_vector(
        self, 
        embedding: List[float], 
        k: int = 4, 
        filter: Optional[Dict[str, Any]] = None,
        executor=None
    ) -> List[Tuple[Document, float]]:
        """异步执行相似度检索并返回得分。"""
        executor = executor or self._get_thread_pool()
        return await run_in_executor(
            executor, 
            super().similarity_search_with_score_by_vector, 
            embedding, 
            k, 
            filter
        )
    
    async def aadd_documents(
        self, 
        documents: List[Document], 
        ids: Optional[List[str]] = None,
        executor=None,
        **kwargs
    ) -> List[str]:
        """异步写入文档列表并返回生成的自定义 ID。"""
        executor = executor or self._get_thread_pool()
        return await run_in_executor(
            executor, 
            super().add_documents, 
            documents, 
            ids=ids,
            **kwargs
        )

    async def adelete_by_file_ids(
        self,
        file_ids: List[str],
        executor=None,
    ) -> int:
        """按文件 ID 批量删除文档并返回受影响条数。"""
        executor = executor or self._get_thread_pool()
        return await run_in_executor(
            executor,
            super().delete_by_file_ids,
            file_ids,
        )

    async def aget_chunk_digests_by_file_id(
        self,
        file_id: str,
        executor=None,
    ) -> Dict[str, str]:
        """获取指定文件 ID 对应的 chunk 摘要映射。"""
        executor = executor or self._get_thread_pool()
        return await run_in_executor(
            executor,
            super().get_chunk_digests_by_file_id,
            file_id,
        )