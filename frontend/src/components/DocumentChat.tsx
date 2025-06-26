"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card } from "@/components/ui/card";
import { Send } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import api from "@/lib/api";
import { useToast } from "@/components/ui/toaster";
import { auth } from "@/lib/firebase";

interface ChatMessage {
  id: string;
  type: string;
  content: string;
  timestamp: string;
}

export default function DocumentChat({ documentId }: { documentId: string }) {
  const [message, setMessage] = useState("");
  const [isGenerating, setIsGenerating] = useState(false);
  const toast = useToast();

  const { data: chatHistory, refetch } = useQuery<ChatMessage[]>({
    queryKey: ["chatHistory", documentId],
    queryFn: async () => {
      const user = auth.currentUser;
      if (!user) throw new Error("Not authenticated");
      const token = await user.getIdToken();
      const { data } = await api.get(`/api/documents/${documentId}/chat/history`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      return data;
    },
  });

  async function handleSendMessage() {
    if (!message.trim()) return;

    setIsGenerating(true);
    try {
      const user = auth.currentUser;
      if (!user) throw new Error("Not authenticated");
      const token = await user.getIdToken();
      await api.post(
        `/api/documents/${documentId}/chat`,
        { message },
        { headers: { Authorization: `Bearer ${token}` } }
      );
      setMessage("");
      refetch();
    } catch (error) {
      toast("Error: " + (error instanceof Error ? error.message : "Unknown error"));
    } finally {
      setIsGenerating(false);
    }
  }

  async function handleGenerateInsight() {
    setIsGenerating(true);
    try {
      const user = auth.currentUser;
      if (!user) throw new Error("Not authenticated");
      const token = await user.getIdToken();
      await api.post(
        `/api/documents/${documentId}/insights`,
        { question: "Summarize the key strategic insights from this document" },
        { headers: { Authorization: `Bearer ${token}` } }
      );
      refetch();
    } catch (error) {
      toast("Error: " + (error instanceof Error ? error.message : "Unknown error"));
    } finally {
      setIsGenerating(false);
    }
  }

  return (
    <Card className="p-6">
      <div className="mb-6 flex items-center justify-between">
        <h2 className="text-xl font-semibold">Document Analysis</h2>
        <Button
          variant="outline"
          onClick={handleGenerateInsight}
          disabled={isGenerating}
        >
          Generate Insights
        </Button>
      </div>

      <div className="mb-4 h-96 overflow-auto rounded-lg border p-4">
        {chatHistory?.length ? (
          <div className="space-y-4">
            {chatHistory.map((msg) => (
              <div
                key={msg.id}
                className={`flex ${msg.type === "user" ? "justify-end" : "justify-start"}`}
              >
                <div
                  className={`max-w-[80%] rounded-lg p-3 ${msg.type === "user" ? "bg-blue-100" : "bg-gray-100"}`}
                >
                  <p className="whitespace-pre-wrap">{msg.content}</p>
                  <p className="mt-1 text-xs text-gray-500">
                    {new Date(msg.timestamp).toLocaleTimeString()}
                  </p>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <div className="flex h-full items-center justify-center text-gray-500">
            Start chatting with your document to analyze its content
          </div>
        )}
      </div>

      <div className="flex gap-2">
        <Input
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          placeholder="Ask a question about this document..."
          onKeyDown={(e) => {
            if (e.key === "Enter") handleSendMessage();
          }}
          disabled={isGenerating}
        />
        <Button
          onClick={handleSendMessage}
          disabled={!message.trim() || isGenerating}
        >
          <Send className="h-4 w-4" />
        </Button>
      </div>
    </Card>
  );
}