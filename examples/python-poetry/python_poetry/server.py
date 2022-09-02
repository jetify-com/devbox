from typing import Dict

import uvicorn
from fastapi import FastAPI

app = FastAPI()


@app.get("/")
async def hello() -> Dict[str, str]:
    return {
        "message": "Hola mundo!",
    }


@app.get("/ping")
async def ping() -> Dict[str, str]:
    return {
        "message": "Pong!",
    }


def run_server():
    uvicorn.run("python_poetry.server:app", host="0.0.0.0", port=8080, reload=True)
