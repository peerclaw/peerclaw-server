import { useState, useEffect, useRef } from "react"
import { useTranslation } from "react-i18next"
import { useParams, useNavigate } from "react-router-dom"
import { MessageSquare, Trash2, Zap } from "lucide-react"
import { usePlayground } from "@/hooks/use-playground"
import { AgentSelector } from "@/components/playground/AgentSelector"
import { ChatMessage } from "@/components/playground/ChatMessage"
import { ChatInput } from "@/components/playground/ChatInput"
import { RawToggle } from "@/components/playground/RawToggle"

export function PlaygroundPage() {
  const { t } = useTranslation()
  const { agentId: paramAgentId } = useParams<{ agentId?: string }>()
  const navigate = useNavigate()

  const [selectedAgentId, setSelectedAgentId] = useState<string | null>(
    paramAgentId ?? null
  )
  const [streamEnabled, setStreamEnabled] = useState(false)
  const [showRaw, setShowRaw] = useState(false)

  const { messages, loading, error, lastRaw, sendMessage, clearMessages } =
    usePlayground(selectedAgentId)

  const messagesEndRef = useRef<HTMLDivElement>(null)

  // Sync URL param to state
  useEffect(() => {
    if (paramAgentId && paramAgentId !== selectedAgentId) {
      setSelectedAgentId(paramAgentId)
    }
  }, [paramAgentId])

  // Auto-scroll on new messages
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" })
  }, [messages])

  const handleSelectAgent = (id: string) => {
    setSelectedAgentId(id)
    navigate(`/playground/${id}`, { replace: true })
  }

  const handleSend = (message: string) => {
    if (!selectedAgentId) return
    sendMessage(selectedAgentId, message, streamEnabled)
  }

  const handleClear = () => {
    clearMessages()
    setShowRaw(false)
  }

  return (
    <div className="mx-auto flex h-[calc(100vh-3.5rem)] max-w-6xl">
      {/* Left panel - Agent selector & settings */}
      <div className="flex w-72 shrink-0 flex-col border-r border-border p-4">
        <div className="mb-4 flex items-center gap-2">
          <MessageSquare className="size-5 text-primary" />
          <h1 className="text-lg font-semibold">{t('playground.title')}</h1>
        </div>

        {/* Agent selector */}
        <div className="mb-4">
          <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
            {t('playground.agent')}
          </label>
          <AgentSelector
            selectedId={selectedAgentId}
            onSelect={handleSelectAgent}
          />
        </div>

        {/* Stream toggle */}
        <div className="mb-4">
          <label className="flex items-center gap-2 text-sm text-muted-foreground">
            <button
              type="button"
              role="switch"
              aria-checked={streamEnabled}
              onClick={() => setStreamEnabled((s) => !s)}
              className={`relative inline-flex h-5 w-9 shrink-0 cursor-pointer items-center rounded-full transition-colors ${
                streamEnabled ? "bg-primary" : "bg-muted"
              }`}
            >
              <span
                className={`inline-block size-3.5 rounded-full bg-white shadow-sm transition-transform ${
                  streamEnabled ? "translate-x-[18px]" : "translate-x-[3px]"
                }`}
              />
            </button>
            <Zap className="size-3.5" />
            {t('playground.stream')}
          </label>
          <p className="mt-1 text-[10px] text-muted-foreground/70">
            {t('playground.streamDesc')}
          </p>
        </div>

        {/* Clear button */}
        <button
          onClick={handleClear}
          disabled={messages.length === 0}
          className="mt-auto flex items-center justify-center gap-2 rounded-lg border border-border px-3 py-2 text-xs font-medium text-muted-foreground transition-colors hover:bg-accent hover:text-foreground disabled:pointer-events-none disabled:opacity-50"
        >
          <Trash2 className="size-3.5" />
          {t('playground.clearConversation')}
        </button>
      </div>

      {/* Right panel - Chat */}
      <div className="flex flex-1 flex-col">
        {/* Messages area */}
        <div className="flex-1 overflow-y-auto p-4">
          {messages.length === 0 ? (
            <div className="flex h-full flex-col items-center justify-center text-muted-foreground">
              <MessageSquare className="mb-3 size-10 opacity-30" />
              <p className="text-sm font-medium">{t('playground.noMessages')}</p>
              <p className="mt-1 text-xs">
                {selectedAgentId
                  ? t('playground.sendToStart')
                  : t('playground.selectAgentFirst')}
              </p>
            </div>
          ) : (
            <div className="space-y-4">
              {messages.map((msg) => (
                <ChatMessage
                  key={msg.id}
                  role={msg.role}
                  content={msg.content}
                  timestamp={msg.timestamp}
                />
              ))}

              {/* Loading indicator */}
              {loading && (
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <div className="flex gap-1">
                    <span className="size-1.5 animate-bounce rounded-full bg-muted-foreground [animation-delay:0ms]" />
                    <span className="size-1.5 animate-bounce rounded-full bg-muted-foreground [animation-delay:150ms]" />
                    <span className="size-1.5 animate-bounce rounded-full bg-muted-foreground [animation-delay:300ms]" />
                  </div>
                  {t('playground.thinking')}
                </div>
              )}

              <div ref={messagesEndRef} />
            </div>
          )}
        </div>

        {/* Error bar */}
        {error && (
          <div className="border-t border-destructive/30 bg-destructive/10 px-4 py-2 text-xs text-destructive">
            {error}
          </div>
        )}

        {/* Raw toggle */}
        <RawToggle
          data={lastRaw}
          show={showRaw}
          onToggle={() => setShowRaw((s) => !s)}
        />

        {/* Input */}
        <div className="border-t border-border p-4">
          <ChatInput
            onSend={handleSend}
            disabled={!selectedAgentId || loading}
            placeholder={
              selectedAgentId
                ? t('playground.typeMessage')
                : t('playground.selectFirst')
            }
          />
        </div>
      </div>
    </div>
  )
}
