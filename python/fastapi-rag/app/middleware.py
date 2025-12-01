from fastapi import Request
from fastapi.responses import JSONResponse


async def security_middleware(request: Request, call_next):
    """
    Middleware to handle security-related tasks such as CORS and request validation.
    """
    userId = request.headers.get("X-User-ID")
    if not userId:
        return JSONResponse(
            status_code=401,
            content={"detail": "X-User-ID header missing"},
        )
    response = await call_next(request)
    return response