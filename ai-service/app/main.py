
from fastapi import FastAPI

from app.api.tickets import router as tickets_router


app = FastAPI(
    title="ResolveFlow AI Service",
    description="AI-powered ticket intelligence for ResolveFlow",
    version="0.1.0",
)


@app.get("/health")
def health_check():
    return {
        "status": "healthy",
        "service": "resolveflow-ai-service",
    }


app.include_router(
    tickets_router,
    prefix="/api/v1",
)
