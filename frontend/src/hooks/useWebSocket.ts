import { useEffect, useRef, useState, useCallback } from 'react'

interface WebSocketMessage {
  type: string
  game_id: string
  payload: any
}

export function useWebSocket(gameId: string, playerId: string) {
  const wsRef = useRef<WebSocket | null>(null)
  const [isConnected, setIsConnected] = useState(false)
  const [lastMessage, setLastMessage] = useState<WebSocketMessage | null>(null)
  const messageHandlersRef = useRef<Map<string, (payload: any) => void>>(new Map())

  const sendMessage = useCallback((type: string, payload: any) => {
    console.log(`[useWebSocket] sendMessage called with type: ${type}, payload:`, payload)
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) {
      const message = {
        type,
        game_id: gameId,
        payload,
      }
      console.log(`[useWebSocket] Sending message:`, JSON.stringify(message))
      wsRef.current.send(JSON.stringify(message))
    } else {
      console.warn(`[useWebSocket] WebSocket not ready. State:`, wsRef.current?.readyState)
    }
  }, [gameId])

  const onMessage = useCallback((messageType: string, handler: (payload: any) => void) => {
    messageHandlersRef.current.set(messageType, handler)
  }, [])

  useEffect(() => {
    if (!gameId || !playerId) return

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const backendHost = import.meta.env.VITE_BACKEND_URL || 'localhost:8080'
    const ws = new WebSocket(
      `${protocol}//${backendHost}/ws?game_id=${gameId}&player_id=${playerId}`
    )

    // Set a connection timeout - if not connected within 10 seconds, close it
    const connectionTimeout = setTimeout(() => {
      if (!ws || ws.readyState !== WebSocket.OPEN) {
        console.warn('WebSocket connection timeout - game may be expired')
        ws?.close()
      }
    }, 10000)

    ws.onopen = () => {
      clearTimeout(connectionTimeout)
      console.log('WebSocket connected')
      setIsConnected(true)
    }

    ws.onmessage = (event) => {
      try {
        const message: WebSocketMessage = JSON.parse(event.data)
        setLastMessage(message)

        const handler = messageHandlersRef.current.get(message.type)
        if (handler) {
          handler(message.payload)
        }
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error)
      }
    }

    ws.onerror = (error) => {
      clearTimeout(connectionTimeout)
      console.error('WebSocket error:', error)
    }

    ws.onclose = () => {
      clearTimeout(connectionTimeout)
      console.log('WebSocket disconnected')
      setIsConnected(false)
    }

    wsRef.current = ws

    return () => {
      clearTimeout(connectionTimeout)
      if (wsRef.current) {
        wsRef.current.close()
      }
    }
  }, [gameId, playerId])

  return {
    isConnected,
    lastMessage,
    sendMessage,
    onMessage,
  }
}
