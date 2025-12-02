"""PostgreSQL 连接池与健康检查工具方法。"""

import asyncpg
from app.config import PG_DSN, logger
from app.constants import PGVector

# PostgreSQL 数据库连接池管理。
class PSQLDatabase:
    """维护全局 asyncpg 连接池的单例封装。"""
    pool = None

    @classmethod
    async def get_pool(cls):
        """惰性创建并返回 asyncpg 连接池。"""
        if cls.pool is None:
            cls.pool = await asyncpg.create_pool(dsn=PG_DSN)
        return cls.pool
    
    @classmethod
    async def close_pool(cls):
        """关闭并清理连接池资源。"""
        if cls.pool is not None:
            await cls.pool.close()
            cls.pool = None

# 确保向量数据库的索引存在。
async def ensure_vector_indexes():
    """在 pgvector 表上创建常用索引，避免重复创建异常。"""
    pool = await PSQLDatabase.get_pool()
    async with pool.acquire() as conn:
        await conn.execute(
            f"""
            CREATE INDEX IF NOT EXISTS {PGVector.INDEX_NAME} 
            ON {PGVector.TABLE_NAME} ({PGVector.COLUMN_NAME});
            """
        )

        await conn.execute(
            f"""
            CREATE INDEX IF NOT EXISTS idx_{PGVector.TABLE_NAME}_file_id 
            ON {PGVector.TABLE_NAME} ((cmetadata->>'file_id'));
            """
        )

        logger.info("Vector database indexes ensured.")

# 检查 PostgreSQL 数据库的健康状态。
async def pg_health_check():
    """执行轻量级查询检测 PostgreSQL 是否可用。"""
    try:
        pool = await PSQLDatabase.get_pool()
        async with pool.acquire() as conn:
            await conn.fetchval("SELECT 1")
        return True
    except Exception as e:
        logger.error(f"Health check failed: {e}")
        return False