"""
WebRTC P2P 最小サンプル - Step 1: STUNなし
シグナリングサーバー（FastAPI + aiortc）
"""

import json
import asyncio
from fastapi import FastAPI, WebSocket
from fastapi.responses import FileResponse
from aiortc import RTCPeerConnection, RTCSessionDescription

app = FastAPI()


@app.get("/")
async def index():
    """index.html を返す"""
    return FileResponse("index.html")


@app.websocket("/ws")
async def websocket_endpoint(ws: WebSocket):
    await ws.accept()
    print("WebSocket connected")

    pc = RTCPeerConnection()

    # DataChannel を受け取る側（ブラウザが作成したチャンネルを受信）
    @pc.on("datachannel")
    def on_datachannel(channel):
        print(f"DataChannel received: {channel.label}")

        @channel.on("open")
        def on_open():
            print("DataChannel open")

        @channel.on("message")
        def on_message(message):
            print(f"Received: {message}")
            # エコーバック
            channel.send(f"Echo: {message}")

    # ICE candidate の状態変化をログ
    @pc.on("iceconnectionstatechange")
    async def on_ice_state_change():
        print(f"ICE state: {pc.iceConnectionState}")

    # === signaling ===
    async def send(msg):
        await ws.send_text(json.dumps(msg))

    async def recv():
        return json.loads(await ws.receive_text())

    # Offer を受け取り → Answer を返す
    offer = await recv()
    print(f"Received offer")
    await pc.setRemoteDescription(RTCSessionDescription(**offer))

    answer = await pc.createAnswer()
    await pc.setLocalDescription(answer)
    print(f"Sending answer")

    await send({
        "sdp": pc.localDescription.sdp,
        "type": pc.localDescription.type
    })

    # keep alive
    try:
        while True:
            await asyncio.sleep(1)
    except Exception as e:
        print(f"Connection closed: {e}")
    finally:
        await pc.close()
        print("PeerConnection closed")
