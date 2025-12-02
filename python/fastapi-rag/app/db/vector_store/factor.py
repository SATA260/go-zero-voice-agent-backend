"""向量存储工厂方法。"""

from langchain_core.embeddings import Embeddings
from .async_pg_vector import AsyncPgVector

def get_vector_store(
    connection_string: str,
    embeddings: Embeddings,
    collection_name: str,
    mode: str = "async",
):
    """根据运行模式返回对应的 pgvector 封装实例。"""
    if mode == "async":
        return AsyncPgVector(
            connection_string=connection_string,
            embedding_function=embeddings,
            collection_name=collection_name,
        )
    else:
        raise ValueError(f"Unsupported mode: {mode}")