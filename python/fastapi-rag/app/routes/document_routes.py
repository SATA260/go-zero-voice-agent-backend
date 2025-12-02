"""定义与文档向量相关的 HTTP 路由。"""

from typing import List

from fastapi import APIRouter, Body, Form, HTTPException, Query, Request

from app.db.database import pg_health_check
from app.models.document import CleanupMethod, QueryMultipleBody, QueryRequestBody
from app.services import (
    delete_documents,
    embed_file,
    get_documents_by_ids,
    query_documents,
    query_multiple_documents,
)


router = APIRouter()


def _require_user_id(request: Request) -> str:
    """校验请求头中是否包含用户标识。"""
    user_id = request.headers.get("X-User-ID")
    if not user_id:
        raise HTTPException(status_code=401, detail="X-User-ID header missing")
    return user_id


def _get_executor(request: Request):
    """从 FastAPI 应用状态中获取线程池用于 CPU 密集任务。"""
    return getattr(request.app.state, "thread_pool", None)


@router.get("/health")
async def health_check():
    """检查服务健康状态与数据库连通性。"""
    postgres_ok = await pg_health_check()
    return {"status": "ok" if postgres_ok else "degraded", "postgres": postgres_ok}


@router.get("/documents")
async def fetch_documents(request: Request, ids: List[str] = Query(...)):
    """根据自定义 ID 查询已存储的文档。"""
    _require_user_id(request)
    return await get_documents_by_ids(ids=ids, executor=_get_executor(request))


@router.delete("/documents")
async def remove_documents(request: Request, ids: List[str] = Body(...)):
    """批量删除指定自定义 ID 的文档。"""
    _require_user_id(request)
    return await delete_documents(ids=ids, executor=_get_executor(request))


@router.post("/embed")
async def embed_file_upload(
    request: Request,
    file_id: str = Form(...),
    bucket_name: str = Form(...),
    file_path: str = Form(...),
    filename: str = Form(None),
    file_content_type: str = Form(None),
    entity_id: str = Form(None),
    cleanup_method: CleanupMethod = Form(CleanupMethod.incremental),
):
    """从 Minio 下载文件并完成向量化写入。"""
    user_id = _require_user_id(request)
    return await embed_file(
        file_id=file_id,
        bucket_name=bucket_name,
        object_path=file_path,
        filename=filename,
        content_type=file_content_type,
        entity_id=entity_id,
        user_id=user_id,
        cleanup_method=cleanup_method,
        executor=_get_executor(request),
    )


@router.post("/query")
async def query_document(request: Request, body: QueryRequestBody):
    """在单文件范围内执行向量检索。"""
    _require_user_id(request)
    return await query_documents(body=body, executor=_get_executor(request))


@router.post("/query-multiple")
async def query_documents_multiple(request: Request, body: QueryMultipleBody):
    """在多个文件范围内执行向量检索。"""
    _require_user_id(request)
    return await query_multiple_documents(body=body, executor=_get_executor(request))