"""定义项目中使用的常量与枚举类型。"""

from enum import Enum


class PGVector(str, Enum):
    """pgvector 表结构名称及常见索引常量。"""
    TABLE_NAME = "langchain_pg_embedding"
    COLUMN_NAME = "custom_id"
    INDEX_NAME = f"idx_{TABLE_NAME}_{COLUMN_NAME}"