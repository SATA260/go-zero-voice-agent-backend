from enum import Enum

class PGVector(str, Enum):
    TABLE_NAME = "langchain_pg_embedding"
    COLUMN_NAME = "custom_id"
    INDEX_NAME = f"idx_{TABLE_NAME}_{COLUMN_NAME}"