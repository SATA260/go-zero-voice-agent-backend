"""定义 RAG 文档相关的数据模型。"""

from enum import Enum
import hashlib
from pydantic import BaseModel
from typing import Optional


class DocumentResponse(BaseModel):
    """查询响应中返回的文档结构。"""
    page_content: str
    metadata: dict

class DocumentModel(BaseModel):
    """用于接收单段文本及元数据的模型。"""
    page_content: str
    metadata: Optional[dict] = None

    def generate_digest(self):
        """基于内容计算 MD5，用于去重。"""
        hash_obj = hashlib.md5(self.page_content.encode())
        return hash_obj.hexdigest()
    
class StoreDocument(BaseModel):
    """描述原始文件信息的结构。"""
    filepath: str
    filename: str
    file_content_type: str
    file_id: str

class QueryRequestBody(BaseModel):
    """单文件范围内的向量检索请求体。"""
    query: str
    file_id: str
    top_k: int = 4
    entity_id: Optional[str] = None

class CleanupMethod(str, Enum):
    """切片写入时的清理策略。"""
    incremental = "incremental"
    full = "full"

class QueryMultipleBody(BaseModel):
    """多文件检索请求体。"""
    query: str
    file_ids: list[str]
    top_k: int = 4