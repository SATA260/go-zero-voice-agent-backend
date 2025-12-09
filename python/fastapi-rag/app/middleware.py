"""自定义 FastAPI 中间件实现。"""

from fastapi import Request
from fastapi.responses import JSONResponse


async def security_middleware(request: Request, call_next):
    """校验请求头中是否包含用户身份标识。"""
    userId = request.headers.get("X-User-Id")
    if not userId:
        return JSONResponse(
            status_code=401,
            content={"detail": "X-User-Id header missing"},
        )
    response = await call_next(request)
    return response