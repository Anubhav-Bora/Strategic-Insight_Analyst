"use client"
import { useState, useRef, useEffect } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Send, Bot, User, Sparkles, Loader2, MessageSquare, Zap } from "lucide-react"
import { useQuery } from "@tanstack/react-query"
import api from "@/lib/api"
import { useToast } from "@/components/ui/toaster"
import { auth } from "@/lib/firebase"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { motion, AnimatePresence } from "framer-motion"

interface ChatMessage {
  id: string
  type: string
  content: string
  timestamp: string
}

export default function DocumentChat({ documentId }: { documentId: string }) {
  const [message, setMessage] = useState("")
  const [isGenerating, setIsGenerating] = useState(false)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const toast = useToast()

  const { data: chatHistory, refetch } = useQuery<ChatMessage[]>({
    queryKey: ["chatHistory", documentId],
    queryFn: async () => {
      const user = auth.currentUser
      if (!user) throw new Error("Not authenticated")
      const token = await user.getIdToken()
      const { data } = await api.get(`/api/documents/${documentId}/chat/history`, {
        headers: { Authorization: `Bearer ${token}` },
      })
      return data
    },
  })

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" })
  }

  useEffect(() => {
    scrollToBottom()
  }, [chatHistory])

  async function handleSendMessage() {
    if (!message.trim()) return

    setIsGenerating(true)
    try {
      const user = auth.currentUser
      if (!user) throw new Error("Not authenticated")
      const token = await user.getIdToken()
      await api.post(
        `/api/documents/${documentId}/chat`,
        { message },
        { headers: { Authorization: `Bearer ${token}` } },
      )
      setMessage("")
      refetch()
    } catch (error) {
      toast.toast({ title: "Error: " + (error instanceof Error ? error.message : "Unknown error") })
    } finally {
      setIsGenerating(false)
    }
  }

  async function handleGenerateInsight() {
    setIsGenerating(true)
    try {
      const user = auth.currentUser
      if (!user) throw new Error("Not authenticated")
      const token = await user.getIdToken()
      await api.post(
        `/api/documents/${documentId}/insights`,
        { question: "Summarize the key strategic insights from this document" },
        { headers: { Authorization: `Bearer ${token}` } },
      )
      refetch()
    } catch (error) {
      toast.toast({ title: "Error: " + (error instanceof Error ? error.message : "Unknown error") })
    } finally {
      setIsGenerating(false)
    }
  }

  const suggestedQuestions = [
    "What are the key insights?",
    "Summarize the main points",
    "What are the recommendations?",
    "Identify potential risks",
  ]

  return (
    <Card className="flex flex-col h-full bg-gradient-to-br from-background to-muted/30 border-0 shadow-xl">
      <CardHeader className="border-b border-border/40 bg-gradient-to-r from-blue-50 to-purple-50 dark:from-blue-950/20 dark:to-purple-950/20">
        <div className="flex items-center justify-between">
          <CardTitle className="flex items-center gap-3">
            <div className="rounded-full bg-gradient-to-r from-blue-500 to-purple-600 p-2.5 shadow-lg">
              <Bot className="h-5 w-5 text-white" />
            </div>
            <div>
              <span className="bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
                AI Document Analysis
              </span>
              <Badge variant="secondary" className="ml-2 text-xs">
                Powered by GPT-4
              </Badge>
            </div>
          </CardTitle>
          <Button
            variant="outline"
            onClick={handleGenerateInsight}
            disabled={isGenerating}
            className="gap-2 bg-background/50 hover:bg-background transition-colors"
          >
            {isGenerating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Sparkles className="h-4 w-4" />}
            Generate Insights
          </Button>
        </div>
      </CardHeader>

      <CardContent className="flex-1 flex flex-col p-0">
        <div className="flex-1 overflow-auto p-6 space-y-6 max-h-96">
          <AnimatePresence>
            {chatHistory?.length ? (
              chatHistory.map((msg, index) => (
                <motion.div
                  key={msg.id}
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -20 }}
                  transition={{ duration: 0.3, delay: index * 0.1 }}
                  className={`flex gap-4 ${msg.type === "user" ? "justify-end" : "justify-start"}`}
                >
                  {msg.type === "assistant" && (
                    <Avatar className="h-10 w-10 mt-1 shadow-md">
                      <AvatarFallback className="bg-gradient-to-r from-blue-500 to-purple-600 text-white">
                        <Bot className="h-5 w-5" />
                      </AvatarFallback>
                    </Avatar>
                  )}
                  <div
                    className={`max-w-[80%] rounded-2xl px-6 py-4 shadow-sm ${
                      msg.type === "user"
                        ? "bg-gradient-to-r from-blue-600 to-purple-600 text-white ml-12"
                        : "bg-muted/80 backdrop-blur-sm"
                    }`}
                  >
                    <p className="whitespace-pre-wrap text-sm leading-relaxed">{msg.content}</p>
                    <p className="mt-3 text-xs opacity-70">{new Date(msg.timestamp).toLocaleTimeString()}</p>
                  </div>
                  {msg.type === "user" && (
                    <Avatar className="h-10 w-10 mt-1 shadow-md">
                      <AvatarFallback className="bg-gradient-to-r from-green-500 to-emerald-500 text-white">
                        <User className="h-5 w-5" />
                      </AvatarFallback>
                    </Avatar>
                  )}
                </motion.div>
              ))
            ) : (
              <motion.div
                className="flex flex-col items-center justify-center h-full text-center py-12"
                initial={{ opacity: 0, scale: 0.9 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ duration: 0.5 }}
              >
                <div className="rounded-full bg-gradient-to-r from-blue-500 to-purple-500 p-6 mb-6 shadow-lg">
                  <MessageSquare className="h-12 w-12 text-white mx-auto" />
                </div>
                <h3 className="font-semibold mb-3 text-lg">Start analyzing your document</h3>
                <p className="text-sm text-muted-foreground max-w-sm mb-6 leading-relaxed">
                  Ask questions about your document or generate insights to get started with AI-powered analysis
                </p>

                {/* Suggested Questions */}
                <div className="grid grid-cols-2 gap-2 max-w-sm">
                  {suggestedQuestions.map((question, index) => (
                    <Button
                      key={index}
                      variant="outline"
                      size="sm"
                      onClick={() => setMessage(question)}
                      className="text-xs h-8 bg-background/50 hover:bg-background transition-colors"
                    >
                      {question}
                    </Button>
                  ))}
                </div>
              </motion.div>
            )}
          </AnimatePresence>
          <div ref={messagesEndRef} />
        </div>

        <div className="border-t border-border/40 p-6 bg-gradient-to-r from-muted/30 to-muted/50">
          <div className="flex gap-3">
            <Input
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              placeholder="Ask a question about this document..."
              onKeyDown={(e) => {
                if (e.key === "Enter" && !e.shiftKey) {
                  e.preventDefault()
                  handleSendMessage()
                }
              }}
              disabled={isGenerating}
              className="flex-1 bg-background/80 border-border/50 focus:bg-background transition-colors"
            />
            <Button
              onClick={handleSendMessage}
              disabled={!message.trim() || isGenerating}
              size="icon"
              className="shrink-0 bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 shadow-lg"
            >
              {isGenerating ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
            </Button>
          </div>

          <div className="flex items-center justify-center gap-2 mt-3 text-xs text-muted-foreground">
            <Zap className="w-3 h-3" />
            <span>AI responses are generated in real-time</span>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}