"""FastAPI 入口脚本，负责应用生命周期与中间件配置。"""

import asyncio
from concurrent.futures import ThreadPoolExecutor
from contextlib import asynccontextmanager
import os

from fastapi import FastAPI
from fastapi.exceptions import RequestValidationError
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

from app.routes import document_routes
import uvicorn


from app.config import (
    VectorDBType,
    RAG_HOST,
    RAG_PORT,
    VECTOR_DB_TYPE,
    LogMiddleware,
    logger,
)
from app.db.database import PSQLDatabase, ensure_vector_indexes

@asynccontextmanager
async def lifespan(app: FastAPI):
    """管理应用启动与销毁时需要执行的资源准备与清理工作。"""
    # Startup logic goes here
    # Create bounded thread pool executor based on CPU cores
    max_workers = min(
        int(os.getenv("RAG_THREAD_POOL_SIZE", str(os.cpu_count()))), 8
    )  # Cap at 8
    app.state.thread_pool = ThreadPoolExecutor(
        max_workers=max_workers, thread_name_prefix="rag-worker"
    )
    logger.info(
        f"Initialized thread pool with {max_workers} workers (CPU cores: {os.cpu_count()})"
    )

    loop = asyncio.get_running_loop()
    loop.set_default_executor(app.state.thread_pool)

    if VECTOR_DB_TYPE == VectorDBType.PGVECTOR:
        await PSQLDatabase.get_pool()
        await ensure_vector_indexes()

    yield

    # Cleanup logic
    logger.info("Shutting down thread pool")
    app.state.thread_pool.shutdown(wait=True)
    logger.info("Thread pool shutdown complete")

    if VECTOR_DB_TYPE == VectorDBType.PGVECTOR:
        await PSQLDatabase.close_pool()

app = FastAPI(lifespan=lifespan)

# CORS configuration
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Adjust as needed for your use case
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.add_middleware(LogMiddleware)

app.include_router(document_routes.router)

@app.exception_handler(RequestValidationError)
async def validation_exception_handler(request, exc):
    """拦截请求校验异常并返回详细错误上下文，便于排查问题。"""
    body = await request.body()
    logger.debug(f"Validation error occurred")
    logger.debug(f"Raw request body: {body.decode()}")
    logger.debug(f"Validation errors: {exc.errors()}")
    return JSONResponse(
        status_code=422,
        content={
            "detail": exc.errors(),
            "body": body.decode(),
            "message": "Request validation failed",
        },
    )

if __name__ == "__main__":
    uvicorn.run(app, host=RAG_HOST, port=RAG_PORT, log_config=None)