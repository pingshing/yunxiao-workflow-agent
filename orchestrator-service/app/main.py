import logging
import threading

from fastapi import FastAPI

from app.config import load_settings
from app.consumer import consume

logging.basicConfig(level=logging.INFO)

app = FastAPI(title="云效工作流助手编排服务")
settings = load_settings()


@app.on_event("startup")
def start_consumer() -> None:
    thread = threading.Thread(target=consume, args=(settings,), daemon=True)
    thread.start()


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {"status": "ok"}
