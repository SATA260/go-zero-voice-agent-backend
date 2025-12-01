import asyncpg
from app.config import PG_DSN, logger
from app.constants import PGVector

# PostgreSQL 数据库连接池管理。
class PSQLDatabase:
    pool = None

    @classmethod
    async def get_pool(cls):
        if cls.pool is None:
            cls.pool = await asyncpg.create_pool(dsn=PG_DSN)
        return cls.pool
    
    @classmethod
    async def close_pool(cls):
        if cls.pool is not None:
            await cls.pool.close()
            cls.pool = None

# 确保向量数据库的索引存在。
async def ensure_vector_indexes():
    pool = await PSQLDatabase.get_pool()
    async with pool.acquire() as conn:
        await conn.execute(
            f"""
            CREATE INDEX IF NOT EXISTS {PGVector.INDEX_NAME} 
            ON {PGVector.TABLE_NAME} ({PGVector.COLUMN_NAME}));
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
    try:
        pool = await PSQLDatabase.get_pool()
        async with pool.acquire() as conn:
            await conn.fetchval("SELECT 1")
        return True
    except Exception as e:
        logger.error(f"Health check failed: {e}")
        return False